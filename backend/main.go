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
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"gotypemyadmin/internal/api"
	"gotypemyadmin/internal/server"
	"gotypemyadmin/internal/session"
	"gotypemyadmin/web"
)

// version is overridden at build time via -ldflags "-X main.version=…".
var version = "dev"

func main() {
	addr := flag.String("addr", envOr("GTMA_ADDR", ":8088"), "listen address")
	staticDir := flag.String("static", envOr("GTMA_STATIC", "../frontend/dist"), "directory of built frontend assets")
	sessionTTL := flag.Duration("session-ttl", 2*time.Hour, "idle lifetime of a database session")
	allowHostsRaw := flag.String("allow-hosts", envOr("GTMA_ALLOW_HOSTS", ""),
		"comma-separated allowlist of database hosts clients may connect to (empty = any)")
	tlsCert := flag.String("tls-cert", envOr("GTMA_TLS_CERT", ""), "TLS certificate file (enables HTTPS)")
	tlsKey := flag.String("tls-key", envOr("GTMA_TLS_KEY", ""), "TLS private key file (enables HTTPS)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("GoTypeMyAdmin %s (%s/%s, %s)\n", version, runtime.GOOS, runtime.GOARCH, runtime.Version())
		return
	}

	allowHosts := splitCSV(*allowHostsRaw)
	if len(allowHosts) == 0 {
		log.Printf("warning: -allow-hosts is empty; any client can make this server connect to arbitrary hosts (SSRF). Set GTMA_ALLOW_HOSTS to restrict.")
	}

	sessions := session.NewStore(*sessionTTL)
	defer sessions.Close()

	// Prefer the frontend embedded into the binary (release builds); otherwise
	// serve it from -static on disk (dev builds).
	srvCfg := server.Config{
		Addr:      *addr,
		StaticDir: *staticDir,
		API:       api.New(sessions, api.Config{AllowHosts: allowHosts}),
	}
	staticSource := *staticDir
	if dist, ok := web.Dist(); ok {
		srvCfg.StaticFS = server.SPAFromFS(dist)
		staticSource = "embedded"
	}
	srv := server.New(srvCfg)

	useTLS := *tlsCert != "" && *tlsKey != ""
	go func() {
		scheme := "http"
		if useTLS {
			scheme = "https"
		}
		log.Printf("GoTypeMyAdmin %s listening on %s%s (frontend: %s)", version, scheme+"://", *addr, staticSource)
		var err error
		if useTLS {
			err = srv.ListenAndServeTLS(*tlsCert, *tlsKey)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
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

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
