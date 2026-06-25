// Package server wires the HTTP listener: API routes plus a SPA-aware static
// file server for the built frontend.
package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds the knobs main() passes in.
type Config struct {
	Addr      string
	StaticDir string
	API       http.Handler
}

// New builds an *http.Server ready to ListenAndServe.
func New(cfg Config) *http.Server {
	mux := http.NewServeMux()

	// Everything under /api is the JSON backend.
	mux.Handle("/api/", http.StripPrefix("/api", cfg.API))

	// Everything else is the single-page app.
	mux.Handle("/", spaHandler(cfg.StaticDir))

	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           logging(mux),
		ReadHeaderTimeout: 15 * time.Second,
	}
}

// spaHandler serves files from dir, falling back to index.html so client-side
// routing works on deep links.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(r.URL.Path)
		full := filepath.Join(dir, clean)
		if info, err := os.Stat(full); err != nil || info.IsDir() {
			if !strings.HasPrefix(clean, "/assets") {
				r.URL.Path = "/"
			}
		}
		fs.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
