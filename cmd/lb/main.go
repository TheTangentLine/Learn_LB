package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thetangentline/lb/internal/engine"
)

func main() {
	listenAddr := flag.String("listen", ":8080", "address the load balancer listens on")
	adminAddr := flag.String("admin", ":9090", "address the admin API listens on")
	backendsFlag := flag.String("backends", "", "comma-separated list of ring:addr pairs, e.g. api:10.0.0.1:8081,default:10.0.0.2:8082")
	flag.Parse()

	orchestrator := engine.NewOrchestrator()

	if *backendsFlag != "" {
		for _, entry := range strings.Split(*backendsFlag, ",") {
			entry = strings.TrimSpace(entry)
			// Format: ring:host:port — split on the first colon only.
			idx := strings.Index(entry, ":")
			if idx < 0 {
				log.Fatalf("invalid backend entry %q: expected ring:addr", entry)
			}
			ringType := engine.RingType(entry[:idx])
			addr := entry[idx+1:]
			if err := orchestrator.AddBackend(ringType, addr); err != nil {
				log.Fatalf("failed to add backend %q to ring %q: %v", addr, ringType, err)
			}
			log.Printf("registered backend %q in ring %q", addr, ringType)
		}
	}

	listener := engine.NewListener(orchestrator)
	adminServer := engine.NewAdminServer(orchestrator)

	// Start load balancer.
	go func() {
		log.Printf("load balancer listening on %s", *listenAddr)
		if err := listener.Start(*listenAddr); err != nil {
			log.Printf("listener stopped: %v", err)
		}
	}()

	// Start admin API.
	go func() {
		log.Printf("admin API listening on %s", *adminAddr)
		if err := adminServer.Start(*adminAddr); err != nil {
			log.Printf("admin server stopped: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println()
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := listener.Shutdown(ctx); err != nil {
		log.Printf("listener shutdown error: %v", err)
	}
	if err := adminServer.Shutdown(ctx); err != nil {
		log.Printf("admin server shutdown error: %v", err)
	}

	log.Println("shutdown complete")
}
