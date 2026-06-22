// Package server turns a normalized spec + config into a live HTTP mock.
//
// Per request it: matches a route, applies config (forced status, error
// injection, latency), selects a response (honouring on-demand status
// control), and fakes a deterministic realistic body.
package server

import (
	"encoding/json"
	"hash/fnv"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/heirs/api-mock-server/internal/config"
	"github.com/heirs/api-mock-server/internal/faker"
	"github.com/heirs/api-mock-server/internal/spec"
)

// Server holds everything needed to serve mock responses.
type Server struct {
	spec *spec.Spec
	cfg  config.Config
}

// New builds a Server.
func New(s *spec.Spec, cfg config.Config) *Server {
	return &Server{spec: s, cfg: cfg}
}

// Handler returns the root http.Handler (admin routes + mock dispatch).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/__mock", s.handleAdmin)
	mux.HandleFunc("/__mock/routes", s.handleRoutesJSON)
	mux.HandleFunc("/", s.dispatch)
	return s.withMiddleware(mux)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.CORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// dispatch matches and serves a mock for a spec-defined route.
func (s *Server) dispatch(w http.ResponseWriter, r *http.Request) {
	op, params, ok := s.match(r.Method, r.URL.Path)
	if !ok {
		s.writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "no mock route matches " + r.Method + " " + r.URL.Path,
			"hint":  "GET /__mock to list available routes",
		})
		return
	}

	key := config.RouteKey(op.Method, op.Path)
	rc := s.cfg.Routes[key]

	// Latency: route override falls back to the global default.
	latency := s.cfg.LatencyMs
	if rc.LatencyMs != nil {
		latency = *rc.LatencyMs
	}
	if latency > 0 {
		time.Sleep(time.Duration(latency) * time.Millisecond)
	}

	// Deterministic rand seeded by config seed + the concrete request, so the
	// same resource id returns stable data but different ids differ.
	rnd := rand.New(rand.NewSource(s.seed(r.Method, r.URL.Path)))

	// Decide which status to return (precedence: client override -> error
	// injection -> forced route status -> declared success).
	status := s.resolveStatus(op, rc, r, rnd)

	resp, found := op.Responses[status]
	if !found {
		// No schema declared for this status — synthesize a plausible body.
		s.writeJSON(w, status, synthBody(status))
		return
	}
	if resp.Schema == nil {
		w.WriteHeader(status)
		return
	}

	body := faker.New(rnd).Value(resp.Schema, lastSegment(op.Path))
	_ = params // reserved: future stateful mode echoes path params into the body
	s.writeJSON(w, status, body)
}

// resolveStatus implements the response-selection precedence.
func (s *Server) resolveStatus(op spec.Operation, rc config.RouteConfig, r *http.Request, rnd *rand.Rand) int {
	// 1. On-demand client override: ?__status=NNN or X-Mock-Status header.
	if v := r.URL.Query().Get("__status"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	if v := r.Header.Get("X-Mock-Status"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	// 2. Injected error.
	if rc.ErrorRate != nil && *rc.ErrorRate > 0 && rnd.Float64() < *rc.ErrorRate {
		if rc.ErrorStatus != nil {
			return *rc.ErrorStatus
		}
		return http.StatusInternalServerError
	}
	// 3. Config-forced status for the route.
	if rc.Status != nil {
		return *rc.Status
	}
	// 4. Declared success status.
	return op.SuccessStatus()
}

// match finds the operation for a method+path and extracts path params.
func (s *Server) match(method, path string) (spec.Operation, map[string]string, bool) {
	reqSegs := splitPath(path)
	for _, op := range s.spec.Operations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		if params, ok := matchSegments(op.Segments, reqSegs); ok {
			return op, params, true
		}
	}
	return spec.Operation{}, nil, false
}

func matchSegments(tmpl, req []string) (map[string]string, bool) {
	if len(tmpl) != len(req) {
		return nil, false
	}
	params := map[string]string{}
	for i, seg := range tmpl {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			params[strings.Trim(seg, "{}")] = req[i]
			continue
		}
		if seg != req[i] {
			return nil, false
		}
	}
	return params, true
}

func (s *Server) seed(method, path string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(method + " " + path))
	return int64(h.Sum64()) ^ s.cfg.Seed
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Mock-Server", "api-mock-server")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(body); err != nil {
		log.Printf("encode error: %v", err)
	}
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func lastSegment(p string) string {
	segs := splitPath(p)
	for i := len(segs) - 1; i >= 0; i-- {
		if !strings.HasPrefix(segs[i], "{") {
			return strings.TrimSuffix(segs[i], "s")
		}
	}
	return ""
}

// synthBody builds a generic body for a status with no declared schema.
func synthBody(status int) map[string]any {
	if status >= 200 && status < 300 {
		return map[string]any{"ok": true, "status": status}
	}
	return map[string]any{
		"error":  http.StatusText(status),
		"status": status,
	}
}
