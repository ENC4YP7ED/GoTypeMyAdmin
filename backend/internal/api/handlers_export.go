package api

import (
	"fmt"
	"net/http"

	"gotypemyadmin/internal/db"
)

func (a *API) handleExportTable(w http.ResponseWriter, r *http.Request) {
	schema := r.PathValue("db")
	table := r.PathValue("table")
	format := db.ExportFormat(r.URL.Query().Get("format"))
	if format == "" {
		format = db.FormatSQL
	}

	body, contentType, err := db.ExportTable(r.Context(), sess(r).DB, schema, table, format)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	serveDownload(w, contentType, fmt.Sprintf("%s.%s.%s", schema, table, format), body)
}

func (a *API) handleExportDatabase(w http.ResponseWriter, r *http.Request) {
	schema := r.PathValue("db")
	body, err := db.ExportDatabase(r.Context(), sess(r).DB, schema)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	serveDownload(w, "application/sql", schema+".sql", body)
}

func serveDownload(w http.ResponseWriter, contentType, filename, body string) {
	w.Header().Set("Content-Type", contentType+"; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

// ---- import ----------------------------------------------------------------

type importReq struct {
	Database string `json:"database"`
	SQL      string `json:"sql"`
}

func (a *API) handleImport(w http.ResponseWriter, r *http.Request) {
	var req importReq
	if err := decode(r, &req); err != nil || req.SQL == "" {
		writeErr(w, http.StatusBadRequest, "sql script required")
		return
	}
	res, err := db.RunScript(r.Context(), sess(r).DB, req.Database, req.SQL)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": res})
}
