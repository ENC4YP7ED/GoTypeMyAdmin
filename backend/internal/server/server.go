// Package server wires the HTTP listener: API routes plus a SPA-aware static
// file server for the built frontend.
package server

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Config holds the knobs main() passes in.
type Config struct {
	Addr      string
	StaticDir string
	// StaticFS, when non-nil, serves the frontend from an embedded filesystem
	// instead of StaticDir on disk (used by self-contained release binaries).
	StaticFS http.Handler
	API      http.Handler
}

// New builds an *http.Server ready to ListenAndServe.
func New(cfg Config) *http.Server {
	mux := http.NewServeMux()

	// Everything under /api is the JSON backend.
	mux.Handle("/api/", http.StripPrefix("/api", cfg.API))

	// Everything else is the single-page app: embedded FS if present, else disk.
	if cfg.StaticFS != nil {
		mux.Handle("/", cfg.StaticFS)
	} else {
		mux.Handle("/", spaHandler(cfg.StaticDir))
	}

	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           secureHeaders(mux),
		ReadHeaderTimeout: 15 * time.Second,
	}
}

// SPAFromFS builds a SPA-aware handler over an embedded filesystem, falling
// back to index.html for unknown non-asset paths so deep links work.
func SPAFromFS(fsys fs.FS) http.Handler {
	fileServer := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(fsys, p); err != nil && !strings.HasPrefix(r.URL.Path, "/assets") {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

// secureHeaders applies defense-in-depth response headers. The CSP is strict —
// the production bundle has no inline scripts/styles, only same-origin assets —
// which neutralizes whole classes of reflected/stored XSS even if a sink slips
// through.
func secureHeaders(next http.Handler) http.Handler {
	// font-src/img-src allow data: because Vite inlines small font/image assets
	// as data URIs; these are inert and cannot execute script.
	const csp = "default-src 'self'; script-src 'self'; style-src 'self'; " +
		"font-src 'self' data:; img-src 'self' data:; connect-src 'self'; " +
		"object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", csp)
		next.ServeHTTP(w, r)
	})
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
