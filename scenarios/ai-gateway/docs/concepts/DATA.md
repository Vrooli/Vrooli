# Data - AI Gateway

AI Gateway should persist operational evidence and operator policy
state, not provider secrets or sensitive prompt/response content.

## Storage Overview

The generated scenario starts with embedded SQLite via `SQLITE_PATH`.
That is sufficient for local route evidence, profile settings, smoke
test history, and conformance scan reports. A future Postgres migration
is reasonable if gateway evidence becomes fleet-scale or shared across
machines, but the schema must keep provider secret material out of this
scenario.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Route evidence | routing | SQLite initially | `api/internal/routing/schema.sql` | Rolling retention, default 30 days unless changed by operator policy. | Stores metadata, route reasons, timings, and redacted provider info. |
| Provider inventory snapshot | inventory | SQLite cache | `api/internal/inventory/schema.sql` | Short TTL cache; live resource policy remains authoritative. | Used for UI responsiveness and drift comparison. |
| Smoke-test results | inventory | SQLite | `api/internal/inventory/schema.sql` | Rolling retention, default 30 days. | Does not store prompt/response bodies by default. |
| Gateway profiles | routing | SQLite or checked-in policy file | `api/internal/routing/profiles.go` or policy artifact | Durable until changed. | Cross-provider profile policy owned by AI Gateway. |
| Conformance scan reports | conformance | SQLite and exported JSON | `api/internal/conformance/schema.sql` | Rolling retention plus explicit exported reports. | Findings should include source paths and fix guidance, not file contents. |
| Operator preferences | operator | SQLite/local storage as appropriate | UI/API settings contract | Durable until reset. | Presentation and filter preferences only. |

## Data That Must Not Be Stored

- OpenRouter API keys or other provider credentials.
- Full prompts, responses, uploaded files, or scenario secrets unless a
  future retention policy explicitly permits them.
- Concrete model catalogs copied from resources as source-of-truth data.
  Snapshots are allowed only as cached evidence with timestamp/source.
- Embedding vectors owned by other scenarios.

## Embedding Metadata Expectations

Embedding governance is about protecting caller data stores from silent
model changes. AI Gateway should report and validate these fields when a
scenario uses embeddings:

- embedding role, such as `embedding.default`
- resolved provider and concrete model when available
- vector dimensions
- input normalization/tokenization policy when available
- content hash/version or corpus version
- index/table/collection name
- migration or retarget status

The gateway should not assume one global embedding length. Hard-coded
dimensions in a scenario may be acceptable only when stored beside
role/model metadata and validated by a retarget plan.

## Schema Map

| Table/Object | Owner | Planned Source | Used By |
|---|---|---|---|
| `route_events` | routing | `api/internal/routing/schema.sql` | route history, UI traces, conformance evidence; metadata-only retention with prompt/response/attachment redaction. |
| `provider_snapshots` | inventory | `api/internal/inventory/schema.sql` | role inventory, drift checks |
| `smoke_results` | inventory | `api/internal/inventory/schema.sql` | provider health UI and CLI |
| `profiles` | routing | `api/internal/routing/schema.sql` or policy artifact | route preview/execution |
| `conformance_runs` | conformance | `api/internal/conformance/schema.sql` | test-genie reports |
| `conformance_findings` | conformance | `api/internal/conformance/schema.sql` | operator UI, exports, migration reports |

Inventory, smoke, profile, and conformance tables are planned data
shapes and should be implemented only when the owning domain is built
and tested. `route_events` is implemented and intentionally stores metadata
only: request/operation identifiers, role/profile/privacy, selected resource
path, policy/failure reasons, fallback state, redaction flags, latency,
timestamp, and—when attachments are routed—image count, byte count, SHA-256
content hash, and declared dimensions. It never stores inline bytes,
references, base64, prompt text, or response text.

`route_events` additionally carries stable machine-readable outcome codes
so route history is analytics-ready without log parsing: `breaker_state`,
`failure_class`, `rejection_reason`, `capacity_verdict`, `capacity_claim_id`,
`capacity_required_bytes`, `capacity_granted_bytes`, `capacity_reclaim_required`,
`input_tokens`, `output_tokens`, `cost_estimate`, and `selected_model`. These
columns are additive with safe defaults; token/cost/model fields stay zero/empty
unless the resource reports them, and capacity fields are populated by the
capacity-aware routing phase. None of these fields contain prompt or response
text. `provider_health` (keyed by provider/role/kind) persists circuit-breaker
state, consecutive failures, last failure class, cooldown, and generation.

On an existing local database, adding a column to a `CREATE TABLE IF NOT EXISTS`
block is a silent no-op, so `EnsureSchemas` runs a post-apply drift check that
fails loudly and instructs a one-shot `ALTER TABLE ... ADD COLUMN` migration
rather than recreating the table (route history is preserved).

The 2026-08-16 sampling columns need that migration on any pre-existing
`route_events` table:

```sql
ALTER TABLE route_events ADD COLUMN sampling_temperature REAL;
ALTER TABLE route_events ADD COLUMN sampling_temperature_support TEXT NOT NULL DEFAULT '';
```

`sampling_temperature` is deliberately nullable while every neighbouring column
is `NOT NULL DEFAULT`. "The gateway sent no temperature" and "the gateway sent
0" are different facts, and storing the first as `0` would make an omitted
control indistinguishable from a deterministic one in every later query.

## Retention And Privacy

Route evidence and scan reports are useful because they are inspectable,
but they can also reveal scenario structure. Retention defaults should
be short, export should be explicit, and sensitive content should be
redacted before persistence.

Any future option to persist prompt/response samples must require an
operator decision, a retention window, and clear labeling in the UI and
CLI output.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`INTEGRATIONS.md`](INTEGRATIONS.md)
- [`../internal/SECURITY.md`](../internal/SECURITY.md)
- [`../reference/validation-provider.md`](../reference/validation-provider.md)
