# Security: Compute Manager

This document records the security and privacy posture of this scenario:
what it holds, who may act on it, what can go wrong, and what is not yet
handled.

> **Status: partially implemented.** Provider credential resolution,
> enrollment delegation, metering boundaries, reservation persistence and
> destructive-operation guards have focused coverage. Provider-live proof and
> the remaining threat-model review are still open.

One thing about this scenario's shape is worth stating before the tables,
because it explains most of the decisions. Compute Manager can create
machines on the internet and charge money for them. Those are two of the
more attractive capabilities in the ecosystem to an attacker, and the
scenario reduces that attraction mainly by refusing to hold things. It
holds no SSH implementation, no node credential, no wallet, no payment
data and no provider credential at rest. What it holds is a record of
what exists and what it cost. A full compromise of this scenario's
database yields an inventory, not a key.

## Purpose Of This Document

Use this document to answer:

- What sensitive data does this scenario hold, and what does it refuse to hold?
- How is access controlled, and where is that enforced?
- Where do secrets come from?
- Which threats are known, and what is still unhandled?

## Data Sensitivity

All rows below are intended. No table exists yet.

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Instance intents | medium | `intent` | A durable record of a request and its idempotency key, written before any provider call. Reveals what was asked for and when. The idempotency key is not secret but must be unguessable, because predicting one could let a caller probe whether a request occurred. |
| Instance records | medium | `instance` | Provider identifier, region, size, state, expiry, and the network address. The address is the most sensitive field: it names a reachable host. |
| Provider receipts | medium | `provider` | What the provider reported and what it charged. Business-identifying in aggregate. |
| Reconciliation findings | medium | `reconcile` | A finding names an instance the scenario cannot account for. Read together with the instance table, findings describe gaps in the fleet's supervision. |
| Rendered first-boot configuration | medium | `enroll` | Carries the bridge onboarding public key. The key itself is public. The document is still sensitive because it describes how a host is brought under management, and because it is the natural place for a future change to add something that is not public. |
| Metering references | low | `meter` | Reservation and settlement identifiers only. No amounts of record, no balances, no payment data. |

**Deliberately not held, and why that is a security property rather than a
limitation:**

| Not held | Lives in | Security consequence |
|---|---|---|
| Provider API credentials at rest | The Vrooli credential authority | A compromise of this scenario's database cannot create or destroy a single machine, because the ability to call the provider is not stored here. |
| SSH keys, host keys, node credentials | `vrooli-bridge` | This scenario contains no SSH implementation at all. It cannot be the origin of a node credential leak because a node credential never arrives here. |
| Wallets, entitlements, invoices, payment instruments | `landing-page-business-suite` | Keeps this scenario outside payment-data scope by construction rather than by policy. |
| Node identity, pairing, scopes, dispatch | `vrooli-bridge` | This scenario cannot grant a machine authority over anything. It can only ask the bridge to onboard one. |
| Agent spending authority | `treasury` | This scenario enforces a capacity ceiling, not a mandate. It cannot widen what an agent may spend. |
| Public hostnames, DNS, ingress | `tunnel-manager` | Creating an instance does not expose it. Exposure is a separate decision owned elsewhere. |

## Auth And Authorization

The generated template does not include an auth provider, and none has
been added. The intended model:

**Authorization is always server-side.** UI and CLI enforce nothing; they
are translation layers over the API. This matters more than usual on the
metering path, because the tier that decides what a caller may provision
must be resolved from the caller's subscription server-side and never
read from the request. A client asserting a higher tier must still be
refused. `COMPUTEM-P0-006` carries a validation for exactly this.

**Two distinct callers, with different rules.** An operator provisioning
capacity for their own fleet and an agent provisioning capacity on
Vrooli's behalf are not the same principal:

| Caller | Intended authority | Bound by |
|---|---|---|
| Operator | Request, extend, inspect and destroy capacity in their own tenancy. | The per-tenant ceiling computed from our own meter (`COMPUTEM-P1-002`). |
| Agent | Request capacity only on the agent-initiated purchase path. | `treasury`. If `treasury` is unavailable, agent-initiated provisioning refuses while operator-initiated provisioning continues. |

