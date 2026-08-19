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

1. **The setpoint is never stored.** It is read from
   `docs/infra-health/strategy/RELIABILITY_TARGETS.md` at query time. A cached
   setpoint would let the board measure against a target the operator has
   already changed.
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
3. **No derived set is ever cached.** The should-be-supervised set is computed
   fresh per read. Caching it would turn a derivation into the roster
   operating-model rule 6 forbids.

Rule 2 is the one deliberate divergence from `meta-optimization-manager`, which
never stores a numerator at all. Uptime over thirty days *is* history, and
`INSTRUMENTATION_ROADMAP.md` Gap 11 names the failure of not keeping it:
*"an outage is indistinguishable from missing data after the fact."*

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Readings | readings | SQLite | `api/internal/readings/schema.sql` | Rolling window, longest target window + margin | Raw observed value, source, timestamp, trust verdict. **No band evaluation.** |
| Trust verdicts | readings | SQLite (on the reading row) | `api/internal/readings/schema.sql` | Same as the reading | A property of the observation, not of the target — and therefore **not recomputable later**, which is why rule 2 exempts it. |
| Band verdicts | readings | **none** | recomputed per query from the reading + current deadband | not stored | In-band / out-of-band. Storing one would strand a judgment against a deadband the operator has since changed. |
| Findings | focus | SQLite | `api/internal/focus/schema.sql` | Until closed, plus an audit window | Ranked error entries with their originating source. |
| Efficacy records | focus | SQLite | `api/internal/focus/schema.sql` | Lifetime of the finding | Finding → named sensor → expected in-band return → observed result. |
| Targets | targets | **none** | the upstream setpoint document | not stored | Parsed per read. Caching it would let the board grade itself against a stale bar. |
| Supervised set | supervision | **none** | derived at read time | not stored | Core-set closure ∪ load-bearing declarations. Never cached. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| readings tables | readings | `api/internal/readings/schema.sql` | readings repository/service/handlers |
| focus tables | focus | `api/internal/focus/schema.sql` | focus repository/service/handlers |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

`targets` and `supervision` own no tables by design and must not acquire any.
An extension that adds a `targets` table is almost certainly caching the
setpoint, which is deviation `D6`.

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships the `notes` domain as a worked CRUD slice with a
binary attachment-upload exception, showing how a real domain owns its
tables, metadata, and opaque blob bytes. Copy its shape, then remove it.

Its Data Ownership rows:

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Notes | notes | SQLite | `api/internal/notes/schema.sql` | Until deleted by future product behavior | Template reference data; remove with notes domain. |
| Attachment metadata | notes | SQLite | `api/internal/notes/schema.sql` | Until parent note or attachment is deleted by future product behavior | Metadata only; bytes are stored through BlobStore. |
| Attachment bytes | notes | Filesystem BlobStore by default | BlobStore implementation in notes handler module | Same lifecycle as metadata | Opaque bytes stay outside proto payloads. |

Its Schema Map row:

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| notes tables | notes | `api/internal/notes/schema.sql` | notes repository/service/handlers |

Its Retention And Deletion row:

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Template notes data | Domain removal or future product delete behavior | Local development data only | Real scenarios must define product-specific deletion semantics. |
<!-- EXAMPLE-DOMAIN:notes END -->

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
| Readings | Rolling age-out | Longest declared target window + margin. Today the longest is 30d, so 45d. **Derive this from the setpoint rather than hardcoding it** — a target that widens its window must widen retention with it. | The derivation is not implemented; an initial fixed 45d is the interim, and a target declaring a longer window would be silently unmeasurable. Tracked as an integrity finding in `targets`. |
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
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
