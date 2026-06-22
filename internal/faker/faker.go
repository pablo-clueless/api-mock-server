// Package faker turns an OpenAPI schema into *realistic* JSON data, not just
// schema-valid noise. Resolution order per field:
//
//	example/examples -> enum -> format -> field-name heuristic -> type fallback
//
// A per-request *rand.Rand is threaded through so output is deterministic for
// a given (seed, request) — the frontend then sees stable data across polls.
package faker

import (
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// Faker generates values. It is stateless apart from the supplied rand source.
type Faker struct {
	r *rand.Rand
}

// New returns a Faker backed by the given deterministic rand source.
func New(r *rand.Rand) *Faker { return &Faker{r: r} }

// maxDepth guards against recursive/self-referential schemas.
const maxDepth = 8

// Value generates a realistic value for schema. fieldName is the property name
// in the parent object ("" at the root) and drives the heuristic layer.
func (f *Faker) Value(schema *openapi3.Schema, fieldName string) any {
	return f.value(schema, fieldName, 0)
}

func (f *Faker) value(schema *openapi3.Schema, name string, depth int) any {
	if schema == nil || depth > maxDepth {
		return nil
	}

	// 1. Honour spec-provided examples first — author intent wins.
	if schema.Example != nil {
		return schema.Example
	}

	// 2. Vendor extension override: x-mock picks a named generator.
	if gen, ok := schema.Extensions["x-mock"].(string); ok {
		if v, ok := f.byGenerator(gen); ok {
			return v
		}
	}

	// 3. Enum — pick one deterministically.
	if len(schema.Enum) > 0 {
		return schema.Enum[f.r.Intn(len(schema.Enum))]
	}

	t := schemaType(schema)

	// Composition: allOf merges, oneOf/anyOf pick a branch.
	if v, ok := f.composition(schema, name, depth); ok {
		return v
	}

	switch t {
	case "object", "":
		if len(schema.Properties) > 0 || t == "object" {
			return f.object(schema, depth)
		}
		// Untyped schema with no properties: best-effort string.
		return f.heuristic(name, "")
	case "array":
		return f.array(schema, name, depth)
	case "boolean":
		return f.boolean(name)
	case "integer":
		return f.integer(schema, name)
	case "number":
		return f.number(schema, name)
	case "string":
		return f.str(schema, name)
	default:
		return nil
	}
}

func (f *Faker) object(schema *openapi3.Schema, depth int) map[string]any {
	out := map[string]any{}
	// Generate in sorted key order: Go map iteration is randomized, and the
	// rand stream must be consumed deterministically for stable responses.
	names := make([]string, 0, len(schema.Properties))
	for propName := range schema.Properties {
		names = append(names, propName)
	}
	sort.Strings(names)
	for _, propName := range names {
		ref := schema.Properties[propName]
		if ref == nil || ref.Value == nil {
			continue
		}
		out[propName] = f.value(ref.Value, propName, depth+1)
	}
	return out
}

func (f *Faker) array(schema *openapi3.Schema, name string, depth int) []any {
	n := 3
	if schema.MinItems > 0 {
		n = int(schema.MinItems)
	}
	if schema.MaxItems != nil && n > int(*schema.MaxItems) {
		n = int(*schema.MaxItems)
	}
	if n < 1 {
		n = 1
	}
	if n > 10 {
		n = 10
	}
	out := make([]any, 0, n)
	if schema.Items == nil || schema.Items.Value == nil {
		return out
	}
	// Singularize the field name so item heuristics fire ("tags" -> "tag").
	itemName := strings.TrimSuffix(name, "s")
	for i := 0; i < n; i++ {
		out = append(out, f.value(schema.Items.Value, itemName, depth+1))
	}
	return out
}

func (f *Faker) composition(schema *openapi3.Schema, name string, depth int) (any, bool) {
	if len(schema.AllOf) > 0 {
		merged := map[string]any{}
		for _, ref := range schema.AllOf {
			if ref == nil || ref.Value == nil {
				continue
			}
			if obj, ok := f.value(ref.Value, name, depth).(map[string]any); ok {
				for k, v := range obj {
					merged[k] = v
				}
			}
		}
		return merged, true
	}
	if branches := pickFirst(schema.OneOf, schema.AnyOf); len(branches) > 0 {
		choice := branches[f.r.Intn(len(branches))]
		if choice != nil && choice.Value != nil {
			return f.value(choice.Value, name, depth), true
		}
	}
	return nil, false
}

func (f *Faker) str(schema *openapi3.Schema, name string) any {
	// format takes precedence over name heuristics.
	if v, ok := f.byFormat(schema.Format); ok {
		return v
	}
	return f.heuristic(name, schema.Format)
}

func (f *Faker) integer(schema *openapi3.Schema, name string) any {
	if v, ok := f.heuristicInt(name); ok {
		return v
	}
	lo, hi := 1, 1000
	if schema.Min != nil {
		lo = int(*schema.Min)
	}
	if schema.Max != nil {
		hi = int(*schema.Max)
	}
	if hi <= lo {
		hi = lo + 1000
	}
	return lo + f.r.Intn(hi-lo)
}

func (f *Faker) number(schema *openapi3.Schema, name string) any {
	if isMoney(name) {
		return f.money()
	}
	lo, hi := 0.0, 1000.0
	if schema.Min != nil {
		lo = *schema.Min
	}
	if schema.Max != nil {
		hi = *schema.Max
	}
	if hi <= lo {
		hi = lo + 1000
	}
	return round2(lo + f.r.Float64()*(hi-lo))
}

func (f *Faker) boolean(name string) bool {
	return f.r.Intn(2) == 0
}

func pickFirst(a, b openapi3.SchemaRefs) openapi3.SchemaRefs {
	if len(a) > 0 {
		return a
	}
	return b
}

func round2(v float64) float64 {
	return float64(int(v*100)) / 100
}

func (f *Faker) money() float64 {
	return round2(1 + f.r.Float64()*9999)
}

// now is overridable in tests; deterministic-enough for mock data.
var now = time.Now

func (f *Faker) recentDate() string {
	// Anchor to the start of the current UTC day so the value is fully
	// deterministic for a given (seed, request) within a run — wall-clock
	// sub-day drift would otherwise make "identical" requests differ.
	base := now().UTC().Truncate(24 * time.Hour)
	d := base.Add(-time.Duration(f.r.Intn(365*24)) * time.Hour)
	return d.Format(time.RFC3339)
}