**Destruction is confirmed against the instance's own name**, never a
generic yes-or-no dialog. This is a usability decision with a security
effect: it makes a mis-targeted destroy hard to perform by reflex, in a
scenario where destroy is irreversible and is also the only way to stop
the meter.

**Enrollment authority is delegated, not held.** This scenario asks
`vrooli-bridge` to onboard an instance. It does not decide what that node
may then do, and it holds no credential that would let it act as the
node.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| Provider API credentials | Vrooli credential authority | P0, per configured adapter | Resolved by reference at call time. Never persisted in this scenario's database. Never read from the process environment. Never passed as a command argument, because argv is readable by any local process. Never returned in a response body. Never logged, including on error paths. |
| Bridge onboarding public key | `vrooli-bridge` | P0 | Public by nature and safe to embed in a first-boot configuration. It is listed here because it is fetched over the wire and its authenticity matters even though its confidentiality does not. |
| Business suite metering credentials | Vrooli credential authority | P0 | Scoped to reservation and settlement. Same handling rules as the provider credentials. |
| Node credentials of any kind | not held | never | The scenario has no SSH implementation and never mints, stores or forwards a node credential. |

No secret is held in `.vrooli/service.json`, in environment defaults, or
in scenario storage.

Two handling rules deserve to be stated as rules rather than left implicit:

- **A provider credential is never placed on a provisioned instance.**
  An instance that could call the provider API could create more
  instances, and a compromised node would become a billing amplifier.
  This is also why the instance-side expiry timer can only power the
  instance off and not delete it.
- **The rendered first-boot configuration is never logged whole.** Its
  current contents are harmless. The habit is not.

## Threat Model

**No threat model review has been performed.** The table below is a first
enumeration written during design, not a reviewed model. Every status is
`designed` or `open`, because nothing has been built to verify.

| Risk | Impact | Intended mitigation | Status |
|---|---|---|---|
| **Provider credential exfiltration** through a log line, an error message, a response body or argv | An attacker can create unlimited billable machines and destroy the fleet. This is the worst outcome available in this scenario. | Credential resolves through the credential authority at call time; never persisted, never in env, never in argv, never logged. Provider call logging records the idempotency key and outcome, not the request. | designed |
| **A machine that boots unmetered** because the metering dependency was unreachable and provisioning proceeded anyway | Unrecoverable cost that grows hourly. Money already spent cannot be reclaimed after the fact. | Fail closed. `landing-page-business-suite` is the one dependency that refuses rather than degrades. There is no bypass flag, and none should be added. | designed |
| **Lost provider response leaves an invisible instance** | Cost the scenario cannot see, and an instance nobody supervises or expires. | Intent and idempotency key are persisted before any provider call. Bidirectional reconciliation matches the orphan back to its intent. | designed |
| **Reconciler destroys a live customer node** because a bug made it look unaccounted | Destruction of a paying customer's machine. Irreversible. | Mark and sweep, not sweep. The reconciler reports and never resolves. Destruction is a separate operator action confirmed against the provider-side identifier. | designed |
| **One-directional reconciliation** silently misses orphans in the other direction | Either invisible spend or a meter running against something that no longer exists. Most implementations build only one direction. | Both directions are a stated P0 requirement with an integration test that asserts findings in each. | designed |
| **Client-asserted tier** on the cost-bearing path | A caller provisions capacity it has not paid for. | Tier is resolved server-side from the subscription and ignored when present in the request. `COMPUTEM-P0-006` validation. | designed |
| **Replayed provisioning request** | Duplicate machines and duplicate charges. | Caller-supplied idempotency key required and persisted before the provider call; a repeated key returns the first record. | designed |
| **Agent manipulated into provisioning** by content it read | Cost, bounded. | The agent path is gated by `treasury`, which bounds what an agent may spend, and separately by the per-tenant ceiling. The blast radius is a ceiling, not a fleet. | designed |
| **Instance-side expiry timer removed or disabled** on a compromised instance | The instance outlives its expiry when the control plane is down. | Accepted and bounded. The scenario-side sweeper is the primary enforcement; the timer is a backstop for control-plane unavailability, not a control against a compromised host. | designed |
| **Provisioned instance is publicly reachable and attacked** | Compromise of a Vrooli node; on some providers, attacker-controlled inbound traffic cost. | Exposure is owned by `tunnel-manager` and is not granted by creating an instance. Provider choice mitigates the cost half: Hetzner bills outbound traffic only, whereas Amazon's small-instance product counts inbound and would turn a fixed cost into one an attacker controls. | designed |
| **Metered-versus-billed drift used as a control** | Acting on lagging data destroys or refuses the wrong thing. | Provider billing lags by hours to more than a day and is treated as a reconciliation signal only, never as a control input. | designed |
| **Provider terms violated by reselling capacity** | Account termination, which is a fleet-wide availability failure. | Reselling terms are checked per service and not per provider, because a general agreement can be overridden by a service annex. Hetzner permits granting third parties use rights with no partner programme; four of seven providers surveyed forbid reselling. | designed |
| Unsafe file upload handling | Malicious or oversized upload could affect storage. | Multipart handler validates metadata and BlobStore seam isolates bytes. | template-reference |

