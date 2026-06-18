# Security — Vrooli Bridge

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

Bridge's security posture is unusually load-bearing: it executes code and runs privileged provisioning on the owner's machines. Getting this right is the single most important thing about the scenario.

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Node credentials | high | pairing | Mutual-auth secret material; hashed at rest; theft = ability to impersonate a node or the control plane. |
| Audit trail | high (integrity) | audit | Append-only record of every dispatch/provision; must be tamper-evident (workspace-sandbox). |
| Job logs / result artifacts | variable | runs | Inherit the sensitivity of whatever the executed command emitted; treated as at least as sensitive as the scenario under test. |
| Node metadata | medium | registry | Machine identities, OS/arch/revision, reachable endpoints, permission scopes — operational intelligence about the owner's fleet. |
| Pairing tokens | high (briefly) | pairing | Single-use, short-TTL; a live token can pair a rogue node. |

## Auth And Authorization

Bridge has three distinct authorization boundaries, all enforced at the API/service layer (UI and CLI never enforce locally):

1. **Owner → control plane.** Access to the control plane is gated by scenario-authenticator (fail-closed, brief validation cache), consistent with device-sync-hub. Only the owner can register/revoke nodes, dispatch jobs, or provision.
2. **Control plane ↔ node (mutual auth).** Every exchange is mutually authenticated — the node proves it is the paired node, and the control plane proves it is the legitimate coordinator (so a node never executes a job from an impostor). **Mechanism (decided 2026-06-18, see `DECISIONS.md`): per-node Ed25519 keypair pinned both directions at pairing.** The node generates an Ed25519 keypair at pairing and registers its public key in `node_credentials`; the control plane holds its own long-lived keypair, and the node pins the control-plane public key at bootstrap (delivered out-of-band alongside the pairing code). Node→CP calls (Connect-RPC: register, heartbeat, run results) carry a signature over the request verifiable against the stored node key; the dial-out SSE token is bound to the node key. CP→node pushes are verifiable against the pinned control-plane key, so a node rejects an impostor coordinator. TLS provides transport confidentiality; the pinned keys provide identity. Full PKI/mTLS-CA was rejected as heavier than a single owner needs.
3. **Per-node verb scopes + trust tiers.** Authorization for *what a node will run* is the manifest-validated verb allowlist plus that node's granted verb-namespaces (e.g. `scenario test*` yes, `secrets*` / `scenario deploy*` no). **Trust-tier mechanism (decided 2026-06-18, see `DECISIONS.md`): the runner executes as a dedicated non-privileged service user with no escalation path; a structurally separate privileged provisioning helper (a distinct root/admin process installed at bootstrap) performs only whitelisted provisioning ops over a local IPC the runner cannot forge.** The two tiers are different OS principals, not a flag. The non-privileged runner cannot invoke the privileged provisioning tier. Revocation is atomic and kills both job and provisioning rights immediately.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| Node mutual-auth credential | minted at pairing | yes | Hashed at rest; rotated/destroyed on revoke. |
| Pairing token | issued out-of-band by owner | yes (bootstrap) | Single-use, short TTL; burned on redeem. |
| Node-bound secrets (for jobs that need them) | secrets-manager | optional (P2) | Bridge never ships secrets ad hoc; validation jobs prefer fixtures over real secrets. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Arbitrary remote code execution | An attacker (or a buggy caller) runs unintended commands on a node. | Typed `{scenario, verb, args}` jobs validated against the CLI manifest + per-node scopes; **no raw shell path exists** (the node-agent execs a validated argv, never `sh -c`). | **built + tested (Phase 3)**: `internal/dispatch/allowlist_test.go`, `agent/internal/exec/typedjob_test.go` |
| Privilege escalation via the runner | Everyday job runner gains provisioning/`sudo` rights. | Structural two-tier separation; the runner has no path to the privileged provisioning tier. | **built + tested (Phase 4)**: `agent/internal/privsep/privsep_test.go::TestPrivilegeSeparation_NoCrossImport` proves `internal/exec` and `internal/privsep` share no in-process call path; the service-install adapter installs each tier under a separate OS principal (`agent/internal/service/service_test.go`) |
| Node impersonation / credential theft | A rogue host receives jobs or exfiltrates results. | Mutual auth both directions; credentials hashed at rest; atomic revocation. | designed, unbuilt |
| Control-plane impersonation (rogue coordinator) | A node executes attacker-supplied jobs. | Mutual auth — the node verifies the control plane, not just vice-versa. | designed, unbuilt |
| Pairing-token replay / rogue node enrollment | Attacker pairs an unauthorized machine. | Single-use, short-TTL, out-of-band tokens; owner-approved enrollment. | designed, unbuilt |
| MITM on the dial-out channel | Job/results intercepted or altered. | TLS in transit + mutual auth; off-LAN via tunnel-manager. | designed, unbuilt |
| Compromised / malicious node | A bad node returns false verdicts or attacks the control plane. | Nodes are the owner's own trusted machines; least-privilege control-plane API; audit of all exchanges; revocation. | partial (trust model) |
| Provisioning tampering | A malicious revision R or setup step. | Provisioning fetches source from the owner's own repo at a pinned revision; privileged, consented, audited; idempotent with rollback. | **built + tested (Phase 4)**: `api/internal/provision/provision_test.go` (audited op lifecycle, idempotent re-sync, rollback outcome) + `agent/internal/privsep/privsep_test.go` (rollback-on-failed-setup, no-shell typed step plan) |
| Node-agent supply chain | Tampered agent binary. | Agent is bridge-built and distributed by the owner; checksums on bootstrap; no third-party binary fetch. | designed, unbuilt |
| Log/artifact exfiltration | Sensitive command output leaks. | Logs inherit run-record access controls + retention; no broadcast; owner-scoped. | designed, unbuilt |

