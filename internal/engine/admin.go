package engine

import (
	"context"
	"encoding/json"
	"net/http"
)

type AdminServer struct {
	orchestrator *Orchestrator
	server       *http.Server
}

type backendRequest struct {
	Ring RingType `json:"ring"`
	Addr string   `json:"addr"`
}

func NewAdminServer(orchestrator *Orchestrator) *AdminServer {
	a := &AdminServer{orchestrator: orchestrator}

	mux := http.NewServeMux()
	mux.HandleFunc("/admin/backends", a.handleBackends)

	a.server = &http.Server{Handler: mux}
	return a
}

func (a *AdminServer) handleBackends(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		a.addBackend(w, r)
	case http.MethodDelete:
		a.removeBackend(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *AdminServer) addBackend(w http.ResponseWriter, r *http.Request) {
	var req backendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Ring == "" || req.Addr == "" {
		http.Error(w, `"ring" and "addr" are required`, http.StatusBadRequest)
		return
	}

	if err := a.orchestrator.AddBackend(req.Ring, req.Addr); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (a *AdminServer) removeBackend(w http.ResponseWriter, r *http.Request) {
	var req backendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Ring == "" || req.Addr == "" {
		http.Error(w, `"ring" and "addr" are required`, http.StatusBadRequest)
		return
	}

	if err := a.orchestrator.RemoveBackend(req.Ring, req.Addr); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *AdminServer) Start(addr string) error {
	a.server.Addr = addr
	return a.server.ListenAndServe()
}

func (a *AdminServer) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
