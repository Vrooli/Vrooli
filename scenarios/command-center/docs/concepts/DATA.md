# Data

**Status:** verified data contract for the current implementation. Authored coverage and live reading trust remain separate axes; integration lifecycle and feature state are exposed alongside the reading projections.

## Storage posture

The P0 set is **filesystem and in-memory only**. There is no database.

- The **outcome registry** is checked-in versioned data.
- The **setpoint** is a checked-in file, parsed per query, never written.
- **Numerators are computed live and never stored** — invariant 4. A stale board is structurally impossible because there is nothing to go stale.
- The per-source TTL cache holds the last upstream response so a failed fetch can degrade to `CACHED` rather than to nothing. It is a freshness buffer, not a store.

Reading history (`CC-P2-002`) is the first thing that earns durable storage, and it earns it for one reason: recomputing in-band verdicts against a changing setpoint requires the readings, not the verdicts. When it lands it uses SQLite via `api-core/storage`, stores readings and trust verdicts, and stores **no band verdicts** — so tightening a target re-grades its own history.

## The registry

`config/outcome-registry.json`, superseding `config/gap-registry.json`.

```json
{
  "$schema": "./outcome-registry.schema.json",
  "schemaVersion": "2.0.0",
  "rooms": [
    {
      "id": "forge",
      "title": "The Forge",
      "category": "engineering-velocity",
      "objectiveRefs": ["I1"],
      "select": { "anyOf": ["source:swarm-manager", "tag:throughput"] },
      "composition": "flow-current",
      "theme": "foundry"
    }
  ],
  "metrics": [
    {
      "id": "swarm_throughput_24h",
      "label": "Items shipped",
      "description": "Backlog items completed by the swarm in the trailing day.",
      "unit": "count",
      "format": "integer",
      "tags": ["throughput"],
      "source": {
        "team": "director-swarm",
        "binding": "scenario:swarm-manager",
        "read": "/api/v1/stats",
        "select": "completed_24h",
        "ttlSeconds": 30
      },
      "coverage": "NOW"
    }
  ],
  "tombstones": []
}
```

### Metric fields

| Field | Purpose |
|---|---|
| `id` | Stable identity. Never reused for a different meaning. Predictions, setpoint bars and history key on it. |
| `label`, `description` | Human meaning. Authored, never derived. |
| `unit`, `format` | How the value is rendered and whether two values may be compared. A unit mismatch against a source is an `UNTRUSTED` verdict, not a silent conversion. |
| `tags` | Free grouping, used by room `select` queries. |
| `source` | Where the reading comes from: the owning team, the binding (a team instrument, a named scenario, or none), the read path, the selector, and the TTL. |
| `coverage` | Authored coverage status — `NOW`, `IN-REACH` or `MISSING`. Joined live but never overwritten by a fetch failure. |
| `owner` | Present when coverage is not `NOW`: the team responsible for closing the gap. |
| `whatIsNeeded` | Present when coverage is not `NOW`: the named missing surface, in one sentence. |
| `firstObservedMissing` | Date the gap was first observed. Drives ageing in the self-report (`CC-P0-011`). |
| `sample` | Authored illustrative value. See [PROVENANCE-MODEL.md](PROVENANCE-MODEL.md). |
| `empirical` | Prediction-ledger verdict riding on this metric — `NONE`, `PENDING`, `HIT`, `MISS` or `UNMEASURABLE`. Joined live from the decision store; never authored in the registry. |

### Sample block

```json
"sample": {
  "value": 12400,
  "series": [8100, 8800, 9600, 10200, 11300, 12400],
  "basis": "hand-authored, mid-scale, reviewed 2026-09-01"
}
```

`basis` is required and is emitted with every sample reading, so nothing downstream can mistake it for a measurement (`CC-P0-003`).

### Tombstones

```json
"tombstones": [
  {
    "id": "scenario_usage_frequency",
    "retired": "2026-09-01",
    "reason": "Superseded by the usage projection on the meta-optimization instrument",
    "supersededBy": "meta-opt:usage.invocations"
  }
]
```

A prediction naming a tombstoned id resolves to `unmeasurable` with this reason attached, rather than dangling.

## A reading on the wire

What the API composes per metric. This replaces `MetricEntry`, which carried six fields and no value.

