# Security — Ecosystem Manager

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

Ecosystem Manager is an **internal Vrooli control plane**. It generates
and improves scenarios and resources by running agent loops (via
`agent-manager`). Its security profile is unusual: the asset at risk is
**not user data** but the **monorepo itself** — the agents it spawns can
modify any code under the Vrooli tree.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

| Data | Sensitivity | Owner | Notes |
|---|---|---|---|
| Auto-steer profiles (`profiles/*/profile.json`, `metadata.json`) | low | ecosystem-manager | Human-authored, version-controlled config (objective functions). No secrets. |
| Task queue (`queue/<status>/*.yaml`) | low | ecosystem-manager | Execution requests and history; directory name encodes status. |
| Execution state / history (embedded SQLite under the storage data root) | low | ecosystem-manager | Operation runs, metrics, iteration counts. No PII. |
| Agent-applied code diffs | high (integrity) | agent-manager | The actual blast radius. Confidentiality is low; **integrity is the concern** — these mutate the monorepo. Diff capture/review is delegated to agent-manager's sandbox + downstream review, not enforced here. |

## Auth And Authorization

There is **no end-user authentication or authorization today**. Ecosystem
Manager is an internal tool operating under a **local-dev trust model**:
anyone who can reach the API can enqueue tasks and trigger agent loops.

`[CODE: api/pkg/server/server.go]` — when no `AllowedOrigins` are
configured the server defaults CORS to `*`. This is acceptable for local
single-operator development and **must be tightened before any shared or
networked deployment**.

UI and CLI must not enforce business authorization locally; if an auth
model is ever added it belongs at the API/service layer.

## Secrets

| Secret | Source | Required? | Notes |
|---|---|---|---|
| Database credentials | — | no | The runtime store is an embedded SQLite file on local disk; there are no database credentials to provision or store. |
| Agent runner credentials (model/API keys) | agent-manager | yes (for loops) | Owned and injected by `agent-manager`; ecosystem-manager never handles them directly. |
| `CORS_ALLOWED_ORIGINS` | environment | no (defaults to `*`) | Not a secret, but a security-relevant control. `[CODE: api/main.go]` parses it; empty → wildcard. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Runaway / unbounded agent loop mutating code | An auto-steer phase iterates indefinitely, accumulating unsafe or incorrect changes across the monorepo. | Per-phase `max_iterations` in profiles `[CODE: profiles/*/profile.json]`; `stop_conditions` on metrics; bounded concurrency in the queue processor `[CODE: api/pkg/queue/]`. | partial — caps exist, no semantic safety gate here |
| Prompt injection into agent loops | Hostile content in a target scenario's files steers a spawned agent into harmful actions. | Delegated to `agent-manager` sandbox isolation; ecosystem-manager does not sanitize prompt context itself. | deferred |
| CORS too open in a non-local deployment | A browser on an untrusted origin drives the control plane. | Set `CORS_ALLOWED_ORIGINS` explicitly; default `*` is local-dev only. | gap — default is open |
| Unauthenticated task enqueue | Any reachable client triggers expensive/destructive agent runs. | Local trust model; no network exposure expected. | gap — no authN |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No authentication or authorization | conditional | Required before any non-local or multi-operator deployment. |
| CORS defaults to `*` | medium | Before binding the API to any non-loopback interface. |
| No audit/review of agent-applied diffs at this layer | medium | Owned by agent-manager sandbox + downstream review; revisit if ecosystem-manager gains direct apply authority. |
| No semantic safety gate on autonomous changes | high (integrity) | Owned by the gameguard promote-safety gate + agent-manager review; revisit if a richer semantic diff gate is added (see CONTROL-MODEL.md). |

## Cross-References

- [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md) — closed-loop controller model and objective functions
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system structure and the agent-manager seam
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt (authN, CORS, safety gating)
- [`DECISIONS.md`](DECISIONS.md) — durable decisions including REST/JSON transport drift
