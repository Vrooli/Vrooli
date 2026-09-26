# Prose Studio

Prose Studio is Vrooli's governed natural-language generation capability. A
consumer supplies a versioned style and profile; every generation routes through
ai-gateway, returns a measured candidate set, records full provenance, and can
be converged into a section-composed document.

The core rules are intentional:

- quality constraints are deterministic eligibility gates, not taste rankings;
- agent mode chooses by rarity above the floor, while human mode shows a spread;
- verbalized ordering hints are stored as uncalibrated round-local ordinals and
  are never read by selection;
- long-form variation happens at outline and section level, never whole-document;
- consumer-owned declaration files are authoritative projections with content
  hashes and explicit invalid/collision/unregistered states.

## Surfaces

- Go Connect-RPC API and JSON-compatible REST projections (`api/`)
- CLI verb parity with JSON output (`cli/`)
- React/Vite operator surfaces: Variation Board, Style Library, Document
  Workspace, and Declaration Registry (`ui/`)
- shared deterministic metrics package (`packages/textmetrics`)

Start with [`docs/START-HERE.md`](docs/START-HERE.md), then read [`PRD.md`](PRD.md)
and [`DESIGN.md`](DESIGN.md). SQLite is the default persistence layer; all model
inference is delegated to ai-gateway and no model credential is stored here.

```bash
make setup
make start
make test
```

## What You Get

Every generation is a measured, comparable candidate set with resolved
profile instructions, model/token provenance, disclosure, and an append-only
session history. Styles and profiles can also arrive as consumer-owned
declaration files, so adding a new voice does not require integration code.

## Documentation Map

- [`docs/START-HERE.md`](docs/START-HERE.md) — first vertical slice and validation
- [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md) — system shape
- [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) — ownership boundaries
- [`docs/concepts/FLOWS.md`](docs/concepts/FLOWS.md) — generation and document flows
- [`docs/internal/FOLLOW-UPS.md`](docs/internal/FOLLOW-UPS.md) — bounded P1/P2 work

## Customize Safely

Add styles, profiles, lexicon terms, and declaration records as data. Keep
sampler and selection kinds in the API so their invariants remain governed;
extend the proto first when adding a new transport operation. Never put model
credentials or vendor-specific logic in this scenario.