```json
{
  "id": "swarm_throughput_24h",
  "label": "Items shipped",
  "description": "Backlog items completed by the swarm in the trailing day.",
  "unit": "count",
  "format": "integer",

  "value": 1284,
  "observedAt": "2026-09-01T15:42:07Z",
  "ttlSeconds": 30,

  "coverage": "NOW",
  "trust": "VALID",

  "source": {
    "team": "director-swarm",
    "binding": "scenario:swarm-manager",
    "instrumentStatus": "partial",
    "instrumentArchetype": "production-ledger"
  },

  "target": { "direction": "up", "bar": 1500, "barRef": "setpoint:forge.throughput" },

  "owner": null,
  "whatIsNeeded": null,
  "firstObservedMissing": null,
  "gapOpenDays": null,
  "sample": null,

  "empirical": "NONE",
  "prediction": null
}
```

Rules the shape enforces:

- `coverage` and `trust` are separate fields with closed vocabularies and no derived composite (`CC-P0-002`). The API never emits a pre-composed "ink" — that would be the forbidden merge moved one layer down.
- `trust: "VALID"` requires `observedAt` inside `ttlSeconds` (`CC-P0-004`).
- `value` is `null` when no reading exists. It is never fabricated, and never filled from `sample` — the sample travels in its own field so a consumer must opt into using it.
- `source.instrumentStatus` carries the owning team's declared instrument state, so *no pipeline* and *no instrument* are distinguishable (`CC-P0-008`).
- `sample` is non-null only when `coverage` is `IN-REACH` or `MISSING`.
- `empirical` is a third independent axis and is never folded into `coverage` or `trust` (`CC-P0-014`). It is always present; `NONE` means no prediction rides on this metric, which is the common case and not a deficiency.
- `prediction` carries the detail — target, direction and horizon — only when `empirical` is `PENDING`. A matured verdict keeps its `empirical` value and drops the block.

## Board shape

`GET /api/v1/board` returns the derived shape rather than a hardcoded room list (`CC-P0-007`).

```json
{
  "schemaVersion": "2.0.0",
  "generatedAt": "2026-09-01T15:42:07Z",
  "rooms": [
    { "id": "forge", "title": "The Forge", "composition": "flow-current", "theme": "foundry", "metricIds": ["..."] }
  ],
  "denominator": {
    "outcomeCategories": 6,
    "confidence": "partial",
    "rationale": "Objective set names seven categories; one (personal agency) is measured outside this board by design."
  },
  "sources": [
    { "team": "marketing-crew", "instrumentStatus": "partial", "readable": false, "reason": "No aggregator declared" }
  ]
}
```

The UI generates routes from this. Adding a room is a registry edit; no route file changes.

## The setpoint

A checked-in file this scenario parses and validates but has no path to write (`CC-P0-005`).

```json
{
  "schemaVersion": "1.0.0",
  "bars": [
    {
      "metricId": "swarm_throughput_24h",
      "direction": "up",
      "bar": 1500,
      "unit": "count",
      "authoredAgainst": 1180,
      "decisionRef": "decision:2026-08-14-forge-throughput"
    }
  ]
}
```

Integrity rules, checked at parse:

- Every `NOW` metric has a bar, or is declared not-gradeable with a stated reason.
- No bar equals the reading it was authored against — a bar set to the current value grades nothing.
- Every changed bar carries a `decisionRef`.
- An unparseable setpoint **fails loudly**. Reporting zero targets as zero problems is the failure mode this rule exists to prevent.

## Caching and freshness

| Source class | Default TTL | On failure |
|---|---|---|
| Team instrument, `live` | 60s | Serve last good as `CACHED` with age; source listed as `UNAVAILABLE` with reason |
| Named scenario, direct | 30s | As above |
| Control plane | 60s | As above |

A failed fetch **never** changes `coverage` (`CC-P0-009`). It changes `trust` and adds an availability entry. This separation is the whole point of the two axes.

## Cross-references

- [COVERAGE-MODEL.md](COVERAGE-MODEL.md) — the vocabularies these fields carry
- [PROVENANCE-MODEL.md](PROVENANCE-MODEL.md) — how a reading becomes a figure
- [OUTCOME-TAXONOMY.md](OUTCOME-TAXONOMY.md) — versioning, ids and migration rules
- [../reference/api-endpoints.md](../reference/api-endpoints.md) — the surfaces these shapes travel on
