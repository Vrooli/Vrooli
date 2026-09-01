# Architecture

**Status:** design contract, ahead of implementation. This document describes the target architecture agreed on 2026-09-01. The code still implements the 2026-04 read-only aggregator; see [../internal/PROBLEMS.md](../internal/PROBLEMS.md) for the gap between the two.

Command Center is `team:director-swarm`'s instrument. What that obliges it to be is in [INSTRUMENT-MODEL.md](INSTRUMENT-MODEL.md); this document is how it is built.

## Shape

```
┌─ sources ───────────────┐   ┌─ command-center ─────────────────┐   ┌─ surfaces ──────┐
│ prompt-manager          │   │                                  │   │                 │
│   graph objectives ─────┼──▶│  derivation                      │   │  board (UI)     │
│   team instrument blocks│   │    objective set → denominator   │   │  describe (API) │
│                         │   │    instrument blocks → fleet map │──▶│  CLI            │
│ meta-optimization-mgr   │   │    live surfaces → bindings      │   │                 │
│ infrastructure-manager  │──▶│                                  │   └─────────────────┘
│ offer-desk / money-ledgr│   │  join                            │
│ swarm-manager           │   │    authored coverage × live read │
│ landing-page-biz-suite  │   │    trust verdict per reading     │
│ vrooli control plane    │   │                                  │
└─────────────────────────┘   │  compose                         │
                              │    rooms · ranked surface        │
   ┌─ checked-in data ────┐   │    open-loop self-report         │
   │ outcome-registry.json│──▶│                                  │
   │ setpoint.json        │   └──────────────────────────────────┘
   └──────────────────────┘
```

Three layers, and the boundaries between them are the architecture:

- **Derivation** turns upstream declarations into the board's shape. It knows nothing about rendering.
- **Join** produces readings with both honesty axes attached. It never mutates authored coverage from a fetch result.
- **Compose** groups readings into rooms and surfaces. It adds no data.

## API

Go on `api-core/server`, per-domain handler files.

| Surface | Purpose |
|---|---|
| `GET /api/v1/health` | Readiness plus per-source dependency status |
| `GET /api/v1/board` | The derived board shape: rooms, denominator with confidence, source availability |
| `GET /api/v1/rooms/{id}` | Composed readings for one room |
| `GET /api/v1/focus` | The one ranked surface — what to build next, ordered, each entry naming its owner |
| `GET /api/v1/open-loop` | Every `MISSING` cell and every `UNREGISTERED` outcome, dated and aged |
| `GET /api/v1/capabilities/describe` | The full projection of this instrument's sensor space, from the shared handler module |
| `GET\|POST /api/v1/debug/render-stats` | Client render telemetry ring buffer |

`/api/v1/dashboards/{id}` and `/api/v1/gaps` remain as compatibility aliases onto `rooms` and a filtered `open-loop`, because the outcomes charter's sensor map cites them by name. They are aliases, not a second model.

**No non-GET route exists outside the debug telemetry sink** (`CC-P0-013`). There is no path that writes the setpoint, mutates an upstream, or files a work item. Sensor implies no authority.

## Source clients

One interface, one behaviour contract:

- A source names itself and degrades **legibly**: an unreadable source produces an availability entry with the reason verbatim, never a zero and never a dropped row.
- A source has a deadline. A slow source is `UNAVAILABLE` with `deadline exceeded`, not a hung board.
- A source is **independently degradable**: one unreadable source never prevents the rest of the board from rendering.
- A client **never re-implements a source's measurement.** It reads the derived output. Re-deriving a peer's number is how two surfaces come to disagree with no way to tell which is right.

## UI

React 18 + Vite + TypeScript, composed from React Component Library assets. See [UI-ARCHITECTURE.md](UI-ARCHITECTURE.md).

Two rules bind the UI to this architecture:

1. **Routes are generated from `/api/v1/board`**, never from a literal list (`CC-P0-007`). A room that appears in the registry appears in the app.
2. **The UI never composes provenance itself.** It receives `coverage` and `trust` and maps the pair to an ink through one shared resolver. A second place that decides how a sample looks is a second source of truth about honesty.

## Configuration

| Input | Kind | Written by |
|---|---|---|
| `config/outcome-registry.json` | Checked-in, versioned, migratable | People, in review |
| `config/setpoint.json` | Checked-in | The operator, elsewhere. Never this scenario. |
| `${SOURCE}_BASE_URL` | Env, assigned by the lifecycle manager | Runtime |
| URL parameters | Per-view | The viewer: `room`, `cycle`, `samples`, `fullscreen` |

## What is deliberately absent

- **No database in the P0 set.** Numerators are live; there is nothing to persist that would not make the board capable of being stale.
- **No write paths.** See above.
- **No derived-set cache.** The board recomputes its shape per request. A cached fleet map is a fleet map that can be wrong.
- **No per-user auth.** This is a local read-only surface; the outward-facing concern is handled by `samples=hide`, not by authentication.
- **No hardcoded room list, metric list or source list.** All three are derived.

## Cross-references

- [INSTRUMENT-MODEL.md](INSTRUMENT-MODEL.md) — the obligations this satisfies
- [DATA.md](DATA.md) — shapes on the wire
- [UI-ARCHITECTURE.md](UI-ARCHITECTURE.md) — the board's client architecture
- [../internal/PROBLEMS.md](../internal/PROBLEMS.md) — where the code diverges from this document today
