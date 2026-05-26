package engine

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Listener struct {
	orchestrator *Orchestrator
	server       *http.Server
}

func NewListener(orchestrator *Orchestrator) *Listener {
	l := &Listener{orchestrator: orchestrator}
	l.server = &http.Server{Handler: l}
	return l
}

func (l *Listener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		clientIP = r.RemoteAddr
	}

	backendAddr, err := l.orchestrator.Route(r.URL.Path, clientIP)
	if err != nil {
		http.Error(w, fmt.Sprintf("no backend available: %v", err), http.StatusServiceUnavailable)
		return
	}

	target, err := url.Parse("http://" + backendAddr)
	if err != nil {
		http.Error(w, "invalid backend address", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}

func (l *Listener) Start(addr string) error {
	l.server.Addr = addr
	return l.server.ListenAndServe()
}

func (l *Listener) Shutdown(ctx context.Context) error {
	return l.server.Shutdown(ctx)
}
