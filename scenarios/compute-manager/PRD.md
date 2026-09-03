# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

Purpose: Compute Manager is the permanent capability for acquiring, tracking and retiring remote compute. It turns "I need a machine" into a machine that is already a trusted Vrooli node, and it is the one place that knows what that machine costs. It owns capacity and cost, and nothing else.

Primary users/verticals: The operator buying capacity for their own fleet without leaving Vrooli or opening a provider console. Scenarios that need capacity the standing fleet does not have, such as validation bursting to an operating system no current node runs, or a deployment targeting a host that does not exist yet. Later, a subscriber who buys managed compute rather than running their own hardware. Connecting a machine the user already owns stays free forever; only capacity Vrooli provisions and pays for is metered.

Deployment surfaces: A Go API server with Connect-RPC over proto-owned wire contracts; a Go CLI with full headless parity for requesting, inspecting, extending and destroying capacity; a React + Vite + TypeScript operator dashboard showing live inventory, elapsed cost and expiry; and two unattended background loops, a bidirectional reconciler and an expiry sweeper, that run with no operator present.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Provider-agnostic instance lifecycle | The scenario shall create, describe, list and destroy an instance through one provider interface, and no caller shall name a provider
- [ ] OT-P0-002 | Intent recorded before the provider is called | When an instance is requested, the scenario shall persist the request and its idempotency key before it calls any provider API
- [ ] OT-P0-003 | Bidirectional reconciliation | The scenario shall compare provider inventory against its own records in both directions on a schedule, and shall report every divergence rather than resolve it silently
- [ ] OT-P0-004 | Expiry enforced twice | Every instance shall carry an expiry enforced both by the scenario and by a timer on the instance itself, so the fleet drains when the scenario is unavailable
- [ ] OT-P0-005 | Unattended enrollment | When an instance boots it shall become a trusted node through the bridge onboarding contract, with no interactive step and no password on any wire
- [ ] OT-P0-006 | Credit reserved before boot | When provisioned compute is requested, the scenario shall reserve credit server-side before it calls a provider, and shall settle measured usage when the instance is destroyed
- [ ] OT-P0-007 | Destroy is the only stop | The scenario shall expose no pause that maps to a provider power-off, because a stopped instance still bills at the full rate on most providers

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Adopt a machine the operator already owns | Given reachable access to an existing host, the scenario should enroll it as a node without creating or billing anything
- [ ] OT-P1-002 | Per-tenant ceiling computed from our own meter | The scenario should refuse a request that would take a tenant past its ceiling, rather than rely on a provider spend alert that only sends mail
- [ ] OT-P1-003 | Daily cost reconciliation against the provider | The scenario should compare metered usage against the provider's own billing daily and raise an alarm when they diverge beyond a threshold
- [ ] OT-P1-004 | A second provider changes no caller | The scenario should support a second provider behind the same interface with no change to any consumer
- [ ] OT-P1-005 | Operator inventory surface | The scenario should render live inventory, elapsed cost and remaining lifetime for every instance it owns

### 🟢 P2 – Future / expansion

- [x] OT-P2-001 | Customer purchase through the subscription | The scenario may sell provisioned compute to a subscriber through the existing entitlement and credit rails rather than a separate payment path
- [x] OT-P2-002 | Ephemeral capacity for validation | The scenario may supply short-lived nodes so the cross-operating-system gate can burst beyond the standing fleet
- [x] OT-P2-003 | Warm pooling against hourly rounding | The scenario may reuse warm instances so provider hour rounding does not dominate the cost of short workloads

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API with Connect-RPC and proto-first wire contracts; SQLite through `api-core/storage`; a Go CLI whose verbs are manifest-declared so they are dispatchable through vrooli-bridge; React + Vite + TypeScript + Tailwind on the `vrooli-default` design kit.
- Data + storage expectations: one scenario-owned SQLite database holding instance intents, instances, provider receipts, credit reservations, reconciliation findings and the cached bridge onboarding key. Provider API credentials resolve through the credential authority at call time and are never written to this database, the process environment, or argv. No customer payment data is stored here at all; the business suite owns wallets, entitlements and invoices.
- Integration strategy: compose, do not reimplement. Enrollment and node trust delegate to `vrooli-bridge` through its existing onboarding contract, so this scenario contains no SSH implementation. Credit reservation and settlement delegate to `landing-page-business-suite` through its existing usage and reservation surface. Agent spend authorization delegates to `treasury`. The sellable definition lives in `offer-desk`. Provider access is one deliberately small adapter interface: create, describe, list, destroy.
- Non-goals / guardrails: will NOT implement SSH — first touch delegates to bridge onboarding and a provisioned instance trusts the bridge key from first boot; will NOT own node identity, pairing, scopes or dispatch (`vrooli-bridge`); will NOT own public exposure, hostnames or DNS (`tunnel-manager`); will NOT deploy scenarios onto a machine (`scenario-to-cloud`, `deployment-manager`); will NOT own subscriptions, entitlements, wallets or invoices (`landing-page-business-suite`); will NOT introduce a second Machine object — bridge owns Machine, this scenario owns Instance; will NOT expose a pause that maps to a provider power-off; will NOT meter from an observer loop that watches what is running, because a dead observer stops billing while the provider keeps charging.

