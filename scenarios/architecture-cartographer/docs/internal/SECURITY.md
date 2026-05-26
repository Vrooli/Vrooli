# Security — Architecture Cartographer

This document records the scenario's security and privacy posture.
The cartographer is a developer-facing analysis tool that reads
source code from target scenarios. Most security risk is around what
the cartographer learns about that code and what it does with that
knowledge, not network-attack surfaces.

## Purpose Of This Document

Use this document to answer:

- What sensitive data does the cartographer touch?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

| Data | Sensitivity | Owner | Notes |
|---|---|---|---|
| Source code of target scenarios | varies (low to high) | target scenario, not cartographer | Cartographer reads source code via `go-code-graph` / `typescript-code-graph`. Bodies may contain credentials, hostnames, customer identifiers, or other regulated data if the scenario maintainer has been careless. |
| Graph snapshots | structural metadata only | `graph` domain | File paths, package names, symbol names, import edges. No source bodies persisted. |
| Conflict evidence | structural metadata + small source excerpts | `conflicts` domain | When an agent runs `arch-cart conflicts show <id>`, the response includes the source range for the conflict location. Streamed to the requesting client; not persisted in analytics. |
| Analytics event log | structural identifiers only | `analytics` domain | File paths and symbol names appear; never source bodies. |
| Manifest contents | low | `manifest` domain | Manifest is part of the target scenario; cartographer caches parsed forms but does not own the source of truth. |
| Apply git commits | depends on what was committed | `apply` domain | Cartographer authors commits with operation lists. Commit messages do not include source content; commit content is whatever the file-move/import-rewrite operations produced. |
| Template notes data | low (placeholder) | notes reference | Local development data only; removed when notes domain is deleted per Gate 7. |

## Auth And Authorization

The cartographer is a local developer tool. It does not require auth
in v1 because:

- The API listens on localhost only via the scenario lifecycle.
- The CLI runs as the local user against the local API.
- Access to the target scenario directory is controlled by the host
  filesystem, not by cartographer.

If the cartographer is ever exposed over a network (e.g., a shared
team server in a future tier), this must change:

- Authenticate the API surface (likely via cli-core's standard
  scenario-API auth pattern).
- Authorize per-target-scenario access (an agent on one team should
  not be able to extract another team's graph).
- Surface this requirement loudly in the deployment doc and in
  `.vrooli/service.json`.

UI and CLI must not enforce authorization locally; if/when auth
arrives, authorization decisions are made at the API/service layer.

## Secrets

| Secret | Source | Required? | Notes |
|---|---|---|---|
| None in v1 | n/a | no | Cartographer has no third-party APIs or paid services. No `OPENAI_API_KEY`, no cloud credentials. |
| Git credentials | Inherited from host | conditional | `git-co-edit` signal and `apply` commits use whatever credentials the host has configured. Cartographer never reads, stores, or transmits credentials. |
| Future Ollama/Qdrant tokens | Local resource | conditional (P2) | Only required if the embedding suggestion ranker (OT-P2-001) is ever built. Tokens would come from the local Vrooli resource configuration, not from cartographer-specific secrets. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Source code leakage via analytics export | Sensitive source could leak into shared `arch-cart analytics events export` outputs. | Analytics never stores source bodies; only structural identifiers. Exports run through a redaction pass that strips `Evidence` values longer than `N=200` characters by default. Documented in [`../concepts/DATA.md`](../concepts/DATA.md) under Privacy Notes. | active |
| Source code leakage via conflict show | `arch-cart conflicts show <id>` returns source ranges. If the agent forwards that output unredacted to an external system, source could leak. | CLI output goes to local stdout. Documented contract that conflict-show output is for human/agent consumption only; not for third-party transmission. | documented |
| Malicious manifest injecting unsafe paths | A crafted manifest could declare a domain path that traverses outside the scenario root (e.g. `../..` segments aimed at host-system files) to trick rewrite operations into touching files the user never intended. | Manifest schema validation rejects paths that escape the scenario root; `apply` operations are sandboxed to the target scenario directory. | active |
| Apply-step running on broken baseline | Agent could `--force` an apply that breaks the build, silently shipping broken code. | `--force` requires `--note` and is logged in analytics. CI integration should run `prd-control-tower prd validate` (or equivalent post-apply check) against finalized migrations. | active (logging); CI integration TBD |
| `go build` / `tsc` executing target-scenario code via init/codegen | Build verification invokes the target scenario's toolchain. A malicious target scenario could run arbitrary code during build. | This is the same trust boundary as any developer running `go build` on a checked-out scenario; not a new threat introduced by cartographer. Documented so it is not surprising. | accepted |
| Resource exhaustion on huge scenarios | A scenario with millions of files could DoS the cartographer (memory, build time). | Performance budgets in [`PERFORMANCE.md`](PERFORMANCE.md) bound graph extraction time; the CLI surfaces a graceful failure message above thresholds. | bounded |
| Git operations on wrong branch | `apply` commits could land on `main` accidentally. | `apply` checks the current branch; refuses on `main`/`master` unless `--allow-protected-branch --note`. Documented in [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md). | active |
| Cycle in cartographer's own architecture | The cartographer is built using the same rules it enforces. Drift would be acutely embarrassing. | Cartographer runs `arch-cart` against its own scenario in CI once apply ships; cycles or forbidden edges in its own code fail the build. | planned (post-MVP) |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No CI gate for source leakage in `arch-cart analytics events export` | medium | Add a redaction-pass unit test and a CI policy that exports never include `Evidence` strings above the length threshold. |
| No `--allow-protected-branch` flag implemented yet | medium | Implement before any apply path lands; default behavior must refuse main/master without explicit acknowledgment. |
| Dogfooding gate (cartographer validates its own architecture) not enforced | low (until v1 ships) | Add as a CI step in Stage 7 of the launch sequence. |
| Network-exposed cartographer scenario | high (when applicable) | Required if cartographer is ever deployed beyond a single developer machine. Add auth, authorization, and audit logging first. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and privacy notes
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and trust boundaries
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PERFORMANCE.md`](PERFORMANCE.md) — resource exhaustion bounds
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) — operational guidance including protected branches
