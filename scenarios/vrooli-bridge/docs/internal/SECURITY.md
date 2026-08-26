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
| Job logs / result artifacts | variable | runs/artifacts | Inherit the sensitivity of whatever the executed command emitted; treated as at least as sensitive as the scenario under test. Node-produced artifacts are bounded, owner-scoped, and stored only for their run. |
| Node metadata | medium | registry | Machine identities, OS/arch/revision, reachable endpoints, permission scopes — operational intelligence about the owner's fleet. |
| Pairing tokens | high (briefly) | pairing | Single-use, short-TTL; a live token can pair a rogue node. |
| Local operator enrollment | high | shared `api-core/operatorsession` contract | The per-user Ed25519 private key is stored owner-only in the local operator-session store; short-lived `OS1` sessions are minted in memory and are never persisted. |
| Break-glass credential | high (briefly) | authenticator/CLI | Issued from owner-only provisioned material into a 0600 token file and sent only as `Authorization: BreakGlass`; it is short-lived, scope-ceilinged and audited. |

## Auth And Authorization

Bridge has three distinct authorization boundaries, all enforced at the API/service layer (UI and CLI never enforce locally):

1. **Owner → control plane.** A machine enrolls once through scenario-authenticator. Bridge stores only the enrollment public-key record and the owner client stores its private key in the local operator-session store. Owner RPCs then use short-lived signed `LocalSession` credentials verified against the Bridge enrollment record, so an enrolled machine does not depend on the authenticator for every request. Only the owner can register/revoke nodes, dispatch jobs, or provision.
2. **Control plane ↔ node (mutual auth).** Every exchange is mutually authenticated — the node proves it is the paired node, and the control plane proves it is the legitimate coordinator (so a node never executes a job from an impostor). **Mechanism (decided 2026-06-18, see `DECISIONS.md`): per-node Ed25519 keypair pinned both directions at pairing.** The node generates an Ed25519 keypair at pairing and registers its public key in `node_credentials`; the control plane holds its own long-lived keypair, and the node pins the control-plane public key at bootstrap (delivered out-of-band alongside the pairing code). Node→CP calls (Connect-RPC: register, heartbeat, run results) carry a signature over the request verifiable against the stored node key; the dial-out SSE token is bound to the node key. CP→node pushes are verifiable against the pinned control-plane key, so a node rejects an impostor coordinator. TLS provides transport confidentiality; the pinned keys provide identity. Full PKI/mTLS-CA was rejected as heavier than a single owner needs.

   **Status: built + tested (Phase 3 / G7).** CP→node: the control plane signs every server frame with its long-lived Ed25519 key and wraps it in a `SignedServerFrame` envelope (`api/internal/channelsign`, signature over the exact serialized frame bytes); the agent verifies each frame against the pinned control-plane key (`agent/internal/cpverify`) before acting on it. The pin is written to `<state-dir>/control_plane.pub` (0600, dir 0700) by `vrooli-bridge pair redeem --state-dir …`, **before** the single-use pairing code is burned; a redeem that cannot pin surfaces the key so the operator can pin by hand. At startup a paired agent that finds **no pin fails hard** (`cpverify.ErrNoPin`, no TOFU fallback), and any frame that fails verification is dropped and surfaced in the `rejected_cp_frames` health counter on the agent heartbeat. Node→CP: node calls carry an Ed25519 signature verifiable against the public key stored in `node_credentials` at pairing.
3. **Per-node execution scopes + trust tiers.** Authorization for *what a node will run* is the manifest-validated typed-verb allowlist intersected with two registry-owned grants: the transport grant `vrooli-bridge:<effect>` and the owning command catalog grant `<namespace>:<effect>`. Catalog grants support `*`, `<namespace>:*`, and `*:<effect>`; command-name grants are rejected. The node's self-reported `capabilities` are evidence only and are never consulted by admission. `secrets` and `scenario deploy` remain absent from the Bridge binding manifest. **Trust-tier mechanism (decided 2026-06-18, see `DECISIONS.md`): the runner executes as a dedicated non-privileged service user with no escalation path; a structurally separate privileged provisioning helper (a distinct root/admin process installed at bootstrap) performs only whitelisted provisioning ops over a local typed IPC boundary the runner cannot forge.** The two tiers are different OS principals, not a flag. Linux and Darwin bind the IPC to kernel-reported peer UIDs. The non-privileged runner cannot invoke the privileged provisioning tier from an arbitrary local process. Revocation is atomic and kills both job and provisioning rights immediately.

   New enrollments hold no execution scopes by default on shared and hosted
   posture. That scope-less state is the presence-only behavior: signed
   heartbeats continue while correctly signed job and provisioning frames are
   refused. `--presence-only` remains an alias that grants no execution
   scopes, and observed capabilities never approve registry grants.

