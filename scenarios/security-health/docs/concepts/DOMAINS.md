# Domains — Security Health

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

> **Migration note.** The template `notes` worked-example domain has been
> removed (START-HERE Gate 7) from code, proto, and docs now that the real
> domains are green. `vrooli scenario orient` passes `example-domain-removed`.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md). Resource contracts belong in
[`INTEGRATIONS.md`](INTEGRATIONS.md).

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| validation | Detect a target scenario's substrates and run the applicable security scanners, returning normalized findings. | Validation / analysis | Scan history (SQLite). | API, CLI, UI | OT-P0-001..006 (MOD-P0-001..006). | `api/internal/validation/`, `api/handlers/validation/`, `cli/domains/validate/`, `ui/src/features/posture/`, `packages/proto/schemas/security-health/v1/validation/` |
| dependencies | Maintain and serve a fleet-wide, vuln-annotated, semantically-searchable index of every scenario's dependencies. | Search / intelligence | Structured dependency table (SQLite) + Qdrant vectors. | API, CLI, UI | OT-P1-001..002 (MOD-P1-007..008). | `api/internal/dependencies/`, `api/handlers/dependencies/`, `cli/domains/deps/`, `ui/src/features/dependencies/`, `packages/proto/schemas/security-health/v1/dependencies/` |
| reindex | Run async jobs that re-discover lockfiles, re-annotate vuln status, and reconcile the dependency index. | Job / orchestration | Job state (in-memory + SQLite checkpoint). | API, CLI | OT-P1-001 (MOD-P1-007). | `api/internal/reindex/`, `api/handlers/reindex/`, `cli/domains/reindex/`, `packages/proto/schemas/security-health/v1/reindex/` |
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/security-health/v1/health/` |

## Domain Details

### validation

- Purpose: the producer behind test-genie's `security` phase. Detect which
  substrates a scenario contains and run only the applicable scanners
  (gitleaks, gosec, govulncheck, pnpm audit, osv-scanner), returning a
  normalized `Finding{rule_id, severity, title, description, remediation,
  file_path, scanner}` list.
- Primary archetype: validation / analysis.
- Owns: substrate detection, the `Scanner` interface and its per-tool runners,
  severity normalization (critical/high → ERROR, medium → WARNING, low/info →
  INFO), and scan-history persistence.
- Does not own: the fleet dependency index (that is the `dependencies` domain),
  test-genie phase wiring (lives in test-genie), or the EM dimension map.
- API: `ValidationService.ValidateScenario` (`api/handlers/validation/`).
- CLI: `security-health validate scenario <name> --json` (`cli/domains/validate/`).
- UI: `ui/src/features/posture/` — severity-grouped findings + re-scan; the
  Secrets view renders gitleaks findings redacted to file:line.
- Storage: SQLite scan-history table; no shared resource required.
- Requirements: MOD-P0-001 … MOD-P0-006.
- Tests: detector unit tests, per-scanner normalization unit tests, handler
  integration tests against fixture scenarios (clean + planted-secret).
- Related docs: [`FLOWS.md`](FLOWS.md), [`../internal/SEAMS.md`](../internal/SEAMS.md).

### dependencies

- Purpose: the justifying feature — answer "which scenarios are exposed to
  CVE-X?" in one query. Index every scenario's resolved dependencies, annotate
  with known-vuln status, and serve AI (Qdrant + Ollama) search with a TEXT
  fallback plus `--ecosystem` / `--vulnerable-only` / `--name` filters.
- Primary archetype: search / intelligence.
- Owns: the `DependencyRecord` corpus, the discovery walk over go.mod /
  pnpm-lock.yaml, vuln annotation (reusing the validation domain's scanners),
  the Qdrant collection `security-health-deps`, and the embedder/vectorstore
  seams (cloned from cli-health's `aisearch`).
- Does not own: the reconcile job lifecycle (that is the `reindex` domain).
- API: `DependencyService.Search`, `DependencyService.Status`
  (`api/handlers/dependencies/`).
- CLI: `security-health deps search <query> [--vulnerable-only] [--ecosystem]
  [--name]`, `security-health deps status` (`cli/domains/deps/`).
- UI: `ui/src/features/dependencies/` — searchable SBOM with AI/TEXT toggle and
  filters; plus the embeddable posture-badge widget.
- Storage: structured dependency table (SQLite) + Qdrant vectors (dimensions
  resolved from `embedding.default`, cosine); degrades to TEXT when
  Qdrant/Ollama are down.
- Requirements: MOD-P1-007, MOD-P1-008 (and the P1-010 badge surface).
- Tests: lockfile-parser unit tests, search AI/TEXT mode unit tests, filter
  integration tests, status reporting tests.
- Related docs: [`DATA.md`](DATA.md), [`INTEGRATIONS.md`](INTEGRATIONS.md).

### reindex

- Purpose: async reconciliation of the dependency corpus on demand and on a
  5-minute background loop, so the `dependencies` search never blocks on a live
  scan.
- Primary archetype: job / orchestration.
- Owns: job lifecycle (pending → running → succeeded/failed/cancelled),
  dry-run planning (planned upsert/delete counts), and the background
  reconcile loop.
- Does not own: discovery or vuln annotation logic (delegates to the
  `dependencies` domain).
- API: `ReindexService.Reindex`, `ReindexStatus`, `ReindexCancel`
  (`api/handlers/reindex/`).
- CLI: `security-health reindex [--dry-run] [--scenario <name>]`
  (`cli/domains/reindex/`).
- Storage: job state in memory with a SQLite checkpoint.
- Requirements: MOD-P1-007.
- Tests: job-state machine unit tests, dry-run planning tests, cancellation
  idempotency tests.

### health

- Purpose: expose API/database/resource readiness and show the UI can read
  live backend state.
- API: `api/handlers/health/`. CLI: built-in `status` via cli-core.
  UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database + resource reachability.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Substrate | A language/framework stack detected in a scenario (Go, pnpm UI, …) that selects which scanners apply. | `validation` domain. |
| Scanner | A pluggable runner wrapping one external tool behind `Applies(substrate)` + `Run(ctx, dir)`. | `validation` domain. |
| DependencyRecord | One resolved dependency of one scenario, annotated with vuln status. | `dependencies` domain. |
| Finding | A normalized security result with severity, remediation, and location. | `validation` domain; consumed by test-genie. |
| Seam | Test-substitutable boundary wired once in production. | [`../internal/SEAMS.md`](../internal/SEAMS.md). |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| Python / JS-TS SAST runners | No Python scenarios exist; semgrep deferred. Designed as pluggable runners (OT-P2-001). | First Python scenario lands, or semgrep is approved. |
| CVE alerting | Builds on the live dependency index (OT-P2-002). | After the reconcile loop + vuln annotation are proven. |
| Secret rotation workflows | v1 detects + advises only (OT-P2-003). | Demand for guided rotation. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/detect/` — substrate detector lives under the `validation` domain, not a standalone product domain.
- `api/internal/scanners/` — scanner runners are owned by `validation`.
- `api/internal/aisearch/` — embedder/vectorstore/reconciler substrate owned by `dependencies`.
- `api/internal/server/`, `api/internal/module/`, `api/internal/modules/` — HTTP composition / registry substrate.
- `api/internal/database/`, `api/internal/testutil/` — cross-cutting infra.
- `ui/src/components/`, `ui/src/test-utils/` — shared presentation + test support.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts (qdrant, ollama, scanners)
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
