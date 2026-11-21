# Integrations — Security Health

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | validation (scan history), dependencies (structured dep table) | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| qdrant | Vrooli resource | no (try_start) | dependencies | `dependencies.resources.qdrant` (collection `security-health-deps`, dimensions resolved from `embedding.default`, cosine) | Dependency search degrades to TEXT mode; reindex deferred. |
| ollama | Vrooli resource | no (try_start) | dependencies | `dependencies.resources.ollama` (`embedding.default` role) | Embedding unavailable; TEXT-mode search still works. |
| gitleaks | host CLI | present | validation | shelled `gitleaks detect --report-format json --redact`; matches in untracked gitignored files (via `git check-ignore` + `git ls-files`) are downgraded to INFO — they cannot be committed, so they never gate | Absent ⇒ secrets scanner emits INFO observation, not failure. |
| gosec | host CLI | present | validation | shelled `gosec -fmt=json`; issues whose flagged line carries a covering `//nolint`/`//nolint:gosec` directive are dropped — standalone gosec only honors `#nosec`, but the repo lints Go through golangci-lint | Absent ⇒ Go SAST scanner emits INFO observation. |
| govulncheck | host CLI | optional (absent) | validation, dependencies | `go install golang.org/x/vuln/cmd/govulncheck@latest` (install-gated) | Absent ⇒ Go vuln scanner emits INFO observation. |
| osv-scanner | host CLI | optional (absent) | validation, dependencies | `go install github.com/google/osv-scanner/cmd/osv-scanner@latest` (install-gated) | Absent ⇒ CVE scanner emits INFO observation. |
| pnpm audit | host CLI | ships with pnpm | validation, dependencies | shelled `pnpm audit --json` | Absent ⇒ JS deps scanner emits INFO observation. |

## Vrooli Resources

`qdrant` and `ollama` are declared in `.vrooli/service.json` as **optional**
(`required:false`, `startup_policy:"try_start"`) because the `dependencies`
domain must degrade to TEXT search when they are down. They mirror the
cli-health / ui-health AI-search substrate exactly.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| qdrant | active (optional) | Vector index for the fleet dependency intelligence search (collection `security-health-deps`). | Required only if semantic search becomes load-bearing. |
| ollama | active (optional) | `embedding.default` embeddings for the dependency index. | — |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| test-genie | downstream consumer | test-genie's `security` phase shells `security-health validate scenario <name> --json`. | CLI `--json` finding shape (see `validation` proto). |
| ecosystem-manager | downstream consumer | EM consumes `FINDING_SOURCE_SECURITY` at maturity-ladder rung R1. | proto `FindingSource` enum + EM `dimensions.json` map. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | Generated scenario has no third-party dependency. | Add when PRD/requirements require external APIs, webhooks, auth, payments, or data feeds. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