### Trust posture and execution scopes

The installation declares its trust stance in `.vrooli/operator-state.json` as
`trust_posture`. The typed reader defaults a missing field to `personal` and
rejects every other value. Posture is readable by scenario agents but there is
no agent write path; changing it is an operator action and must be recorded as
the typed `trust_posture.transition` event by the control-plane operator
workflow.

The shared defaults table is:

| Posture | Access-token TTL | Break-glass | Default node execution scopes | JWKS cache grace |
|---|---:|---|---|---:|
| `personal` | 60 minutes | available, 15-minute credential | `vrooli-bridge:read`, `vrooli-bridge:write` | 24 hours |
| `shared` | 15 minutes | available, 10-minute credential | none | 4 hours |
| `hosted` | 10 minutes | unavailable | none | 1 hour |

These values tune duration and newly enrolled-node defaults only. Every
posture verifies normal tokens. A cached JWKS is usable only within its
posture-selected grace window; an unavailable authenticator never grants
access.

Break-glass is a separate, positive capability. The protected cleanup flow
provisions an Ed25519
private key with owner-only permissions and pins its public half on the
verifier. The `BreakGlass` authorization scheme verifies the signed,
time-boxed credential offline, requires an explicit scope list, and applies the
account's scope ceiling before issuance. Every accepted use appends the typed
`ActionBreakGlass` audit record; an audit failure refuses the request. It is not
a fail-open branch and cannot be obtained merely by taking the authenticator
offline.

The refusal matrix is intentionally explicit: a target, machine, node, cleanup
scope, frozen plan hash, operation id, audience, pinned key, or lifetime
mismatch is rejected with a named field or typed reason. A valid owner session
does not authorize apply by itself; the operation also requires the opaque
node-bound passphrase envelope and operator confirmation. A node revoked after
planning is re-checked before confirmation and every resume dispatch. The
chosen capability clock-skew tolerance is two minutes; outside it verification
returns the clock-skew reason.

Residual risks are bounded but not eliminated: an already-running child process
cannot be retroactively unwound by a later revocation, and unit tests cannot
provide forensic guarantees against a privileged operating-system memory
inspector. The implementation therefore never puts the passphrase in argv,
environment, durable Bridge state, or emitted events; only the node helper
opens the sealed envelope and zeroes its working buffers.

Enrollment is the only normal path that contacts scenario-authenticator: the
CLI may use its Unix-socket machine-principal exchange or an explicitly
supplied login/token for that one enrollment RPC. The returned provider token
is cleared immediately and is never written to CLI config, operator-state,
browser storage, or Bridge state. Every later CLI and UI request mints an
`OS1` LocalSession locally. Bridge checks enrollment revocation on each request,
and the 15-minute local-session lifetime is the maximum revocation delay for a
session minted before revocation. A rejected credential remains
`unauthenticated`; an authenticator outage during first enrollment is a typed
provider-unavailable diagnosis.

The node registry is the sole owner of approved execution scopes. The
dispatch manifest is derived from the shared CLI governance catalog and then
intersected with those grants; it is not widened by a Go literal or by a
self-reported capability. Interactive sessions now use a separate, signed
session wire contract and owner-authenticated control surface (Mode B); they
are not admitted through the no-shell typed-job allowlist. The no-shell typed
argv path remains Mode A, while session backends remain independently scoped
and auditable.

Presence-only is represented by an empty execution-scope grant. The
`--presence-only` configuration remains a compatibility alias for that empty
grant. On personal posture a newly enrolled node may receive operational and
test scopes; shared and hosted posture receive none by default. Gap G8 stays
open: the live onboarding path has not installed a separate privileged helper
under a second operating-system principal. The implementation now has native
Linux and Darwin peer-UID checks; only host evidence can close the gap.

