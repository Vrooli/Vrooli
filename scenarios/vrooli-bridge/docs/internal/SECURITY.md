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

   **Status: built + tested (Phase 3 / G7).** CP→node: the control plane signs every server frame with its long-lived Ed25519 key and wraps it in a `SignedServerFrame` envelope (`api/internal/channelsign`, signature over the exact serialized frame bytes); the agent verifies each frame against the pinned control-plane key (`agent/internal/cpverify`) before acting on it. The pin is written to `<state-dir>/control_plane.pub` (0600, dir 0700) by `vrooli-bridge pair redeem --state-dir …`, **before** the single-use pairing code is burned; a redeem that cannot pin surfaces the key so the operator can pin by hand. At startup a paired agent that finds **no pin fails hard** (`cpverify.ErrNoPin`, no TOFU fallback), and any frame that fails verification is dropped and surfaced in the `rejected_cp_frames` health counter on the agent heartbeat. Node→CP: node calls carry an Ed25519 signature verifiable against the public key stored in `node_credentials` at pairing.
3. **Per-node verb scopes + trust tiers.** Authorization for *what a node will run* is the manifest-validated verb allowlist plus that node's granted verb-namespaces (e.g. `scenario test*` yes, `secrets*` / `scenario deploy*` no). **Trust-tier mechanism (decided 2026-06-18, see `DECISIONS.md`): the runner executes as a dedicated non-privileged service user with no escalation path; a structurally separate privileged provisioning helper (a distinct root/admin process installed at bootstrap) performs only whitelisted provisioning ops over a local IPC the runner cannot forge.** The two tiers are different OS principals, not a flag. The non-privileged runner cannot invoke the privileged provisioning tier. Revocation is atomic and kills both job and provisioning rights immediately.

   New enrollments are additionally **presence-only by default** (`--presence-only=true` / `BRIDGE_PRESENCE_ONLY=true`): the agent continues signed heartbeats but drops even correctly signed job and provisioning frames. Enabling control actions is a separate, explicit policy-approved change; profile selection or self-reported capability cannot enable it implicitly.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| Node mutual-auth credential (Ed25519) | minted at pairing on the node | yes | Public key registered in `node_credentials`; private key stays on the node; destroyed on revoke/re-onboard. |
| Control-plane signing key (Ed25519) | long-lived, control-plane-owned | yes | Private key never leaves the control plane; the **public** half is pinned on each node at redeem. |
| Pairing code | issued out-of-band by owner (`pair issue`), or server-side during one-shot onboarding | yes (onboarding) | Single-use, short TTL; burned on redeem. During onboarding it is injected over SSH **stdin → `BRIDGE_PAIRING_CODE`**, never argv/logs/DB. |
| SSH first-touch password | transient, owner-supplied for one named host | onboarding only | Held in memory for the single key-install dial, then **zeroed**; never written to disk, logs, or the DB. See Transient-Credential Posture. |
| Node-bound secrets (for jobs that need them) | secrets-manager | optional (P2) | Bridge never ships secrets ad hoc; validation jobs prefer fixtures over real secrets. |

### Transient-Credential Posture

Onboarding a fresh node uses two short-lived credentials that are
**deliberately never persisted**:

- **SSH first-touch password.** `api/internal/onboard/ssh.FirstTouch`
  takes an owner-supplied password for one reachable host, uses it for a
  single `ssh-copy-id`-style key-install dial, and zeroes the buffer on
  every exit path (asserted by test). It is never written to disk, logs,
  or the DB — the whole point is to establish key-based passwordless SSH,
  after which the password is worthless. If the host already trusts the
  bridge key, `FirstTouch` short-circuits and **no password is used at
  all**. The generated per-node keypair + `known_hosts` persist 0600
  (dir 0700) under the bridge-owned storage namespace, never the
  operator's `~/.ssh`.
- **Pairing code.** During one-shot onboarding the control plane issues
  the single-use code **server-side** and injects it over SSH **stdin**
  into `BRIDGE_PAIRING_CODE` in the remote shell — never on argv (so it
  cannot leak through `ps`), never echoed, never stored. The `onboard`
  domain has no DB column or step-event field for either the SSH password
  or the pairing code, and tests assert both are absent from DB rows,
  step events, and captured logs.

### Credential Custody Decision (2026-07-14)

Onboarding and privileged provisioning handle several classes of credential;
this records **where each lives and why**, so a future agent does not
accidentally move a secret to weaker custody.

