package engine

import (
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
)

type RingType string

const (
	API     RingType = "api"
	Default RingType = "default"
)

type Ring struct {
	mu         sync.RWMutex
	serverList []int32
	servers    map[int32]string
}

func newRing() Ring {
	return Ring{servers: make(map[int32]string)}
}

func hashAddr(addr string) int32 {
	h := fnv.New32a()
	h.Write([]byte(addr))
	return int32(h.Sum32())
}

func (r *Ring) addServer(addr string) error {
	hash := hashAddr(addr)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.servers[hash]; exists {
		return fmt.Errorf("server %q already in ring", addr)
	}

	r.servers[hash] = addr

	idx := sort.Search(len(r.serverList), func(i int) bool {
		return r.serverList[i] >= hash
	})
	r.serverList = append(r.serverList, 0)
	copy(r.serverList[idx+1:], r.serverList[idx:])
	r.serverList[idx] = hash

	return nil
}

func (r *Ring) removeServer(addr string) {
	hash := hashAddr(addr)

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.servers, hash)

	idx := sort.Search(len(r.serverList), func(i int) bool {
		return r.serverList[i] >= hash
	})
	if idx < len(r.serverList) && r.serverList[idx] == hash {
		r.serverList = append(r.serverList[:idx], r.serverList[idx+1:]...)
	}
}

// findServer returns the backend address for the given client hash using
// consistent hashing: walk clockwise on the ring and wrap around.
func (r *Ring) findServer(clientHash int32) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.serverList) == 0 {
		return "", fmt.Errorf("ring is empty")
	}

	idx := sort.Search(len(r.serverList), func(i int) bool {
		return r.serverList[i] >= clientHash
	})

	// Wrap around to the first node.
	if idx == len(r.serverList) {
		idx = 0
	}

	return r.servers[r.serverList[idx]], nil
}