## Security Gaps

These are open because this is the documentation-first foundation — the threat model is designed but not yet implemented or tested.

| Gap | Severity | Revisit Trigger |
|---|---|---|
| ~~Mutual-auth mechanism not yet chosen (mTLS vs signed tokens)~~ **RESOLVED 2026-06-18** | ~~high~~ closed | Decided: per-node Ed25519 keypair pinned both directions at pairing (`DECISIONS.md`). Implemented in Phase 2 (pairing/auth). |
| ~~Runner OS-level sandboxing/least-privilege user not yet specified~~ **RESOLVED 2026-06-18** | ~~high~~ closed | Decided: dedicated non-privileged service user runs the runner; structurally separate privileged provisioning helper (`DECISIONS.md`). Implemented in Phase 0 (agent skeleton) + Phase 4 (provisioning helper). |
| ~~Implementation/tests still pending for each mitigation above~~ **mostly closed for the P0 set** | high → low | Phase 2 (mutual auth), Phase 3 (allowlist + no-shell-path + audit), and **Phase 4 (privilege separation + provisioning + cross-platform agent)** now ship with tests. Remaining "designed, unbuilt" rows (impersonation, MITM, supply chain, exfiltration) are covered by the Phase-2/3 mutual-auth + TLS + audit machinery; their dedicated regression tests and the P1 hardening (update/rollback, artifact distribution) land in Phase 5. Track in `requirements/`. |
| Audit tamper-evidence depends on workspace-sandbox guarantees | medium | The audit domain is built behind the `audit.Sink` seam (local append-only SQLite store today); a workspace-sandbox-backed Sink is a drop-in once that scenario is green. See PROBLEMS.md (2026-06-18 deferred entry). |

## Allowlist

The allowlist gate (OT-P0-004, `internal/dispatch`) is the only path to remote
execution and is enforced in two layers:

1. **Control plane (`dispatch.Allow`)** — a job's `verb` must be a recognised
   manifest verb (`dispatch.DefaultManifest`: the safe operational/test
   namespaces — `scenario test/build/start/stop/status/logs`; `scenario deploy`
   and `secrets …` are deliberately absent and therefore never dispatchable),
   AND covered by the target node's granted scopes (glob), AND free of shell
   metacharacters in every token. Each failure is a distinct typed rejection
   (`ErrVerbNotInManifest` / `ErrVerbOutOfScope` / `ErrUnsafeToken`), is
   **audited as rejected**, and surfaces as `PermissionDenied` before any run is
   created or anything is pushed.
2. **Node-agent (`agent/internal/exec.BuildArgv`)** — re-validates and builds a
   typed `[]string{bin, tokens...}` argv executed via `os/exec` directly; there
   is no `sh -c` anywhere in the path, and any smuggled metacharacter is rejected
   before execution.

Tests: `internal/dispatch/allowlist_test.go`, `internal/dispatch/service_test.go`,
`agent/internal/exec/typedjob_test.go`.

## Audit

Every dispatch — accepted or rejected — is written to the append-only audit
trail (OT-P0-008, `internal/audit`) as an immutable record (actor, node,
verb/args, outcome, linked run id). The domain exposes only `Sink.Append`
(write, held by dispatch/provision) and `Reader.List` (owner-gated read, the
`AuditService`); there is **no proto RPC, UPDATE, or DELETE** path, so a record
cannot be forged or mutated after the fact. Acceptance auditing is fail-closed:
if a dispatch cannot be recorded, the run is aborted and the dispatch refuses
rather than running un-audited.

Tests: `internal/audit/audit_test.go`,
`internal/audit/sandbox_integration_test.go`, plus the audit assertions in
`internal/dispatch/service_test.go`.

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
