# Domains — Template Manager

Template Manager is a meta-scenario: it owns the template lifecycle as a reusable Vrooli capability rather than as scattered vrooli CLI code. This map names the bounded contexts that will own API, CLI, UI, storage, validation, and test-genie behavior as the plan lands.

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Source Paths |
|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Prove the scenario process and store are reachable. | No product data. | reporting | query | `api/handlers/health/`, `ui/src/features/health/` |
| registry | Catalog scenario templates, design kits, and resource templates. | Provide the authoritative inventory every lifecycle operation and dashboard uses. | Template records, kind/version metadata, manifest hashes. | catalog | service | `api/internal/registry/`, `cli/domains/registry/`, `ui/src/features/registry/` |
| validation | Run and record template validation, drift, and version-lag checks. | Turn throwaway validation output into durable evidence. | Validation runs, phase results, findings, drift snapshots, version lag. | workflow | runner | `api/internal/validation/`, `cli/domains/validation/`, `ui/src/features/validation/` |
| debt | Track stable inherited-template defects and remediation state. | Make template quality measurable and drive recurring debt burn-down. | Debt entries, status transitions, defect keys, source links, history. | ledger | reporting | `api/internal/debt/`, `cli/domains/debt/`, `ui/src/features/debt/` |
| phase-provider | Serve test-genie's `templates` phase. | Report template standing for every scenario suite without triggering deep validation. | Provider standing cache only if needed; primary source is target scenario metadata plus registry state. | integration | validation-provider | `api/handlers/validation/`, `.vrooli/test-genie.json` |
| guidance | Evaluate orientation gates and return next work-order data. | Let small-model execution agents act on structured gates rather than START-HERE prose. | Gate definitions, check results, remediation pointers. | advisory | evaluator | `api/internal/guidance/`, `cli/domains/guidance/` |
| docs | Serve and index factory documentation. | Make template maintenance, validation, drift, and migration protocols discoverable. | Search provider metadata and doc registry rows. | knowledge | index | `docs/`, `.vrooli/search.json` |
| monitor | Schedule recurring deep validation and expose monitor status. | Force the template improvement loop to run without waiting for ad hoc generation. | Schedule settings, last/next run, green streak, in-flight state. | scheduler | operations | `api/internal/monitor/`, `cli/domains/monitor/` |
| engine | Own generation, orientation, detemplate, drift, cleanup, design-kit, and resource-template execution after cutover. | Remove the split-brain between templates and the vrooli CLI. | No independent business data; writes generated scenario/resource files and records run evidence through validation/debt domains. | orchestrator | migration | `api/internal/templateengine/`, `cli/domains/template/`, `cli/domains/resource-template/` |

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Template record | A scenario template, design kit, or resource template known to Template Manager. | registry |
| Validation run | A shallow/deep validation or drift execution with parsed findings and provenance. | validation |
| Debt entry | A stable, deduplicated defect or contract gap inherited from template content. | debt |
| Template standing | Static per-scenario assessment reported into test-genie. | phase-provider |
| Orientation gate | A declarative unit of scenario initialization work with machine-checkable criteria. | guidance |
| Engine operation | A lifecycle command such as generate, orient, detemplate, validate, drift, cleanup, or resource-template generate. | engine |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| entitlement | Template Manager is a platform meta-capability, not a paid product surface in Phase 1. | A future bundle decision needs metered/gated template operations. |
| collaboration | Debt assignment and review workflows can reuse existing issue/record systems for now. | Operators need multi-user triage state inside Template Manager. |

## Non-Domains

- `api/internal/server/`, `api/internal/module/`, and `api/internal/modules/` remain scenario substrate.
- `ui/src/components/` and `ui/src/test-utils/` remain shared UI/testing infrastructure.
- `templates/` remains the content home; Template Manager owns handling and governance, not a content relocation.
- The old vrooli CLI template/design/resource handlers are temporary source material until the cutover phases move them.

## Cross-References

- [`DATA.md`](DATA.md) — persisted records and ownership
- [`FLOWS.md`](FLOWS.md) — lifecycle workflows and scheduler behavior
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — scenario and platform dependencies
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — implementation seams as they land
