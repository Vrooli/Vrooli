# Domains — API Health

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

API Health is a provider scenario. The `validation` domain is the
keystone: it coordinates target discovery, rule execution, assessment
mapping, and the shared Test Genie provider surface. The `probe` and
`remediation` domains add execution evidence and deterministic repair
without changing the validation boundary.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report API Health's own runtime readiness and dependency reachability. | Keep the provider observable through the standard scenario health surface. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/api-health/v1/shared/health.proto` |
| validation | Resolve a target scenario, discover API applicability, run static API readiness checks, map findings into maturity capabilities, and serve the shared validation RPC. | Give Test Genie and agents one authoritative API-readiness verdict. | No persistent data; reads target scenario trees. | provider | validation, reporting | Target, Capability, Finding, Assessment, NativeDetail | `api/internal/validation/`, `api/handlers/validation/`, `cli/domains/validate/`, `packages/proto/schemas/api-health/v1/validation/` |
| probe | Execute bounded live API health probes through lifecycle-discovered URLs and return timestamped evidence. | Prove `/health` works at runtime without turning API Health into a load tester. | Optional probe history if enabled; otherwise evidence is response-local. | service | validation, reporting | Probe, HealthPayload, Evidence, Timeout | `api/internal/probe/`, `api/handlers/probe/`, `cli/domains/probe/`, `packages/proto/schemas/api-health/v1/probe/` |
| remediation | Preview and apply deterministic fixes for unambiguous API Health findings through the shared Fix RPC. | Reduce fixable API-readiness debt without asking agents to hand-edit boilerplate. | No product data; edits target scenario trees only on explicit apply. | mutation | service | FixCandidate, Preview, Apply, Idempotency | `api/internal/autofix/`, `api/handlers/validation/`, `cli/domains/fix/` |
| migration | Maintain the scenario-auditor API-rule migration ledger and parity-or-better fixtures. | Retire legacy API rules deliberately, with every old rule accounted for. | No persistent data; fixture/golden files only. | reporting | validation | LegacyRule, Decision, Parity | `api/internal/migration/`, `docs/reference/scenario-auditor-api-migration.md` |
| workbench | Present capability summaries, findings, live probe evidence, and fix previews in the UI. | Make API readiness understandable to operators and agents. | Browser state only. | reporting | query | Workbench, FindingTable, ProbePanel, FixPreview | `ui/src/features/validation/`, `ui/src/api/validation.ts` |

## Domain Details

### health

- Purpose: expose API Health's own provider readiness.
- Primary archetype: reporting / query.
- Owns: health response construction and dependency status mapping.
- Does not own: target scenario API validation.
- API: `api/handlers/health/`.
- CLI: built-in `api-health status`.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none.
- Requirements: scaffold/provider self-health only.

### validation

- Purpose: one provider verdict for target API readiness.
- Primary archetype: provider / validation.
- Owns: target resolution, API surface classification, lifecycle checks,
  HTTP semantics checks, runtime hygiene checks, finding normalization,
  provider-native detail, and assessment construction.
- Does not own: adjacent provider policy. Security headers remain in
  security-health; lint/type policy remains in quality-health; CLI/proto/UI
  contracts remain in their existing providers.
- API: shared `ScenarioValidationService.ValidateScenario` plus planned
  native validation service for UI/CLI detail.
- CLI: `validate scenario <target>`.
- Requirements: `APIH-PROV-*`, `APIH-LIFE-*`, `APIH-HTTP-*`,
  `APIH-RUN-*`.

### probe

- Purpose: a bounded, single-operation live check of the target API health
  endpoint when validation is asked to include execution.
- Primary archetype: service / validation.
- Owns: lifecycle URL resolution, one-shot HTTP client timeout, response
  capture, schema validation, and native evidence payload.
- Does not own: starting processes directly, retry loops, performance
  benchmarking, or product-specific endpoint assertions.
- CLI: `probe health <target>`.
- Requirements: `APIH-HEALTH-*`.

### remediation

- Purpose: deterministic repair of findings that are local and
  unambiguous.
- Primary archetype: mutation / service.
- Owns: fixer registry, dry-run preview, explicit apply, and idempotency
  checks.
- Does not own: deciding to redesign an API path, logging vocabulary,
  timeout budget, or context propagation strategy.
- CLI: `fix preview <target>`, `fix apply <target>`.
- Requirements: `APIH-FIX-*`.

### migration

- Purpose: preserve the useful intent of scenario-auditor API rules while
  rejecting brittle implementation details.
- Primary archetype: reporting / validation.
- Owns: the migration ledger and parity fixtures.
- Does not own: runtime dependency on scenario-auditor.
- Requirements: `APIH-MIG-001`.

### workbench

- Purpose: operator and agent UI for the validation report.
- Primary archetype: reporting / query.
- Owns: capability summary, findings table, live probe evidence display,
  and fix preview display.
- Does not own: independent validation logic.
- Requirements: `APIH-UX-001`.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| API readiness | The platform-facing confidence that a scenario API can start, report health, answer HTTP consistently, and shut down cleanly. | API Health validation domain. |
| Live probe | A bounded one-shot HTTP check of a lifecycle-discovered health URL. | probe domain. |
| Capability | A maturity ladder area declared in `.vrooli/maturity.json`. | validation domain plus `packages/maturity-go`. |
| Finding | A normalized provider issue mapped to one capability and one fixability declaration. | validation domain. |
| Fix candidate | A deterministic preview/apply edit for one finding. | remediation domain. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| fleet | Fleet API-readiness ranking is useful but not needed for first provider viability. | P2 target OT-P2-001 starts. |
| fact-export | Cross-provider API-surface facts may help other providers, but schema needs consumers first. | quality-health/security-health/performance-health asks for shared API facts. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — provider HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — local provider database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using API Health vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
