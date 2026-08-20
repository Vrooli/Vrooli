# Data — Infrastructure Manager

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

The template default is embedded SQLite through `modernc.org/sqlite`.
The lifecycle sets `SQLITE_PATH` through `.vrooli/service.json`, and
the API applies schemas on startup through `api-core/database`.

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

Each domain owns its own tables and is the source of truth for its data.
The `health` domain owns no product data — it only probes configured
database reachability. As you build real domains, add a row per data
shape they persist: name it, name the owning domain, the storage backend,
the schema file that is the source of truth, the retention rule, and any
remarks. Keep blob/opaque bytes outside proto payloads, behind a seam
such as BlobStore.

**This scenario persists almost nothing, and the exceptions are deliberate.**
Three rules govern every row below:

1. **Neither half of the denominator is ever stored.** Each control layer's space
   is read live through its `space --projection <p> --json` verb; the setpoint is a
   checked-in file parsed at query time. A cached space goes stale on every owner
   change, and a cached setpoint would let the board measure against a bar the
   operator has already moved. **There is also no write path to the setpoint** —
   not an endpoint, not a table, not a migration. Its absence is the `D6` defence.
2. **Band verdicts are never stored — only readings.** A reading is raw observed
   fact plus a timestamp and a trust verdict. In-band status is recomputed at
   query time against the *current* deadband, so tightening a target re-grades
   history instead of stranding stale judgments.

   **The trust verdict is the deliberate exception, and the reason is that it is
   not recomputable.** A band verdict is a statement about the *target*: given the
   value, was it inside the deadband? Ask again tomorrow with a different deadband
   and you get a correct new answer from the same stored value. A trust verdict is a
   statement about the *observation*: was this check saturated, ghosted, or shelved
   at the moment it was read? That context is gone once the moment passes, so a
   trust verdict deferred is a trust verdict lost. Storing it is what keeps rule 2
   from quietly discarding evidence it cannot get back.
3. **No derived set is ever cached.** The condition leg population and the
   should-be-supervised set are both computed fresh per read. Caching either would
   turn a derivation into the roster operating-model rule 6 forbids.

Rule 2 is the one deliberate divergence from `meta-optimization-manager`, which
never stores a numerator at all. Uptime over thirty days *is* history, and
`INSTRUMENTATION_ROADMAP.md` Gap 11 names the failure of not keeping it:
*"an outage is indistinguishable from missing data after the fact."*

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Readings | condition | SQLite | `api/internal/condition/schema.sql` | Rolling window, longest declared cell window + margin | Raw observed value, cell ref, source, timestamp, trust verdict. **No band evaluation.** |
| Trust verdicts | condition | SQLite (on the reading row) | `api/internal/condition/schema.sql` | Same as the reading | A property of the observation, not of the target — and therefore **not recomputable later**, which is why rule 2 exempts it. |
| Band verdicts | condition | **none** | recomputed per query from the reading + current deadband | not stored | `IN_BAND` / `OUT_OF_BAND` / `PENDING_SUSTAIN` / `NEEDS_BASELINE` / `NOT_EVALUATED`. Storing one would strand a judgment against a deadband the operator has since changed. |
| Findings | focus | SQLite | `api/internal/focus/schema.sql` | Until closed, plus an audit window | Ranked error entries with their originating source. |
| Efficacy records | focus | SQLite | `api/internal/focus/schema.sql` | Lifetime of the finding | Finding → named sensor → expected in-band return → observed result. |
| Projection spaces | coverage | **none** | each owner's `docs/spaces/<projection>-space.md`, read via `space --projection <p> --json` | not stored | Read live per query. A cached space goes stale on every owner change. |
| Setpoint bars | coverage | **none** | the checked-in setpoint file in this scenario | not stored | Parsed per read. **No write path exists**; changing a bar is a reviewed commit. |
| Cell grid | coverage | **none** | computed per query from spaces × setpoint × live join | not stored | A `coverage` table is a cached denominator, which is `D6`. |
| Leg population | condition | **none** | derived per read from cells that resolved `NOW` | not stored | Cannot drift out of sync with coverage because there is no second list. |
| Supervised set | condition (`supervision` projection) | **none** | derived at read time | not stored | Core-set closure ∪ load-bearing declarations. Never cached, and never falls back to an enumerated list when the closure source is unavailable. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| condition tables | condition | `api/internal/condition/schema.sql` | condition repository/service/handlers |
| focus tables | focus | `api/internal/focus/schema.sql` | focus repository/service/handlers |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |
| setpoint | coverage (read-only) | `setpoint/reliability-setpoint.json` — a checked-in file, not a table | coverage parser |

`coverage` owns no tables by design and must not acquire any. An extension that
adds a `coverage` table is almost certainly caching a space, the setpoint, or the
cell grid — all three are deviation `D6`. A migration that creates a setpoint
table is the same violation wearing a different hat.

## Migrations And Compatibility

The generated template uses idempotent schema bootstrap. Domain schema
files should use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| None yet. | n/a | n/a | Add when product requirements include import/export. |

## Retention And Deletion

**Retention here is a correctness constraint, not housekeeping.** A target
banded on "99.5% uptime over 30 days" cannot be evaluated from a 7-day
window, so the retention floor is set by the longest window any target
declares, plus margin. Trimming readings below that floor silently converts
a measurable target into an unmeasurable one — and per
[`TRUST-MODEL.md`](TRUST-MODEL.md), unmeasurable must be *reported*, never
quietly reported as in-band.

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Readings | Rolling age-out | Longest declared cell window + margin. Today the longest is 30d, so 45d. **Derive this from the setpoint rather than hardcoding it** — a cell that widens its window must widen retention with it. | The derivation is not implemented; an initial fixed 45d is the interim, and a cell declaring a longer window would be silently unmeasurable. Tracked as a setpoint-integrity check in `coverage`. |
| Trust verdicts | With the parent reading | Same as readings | None. |
| Findings | Closed and past the audit window | Kept while open; 90d after close so efficacy trends survive. | None. |
| Efficacy records | With the parent finding | Same as findings | None. |

Nothing here is user data, so deletion has no privacy driver — only an
evidence-integrity one. Storage pressure is not a valid reason to trim below
the floor; the correct response is to say so and route it, exactly as the
scenario would for any other instrumentation shortfall.

## Privacy Notes

**This scenario stores no personal, customer, financial, or regulated data,
and should never begin to.** Everything it persists is a measurement of local
infrastructure: a numeric or enumerated reading, its source, a timestamp, a
trust verdict, and findings derived from those.

Two sensitivities are worth naming even so:

- **Readings are operational intelligence about the host.** Uptime patterns,
  restart frequency, capacity claims, and storage growth describe how the
  operator's machine behaves. They stay local, in embedded SQLite, and are
  never transmitted anywhere.
- **Findings may quote a source's error text verbatim**, and an upstream
  error could incidentally contain a path or hostname. Findings are not a
  general log sink — they carry a stated reason, not raw output — and no
  reading path should ever capture credentials, since every source is read
  through a typed surface rather than by scraping logs.

If either property changes, update this document and
[`../internal/SECURITY.md`](../internal/SECURITY.md) in the same change.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`COVERAGE-MODEL.md`](COVERAGE-MODEL.md) — why neither half of the denominator is stored
- [`CONDITION-MODEL.md`](CONDITION-MODEL.md) — why band verdicts are never persisted
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
