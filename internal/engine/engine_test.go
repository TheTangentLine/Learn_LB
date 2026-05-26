package engine

import (
	"testing"
)

// ----------------------------------------------------------------------------
// Ring tests
// ----------------------------------------------------------------------------

func TestRing_AddAndFind(t *testing.T) {
	r := newRing()
	backends := []string{"10.0.0.1:8080", "10.0.0.2:8080", "10.0.0.3:8080"}
	for _, addr := range backends {
		if err := r.addServer(addr); err != nil {
			t.Fatalf("addServer(%q) unexpected error: %v", addr, err)
		}
	}

	got, err := r.findServer(hashAddr("192.168.1.1"))
	if err != nil {
		t.Fatalf("findServer returned error: %v", err)
	}

	found := false
	for _, addr := range backends {
		if got == addr {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("findServer returned %q which is not a registered backend", got)
	}
}

func TestRing_EmptyRing(t *testing.T) {
	r := newRing()
	_, err := r.findServer(42)
	if err == nil {
		t.Fatal("expected error from empty ring, got nil")
	}
}

func TestRing_AddDuplicate(t *testing.T) {
	r := newRing()
	if err := r.addServer("10.0.0.1:8080"); err != nil {
		t.Fatalf("first addServer failed: %v", err)
	}
	if err := r.addServer("10.0.0.1:8080"); err == nil {
		t.Fatal("expected error on duplicate addServer, got nil")
	}
}

func TestRing_Remove(t *testing.T) {
	r := newRing()
	addr := "10.0.0.1:8080"
	if err := r.addServer(addr); err != nil {
		t.Fatalf("addServer failed: %v", err)
	}
	r.removeServer(addr)

	_, err := r.findServer(hashAddr("client"))
	if err == nil {
		t.Fatal("expected error after removing the only server, got nil")
	}
}

func TestRing_WrapAround(t *testing.T) {
	r := newRing()
	// Use a known address whose hash we can control indirectly: just add one
	// server and verify that any client hash (including math.MaxInt32) still
	// resolves to it via wrap-around.
	addr := "10.0.0.1:8080"
	if err := r.addServer(addr); err != nil {
		t.Fatalf("addServer failed: %v", err)
	}

	// A client hash of math.MaxInt32 is very likely greater than the single
	// server hash, triggering the wrap-around path.
	got, err := r.findServer(int32(^uint32(0) >> 1)) // math.MaxInt32
	if err != nil {
		t.Fatalf("findServer error: %v", err)
	}
	if got != addr {
		t.Errorf("wrap-around: expected %q, got %q", addr, got)
	}
}

// ----------------------------------------------------------------------------
// Parser tests
// ----------------------------------------------------------------------------

func TestParser_APIPrefix(t *testing.T) {
	p := Parser{}
	if got := p.parsing("/api/users"); got != API {
		t.Errorf("expected API, got %q", got)
	}
}

func TestParser_DefaultFallback(t *testing.T) {
	p := Parser{}
	if got := p.parsing("/other/path"); got != Default {
		t.Errorf("expected Default, got %q", got)
	}
}

// ----------------------------------------------------------------------------
// Orchestrator tests
// ----------------------------------------------------------------------------

func TestOrchestrator_RouteToAPI(t *testing.T) {
	o := NewOrchestrator()
	backend := "10.0.0.1:8080"
	if err := o.AddBackend(API, backend); err != nil {
		t.Fatalf("AddBackend failed: %v", err)
	}

	got, err := o.Route("/api/users", "192.168.1.100")
	if err != nil {
		t.Fatalf("Route returned error: %v", err)
	}
	if got != backend {
		t.Errorf("expected %q, got %q", backend, got)
	}
}

func TestOrchestrator_FallbackToDefault(t *testing.T) {
	o := NewOrchestrator()
	backend := "10.0.0.9:8080"
	if err := o.AddBackend(Default, backend); err != nil {
		t.Fatalf("AddBackend(Default) failed: %v", err)
	}

	// /api/x → parser resolves to API, but no API ring → falls back to Default.
	got, err := o.Route("/api/x", "192.168.1.100")
	if err != nil {
		t.Fatalf("Route returned error: %v", err)
	}
	if got != backend {
		t.Errorf("expected fallback to %q, got %q", backend, got)
	}
}

func TestOrchestrator_NoBackends(t *testing.T) {
	o := NewOrchestrator()
	_, err := o.Route("/api/users", "192.168.1.100")
	if err == nil {
		t.Fatal("expected error when no backends are registered, got nil")
	}
}

func TestOrchestrator_RemoveBackend_UnknownRing(t *testing.T) {
	o := NewOrchestrator()
	if err := o.RemoveBackend("nonexistent", "10.0.0.1:8080"); err == nil {
		t.Fatal("expected error removing backend from unknown ring, got nil")
	}
}
