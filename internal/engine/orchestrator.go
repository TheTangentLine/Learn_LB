package engine

import (
	"fmt"
	"sync"
)

type Orchestrator struct {
	mu       sync.RWMutex
	ringList map[RingType]*Ring
	parser   Parser
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		ringList: make(map[RingType]*Ring),
	}
}

func (o *Orchestrator) find(ringType RingType) (*Ring, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	ring, ok := o.ringList[ringType]
	return ring, ok
}

// Route resolves the backend address for a given request URL and client IP.
// It parses the URL to determine the target ring, then uses consistent hashing
// on the client IP to pick a backend server.
func (o *Orchestrator) Route(url, clientIP string) (string, error) {
	ringType := o.parser.parsing(url)

	ring, ok := o.find(ringType)
	if !ok {
		// Fall back to the default ring before giving up.
		if ringType != Default {
			ring, ok = o.find(Default)
		}
		if !ok {
			return "", fmt.Errorf("no backends registered for ring %q", ringType)
		}
	}

	return ring.findServer(hashAddr(clientIP))
}

// AddBackend registers addr in the ring for ringType, creating the ring if needed.
func (o *Orchestrator) AddBackend(ringType RingType, addr string) error {
	o.mu.Lock()
	if _, ok := o.ringList[ringType]; !ok {
		r := newRing()
		o.ringList[ringType] = &r
	}
	ring := o.ringList[ringType]
	o.mu.Unlock()

	return ring.addServer(addr)
}

// RemoveBackend deregisters addr from the ring for ringType.
func (o *Orchestrator) RemoveBackend(ringType RingType, addr string) error {
	ring, ok := o.find(ringType)
	if !ok {
		return fmt.Errorf("ring %q not found", ringType)
	}
	ring.removeServer(addr)
	return nil
}