## 🤝 Dependencies & Launch Plan

- Required resources: none. This scenario runs no local service; its state is SQLite and its reach is outbound HTTPS.
- Scenario dependencies: `landing-page-business-suite` (required, fail closed — provisioning refuses without a reservation, because a machine that boots unmetered is cost that grows hourly and cannot be recovered afterwards); `vrooli-bridge` (required, degrades — enrollment queues and retries while the instance is still created, metered and expiring on schedule, since blocking capacity on the trust plane would make capacity unavailable for a reason unrelated to capacity); `treasury` (conditional — only on the path where Vrooli's own agents buy capacity, bounding what an agent may spend; absent, agent-initiated provisioning refuses and operator-initiated provisioning continues); `offer-desk` (catalog only — holds the sellable definition, with no runtime path).
- External dependencies: one cloud provider HTTP API per adapter. Hetzner Cloud is first, chosen because its standard terms permit granting third parties use rights with no partner programme and no per-customer acceptance, it bills outbound traffic only, and it leaves inbound UDP unencumbered. DigitalOcean is the intended second, for per-second billing and geographic diversification.
- Operational risks: an instance created while the response was lost becomes cost the scenario cannot see, mitigated by writing intent before acting and reconciling in both directions; most providers round a partial hour up to a full one, so short-lived instances cost more than they earn, mitigated by a minimum billable unit and warm pooling; a stopped instance still bills, mitigated by having no pause at all; provider billing data lags by hours to more than a day, so it is a reconciliation signal and never a control; reselling terms differ sharply by provider and are checked per service, not per provider, because a general agreement can be overridden by a service annex.
- Launch sequencing: build the reserve-provision-settle spine, intent-before-action, bidirectional reconciliation and double-enforced expiry against a fake provider first, because those four are the failure modes and none of them needs a real API key. Wire Hetzner only once they hold. Unattended enrollment lands after vrooli-bridge publishes its onboarding public key. The customer sale is last and requires an operator, not an agent, to promote the offer.

## 🎨 UX & Branding

- Look & feel: the `vrooli-default` design kit, rendered as an inventory-first dashboard. Three facts carry the page: what exists right now, what it has cost so far, and when it expires. Cost and expiry lead because both are irreversible if missed. Provider identity is metadata, not an organizing principle, so the operator reads capacity rather than reading vendors.
- Accessibility: every state is distinguishable without colour alone, which matters here because expiring-soon and over-budget are the two states that most need attention and both are conventionally red. Full keyboard reachability. Tabular numerals wherever cost or elapsed time is rendered so columns of figures compare by eye. Live regions announce a provisioning result rather than silently swapping a row.
- Voice & messaging: plain and unalarming, but never vague about money. Name the amount and the unit rather than saying usage is high. A refusal states which ceiling was reached and what would raise it. Destruction is irreversible and is confirmed against the instance's own name, never a generic yes-or-no dialog. An instance the reconciler cannot account for is called out as unaccounted rather than hidden until someone reads a bill.
- Branding hooks: none beyond the shared design kit; this is operator infrastructure, not a marketed surface.
