// Package api is the HTTP adapter: it speaks OGC API - Features on the
// outside and core.FeatureStore on the inside. It imports core and
// nothing else — not the storage adapters, not the catalog, not the
// config format. Swapping a backend, or changing where the catalog comes
// from, never touches this package.
package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/VallarisMapsPlatforms/meros/internal/core"
)

// Server serves the OGC API - Features read path (Part 1).
type Server struct {
	cols  []core.Collection          // advertised order, as declared upstream
	byID  map[string]core.Collection // same values, keyed for lookup
	store core.FeatureStore
	log   *slog.Logger
	title string
	desc  string
}

// Option configures a Server.
type Option func(*Server)

// WithLogger sets the request logger.
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) { s.log = l }
}

// WithServiceInfo sets the landing-page title and description.
func WithServiceInfo(title, description string) Option {
	return func(s *Server) { s.title, s.desc = title, description }
}

// New builds the API over the collections it should advertise and the
// FeatureStore that answers them. Both arrive as neutral core types: the
// composition root turns configuration into domain values, so this
// package never learns the catalog's format or where it came from.
func New(cols []core.Collection, store core.FeatureStore, opts ...Option) *Server {
	byID := make(map[string]core.Collection, len(cols))
	for _, c := range cols {
		byID[c.ID] = c
	}
	s := &Server{
		cols:  cols,
		byID:  byID,
		store: store,
		log:   slog.Default(),
		title: "Meros",
		desc:  "A composable, read-only OGC API - Features server.",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Handler returns the routed HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.landing)
	mux.HandleFunc("GET /api", s.apiDefinition)
	mux.HandleFunc("GET /conformance", s.conformance)
	mux.HandleFunc("GET /collections", s.collections)
	mux.HandleFunc("GET /collections/{id}", s.collection)
	mux.HandleFunc("GET /collections/{id}/items", s.items)
	mux.HandleFunc("GET /collections/{id}/items/{fid}", s.item)
	return s.withLogging(withCORS(mux))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.log.Info("request",
			"method", r.Method,
			"path", r.URL.RequestURI(),
			"status", rec.status,
			"duration", time.Since(start).Round(time.Millisecond).String(),
		)
	})
}
