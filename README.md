# api-mock-server

A project-agnostic OpenAPI mock server. Point it at an OpenAPI 3.x spec and it
serves **realistic, deterministic** mock data — with config-driven latency,
error injection, and on-demand status control. Built in Go (single binary, no
runtime deps).

> A part of the suite of internal dev toolkit. **v1 = mock server.** Multi-language model
> codegen (js/ts/py/rs/go/c#) is the planned v2 and reuses the same spec loader.

## Quick start

```sh
go build -o bin/mock.exe .
./bin/mock.exe serve --spec examples/petstore.yaml --port 4010
```

Then:

- `GET http://localhost:4010/pets` → realistic list of pets
- `http://localhost:4010/__mock` → HTML route explorer
- `http://localhost:4010/__mock/routes` → route table as JSON

## Why it's useful

**Realistic, not random.** A field named `email` returns an email, `createdAt`
a recent timestamp, `price` money, `phone` a phone number, enums are respected,
`format: uuid` yields a UUID. Resolution order per field:

```
example/examples  ->  x-mock ext  ->  enum  ->  format  ->  name heuristic  ->  type fallback
```

**Deterministic.** Data is seeded by `(seed, method, path)`, so the same
resource id returns the same data across restarts (no UI flicker on poll) while
different ids differ. Change `seed` in config to reshuffle everything.

**On-demand status control** (great for building error states):

```sh
curl 'localhost:4010/pets/123?__status=404'      # force any status
curl -H 'X-Mock-Status: 500' localhost:4010/pets # same, via header
```

## Configuration — `mockserver.json`

Everything is JSON-driven; a missing config is fine (CLI flags suffice).

```json
{
  "spec": "examples/petstore.yaml",
  "port": 4010,
  "seed": 42,
  "cors": true,
  "latencyMs": 0,
  "routes": {
    "GET /pets":          { "latencyMs": 150 },
    "POST /pets":         { "errorRate": 0.1, "errorStatus": 422 },
    "GET /pets/{petId}":  { "latencyMs": 80 }
  }
}
```

| Field          | Scope        | Effect                                            |
| -------------- | ------------ | ------------------------------------------------- |
| `seed`         | global       | Deterministic data seed                           |
| `latencyMs`    | global/route | Sleep before responding (simulate slow backend)   |
| `errorRate`    | route        | `0.0–1.0` chance of returning `errorStatus`        |
| `errorStatus`  | route        | Status used by injected errors (default 500)       |
| `status`       | route        | Force every response on the route to this status   |
| `cors`         | global       | Permissive CORS headers for browser frontends      |

Route keys are `"METHOD /openapi/{template}/path"`.

### Vendor extension

Annotate a schema field to pin its generator:

```yaml
properties:
  ref:
    type: string
    x-mock: uuid   # uuid|email|name|phone|city|company|url|money|sentence|...
```

## CLI

```
mock serve [flags]
  --config   path to mockserver.json (default "mockserver.json", optional)
  --spec     path to OpenAPI 3.x spec (overrides config)
  --port     listen port (overrides config)
```

## Layout

```
main.go                  CLI entry (serve)
internal/config          mockserver.json loading + route keys
internal/spec            OpenAPI load/normalize -> internal Operation model
internal/faker           schema -> realistic value (heuristics + formats + seeding)
internal/server          routing, config application, status control, /__mock admin
examples/petstore.yaml   sample spec
```

## Roadmap

- **v2 — model codegen**: emit typed models from component schemas in
  ts, js, py, rs, go, c# (thin templated emitter over the same spec model).
- Stateful resources: opt-in in-memory store so POST/GET/PUT/DELETE behave like
  a real REST resource (path params already extracted by the router).
- Request-body validation against the spec; hot-reload on spec change.
```