## Security Gaps

Every gap here is OPEN. Nothing in this section has been closed.

| Gap | Severity | Revisit Trigger |
|---|---|---|
| **Provider-live unattended enrollment is not yet proven.** Bridge now publishes the onboarding public key through an owner-gated contract, and compute-manager delegates machine creation and onboarding without carrying SSH code or passwords. | high | Blocks final live evidence for `COMPUTEM-P0-005` until a real provisioned host reaches the online state. Keep the fake-boundary tests and do not add a private SSH path. |
| **Provider credential handling is unimplemented and unproven.** The rule is that credentials resolve through the credential authority and never touch the environment, argv, the database or a log line. Nothing enforces that today, and the template offers an environment variable as the path of least resistance. | high | Before the first real provider credential is configured. Add an automated check that the credential value appears in no log, no response, no argv and no column, rather than relying on review. |
| **The metering enforcement boundary is upstream and has known defects.** Four are known and are prerequisites rather than integration work: the reservation window is hard-coded to ten minutes, which is shorter than an hour of compute; refunds silently do nothing for app-scoped charges, because the adjustment query filters to rows with no app key; the convenience charge helper creates no reservation, takes no idempotency key, has no release path and does not refund on provider failure; and the reference metered client discards the response body on a non-2xx, so out-of-credit is indistinguishable from a server error and counts toward its circuit breaker. | high | Before `COMPUTEM-P0-006` can be claimed. Use the reservation path and not the convenience helper. Parameterise the window or prove heartbeat re-reservation. Fix response-body handling at this scenario's client boundary rather than waiting for upstream. |
| **No threat model has been reviewed.** The table above is a first enumeration written during design. No adversarial review, no second reader, no red-team pass. | high | Before the first real provider credential is configured, and again before any hosted or multi-tenant tier. |
| **No auth model exists.** The template ships no auth provider, and the operator/agent split above is described but not enforced anywhere. | high | Before any deployment beyond a single-operator local stack. |
| **No per-tenant isolation.** Without it the ceiling cannot be enforced and one tenant's spend is not separable from another's. | high | `COMPUTEM-P1-002`, and required before any managed or multi-tenant tier. |
| **No rate limiting on the provisioning surface.** A caller that cannot exceed its ceiling can still generate unbounded refusals, intents and provider calls. | medium | Before the surface is reachable by anything other than a local operator. |
| **Database backup and restore are undefined**, and a stale restore is actively dangerous: it makes live instances look unaccounted and destroyed ones look live. | medium | Before the first real provider credential is configured. See [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md). |
| **Retention for findings, receipts and instance records is undefined.** Cost and audit questions are asked weeks later. | low | Define with the storage schema. |
| **The scenario has no product surface to review.** `template-manager detemplate compute-manager` removed the template `notes` worked example and its multipart upload path, so the reachable surface is `/health` and the capabilities probe. This narrows the attack surface to almost nothing today and means every row above is unreviewed against real code. | low | Re-enumerate as each domain lands, starting with the provisioning surface. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) - data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) - external services and secrets
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md) - the operator procedures these controls depend on
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) - what must never be logged
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) - error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) - unresolved security debt
