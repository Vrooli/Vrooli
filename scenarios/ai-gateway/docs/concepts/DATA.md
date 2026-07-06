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
| `route_events` | routing | `api/internal/routing/schema.sql` | route history, UI traces, conformance evidence |
| `provider_snapshots` | inventory | `api/internal/inventory/schema.sql` | role inventory, drift checks |
| `smoke_results` | inventory | `api/internal/inventory/schema.sql` | provider health UI and CLI |
| `profiles` | routing | `api/internal/routing/schema.sql` or policy artifact | route preview/execution |
| `conformance_runs` | conformance | `api/internal/conformance/schema.sql` | test-genie reports |
| `conformance_findings` | conformance | `api/internal/conformance/schema.sql` | operator UI, exports, migration reports |

These are planned data shapes. They should be implemented only when the
owning domain is built and tested.

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
