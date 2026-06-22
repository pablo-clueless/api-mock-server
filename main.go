// Command mock is a project-agnostic OpenAPI mock server: point it at a spec
// and it serves realistic, deterministic mock data with config-driven latency,
// error injection, and on-demand status control.
//
// Usage:
//
//	mock serve [--config mockserver.json] [--spec api.yaml] [--port 4010]
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/heirs/api-mock-server/internal/config"
	"github.com/heirs/api-mock-server/internal/spec"
	"github.com/heirs/api-mock-server/internal/server"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		serveCmd(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func serveCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	cfgPath := fs.String("config", "mockserver.json", "path to mockserver.json (optional)")
	specPath := fs.String("spec", "", "path to OpenAPI 3.x spec (overrides config)")
	port := fs.Int("port", 0, "listen port (overrides config)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if *specPath != "" {
		cfg.Spec = *specPath
	}
	if *port != 0 {
		cfg.Port = *port
	}
	if cfg.Spec == "" {
		log.Fatal("no spec: pass --spec or set \"spec\" in mockserver.json")
	}

	sp, err := spec.Load(cfg.Spec)
	if err != nil {
		log.Fatalf("spec: %v", err)
	}

	srv := server.New(sp, cfg)
	addr := fmt.Sprintf(":%d", cfg.Port)

	log.Printf("api-mock-server · %s v%s", sp.Title, sp.Version)
	log.Printf("spec   %s (%d routes)", cfg.Spec, len(sp.Operations))
	log.Printf("listen http://localhost%s", addr)
	log.Printf("admin  http://localhost%s/__mock", addr)

	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `api-mock-server — realistic OpenAPI mocks

Usage:
  mock serve [flags]

Flags:
  --config   path to mockserver.json (default "mockserver.json", optional)
  --spec     path to OpenAPI 3.x spec (overrides config)
  --port     listen port (overrides config)

Examples:
  mock serve --spec examples/petstore.yaml --port 4010
  mock serve --config mockserver.json

Per-request controls:
  ?__status=404            force a specific response status
  X-Mock-Status: 500       same, via header
`)
}