The reserved `scenario screenshot` verb is a typed Mode-A job. On macOS it
requires an active GUI login/window-server session; an SSH-only launchd user
domain is not treated as screenshot evidence and the run fails closed when no
display is available. The node runs
the operator-configured `vrooli` binary, which selects the platform capture
utility (`screencapture` on macOS) with an explicit output path; no shell or
arbitrary binary dispatch is introduced. Its manifest declares the expected
output name, media type, output flag, and 32 MiB bound. The node resolves the
output path inside its private run directory, uploads the bounded bytes through
the node-authenticated `ArtifactsService.UploadRunArtifact` RPC, and emits the
opaque `bridge://run/<run>/<name>` reference in the run event stream. The owner
retrieves bytes only through the owner-gated `ArtifactsService.GetRunArtifact`
RPC (for example, `vrooli-bridge artifacts get-run`); arbitrary filesystem
paths and arbitrary output flags never cross the dispatch boundary. The
distribution path remains metadata-only and delegates inbound byte transport
to device-sync-hub.

## Secrets

Credential values are never dispatch arguments, dispatch verbs, audit fields,
or node-registration material. Fleet delivery uses an explicit grant and a
typed `CredentialPush` channel frame sealed to the node's independent X25519
encryption key. The Ed25519 node identity authenticates the key-add exchange
but is never converted into an encryption key. Revocation stops future pushes
and purges reachable nodes; an unreachable node may retain a durable copy, so
the operator must rotate the source credential after a lost-node event.

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
- **Durable Bridge trust = metadata plus a Bridge-owned key.** A successful
  `onboard connect` binds exactly one Machine to the target host/account/port,
  an opaque client-key reference and fingerprint, and a verified host-key
  fingerprint. The private key remains only in the Bridge-owned 0700/0600
  state directory. Preflight performs an exact key-only check before
  passwordless reconnect; onboarding performs another key-only check after
  bootstrap and pairing before reporting success. Missing authorization,
  account changes, ambiguous Machine evidence, and changed host keys fail
  closed; recovery requires an explicit password-bearing request and never
  silently creates a replacement identity.
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
  returns the console to the sign-in screen. The authenticator already exposes
  rotating refresh tokens; the bridge identity facade and CLI now expose the
  rotation. The CLI keeps the opaque refresh token in its per-user config,
  never in the bridge API, and retries one expired owner call transparently
  before reporting the original operation's result.
- **CLI password intake — masked TTY or stdin only.** The bridge owner login
  and authenticator account commands do not declare a `--password` value flag;
  interactive use reads a masked terminal prompt and unattended use reads
  `--password-stdin`. Passwords are not placed in argv, environment variables,
  reports, or logs.
- **Credential authority is the ordinary boundary.** A dev-mode, root-token
  Vault on the same host is not stronger than the native credential authority
  or operator-controlled encrypted backup and adds a hard runtime dependency to
  onboarding. Durable named secrets and third-party API tokens therefore use
  the canonical credential authority; the bridge's own onboarding credentials
  are never routed through a root-token Vault.

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
| Privilege escalation via the runner | Everyday job runner gains provisioning/`sudo` rights. | Typed local IPC; the runner sends only `ProvisionCommand`, Linux/Darwin peer credentials bind the caller UID, and the helper alone owns `StepRunner`. | **implemented + unit-tested**: `agent/internal/privsep/ipc_test.go` round-trips typed events and rejects the wrong peer; live two-principal install evidence remains required before this is marked closed. |
| Node impersonation / credential theft | A rogue host receives jobs or exfiltrates results. | Node→CP calls carry an Ed25519 signature over the request, verified against the public key registered at pairing; the dial-out token is bound to the node key; credentials hashed at rest; atomic revocation. | **built + tested**: node signing + `node_credentials`; verify path in the node-facing handlers |
| Control-plane impersonation (rogue coordinator) | A node executes attacker-supplied jobs. | CP→node pushes are wrapped in a `SignedServerFrame` (`api/internal/channelsign`, Ed25519 over the exact frame bytes) and verified by the agent against the **pinned** control-plane key (`agent/internal/cpverify`). No pin ⇒ hard startup error (no TOFU fallback); a frame that fails verification is dropped and counted in the `rejected_cp_frames` health signal. | **built + tested (Phase 3 / G7)**: `channelsign`, `cpverify`, `agent/internal/channel` counter |
| Pairing-token replay / rogue node enrollment | Attacker pairs an unauthorized machine. | Single-use, short-TTL, out-of-band pairing code; the code is burned on redeem and never accepted twice; owner-issued via `pair issue`. | **built**: single-use code issuance + redeem |
| MITM on the dial-out channel | Job/results intercepted or altered. | TLS in transit + signed frames both directions (above); off-LAN via tunnel-manager. | **built** (signing) / TLS at the transport edge |
| Compromised / malicious node | A bad node returns false verdicts or attacks the control plane. | Nodes are the owner's own trusted machines; least-privilege control-plane API; audit of all exchanges; revocation. | partial (trust model) |
| Provisioning tampering | A malicious revision R or setup step. | Provisioning fetches source from the owner's own repo at a pinned revision; privileged, consented, audited; idempotent with rollback. | **built + tested (Phase 4)**: `api/internal/provision/provision_test.go` (audited op lifecycle, idempotent re-sync, rollback outcome) + `agent/internal/privsep/privsep_test.go` (rollback-on-failed-setup, no-shell typed step plan) |
| Node-agent supply chain | Tampered agent binary. | Agent is bridge-built and distributed by the owner; checksums on bootstrap; no third-party binary fetch. | designed, unbuilt |
| Log/artifact exfiltration | Sensitive command output leaks. | Logs inherit run-record access controls + retention; no broadcast; owner-scoped. | designed, unbuilt |

