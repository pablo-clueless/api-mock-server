// Package spec loads and normalizes an OpenAPI 3.x document into a small
// internal model (Operations) that both the mock server and the future
// codegen emitter can consume without re-walking kin-openapi types.
package spec

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Response is one declared response for an operation, keyed by status.
type Response struct {
	Status int               // numeric HTTP status (default -> 0)
	Schema *openapi3.Schema  // JSON body schema, may be nil (no content)
	Media  string            // media type the schema came from
}

// Operation is a single method+path the server can serve.
type Operation struct {
	Method    string             // GET, POST, ...
	Path      string             // OpenAPI template path, e.g. /pets/{petId}
	OpID      string             // operationId if present
	Summary   string             // human description for the admin view
	Segments  []string           // pre-split path segments for matching
	Responses map[int]Response   // status -> response (0 == "default")
	// successStatus is the preferred status to return when nothing forces one.
	successStatus int
}

// Spec is the normalized document.
type Spec struct {
	Title      string
	Version    string
	Operations []Operation
	Doc        *openapi3.T // retained for codegen / schema component access
}

var methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE"}

// Load parses and resolves the spec at path.
func Load(path string) (*Spec, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading spec %s: %w", path, err)
	}
	// Validate but don't hard-fail on minor issues — mock servers are often
	// pointed at drafts. We surface validation only via the error path here.
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("invalid spec %s: %w", path, err)
	}

	s := &Spec{Doc: doc}
	if doc.Info != nil {
		s.Title = doc.Info.Title
		s.Version = doc.Info.Version
	}

	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			s.Operations = append(s.Operations, normalizeOp(method, path, op))
		}
	}

	// Stable, literal-segments-first ordering so concrete routes like
	// /pets/health win over templated /pets/{petId} during matching.
	sort.Slice(s.Operations, func(i, j int) bool {
		a, b := s.Operations[i], s.Operations[j]
		if pa, pb := paramCount(a.Segments), paramCount(b.Segments); pa != pb {
			return pa < pb
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Method < b.Method
	})

	return s, nil
}

func normalizeOp(method, path string, op *openapi3.Operation) Operation {
	o := Operation{
		Method:    method,
		Path:      path,
		OpID:      op.OperationID,
		Summary:   op.Summary,
		Segments:  splitPath(path),
		Responses: map[int]Response{},
	}

	best := -1
	if op.Responses != nil {
		for code, ref := range op.Responses.Map() {
			status := parseStatus(code)
			r := Response{Status: status}
			if ref.Value != nil {
				if mt := ref.Value.Content.Get("application/json"); mt != nil && mt.Schema != nil {
					r.Schema = mt.Schema.Value
					r.Media = "application/json"
				} else {
					// fall back to the first declared media type with a schema
					for media, m := range ref.Value.Content {
						if m.Schema != nil {
							r.Schema = m.Schema.Value
							r.Media = media
							break
						}
					}
				}
			}
			o.Responses[status] = r

			// Prefer the lowest 2xx as the default success response.
			if status >= 200 && status < 300 && (best == -1 || status < best) {
				best = status
			}
		}
	}
	if best == -1 {
		best = 200 // spec declared no 2xx; synthesize one
	}
	o.successStatus = best
	return o
}

// SuccessStatus is the status returned when the request doesn't force one.
func (o Operation) SuccessStatus() int { return o.successStatus }

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func paramCount(segs []string) int {
	n := 0
	for _, s := range segs {
		if strings.HasPrefix(s, "{") {
			n++
		}
	}
	return n
}

func parseStatus(code string) int {
	if code == "default" {
		return 0
	}
	var n int
	if _, err := fmt.Sscanf(code, "%d", &n); err != nil {
		return 0
	}
	return n
}
