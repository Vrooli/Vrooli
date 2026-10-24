# Domains — Quality Health

## Purpose Of This Document

This document names the bounded contexts that replaced the generated example domain and now back the API, CLI, and UI.

## Domain Inventory

| Domain | Purpose | Primary OT | Surfaces | Source Paths |
|---|---|---|---|---|
| surfaces | Convert Code Facts output into Quality Health's normalized surface inventory. | OT-P0-001 | API, tests, UI | `api/internal/surfaces`, `api/handlers/audit`, `ui/src/features/audit` |
| rules | Register static-quality rules by language/framework/surface/tooling, including fix class and evaluator seams. | OT-P0-002, OT-P0-006, OT-P0-007 | API, tests | `api/internal/rules`, `api/internal/audit`, `api/internal/autofix` |
| contracts | Expose contract packs derived from the rule registry. | OT-P0-002, OT-P0-003 | API, CLI list, UI | `api/internal/contracts`, `api/handlers/audit`, `cli/domains/contracts`, `ui/src/features/audit` |
| audit | Orchestrate discovery, contract evaluation, command execution, findings, maturity, and next steps. | OT-P0-001, OT-P0-004 | API, CLI, UI | `api/internal/audit`, `api/handlers/audit`, `cli/domains/audit`, `ui/src/features/audit` |
| commands | Resolve and run bounded lint/type commands, returning structured command results. | OT-P0-001 | API, UI | `api/internal/commands`, `ui/src/features/audit` |
| autofix | Preview and apply deterministic config edits for supported rules. | OT-P0-005 | API, CLI, UI | `api/internal/autofix`, `api/handlers/audit`, `cli/domains/autofix`, `ui/src/features/audit` |
| explain | Return detailed remediation for stable finding IDs. | OT-P1-003 | API, CLI, UI | `api/internal/contracts`, `api/handlers/audit`, `cli/domains/explain`, `ui/src/features/audit` |

## Domain Details

### surfaces

Owns the Code Facts client seam and the `QualitySurface` model. It should request surface, language, framework, package-manager, root, and parse-unit facts. If Code Facts is unavailable, it returns degraded inventory status; it must not claim a clean pass based on heuristic discovery.

### rules

Owns rule definitions, applicability, severity, expected evidence, fix class, and evaluators. Rules are **language-first**: applicability is keyed on language (primary), with framework/surface-kind/tooling as optional narrowing — never on surface name or scenario template. `audit` dispatches through `rules.SurfaceRules` and `rules.ScenarioRules`; `contracts` groups the same registry into operator-facing packs. A discovered surface matching no rule is reported `uncovered` with a `QUALITY_COVERAGE_GAP` info finding — never a clean pass — and caps maturity at L2.

### contracts

Owns the contract-pack read model for API/CLI/UI consumers. The registry ships three packs derived from rule `ContractID` values: `typescript-static-quality` (any ts/js surface), `go-static-quality` (any Go surface), and `scenario-quality-gates` (the scenario root).

### audit

Owns `AuditQuality`. It composes surfaces, contracts, optional command execution, optional autofix preview, finding normalization, summary counts, next steps, degraded reasons, and maturity.

### commands

Owns command resolution and bounded execution. It should never run unbounded shell commands. Results include command, working directory, timeout, exit code, stdout/stderr excerpts, status, and failure reason.

### autofix

Owns deterministic config edit planning. v1 mutates only supported config files when `--apply` is explicit. Source suppression edits are out of scope. Preview must show the files and hunks that would change. Finding-level `autofix_available` is true only when this domain can preview a safe candidate for the specific finding.

### explain

Owns stable finding lookup and remediation explanations. It can derive explanations from the contract registry and the latest audit context; persistent history is optional in v1.

## Shared Concepts

- `QualitySurface`: discovered target with ID, kind, language, framework, root, package manager, and confidence.
- `QualityContract`: applicable rule pack keyed by surface metadata.
- `QualityFinding`: stable, agent-readable violation.
- `MaturitySummary`: deterministic L0-L5 quality rung.
- `AutofixCandidate`: safe config edit with dry-run/apply state.

## Deferred Domains

- run history and trend analysis,
- fleet dashboard,
- expanded Python and non-Vite Node contracts,
- suppression growth analysis over time.

## Non-Domains

- Unit test execution belongs outside Quality Health.
- Maintainability debt belongs to Tidiness Manager.
- General standards belong to Scenario Auditor unless they are static-quality contracts.

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [FLOWS.md](FLOWS.md)
- [DATA.md](DATA.md)
- [SEAMS.md](../internal/SEAMS.md)