## Security Gaps

These are the remaining security gaps after the implemented bridge foundation.
Each row is either a deliberately deferred capability or a hardening item with
an explicit revisit trigger.

| Gap | Severity | Revisit Trigger |
|---|---|---|
| ~~Mutual-auth mechanism not yet chosen (mTLS vs signed tokens)~~ **RESOLVED 2026-06-18** | ~~high~~ closed | Decided: per-node Ed25519 keypair pinned both directions at pairing (`DECISIONS.md`). Implemented in Phase 2 (pairing/auth). |
| ~~Runner OS-level sandboxing/least-privilege user not yet specified~~ **RESOLVED 2026-06-18** | ~~high~~ closed | Decided: dedicated non-privileged service user runs the runner; structurally separate privileged provisioning helper (`DECISIONS.md`). Implemented in Phase 0 (agent skeleton) + Phase 4 (provisioning helper). |
| ~~Implementation/tests still pending for each mitigation above~~ **closed for the P0 set + mutual auth** | high → low | Mutual auth is now **built + tested both directions** (Phase 3 / G7: node→CP signing verified against `node_credentials`; CP→node `SignedServerFrame` verified against the pinned key, `rejected_cp_frames` counter, no-TOFU hard-fail). Allowlist + no-shell-path + audit (Phase 3), privilege separation + provisioning (Phase 4), and one-shot onboarding (Phases 1/3/5) ship with tests. Remaining "designed, unbuilt" rows (agent supply-chain checksums, artifact-exfiltration retention controls) are P1/P2 hardening tracked in `requirements/`. |
| **Privileged-helper install divergence (G8 — runtime evidence pending)** | high | The code path now installs a distinct `vrooli-bridge-provisioner` machine-wide unit when `BRIDGE_PROVISION_SERVICE_USER` is configured, uses a local typed IPC boundary, and refuses to silently fall back to in-process provisioning. The current environment lacks passwordless elevation and a second OS principal, so the required live process-owner proof has not yet been recorded. This remains open until a real Linux/macOS host reports runner and provisioner owners during one provisioning action. |
| Audit tamper-evidence depends on workspace-sandbox guarantees | medium | The `audit.HTTPStore` workspace-sandbox sink is implemented and selected by `VROOLI_WORKSPACE_SANDBOX_AUDIT_URL`; the local append-only SQLite store remains the fallback. The live write/read evidence from workspace-sandbox is still required. |

## Allowlist

The allowlist gate (OT-P0-004, `internal/dispatch`) is the only path to remote
execution and is enforced in two layers:

1. **Control plane (`dispatch.Allow`)** — a job's `verb` must be in the
   manifest resolved from CLI governance and the target node's granted
   scopes. `scenario deploy` and `secrets …` remain absent unless their source
   governance declares them. The verb must also be free of shell
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
- Archive Machine, revoke Node, record legacy cleanup intent, remove Machine,
  and exceptional purge are separate authorized/audited effects. Remote
  cleanup is a Bridge-managed operation whose frozen plan and receipt are
  separate from the historical tombstone record.
  Machine-effect audit records are append-only and contain actor, action, safe
  detail, and timestamp only; they never carry secrets or raw command input.
  Local revocation is durable before any remote cleanup request; legacy cleanup
  records migrate to terminal `not_applicable` history with an explicit reason.
- The existing agent remains a restricted Presence client. It cannot gain a
  shell, provisioning, or scope-approval path from Machine enrollment.
