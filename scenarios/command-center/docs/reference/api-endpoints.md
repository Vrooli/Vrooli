# API Endpoints

**Status:** verified implementation contract. Ports are assigned by the lifecycle manager and exposed as `API_PORT`; never hardcode one.

Typed integration state is exposed through generated Connect-RPC, with JSON REST projections for operator-readable surfaces. Recovery endpoints only return allowlisted owner guidance and never execute business mutations.

## `GET /api/v1/health`

Readiness plus per-source dependency status. An unreadable source appears here with its reason and does not prevent readiness. This remains a deliberately small operational REST probe; it is not a proto-owned business payload.

## Integration state

The generated `IntegrationsService` is mounted under `/vrooli.command_center.v1.integrations.IntegrationsService/` and provides `List`, `Get`, `Refresh`, and `RunAction`. The JSON projections are:

- `GET /api/v1/integrations` — cached lifecycle and feature state.
- `POST /api/v1/integrations/refresh` — explicitly refreshes state.
- `GET /api/v1/integrations/{id}` — one integration and its feature states.
- `GET /api/v1/integrations/{id}/features/{feature}` — one feature with lifecycle state kept separate.
- `POST /api/v1/integrations/{id}/action` — confirmed, allowlisted lifecycle action or non-actuating owner guidance.

Lifecycle availability does not imply feature compatibility. A healthy process with no producer-side feature probe reports feature status `unknown`; a missing feature is not repaired by restarting its process.

The current typed probes report compatibility only after the producer contract
answers successfully:

- Swarm Manager's `StatsService.GetPortfolioStats` proves its canonical
  portfolio and agent-stat features.
- LPBS's `MetricsService.GetAnalyticsSummary` proves only `visitors`,
  `cta_clicks`, `conversions`, and `variant_ab`; other declared LPBS features
  remain `unknown` until their producer contract returns them.
- Prompt Manager's `MemberflowService.GetInstruments` proves
  `team_instrument`.

Compatibility is not inferred from `/health`. A typed adapter rejects a
canonical read when the generated producer contract is unavailable; it does
not fall back to an untyped REST payload.

## `GET /api/v1/board`

The derived board shape (`CC-P0-007`). Rooms, the outcome denominator with its confidence and rationale, and per-source availability. The UI generates its routes from this response.

```json
{
  "schemaVersion": "2.0.0",
  "generatedAt": "2026-09-01T15:42:07Z",
  "rooms": [
    { "id": "forge", "title": "The Forge", "composition": "flow-current", "theme": "foundry", "metricIds": ["swarm_throughput_24h"] }
  ],
  "denominator": {
    "outcomeCategories": 6,
    "confidence": "partial",
    "rationale": "Objective set names seven categories; one is measured outside this board by design."
  },
  "sources": [
    { "team": "marketing-crew", "instrumentStatus": "partial", "readable": false, "reason": "No aggregator declared" }
  ]
}
```

## `GET /api/v1/rooms/{id}`

Composed readings for one room. Each reading carries both honesty axes, its source's instrument state, and — where applicable — its sample, target and prediction. Full shape in [../concepts/DATA.md](../concepts/DATA.md).

Query parameters:

| Parameter | Values | Effect |
|---|---|---|
| `samples` | `hide` \| `mark` \| `full` | Whether illustrative readings are withheld, marked, or returned plain. Default `mark`. |

## `GET /api/v1/focus`

The one ranked surface (`CC-P0-010`). Merges coverage gaps, untrusted readings, unavailable sources and unregistered outcomes into a single ordered list. Each entry names its owning team and states why it ranks where it does.

Ranking is stated, not implied, and **source integrity outranks coverage breadth** — a sensor reporting a value nobody can trust is a worse problem than a sensor that does not exist yet, because the first one is believed.

```json
{
  "generatedAt": "2026-09-01T15:42:07Z",
  "entries": [
    {
      "rank": 1,
      "kind": "no-instrument",
      "owner": "team:marketing-crew",
      "statement": "No aggregator declared; social-scheduling capability is owned by no scenario.",
      "affects": ["broadcast"],
      "gapOpenDays": 141,
      "reason": "A missing control loop outranks a missing pipeline: it blocks a room rather than a reading."
    },
    {
      "rank": 2,
      "kind": "no-pipeline",
      "owner": "team:monetization",
      "statement": "Revenue surface not exposed by the monetization instrument.",
      "affects": ["ledger", "mission-control"],
      "gapOpenDays": 136,
      "reason": "Substrate exists; one pipeline closes six readings."
    }
  ]
}
```

Entry kinds: `untrusted-reading`, `source-unavailable`, `no-instrument`, `no-pipeline`, `unregistered-outcome`.

## `GET /api/v1/open-loop`

The self-report (`CC-P0-011`). Every `MISSING` cell and every `UNREGISTERED` outcome, each with `firstObservedMissing` and `gapOpenDays`. Includes this scenario's own blind spots.

```json
{
  "missing": [ { "metricId": "revenue_mrr", "owner": "team:monetization", "firstObservedMissing": "2026-04-18", "gapOpenDays": 136 } ],
  "unregistered": [ { "outcome": "T2 personal agency", "objectiveRef": "T2", "reason": "Measured outside this board by design", "notedOn": "2026-08-09" } ],
  "self": [ { "statement": "This instrument does not measure its own read latency per source.", "notedOn": "2026-09-01" } ]
}
```

## `GET /api/v1/capabilities/describe`

The full projection of this instrument's sensor space, from the shared `api/handlers/capabilities` module — the same surface `infrastructure-manager` and `offer-desk` expose (`CC-P0-012`). This is how the team reads the instrument programmatically rather than by looking at a television.

## `GET|POST /api/v1/debug/render-stats`

Client render telemetry ring buffer, used by the capability ladder and the frame-budget check. The only non-`GET` surface, and it accepts telemetry only — it mutates nothing outside its own buffer.

## Compatibility aliases

| Alias | Maps to | Why it exists |
|---|---|---|
| `GET /api/v1/dashboards/{id}` | `rooms/{id}` | The outcomes charter's sensor map cites this path by name as the live sensor for a category. |
| `GET /api/v1/gaps` | `open-loop`, filtered to non-`NOW` coverage grouped by room | Same; the charter's gap-closure loop step 1 names it. |

Aliases, not a second model. They return the new reading shape.

## Error and degradation posture

- An unreadable source is an availability entry with the reason verbatim — never a zero, never a dropped row, never a `MISSING` coverage status (`CC-P0-009`).
- A slow source is `UNAVAILABLE` with `deadline exceeded`. Sources are independently degradable; one failure never blocks the response.
- An unparseable setpoint or registry **fails loudly** with a non-zero exit at startup. Reporting zero targets as zero problems is worse than not starting.
- No response exposes secrets, tokens, stack traces or absolute local paths. The board may be visible in public.

## Cross-references

- [../concepts/DATA.md](../concepts/DATA.md) — full shapes
- [../concepts/COVERAGE-MODEL.md](../concepts/COVERAGE-MODEL.md) — the vocabularies
- [cli-commands.md](cli-commands.md) — the same reads on the CLI