- **Operator SSH password — transient, zeroed, never persisted.** Intake is
  explicit and promptless by default: the CLI reads it from `--password-stdin`,
  an opt-in `--prompt-password` masked TTY prompt, or `$BRIDGE_SSH_PASSWORD` —
  never as a flag value (argv is `ps`-visible) and never via an unrequested
  prompt (unattended runs must not hang). The UI onboard form holds it only in
  ephemeral component state and clears it when the request settles. The
  owner-supplied first-touch password lives only in memory for the key-install
  dial and is zeroed on **every** exit path (asserted by test). It is never
  written to disk, logs, or the DB. The sudo-provisioning work (2026-07-14)
  extended the in-memory **hold window** inside `ssh.FirstTouch` — the password
  is now also used to install the `sudoers.d` drop-in in the same touch — but it
  is still zeroed on all paths and never persisted; the hold is longer, not
  durable.
- **Durable credential = a per-Machine bridge keypair, on-disk at `0600` (dir
  `0700`).** New Machine enrollment selects a stable Machine-scoped key name
  under the bridge-owned trust-store namespace (never the operator's `~/.ssh`).
  The DB stores **only** an opaque `ssh-key://…` reference plus public client-key
  and server-host-key fingerprints — never a filesystem path exposed as an API
  credential and never private key bytes. The filesystem's owner-only permission
  bits are the custody boundary; a trust record is metadata, not the secret.
- **Recorded privilege artifact = the `sudoers.d` drop-in.** The scoped
  passwordless-sudo grant installed at first touch is itself the auditable record
  that the node was elevated; it lives where the OS expects a sudoers fragment
  (under the system `sudoers.d` directory), owned `root:root` `0440` like any
  sudoers file, and is the reviewable trace of the privilege handover.
- **Owner JWT in the browser — `localStorage`, origin-bounded, cleared on
  sign-out or expiry.** The console signs in through the same-origin
  `IdentityService` facade (which forwards to scenario-authenticator and stores
  nothing server-side) and keeps the owner JWT plus display email in
  `localStorage` under `vrooli-bridge.session`, mirroring device-sync-hub. The
  custody boundary is the browser origin — the standard SPA bearer-token
  posture, with XSS as the threat model. The token is short-lived (authenticator
  expiry), attached per request as `Authorization: Bearer`, never logged by the
  control plane, and a `401` on a token-bearing request clears the session and
  returns the console to the sign-in screen. There is deliberately no refresh
  flow: expiry means signing in again.
- **Vault rejected for now — named upgrade path.** A dev-mode, root-token Vault
  on the same host is not stronger than `0600` files and adds a hard runtime
  dependency to onboarding; it becomes the upgrade path only once Vault grows
  real policies, rotation, and an SSH-CA. Until then, resource-vault content KV
  remains the right home for **durable named secrets** (the kopia precedent), and
  third-party API tokens stay in env vars — but the bridge's own onboarding
  credentials do **not** gain from routing through a root-token dev Vault.

**Known residue (accepted).** The SSH key-copy seam materializes the password as
an **immutable Go string** to satisfy the `KeyCopier` interface, so that copy is
**unzeroable until GC** even though `FirstTouch` zeroes its own `[]byte`. This is
accepted residue, not a persisted secret: it is process-memory-only and dies with
the process. The named future fix is changing the `KeyCopier` interface to take a
`[]byte` so the whole path stays zeroable end-to-end.

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Arbitrary remote code execution | An attacker (or a buggy caller) runs unintended commands on a node. | Typed `{scenario, verb, args}` jobs validated against the CLI manifest + per-node scopes; **no raw shell path exists** (the node-agent execs a validated argv, never `sh -c`). | **built + tested (Phase 3)**: `internal/dispatch/allowlist_test.go`, `agent/internal/exec/typedjob_test.go` |
| Privilege escalation via the runner | Everyday job runner gains provisioning/`sudo` rights. | Structural two-tier separation; the runner has no path to the privileged provisioning tier. | **built + tested (Phase 4)**: `agent/internal/privsep/privsep_test.go::TestPrivilegeSeparation_NoCrossImport` proves `internal/exec` and `internal/privsep` share no in-process call path; the service-install adapter installs each tier under a separate OS principal (`agent/internal/service/service_test.go`) |
| Node impersonation / credential theft | A rogue host receives jobs or exfiltrates results. | Node→CP calls carry an Ed25519 signature over the request, verified against the public key registered at pairing; the dial-out token is bound to the node key; credentials hashed at rest; atomic revocation. | **built + tested**: node signing + `node_credentials`; verify path in the node-facing handlers |
| Control-plane impersonation (rogue coordinator) | A node executes attacker-supplied jobs. | CP→node pushes are wrapped in a `SignedServerFrame` (`api/internal/channelsign`, Ed25519 over the exact frame bytes) and verified by the agent against the **pinned** control-plane key (`agent/internal/cpverify`). No pin ⇒ hard startup error (no TOFU fallback); a frame that fails verification is dropped and counted in the `rejected_cp_frames` health signal. | **built + tested (Phase 3 / G7)**: `channelsign`, `cpverify`, `agent/internal/channel` counter |
| Pairing-token replay / rogue node enrollment | Attacker pairs an unauthorized machine. | Single-use, short-TTL, out-of-band pairing code; the code is burned on redeem and never accepted twice; owner-issued via `pair issue`. | **built**: single-use code issuance + redeem |
| MITM on the dial-out channel | Job/results intercepted or altered. | TLS in transit + signed frames both directions (above); off-LAN via tunnel-manager. | **built** (signing) / TLS at the transport edge |
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
| ~~Implementation/tests still pending for each mitigation above~~ **closed for the P0 set + mutual auth** | high → low | Mutual auth is now **built + tested both directions** (Phase 3 / G7: node→CP signing verified against `node_credentials`; CP→node `SignedServerFrame` verified against the pinned key, `rejected_cp_frames` counter, no-TOFU hard-fail). Allowlist + no-shell-path + audit (Phase 3), privilege separation + provisioning (Phase 4), and one-shot onboarding (Phases 1/3/5) ship with tests. Remaining "designed, unbuilt" rows (agent supply-chain checksums, artifact-exfiltration retention controls) are P1/P2 hardening tracked in `requirements/`. |
| **Privileged-helper install divergence (G8 — REMAINS, documented not fixed)** | medium | The two-trust-tier separation is proven **structurally in-process** (AST import-graph test: `internal/exec` ⊥ `internal/privsep`) and the service renderer **supports** installing the privileged helper under a separate OS principal (`--service-user`). But the live `vrooli-bridge-agent service install` / onboarding path installs **only the single non-privileged runner unit** (`systemctl --user` on Linux, a launchd LaunchAgent on macOS) — it does **not** install a separate root/admin provisioning helper as a distinct OS principal. On the live path today, provisioning runs in the same user-level agent context rather than the designed second principal. The design decision (two OS principals, see `DECISIONS.md`) and the in-process proof stand; realizing the separate privileged principal at install time (per-OS root helper install + local IPC boundary) is the open work. Revisit trigger: before a fleet runs untrusted provisioning, or when per-OS privileged-service install is implemented. |
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

## Setup-managed host actions

Bridge firewall remediation crosses a deliberately narrow privilege boundary.
An elevated `vrooli setup` installs the root-owned local privilege broker; the
Bridge API remains unprivileged and owner-authenticates every request before it
contacts the broker. The broker obtains Unix `SO_PEERCRED`, permits only the
setup-selected caller uid, has no TCP listener, and accepts no command/argv
field. Its v1 policy permits only UFW inspect/allow/verify/revoke for scenario
`vrooli-bridge`, port `18767`, and a routable IP bound to the latest durable
failed admission. It writes redacted audit events and supports exact managed
rule rollback. See [`../../../../docs/architecture/PRIVILEGE_BROKER.md`](../../../../docs/architecture/PRIVILEGE_BROKER.md).

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt

## Machine enrollment boundaries

- Pairing codes and passwords are transient. An attempt may retain an opaque
  correlation ID and safe diagnostics, never either secret.
- Every new Machine uses a per-Machine client key. Private bytes remain in the
  trust store; database, API, CLI, UI, audit, and logs see only opaque
  references and public client-key fingerprints.
- SSH server host-key fingerprints and Bridge client-key fingerprints are
  distinct typed values. A changed host key fails closed until an explicit
  review; no retry or locator match may silently accept it. Review clears only
  that Machine's Bridge-owned `known_hosts` pin, then the next strict
  first-touch re-verifies and re-pins the replacement key. It never exposes a
  shell, private key material, or arbitrary known-hosts file operation.
- Observed capabilities can suggest scopes but cannot approve them. Registry is
  the sole owner of approved scopes, and profile selection is never authority.
- A policy transition that removes required capability or suggested-scope intent
  requires an explicit operator confirmation. The confirmation is recorded with
  the immutable policy decision; it never approves or revokes Registry scope.
- Archive Machine, revoke Node, remove SSH access, remove Machine, cleanup
  tombstone, and exceptional purge are separate authorized/audited effects.
  Machine-effect audit records are append-only and contain actor, action, safe
  detail, and timestamp only; they never carry secrets or raw command input.
  Local revocation is durable before remote cleanup; cleanup is `pending`,
  `confirmed`, `not_applicable`, or `abandoned_with_acknowledgement`.
- The existing agent remains a restricted Presence client. It cannot gain a
  shell, provisioning, or scope-approval path from Machine enrollment.
