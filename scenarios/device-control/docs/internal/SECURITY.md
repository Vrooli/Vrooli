# Security — Device Control

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

This scenario has an unusually high security ceiling for its size. It can
tap anything on a personal phone, read anything rendered on its screen, and
install software. Treat the control capability itself as a credential.

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Screen captures (frames, recordings) | **high** | flows | The single highest-risk artifact here. A frame from a personal device can contain messages, authentication codes, tokens, financial detail, or health data — none of which this scenario asked for and all of which it can incidentally record. Redaction status must be verified before a capture leaves the producer; unverified captures are withheld, not displayed. |
| Device logs | high | sessions | Application logs pulled from a device routinely contain tokens and personal identifiers. Same redaction obligation as frames. |
| Clipboard contents | high | sessions | A copy/paste step can move credentials between apps. Clipboard values are never persisted to a run record; only the fact of the transfer is. |
| Verb audit | medium | sessions | Reveals what was done to which device, when, by whom. Not secret, but it is the accountability record and must not be silently truncated. |
| Lease records | medium | sessions | Show who held a device and when. Retained past expiry deliberately. |
| Capability snapshots | low | devices | What a device can do; no user content. |
| Flow definitions | low–medium | flows | Ordinarily benign, but a flow may embed a target string that is itself sensitive (an account name, a search term). Treated as owner data, not shared by default. |
| Strategy registrations | low | strategies | Adapter metadata only. |
| Authentication profiles | high metadata | auth | Device binding, method, policy, and a credential-authority reference only. No credential value, pattern points, screen text, or capture path is stored. |

Lease tokens are bearer credentials. `session acquire` returns the token once;
live-session listings, kill responses, release responses, and audit surfaces
return only the session id and lease metadata. Callers use that id for kill and
release, while flows continue to send the token they received at acquisition.

## Threat Model

| Threat | Mitigation | Status |
|---|---|---|
| A consumer drives a device it was never granted | `vrooli-bridge` owns scopes, allowlisted verbs, and the short-lived device-lease authorization record; this scenario owns lease lifecycle semantics. Device-scoped dispatch is refused and audited unless bridge's held record matches the device and token. | Designed; `DVC-P0-004`. |
| Two consumers silently interleave on one device | Exclusive leases with refusal (not queueing) on contention. Several strategies are physically single-session, so collision would otherwise corrupt evidence rather than error. | Designed; `DVC-P0-004`. |
| A secret is captured in a frame and then distributed | Redaction verified before a capture leaves the producer; consumers receive `common/v1` `EvidenceRef` (checksum, size, kind) and never bytes or filesystem paths. | Designed; `DVC-P0-008`. |
| An agent run goes further than intended | Bounds on step count, cost, and lease scope; abort at any moment; every action audited as a flow step. | Designed; `DVC-P1-005`. |
| A session is left running unnoticed | Live sessions are persistently visible with holder and expiry, leases expire on their own, and kill is one action from CLI or UI. | Designed; `DVC-P0-009`. |
| Screen content leaks to an inference provider | All inference routes through `ai-gateway`, which owns provider policy, privacy class, and route evidence. This scenario holds no provider client and cannot exfiltrate directly. | **Mitigated** — the vision rung uses the generated Connect client and caller-owned downscaling; unavailable routes are typed and never bypass the boundary. |
| Provider secrets leak from this scenario | This scenario stores no provider credential of any kind. Enforced by an AST check, mirroring `ai-gateway`'s conformance rule. | Designed; `DVC-P0-007`. |

## Auth And Authorization

Authorization is two-layer, and neither layer substitutes for the other.

1. **Reach authorization — owned by `vrooli-bridge`.** Whether this caller
   may address a given node or attached device at all: pairing, trust,
   per-node scopes, and the allowlisted verb manifest. This scenario asks;
   it never decides.
2. **Exclusivity — owned by this scenario.** Whether this caller may act on
   that device *right now*, given that someone else may hold it. That is the
   lease (`DVC-P0-004`). For a device-scoped verb, bridge also requires its
   own held lease record to match the device and presented token; an opaque
   token alone is never accepted.

**Key invariant: no verb reaches a strategy without both a bridge-authorized
reach and a held, unexpired lease.** A verb that passes one check and fails
the other is refused, audited, and reported with which check failed.

Authorization is enforced at the API/service layer. The CLI and UI render
the decision; they never make it locally. This matters more than usual here,
because the CLI is the agent-facing control surface — a check that lives
only in the CLI is a check an agent can be steered around.

There is no end-user auth model because there is exactly one owner. If the
fleet ever spans owners, that is a security architecture change rather than
a configuration change; see obstacle 1 in
[`../business/MONETIZATION.md`](../business/MONETIZATION.md).

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| `vrooli-bridge` API token | secrets-manager, injected by the lifecycle | yes | Authenticates dispatch to the reach plane under this scenario's consumer identity. Never logged and never written into evidence. |
| Provider API keys | **not held** | no | This scenario stores no provider credential of any kind. All inference routes through `ai-gateway`, which owns provider policy. Enforced by an AST check (`DVC-P0-007`). |
| ADB device keys | host, `~/.android` | no | Host-level trust between a host node and an attached Android device. Owned by the host; this scenario consumes the resulting reachability, not the key. |
| iOS signing identity | host keychain on the macOS node | no | Needed by `ios-xcuitest` to sign WebDriverAgent. The private key never leaves the node and is never handled by this scenario. |

## Open Security Decisions

## Device authentication boundary

Device unlock is distinct from ADB trust, display power, app login, and human
authentication. Profiles are reference-only rows in device-control SQLite;
Vrooli's credential authority is the sole secret store. Provisioning accepts a
bounded stdin stream, and unlock resolution exists only in memory for one
lease-scoped transaction. The Android adapter maps numeric digits to named
key-code events, never `adb input text`, and clears its temporary buffer.

An unlock is successful only after a fresh keyguard probe reports
`lock_state=unlocked`. A successful ADB command, a dismissed animation, a
foreground package, or a screenshot cannot prove authentication. The default
policy is one attempt with no automatic retry. Provider absent, provider
unavailable, and unconfigured are separate outcomes. Password, pattern,
biometric, and unknown OEM paths fail closed or report `human_required`.

Unlock requests require the same exclusive device lease as every other verb.
Safe audit metadata records profile id, method, provider state, attempts,
outcome, and state transition; it never records the credential reference value
or resolved secret. UI and CLI surfaces show only profile metadata and typed
next actions.

These need a deliberate answer before the first physical-device strategy
ships, not after:

1. **Redaction policy.** What is redacted from a frame by default, and who
   can view an unredacted capture? A default-permissive answer here is how
   a personal device's screen ends up in a shared evidence store.
2. **Unattended agent control.** Should an agent ever hold a device lease
   without a human present? This is a policy question, not a technical one;
   the bounds and audit make it *safe to observe*, not automatically
   *appropriate to allow*.
3. **Grant granularity.** Per-consumer scoped grants — by device, by verb
   class, by time window — versus a single all-or-nothing control grant.

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Redaction policy needs broader owner review | **high** | Before additional consumers or retention policies expose real-device evidence beyond the current verified producer rules. |
| No per-consumer grant granularity | medium | Before any consumer beyond the delivery ramps holds a lease. |
| Unattended agent control unresolved | medium | Before `DVC-P1-005` (agent mode) ships. |
| No visual-understanding route in `ai-gateway` | blocking for `ai.*` only | Declared dependency, not a workaround target. `ai.*` steps report `unavailable` naming it until the gateway capability exists. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
