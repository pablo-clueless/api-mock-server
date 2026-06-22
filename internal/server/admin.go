package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"sort"

	"github.com/heirs/api-mock-server/internal/config"
)

type routeView struct {
	Method  string `json:"method"`
	Path    string `json:"path"`
	OpID    string `json:"operationId,omitempty"`
	Summary string `json:"summary,omitempty"`
	Status  int    `json:"successStatus"`
	Latency int    `json:"latencyMs,omitempty"`
}

func (s *Server) routeViews() []routeView {
	views := make([]routeView, 0, len(s.spec.Operations))
	for _, op := range s.spec.Operations {
		rv := routeView{
			Method:  op.Method,
			Path:    op.Path,
			OpID:    op.OpID,
			Summary: op.Summary,
			Status:  op.SuccessStatus(),
			Latency: s.cfg.LatencyMs,
		}
		if rc, ok := s.cfg.Routes[config.RouteKey(op.Method, op.Path)]; ok && rc.LatencyMs != nil {
			rv.Latency = *rc.LatencyMs
		}
		views = append(views, rv)
	}
	sort.Slice(views, func(i, j int) bool {
		if views[i].Path != views[j].Path {
			return views[i].Path < views[j].Path
		}
		return views[i].Method < views[j].Method
	})
	return views
}

// handleRoutesJSON serves the route table as JSON (for tooling/devtools).
func (s *Server) handleRoutesJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{
		"title":   s.spec.Title,
		"version": s.spec.Version,
		"routes":  s.routeViews(),
	})
}

// handleAdmin serves a minimal HTML route explorer at /__mock.
func (s *Server) handleAdmin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = adminTmpl.Execute(w, map[string]any{
		"Title":   s.spec.Title,
		"Version": s.spec.Version,
		"Routes":  s.routeViews(),
		"Seed":    s.cfg.Seed,
	})
}

var adminTmpl = template.Must(template.New("admin").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>{{.Title}} — Mock Server</title>
<style>
 body{font:14px/1.5 ui-monospace,SFMono-Regular,Menlo,monospace;margin:0;background:#0d1117;color:#c9d1d9}
 header{padding:20px 28px;border-bottom:1px solid #21262d}
 h1{margin:0;font-size:16px} .sub{color:#8b949e;font-size:12px;margin-top:4px}
 table{width:100%;border-collapse:collapse} td,th{text-align:left;padding:10px 28px;border-bottom:1px solid #161b22}
 th{color:#8b949e;font-weight:600;font-size:12px;text-transform:uppercase;letter-spacing:.04em}
 .m{display:inline-block;min-width:54px;padding:2px 8px;border-radius:5px;font-size:11px;font-weight:700;text-align:center}
 .GET{background:#0d419d;color:#cae8ff}.POST{background:#196c2e;color:#aff5b4}.PUT{background:#9e6a03;color:#ffe1a6}
 .PATCH{background:#9e6a03;color:#ffe1a6}.DELETE{background:#a40e26;color:#ffd7d5}
 a{color:#58a6ff;text-decoration:none} .path{color:#e6edf3} .sum{color:#8b949e}
</style></head><body>
<header><h1>{{.Title}} <span class="sub">v{{.Version}}</span></h1>
<div class="sub">api-mock-server · seed {{.Seed}} · append <code>?__status=4xx</code> to any route to force a response</div></header>
<table><thead><tr><th>Method</th><th>Path</th><th>Success</th><th>Latency</th><th>Summary</th></tr></thead><tbody>
{{range .Routes}}<tr>
 <td><span class="m {{.Method}}">{{.Method}}</span></td>
 <td class="path">{{.Path}}</td>
 <td>{{.Status}}</td>
 <td>{{if .Latency}}{{.Latency}}ms{{else}}—{{end}}</td>
 <td class="sum">{{.Summary}}</td>
</tr>{{end}}
</tbody></table></body></html>`))
