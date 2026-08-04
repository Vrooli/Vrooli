# Validate Space — Validation Concern Coverage

> **Model & terminology** — the projection model, status legend, and how coverage (the numerator)
> is computed are defined once in `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md`.
> This document is the *Validate* denominator only.

## Purpose

The **denominator** for the *Validate* projection: the validation concerns the project should
check, each mapped to its test-genie phase and owning health scenario. The numerator — which
phases are red, the autofix `implemented`-vs-`pending` split, and reliability — is computed live
from `test-genie health` + `test-genie fleet status`, not stored here.

## Target model

The Validate space is evaluated over the repository target inventory, not only
scenarios. Targets use the stable tuple `{kind, id, root}` and are enumerated
from `.vrooli/repo-contract.json`; the current inventory covers scenarios,
resources, tools, safeguards, teams, packages, control-plane code, and docs.
Providers declare the kinds they cover in `.vrooli/test-genie.json`. A missing
declaration, unresolvable target glob, unsupported execution language, or
finding attributed outside declared coverage is a conformance failure.

The two axes are intentionally separate: target coverage answers *what* is
validated, while phase applicability answers *which concern* applies to that
target. Rebase/compare and freshness decisions preserve the target tuple so a
result for `package:api-core` can never satisfy a scenario run by accident.

## This Space

| | |
|---|---|
| Projection | Validate |
| Owner | `test-genie` (owns the phase catalog this extends) |
| Denominator confidence | Covered set: `AUTHORITATIVE` (mirrors the phase catalog). Candidate-missing delta: `SKETCH` (what *else* should be validated is judgment-heavy; full population deferred to the initiative). |
| Sibling spaces | `search-hub/docs/spaces/answer-space.md`, `prompt-manager/docs/spaces/guide-space.md` |
| Legend | `NOW` phase exists + works · `IN-REACH` phase exists but has a substantive coverage gap (missing sub-concern or autofix) · `MISSING` concern not yet a phase · Autofix intent: `mechanical` / `partial` / `manual`. |

## Coverage Grid

### Current phases (the covered set — `AUTHORITATIVE`)

| # | Concern (phase) | Owner (health scenario) | Status | Autofix | Notes |
|---|---|---|---|---|---|
| V1 | Structure | `structure-health` | NOW | mechanical | Skeleton + lifecycle wiring vs `service.json`. |
| V2 | Contracts | `cli-health` | NOW | partial | cli/manifest.json bound to proto descriptors — the SSOT that keeps API/CLI/UI surfaces in agreement (surface-parity is already solved here). |
| V3 | Focused health validators | provider-owned health scenarios | NOW | active | PRD/service.json/proxy/lifecycle rules are moving to focused providers. |
| V4 | Architecture | `architecture-cartographer` | NOW | partial | Structural cohesion; gates only at high-confidence authority. |
| V5 | Dependencies | `scenario-dependency-analyzer` | NOW | partial | Readiness, governance, release-age, security index, graph drift. |
| V6 | Quality | `quality-health` | NOW | mechanical | Lint/type policy, strict config. |
| V7 | DOCS | `knowledge-observatory` | NOW | mechanical | Markdown, mermaid, links, references, manifest. |
| V8 | Unit | `unit-health` | NOW | manual | Execution, coverage, test architecture/quality, flake. |
| V9 | Storage | `storage-manager` | NOW | partial | Schema, migrations, persistence/isolation seams. |
| V10 | Business | _(native — test-genie)_ | NOW | manual | The business-logic ladder: operational targets (`PRD.md`) → requirements (`requirements/`) → linked tests. |
| V11 | Tidiness | `tidiness-manager` | NOW | mechanical | File/function quality. |
| V12 | Security | `security-health` | NOW | partial | Secrets, Go SAST, vuln-DB, JS deps. |
| V13 | Measures | `measures-health` | NOW | partial | Stateful-domain coverage, per-measure tiers. |
| V14 | Proto | `proto-health` | NOW | mechanical | Proto contract validation. |
| V15 | UI Health | `ui-health` | NOW | partial | Manifest/interop/standards + BAS render/handshake. |
| V16 | Performance | `performance-health` | NOW | manual | Go/UI build benchmarks + Lighthouse. |
| V17 | Experience | `ui-health` (UX) + _(native BAS flows)_ | IN-REACH | manual | Reframes "Playbooks". User-flow validation (BAS test cases) is the runtime rung; the UX-spec ladder is the successor (see Known Gaps). |

### Candidate concerns (the denominator delta — `SKETCH`, deferred to initiative)

| # | Candidate concern | Status | Autofix | Notes |
|---|---|---|---|---|
| V18 | Observability / telemetry wiring | MISSING | partial | No dedicated phase; logs/metrics/trace wiring largely unvalidated. |
| V19 | Error-handling conventions | MISSING | partial | Connect error mapping + recovery paths; partly in `ERROR-HANDLING.md`, not gated. |
| V20 | Concurrency / replay safety | MISSING | manual | Race + idempotency hardening; a Guide skill exists, but no validator. |
| V21 | Configuration / env validation | MISSING | mechanical | env vars + `service.json` shape vs declared/used; partly Structure. |

These candidates are illustrative, not authoritative — confirming "what should be validated" is
the Validate-denominator work deferred to the swarm-manager initiative.

## Known Gaps & Approaches

- **Experience validation ladder (the headline gap)** — reframe the `workflow` phase as **Experience** (V17). The business side already has a validation ladder: operational targets (`PRD.md`) → technical requirements (`requirements/`) → linked tests → validated code. The experience side needs the parallel ladder: `DESIGN.md` (high-level UX) → a new per-page UI specification (not yet implemented) → validated via accessibility-tree + screenshot analysis. Current BAS workflow cases are the runtime-flow rung; the per-page UI-spec rung is the successor to build.
- **Autofix `pending` set** — every phase carries an autofix intent (above); `test-genie health` `ProviderConformance.AutofixCoverage` reports the live `implemented` vs `pending` split. The `pending` set is the autofix backlog.
- **Candidate concerns (V18–V21)** — judgment-heavy; the swarm-manager initiative owns deciding which become phases.

## Sources Of Truth

- `test-genie health` (`GetSelfHealth`) — `CatalogSummary` (native vs delegated), `ProviderConformance.AutofixCoverage` (implemented/pending), `ReliabilityLedger`.
- `test-genie fleet status` (`GetFleetHealth`) — most-errored scenarios, finding-source clustering, `never_tested` coverage gaps.
- `test-genie` phase catalog (`api/internal/orchestrator/phases/catalog.go`).

## Cross-References

- `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md` — canonical model + legend.
- `search-hub/docs/spaces/answer-space.md`, `prompt-manager/docs/spaces/guide-space.md` — sibling denominators.
