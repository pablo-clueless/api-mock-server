// Package config loads the JSON-driven mock server configuration.
//
// The whole tool is project-agnostic: behaviour is controlled by an
// OpenAPI spec (the contract) plus a mockserver.json (the runtime knobs).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// RouteConfig overrides behaviour for a single "METHOD /path" route.
// Pointers are used so "unset" is distinguishable from a zero value and
// falls back to the global default.
type RouteConfig struct {
	// LatencyMs simulates network/backend latency before responding.
	LatencyMs *int `json:"latencyMs,omitempty"`
	// ErrorRate (0.0-1.0) injects a random failure with ErrorStatus.
	ErrorRate *float64 `json:"errorRate,omitempty"`
	// ErrorStatus is the status returned when an injected error fires.
	ErrorStatus *int `json:"errorStatus,omitempty"`
	// Status forces every response on this route to a specific status code.
	Status *int `json:"status,omitempty"`
}

// Config is the parsed mockserver.json plus any CLI overrides.
type Config struct {
	// Spec is the path to the OpenAPI 3.x document driving the routes.
	Spec string `json:"spec"`
	// Port the mock server listens on.
	Port int `json:"port"`
	// Seed makes faked data deterministic across restarts.
	Seed int64 `json:"seed"`
	// CORS enables permissive CORS headers (handy for browser frontends).
	CORS bool `json:"cors"`
	// LatencyMs is the global default latency applied to every route.
	LatencyMs int `json:"latencyMs"`
	// Routes holds per-route overrides keyed by "METHOD /path", e.g.
	// "GET /pets" or "POST /pets/{petId}".
	Routes map[string]RouteConfig `json:"routes"`
}

// Default returns a config with sensible zero-config defaults.
func Default() Config {
	return Config{
		Port:   4010,
		Seed:   42,
		CORS:   true,
		Routes: map[string]RouteConfig{},
	}
}

// Load reads a mockserver.json file. A missing file is not an error — the
// tool is usable with only a --spec flag, so we return defaults instead.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", path, err)
	}
	if cfg.Routes == nil {
		cfg.Routes = map[string]RouteConfig{}
	}
	return cfg, nil
}

// RouteKey builds the lookup key used in Config.Routes.
func RouteKey(method, path string) string {
	return strings.ToUpper(method) + " " + path
}
