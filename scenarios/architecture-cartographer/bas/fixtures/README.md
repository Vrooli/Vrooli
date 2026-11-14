# Architecture Cartographer Fixtures

Hand-curated fixtures consumed by the cartographer's integration
tests under `api/internal/conflicts/integration_test.go` (and the
similar tests landing alongside future phases). Each subdirectory is a
self-contained scenario with the inputs the cartographer would see
plus the expected outputs:

- `<name>/manifest.yaml` — declares domains, paths, allowed dependencies, and any signal-weight overlays.
- `<name>/expected-graph.json` — the canonical `GraphSnapshot` shape the `CodeGraphAdapter` would return; `FakeCodeGraphAdapter` is seeded from this file so integration tests stay deterministic regardless of whether `go-code-graph` / `typescript-code-graph` are running.
- `<name>/expected-conflicts.json` — the list the registered Detectors must emit when run against the (graph, manifest) pair. Tests assert semantic equality (envelope shape, type, locations, severity).

## Fixtures

| Name | Language | Exercises |
|---|---|---|
| [`go-cycles/`](go-cycles/) | Go | `cycle` detector — a deliberate cross-package import cycle |
| [`go-mislocated/`](go-mislocated/) | Go | `glossary_drift` detector — symbol-backed foreign domain vocabulary |
| [`ts-junk-drawer/`](ts-junk-drawer/) | TypeScript | `layering` detector — shared substrate importing product code |
| `medium-realistic/` | mixed | _planned_ — ~200-file mixed scenario for performance + multi-detector regression |

## Why fixtures live under `bas/`

`bas/` (browser-automation studio) hosts every scenario's hand-curated
test data and recorded flows. The architecture-cartographer reuses the
same convention rather than inventing a parallel directory: any test
that needs deterministic input data finds it under `bas/fixtures/`,
and `vrooli scenario test` knows to mount it.
