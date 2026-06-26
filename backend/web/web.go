// Package web optionally embeds the built frontend so release binaries are
// fully self-contained (one file per platform, no dist/ to ship alongside).
//
// Plain `go build` only compiles in the placeholder under dist/, so Dist()
// reports false and the server falls back to serving -static from disk. The
// release script copies frontend/dist into ./dist before building, which makes
// Dist() return the real single-page app.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

// Dist returns the embedded frontend filesystem and true when a real build was
// compiled in (detected by the presence of index.html).
func Dist() (fs.FS, bool) {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}
