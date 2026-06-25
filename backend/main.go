// GoTypeMyAdmin — a modern Go + TypeScript reimagining of phpMyAdmin.
//
// This binary is the backend: a stateless-ish REST API that proxies MySQL /
// MariaDB connections on behalf of browser sessions, plus an embedded static
// file server for the built frontend.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gotypemyadmin/internal/api"
	"gotypemyadmin/internal/server"
	"gotypemyadmin/internal/session"
)

func main() {
	addr := flag.String("addr", envOr("GTMA_ADDR", ":8088"), "listen address")
	staticDir := flag.String("static", envOr("GTMA_STATIC", "../frontend/dist"), "directory of built frontend assets")
	sessionTTL := flag.Duration("session-ttl", 2*time.Hour, "idle lifetime of a database session")
	flag.Parse()

	sessions := session.NewStore(*sessionTTL)
	defer sessions.Close()

	apiHandler := api.New(sessions)
	srv := server.New(server.Config{
		Addr:      *addr,
		StaticDir: *staticDir,
		API:       apiHandler,
	})

	go func() {
		log.Printf("GoTypeMyAdmin listening on %s (serving static from %s)", *addr, *staticDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
