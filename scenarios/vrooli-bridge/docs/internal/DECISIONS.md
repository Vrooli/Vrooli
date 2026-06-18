# Decisions — Vrooli Bridge

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-18 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-06-18 | **Dial-out connection direction.** Node-agents dial OUT to the control plane and hold a persistent channel; the control plane never dials in. | Nodes are the owner's machines, often behind NAT/firewalls or in another location. | No inbound ports on nodes; firewall/NAT-proof; reuses device-sync-hub's presence model; off-LAN reach via tunnel-manager. This is the industry-standard agent pattern (Tailscale, GitHub Actions runners, Buildkite). | If a future model requires the control plane to initiate (e.g. push to a fully passive node), reconsider — but expect to keep dial-out as the default. |
| 2026-06-18 | **Two trust tiers.** A privileged *provisioning* path is structurally separated from a non-privileged *job runner*. | Remote code execution and remote provisioning are different risk classes. | The everyday runner can never escalate privilege; provisioning is explicit, consented, and audited. More code than one path, but safe by construction. | If provisioning is ever delegated to an external orchestrator. |
| 2026-06-18 | **Allowlist = typed, manifest-declared CLI verbs — never raw shell.** A job is `{scenario, verb, args}` validated against the scenario-CLI manifest plus per-node scopes. | The "run commands on devices" capability is the dangerous part; arbitrary shell would be a standing RCE. | Remote execution is constrained to declared operations; leans on the existing scenario-CLI-manifest work as the allowlist surface. | If a use case genuinely needs an un-declared operation, add it to the manifest — do not add a shell escape hatch. |
| 2026-06-18 | **Node-agent is bridge-owned, not baked into the root `vrooli` CLI.** | Convenience argued for a `vrooli node join` primitive, but that puts one scenario's fleet protocol into the shared platform CLI. | Root CLI stays scenario-agnostic; bridge owns the control-plane↔node protocol end to end; the node side stays thin (hold channel, validate job, call local `vrooli`). | If the platform later defines a generic node-agent the root CLI could host. |
| 2026-06-18 | **A node is a versioned build/test environment, not a dumb runner.** Provisioning brings it to revision R; it builds/tests natively per OS. | You cannot cross-compile a macOS app on Linux and call it "validated on macOS" — codesigning, native deps, and real OS behavior only exist on the OS itself. | Source reaches a node via `git@R`; non-git artifacts via device-sync-hub; nodes need the toolchain (`vrooli setup`). Provisioning is heavier than a thin runner would be. | If a lighter pre-built-artifact smoke path proves sufficient for some gates. |
| 2026-06-18 | **Compose, don't reinvent.** Delegate byte transport (device-sync-hub), durable runs (test-genie), off-LAN reach (tunnel-manager), secrets (secrets-manager), owner auth (scenario-authenticator), audit (workspace-sandbox). | Each concern already has an owner scenario; re-implementing them would create drift and risk. | Bridge is an orchestrator, not a re-implementation; tighter coupling to those scenarios, accepted. | If a dependency proves unfit for bridge's needs, revisit that single seam — not the principle. |
| 2026-06-18 | **One-touch bootstrap; reject pre-trusting arbitrary SSH.** Exactly one manual touch (the installer) on a fresh machine; everything after is remote. | You cannot fully provision a machine with no pre-existing execution channel; standing SSH/WinRM into personal machines is unconstrained by default. | The first touch is unavoidable but bounded; all subsequent provisioning/updates are control-plane-driven and audited. | If a fleet operates in an environment with managed pre-provisioning (cloud-init), an automated first touch could be added. |
| 2026-06-18 | **Single-owner fleet in v1; control-plane-on-Mac/Windows is gated P2.** | Bridge controls the owner's own machines; multi-tenant adds large surface. Control-plane portability depends on Vrooli-the-platform running on Mac/Win. | v1 is single-owner; bridge is written cross-platform from day one so it is never the blocker, but full Mac/Win control plane waits on a separate platform-level track. | When multi-owner demand is real, or when the platform becomes installable on Mac/Win. |
| 2026-06-18 | **SQLite (`api-core/storage`) + proto-versioned node protocol.** Control-plane metadata in SQLite (Postgres-compatible schema); the node↔control-plane wire protocol is proto-versioned with a back-compat policy. | Single-owner fleet does not need a server DB; nodes may run older agents. | Embedded simplicity now, forward-compatible later; a node on an old protocol either interoperates or is flagged needs-update. | If the fleet outgrows single-owner/single-instance. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| 2026-06-18 | The original `vrooli-bridge` scope: inject Vrooli integration docs (CLAUDE.md / VROOLI_INTEGRATION.md) into external code projects to make them "Vrooli-aware." | The fleet control plane described in `PRD.md`. | The doc-injection scenario was stale (vanilla-JS, ~half-built, untouched by dedicated work since Nov 2025) and its value was largely superseded by the skills-publication system + search-hub. The scenario was removed and regenerated fresh from `react-vite` with the fleet-control-plane scope. The evocative name now matches the capability. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
