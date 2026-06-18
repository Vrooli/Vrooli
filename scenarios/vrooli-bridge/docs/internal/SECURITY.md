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

<!-- EXAMPLE-DOMAIN:notes START -->
The shipped worked-example `notes` domain carries placeholder data only
(removed by `vrooli scenario detemplate`):

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Template notes data | low | notes reference | Local development data only; replace with real scenario data classification. |
| Attachment bytes | unknown | notes reference | Treat as potentially sensitive if retained in product scope. |
<!-- EXAMPLE-DOMAIN:notes END -->

## Auth And Authorization

Bridge has three distinct authorization boundaries, all enforced at the API/service layer (UI and CLI never enforce locally):

1. **Owner → control plane.** Access to the control plane is gated by scenario-authenticator (fail-closed, brief validation cache), consistent with device-sync-hub. Only the owner can register/revoke nodes, dispatch jobs, or provision.
2. **Control plane ↔ node (mutual auth).** Every exchange is mutually authenticated — the node proves it is the paired node, and the control plane proves it is the legitimate coordinator (so a node never executes a job from an impostor). The concrete mechanism (mTLS vs signed-both-ways tokens) is a near-term decision recorded in `DECISIONS.md` once chosen.
3. **Per-node verb scopes + trust tiers.** Authorization for *what a node will run* is the manifest-validated verb allowlist plus that node's granted verb-namespaces (e.g. `scenario test*` yes, `secrets*` / `scenario deploy*` no). The non-privileged runner cannot invoke the privileged provisioning tier. Revocation is atomic and kills both job and provisioning rights immediately.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| Node mutual-auth credential | minted at pairing | yes | Hashed at rest; rotated/destroyed on revoke. |
| Pairing token | issued out-of-band by owner | yes (bootstrap) | Single-use, short TTL; burned on redeem. |
| Node-bound secrets (for jobs that need them) | secrets-manager | optional (P2) | Bridge never ships secrets ad hoc; validation jobs prefer fixtures over real secrets. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Arbitrary remote code execution | An attacker (or a buggy caller) runs unintended commands on a node. | Typed `{scenario, verb, args}` jobs validated against the CLI manifest + per-node scopes; **no raw shell path exists**. | designed, unbuilt |
| Privilege escalation via the runner | Everyday job runner gains provisioning/`sudo` rights. | Structural two-tier separation; the runner has no path to the privileged provisioning tier. | designed, unbuilt |
| Node impersonation / credential theft | A rogue host receives jobs or exfiltrates results. | Mutual auth both directions; credentials hashed at rest; atomic revocation. | designed, unbuilt |
| Control-plane impersonation (rogue coordinator) | A node executes attacker-supplied jobs. | Mutual auth — the node verifies the control plane, not just vice-versa. | designed, unbuilt |
| Pairing-token replay / rogue node enrollment | Attacker pairs an unauthorized machine. | Single-use, short-TTL, out-of-band tokens; owner-approved enrollment. | designed, unbuilt |
| MITM on the dial-out channel | Job/results intercepted or altered. | TLS in transit + mutual auth; off-LAN via tunnel-manager. | designed, unbuilt |
| Compromised / malicious node | A bad node returns false verdicts or attacks the control plane. | Nodes are the owner's own trusted machines; least-privilege control-plane API; audit of all exchanges; revocation. | partial (trust model) |
| Provisioning tampering | A malicious revision R or setup step. | Provisioning fetches source from the owner's own repo at a pinned revision; privileged, consented, audited; idempotent with rollback. | designed, unbuilt |
| Node-agent supply chain | Tampered agent binary. | Agent is bridge-built and distributed by the owner; checksums on bootstrap; no third-party binary fetch. | designed, unbuilt |
| Log/artifact exfiltration | Sensitive command output leaks. | Logs inherit run-record access controls + retention; no broadcast; owner-scoped. | designed, unbuilt |

## Security Gaps

These are open because this is the documentation-first foundation — the threat model is designed but not yet implemented or tested.

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Mutual-auth mechanism not yet chosen (mTLS vs signed tokens) | high | Decide before the pairing/auth domain is implemented; record in `DECISIONS.md`. |
| Runner OS-level sandboxing/least-privilege user not yet specified | high | Specify before the node-agent runner is built. |
| No implementation/tests yet for any mitigation above | high | Each mitigation must ship with tests (see `requirements/`). |
| Audit tamper-evidence depends on workspace-sandbox guarantees | medium | Confirm workspace-sandbox provides the needed append-only/integrity properties. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
