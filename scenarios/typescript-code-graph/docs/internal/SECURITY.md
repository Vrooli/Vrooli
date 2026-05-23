# Security — TypeScript Code Graph

This document records the scenario's security and privacy posture. Update it before adding auth, user data, external APIs, payment flows, secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

TypeScript Code Graph reads source code from target TS projects. The graph response contains structure (file paths, declaration names, import edges, **leading comments verbatim**) — the leading-comment surface deserves special attention because it can expose data that mere structure extraction would not.

| Data | Sensitivity | Owner | Notes |
|---|---|---|---|
| Source code of target project | varies (low to high) | target scenario, not typescript-code-graph | Source bodies are read by `ts-morph` inside the Node sidecar. They are never persisted by this scenario and never returned in the graph response (except via leading comments). |
| Leading-comment metadata | **conditional, potentially high** | extracted graph | `leading_comments: string[]` is verbatim from source. A developer's `@example` JSDoc snippet showing usage of an internal API may contain credentials or hostnames. This is load-bearing for `react-component-library` (see PRD OT-P0-003) but is the largest privacy surface of this scenario. |
| Declaration names | varies (typically low) | extracted graph | Names like `apiKey`, `dbPassword` appear verbatim. |
| File paths | varies (typically low) | extracted graph | Absolute paths to source files appear in node `location` fields. |
| Operation Log (P1) | low | rewrite domain | Records `scenario_path` + normalized ops + per-op status. |
| Plan registry | low | rewrite domain | Same content as Operation Log entries; ephemeral. |
| Sidecar IPC traffic | varies | sidecar domain | All `Extract` and `Rewrite` payloads flow through the sidecar IPC channel. JSON-over-stdio means the data lives transiently in the Node child's memory. |

## Auth And Authorization

The scenario does not include an auth provider. It is a **local-operator scenario** — accessed only from inside a single Vrooli stack on a single host, on a port assigned by the lifecycle. There is no multi-tenancy and no remote access path.

If typescript-code-graph is ever exposed beyond a single-host local install, authentication must be added at the API layer before deployment. The CLI and UI must not enforce business authorization locally; authorization belongs at the API/service layer.

## Secrets

| Secret | Source | Required? | Notes |
|---|---|---|---|
| None | n/a | no | The scenario does not authenticate to anything and does not make outbound API calls. Source code being read may *contain* credentials, but those are inputs the scenario should not interpret — see Threat Model. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Source containing secrets is parsed and surfaced in graph node names or leading comments | A node label or `leading_comments` entry could carry a credential literal. The graph is shared with consumers (cartographer, rcl). | Acceptable for local-operator use because secrets in source are a worse problem than secrets in graph nodes — secrets in source are the upstream bug. Document this: every consumer's display surface should treat node labels and leading comments as "may contain code-as-data". | accepted-for-local-use |
| Malicious symlinks in target project escape into other filesystem trees | `ts-morph` follows symlinks. A crafted project could symlink into system directories and surface unintended files in the graph. | Sidecar validates that resolved source-file paths are inside the target project's root. Reject otherwise. | required at implementation |
| Path traversal via `ImportRewrite.new_path` | A consumer could attempt to write outside the target project by passing a relative path with parent-directory escapes. | `RewritePlan` validates every `new_path` resolves inside the target project's root. Reject otherwise. | required at implementation |
| Path traversal via `FileMove.to` | Same as above for file moves. | `RewritePlan` validates every `to` resolves inside the target project's root. Reject otherwise. | required at implementation |
| Race between plan and apply (TOCTOU) | Source code changes between `RewritePlan` and `RewriteApply`; apply mutates a different graph than was planned. | Apply recomputes the content hash from the current ops and compares to the supplied `plan_id`. Mismatch → `plan_content_mismatch` error. | required at implementation |
| Mid-apply crash leaves working tree corrupted | A panic or sidecar crash between op N and op N+1 leaves disk torn. | Accepted by design. Operator recovers via `git restore .`. Documented explicitly in FLOWS.md and PRD. | accepted |
| Sidecar IPC channel hijacked by another local process | Local stdio pipes are inherited by the spawned Node child; a different process cannot inject into them. Local Unix sockets (if used) are placed inside a per-user runtime directory with 0600 perms. | Use stdio by default. If/when Unix sockets are introduced, enforce 0600 + parent-dir constraints. | accepted-by-design (stdio) |
| Sidecar code is replaced by a hostile actor | An attacker with local filesystem write access could swap the sidecar binary. | Out of scope. Local filesystem write is already game-over for any scenario. | not-applicable |
| Denial of service via giant project | A consumer points `Extract` at a huge project and consumes server resources. | Per-path mutex prevents same-path DoS. Different-path concurrency is bounded by sidecar concurrency. No explicit per-call resource limits in v1. | accepted-for-local-use |
| Sidecar version mismatch with API | Hot-swap or partial deployment leaves the sidecar at an incompatible version. | Handshake on sidecar startup rejects incompatible versions. `SidecarVersionMismatch` surfaces clearly. | required at implementation |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No path-traversal validation on `Rewrite` ops | high | Required before `Rewrite` is implemented. Listed in REQ-P0-004 acceptance criteria. |
| No symlink-escape mitigation in `Extract` | medium | Implement during sidecar bring-up. Default-deny outside the user's home directory is the planned approach. |
| No remote-exposure auth model | conditional | Required only if the scenario is ever deployed beyond single-host local use. |
| Leading-comment surface privacy not assessed for each consumer | low (local-use) / high (remote-use) | Audit when remote exposure is considered. The verbatim comment contract is load-bearing for rcl. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
