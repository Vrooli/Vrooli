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

## Threat Model

| Threat | Mitigation | Status |
|---|---|---|
| A consumer drives a device it was never granted | `vrooli-bridge` owns scopes and allowlisted verbs; this scenario refuses any verb without a bridge-authorized reach *and* a held lease. | Designed; `DVC-P0-004`. |
| Two consumers silently interleave on one device | Exclusive leases with refusal (not queueing) on contention. Several strategies are physically single-session, so collision would otherwise corrupt evidence rather than error. | Designed; `DVC-P0-004`. |
| A secret is captured in a frame and then distributed | Redaction verified before a capture leaves the producer; consumers receive `common/v1` `EvidenceRef` (checksum, size, kind) and never bytes or filesystem paths. | Designed; `DVC-P0-008`. |
| An agent run goes further than intended | Bounds on step count, cost, and lease scope; abort at any moment; every action audited as a flow step. | Designed; `DVC-P1-005`. |
| A session is left running unnoticed | Live sessions are persistently visible with holder and expiry, leases expire on their own, and kill is one action from CLI or UI. | Designed; `DVC-P0-009`. |
| Screen content leaks to an inference provider | All inference routes through `ai-gateway`, which owns provider policy, privacy class, and route evidence. This scenario holds no provider client and cannot exfiltrate directly. | **Blocked** — `ai-gateway` has no visual-understanding request kind yet. `ai.*` steps stay `unavailable` until it exists; see `INTEGRATIONS.md`. |
| Provider secrets leak from this scenario | This scenario stores no provider credential of any kind. Enforced by an AST check, mirroring `ai-gateway`'s conformance rule. | Designed; `DVC-P0-007`. |

## Open Security Decisions

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

<!-- EXAMPLE-DOMAIN:notes START -->
The shipped worked-example `notes` domain carries placeholder data only
(removed by `template-manager detemplate`):

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Template notes data | low | notes reference | Local development data only; replace with real scenario data classification. |
| Attachment bytes | unknown | notes reference | Treat as potentially sensitive if retained in product scope. |
<!-- EXAMPLE-DOMAIN:notes END -->

## Auth And Authorization

The generated template does not include an auth provider. Add auth only
when product requirements identify protected data or user-specific
behavior. UI and CLI must not enforce business authorization locally;
authorization belongs at the API/service layer.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| None by default | n/a | no | Add entries when resources or third-party APIs require secrets. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Unsafe file upload handling | Malicious or oversized upload could affect storage. | Multipart handler validates metadata and BlobStore seam isolates bytes. | template-reference |
| Missing auth for product data | User/customer data could be exposed if added without access control. | Add API-layer auth before storing protected data. | deferred |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No product-specific data classification | medium | Fill after PRD/domain map defines real data. |
| No auth model | conditional | Required before protected or multi-user data. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
