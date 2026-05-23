# Security — Go Code Graph

This document records the scenario's security and privacy posture. Update it before adding auth, user data, external APIs, payment flows, secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

Go Code Graph reads source code from target Go modules. The graph response contains structure (file paths, declaration names, import edges) but not source bodies. However, source-derived identifiers may carry sensitivity that the scenario cannot detect.

| Data | Sensitivity | Owner | Notes |
|---|---|---|---|
| Source code of target module | varies (low to high) | target scenario, not go-code-graph | Source bodies are read in-process by `golang.org/x/tools/go/packages`. They are never persisted by this scenario and never returned in the graph response. |
| Declaration names | varies (typically low) | extracted graph | Names like `apiKey`, `dbPassword`, `internalAPIToken` appear verbatim in node labels. Acceptable for a local operator but would leak if the API were exposed beyond a single-operator install. |
| File paths | varies (typically low) | extracted graph | Absolute paths to source files appear in node `location` fields. Same exposure profile as declaration names. |
| Leading-comment metadata | conditional | extracted graph (P1 method-set coverage) | If/when the Go scenario surfaces leading comments (it doesn't in v1; TS scenario does from day one), credentials accidentally pasted into comments would be exposed. |
| Operation Log (P1) | low | rewrite domain | Records `scenario_path` + normalized ops + per-op status. Same path-exposure profile as declaration names. |
| Plan registry | low | rewrite domain | Same content as Operation Log entries; ephemeral (5-min TTL, in-process). |

## Auth And Authorization

The scenario does not include an auth provider. It is a **local-operator scenario** — accessed only from inside a single Vrooli stack on a single host, on a port assigned by the lifecycle. There is no multi-tenancy and no remote access path.

If go-code-graph is ever exposed beyond a single-host local install (e.g. as a hosted service for cross-org cartographer use), authentication must be added at the API layer before deployment. The CLI and UI must not enforce business authorization locally; authorization belongs at the API/service layer.

## Secrets

| Secret | Source | Required? | Notes |
|---|---|---|---|
| None | n/a | no | go-code-graph does not authenticate to anything, does not make outbound API calls, and does not handle user credentials. Source code being read may *contain* credentials, but those are inputs the scenario should not interpret — see Threat Model below. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Source containing secrets is parsed and surfaced in graph node names | A node label could carry a credential literal (e.g. an exported `const APIKey = "..."`). The graph is shared with consumers (cartographer). | Acceptable for local-operator use because secrets in source are a worse problem than secrets in graph nodes — secrets in source are the upstream bug. Document this in the response: every consumer's display surface should treat node labels as "may contain code-as-data". | accepted-for-local-use |
| Malicious symlinks in target module escape into other filesystem trees | `packages.Load` follows symlinks. A crafted module could symlink to a system directory and surface system files in the graph. | `packages.Config.Dir` is set to the absolute scenario_path; we reject paths outside the user's home directory by default (planned check). | planned mitigation |
| Path traversal via `ImportRewrite.new_path` | A consumer could attempt to write outside the target module by passing a relative path with parent-directory escapes. | `RewritePlan` validates every `new_path` resolves inside the target module's root. Reject otherwise. | required at implementation |
| Path traversal via `FileMove.to` | Same as above for file moves. | `RewritePlan` validates every `to` resolves inside the target module's root. Reject otherwise. | required at implementation |
| Race between plan and apply (TOCTOU) | Source code changes between `RewritePlan` and `RewriteApply`; apply mutates a different graph than was planned. | Apply recomputes the content hash from the current ops and compares to the supplied `plan_id`. Mismatch → `plan_content_mismatch` error. | required at implementation |
| Mid-apply crash leaves working tree corrupted | A panic between op N and op N+1 leaves disk torn. | Accepted by design. Operator recovers via `git restore .`. Documented explicitly in FLOWS.md and PRD. | accepted |
| Denial of service via giant module | A consumer points `Extract` at a huge module (10k+ files) and consumes server resources. | Per-path mutex prevents same-path DoS. Different-path concurrency is bounded by `GOMAXPROCS` in practice. No explicit per-call resource limits in v1. | accepted-for-local-use |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No path-traversal validation on `Rewrite` ops | high | Required before `Rewrite` is implemented. Listed in REQ-P0-003 acceptance criteria. |
| No symlink-escape mitigation in `Extract` | medium | Implement once a real consumer hits the failure mode. Default-deny outside `$HOME` is the planned approach. |
| No remote-exposure auth model | conditional | Required only if the scenario is ever deployed beyond single-host local use. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
