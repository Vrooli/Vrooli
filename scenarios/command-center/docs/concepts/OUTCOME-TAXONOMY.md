# Outcome Taxonomy

**Status:** contract canon for this scenario. The taxonomy itself is owned by `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md`, which derives from `path:docs/director-swarm/strategy/OBJECTIVES.md`. This document states how Command Center *reads* that taxonomy without owning it.

## The rooms are not a theme list

Each room on the board is an **outcome category**. From the charter: "Outcomes are organized by the six Command Center dashboard pages. Each page is a category." Every category traces upward to an objective, and "an outcome worth pursuing that no category can hold is a finding against this charter rather than an outcome that does not count."

The direction of derivation matters and is stated in the charter: objectives say what Vrooli is for; the charter says how progress toward them is observed; **a dashboard's page list is an instrumentation decision, not a statement of intent.** This scenario is downstream of both. It renders the taxonomy; it does not author it.

The six categories as of 2026-09-01:

| Room | Category | Primary question |
|---|---|---|
| Mission Control | System overview | Is the platform running and producing? |
| The Hive | Scenario ecosystem | Is the capability portfolio growing and healthy? |
| The Forge | Engineering velocity | Is work flowing? |
| Ledger | Revenue and subscriptions | Is the business working? |
| Broadcast | Marketing and growth | Is anyone arriving? |
| Panorama | Aggregate view | What is the state of the whole, and what is unmeasurable? |

The charter also records one outcome that fits no room and deliberately has no home: personal agency, which is measured by the scenario that serves it rather than by a platform board. That entry is the taxonomy working correctly — an honest hole in the top-level hierarchy, visible because objectives sit above it.

## Nothing here is fixed

The operator decision of 2026-09-01 is explicit: **the taxonomy is open, and the implementation must make changing it cheap.** Adding a seventh category, splitting a room, retiring a metric or re-pointing a source must be a data change with a migration, never a code change with a release.

That rules out the previous implementation, in which the room list lived in `App.tsx` as six literal routes and the metric list lived in a hand-maintained JSON file with no version, no ids policy and no migration path.

## The board derives its own shape

The room list, the metric set and every source binding are computed at read time (`CC-P0-007`) from three reads:

### 1. The objective set — read as a transmitter

Instrumental and terminal objectives are read through `prompt-manager graph objectives`. The join **stays where it is**; this scenario reads it over the standardised verb rather than absorbing it.

This is the fork named in the team's own gap marker, and the target model settles it: a *transmitter* puts a raw signal on a shared bus in a standard form, and "the verb is a bus contract — its value scales with the number of conforming devices, not with the instrument's cleverness." Invariant 2 forbids the alternative outright: an observer that writes its own reference model is confirming itself.

The objective set supplies the denominator. It is what makes `UNREGISTERED` computable — an outcome named upstream with no row here.

### 2. Every team's instrument declaration

Each team's record carries an `instrument` block: `status`, `archetype`, `scenario`, `coversScenarios` and a dated `gapMarker`. Read live, this is the fleet map. It tells the board which teams have an aggregator worth reading, which have scattered sources, and which have no control loop at all — see [SOURCE-MAP.md](SOURCE-MAP.md).

### 3. Each live instrument's own surface

For teams that have one, read its standard verb — `/api/v1/capabilities/describe` from the shared handler module, or the projection verb. **Never re-implement a source's measurement; read its derived output.** Re-deriving a peer's number is how two surfaces come to disagree with no way to tell which is right.

## Evolution rules

These are the properties that make the taxonomy migratable rather than merely editable.

### Stable identity

Metric ids are stable and separate from labels. Predictions, setpoint bars and reading history all key on the id. Labels, groupings and room membership can be rearranged freely.

**An id is never reused for a different meaning.** Reuse silently re-points every historical reading and every prediction that named it.

### Versioned registry with migrations

The registry carries a schema version. A version bump requires a migration path, and a bump without one fails validation (`CC-P0-006`). This is what allows the reading shape to grow — adding the trust axis, adding prediction binding — without orphaning what is already authored.

### Rooms are a grouping, not a type

A room is a query over the metric set plus a composition binding. Moving a metric between rooms, splitting a room, or adding a seventh is configuration:

```json
{
  "id": "forge",
  "title": "The Forge",
  "category": "engineering-velocity",
  "objectiveRefs": ["I1"],
  "select": { "anyOf": ["source:swarm-manager", "tag:throughput"] },
  "composition": "flow-current",
  "theme": "foundry"
}
```

No route file, no component registry, no rebuild. The UI generates its routes from the fetched board shape (`CC-P0-007`), which is also what lets a newly-instrumented team's room appear without a Command Center release.

### Retirement leaves a tombstone

A metric that goes away keeps a dated record of why:

```json
{
  "id": "scenario_usage_frequency",
  "retired": "2026-09-01",
  "reason": "Superseded by the usage projection on the meta-optimization instrument",
  "supersededBy": "meta-opt:usage.invocations"
}
```

A prediction naming a tombstoned metric resolves to **unmeasurable with a reason**, matching prediction-ledger rule 4, rather than dangling as a broken pointer. Silent deletion is what makes a prediction ledger unscoreable in a way nobody notices.

### The composition catalogue is open too

Each room binds a composition by name. Compositions are registered assets, so a new visual idea is a new entry, and a room can change its composition without changing its data. A composition that a device tier cannot render falls back through the capability ladder to a composed still (`CC-P1-012`), never to a blank surface.

## What stays authored by hand

Genericity has a boundary, and pretending otherwise produces a worse system:

- **What a metric *means*** — its label, description, unit and what would need to exist to make it real. No derivation writes prose.
- **Sample values** — authored, reviewed, stamped. See [PROVENANCE-MODEL.md](PROVENANCE-MODEL.md).
- **Setpoint bars** — authored by the operator, elsewhere. This scenario reads them.
- **Composition design** — which visual idea a room uses, and what its quiet zones are.

The derivation covers *which rows exist and where they come from*. It does not invent meaning.

## Cross-references

- [SOURCE-MAP.md](SOURCE-MAP.md) — the fleet the taxonomy is joined against
- [COVERAGE-MODEL.md](COVERAGE-MODEL.md) — how `UNREGISTERED` is computed from the objective set
- [DATA.md](DATA.md) — registry and board-shape schemas
- `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md` — the taxonomy's owner
