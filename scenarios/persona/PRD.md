# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario persona`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Adds the permanent capability to act on the outside world as a *declared identity* rather than as an anonymous process. A persona is a durable object naming the human or legal entity an agent acts for, carrying the concrete artifacts a real transaction requires: an address the persona controls and can read, a route to the one-time codes a machine cannot receive on its own, postal and billing addresses, and a binding to identity documents held elsewhere. Where a wall cannot be crossed by any machine, this scenario turns that wall into a typed, resumable handoff instead of a failure. It is deliberately **not** an identity provider and **not** a document store.

- **The distinction that justifies the scenario**: `agent-manager` already owns *workload* identity — `Claims.Subject`, an attenuated scope list, and `Attenuate()` enforcing one-way narrowing for child runs. That answers "which run is this and what may it do inside Vrooli". A persona answers a different question — "who is transacting with the outside world" — and the two objects differ in every dimension that matters: lifetime (24 hours versus years), count (thousands versus a handful), subject (a Vrooli account versus a legal person), and verifier (agent-manager versus a passport, a D-U-N-S number, or a bank). This scenario adds one outer link to the chain agent-manager already owns; it does not replace any part of it.

- **Primary users/verticals**: Vrooli agents performing any outbound action that requires being someone — purchasing, enrolment, signup, trial activation, marketplace registration, support correspondence. The operator who must approve and complete the steps no machine may take. Beyond Vrooli: teams running agents against real services who refuse to place a passport scan and a mailbox credential inside a third-party SaaS; solo operators and small studios juggling several legal entities; and compliance-sensitive teams needing every outbound action attributable to an authorising human.

- **Deployment surfaces**: Go API (Connect-RPC), typed Go CLI, React operator console, and adapters satisfying the OTP-retrieval and handoff-delivery contracts.

- **Value promise**: An agent that can be *somebody* — accountably. Every outbound action carries an unbroken chain back to the human who authorised it; every code a machine cannot receive has one contract for retrieving it; and every wall a machine must not cross becomes a single pre-filled action for a person rather than a dead end. The differentiator is custody: the persona, its mailbox, and the pointer to its documents live on the operator's own machine, which no hosted Know-Your-Agent product offers.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Persona is a durable object with a declared legal basis | Every persona names the human or legal entity it acts for at creation time; a persona with no declared basis cannot be created, and the basis is immutable thereafter
- [x] OT-P0-002 | The delegation chain is one chain, not two | A persona binds into agent-manager's signed claims through persona_id in Claims.Meta and the persona.act-as scope, so the chain runs legal person to persona to account subject to run token to child token without a second identity system
- [x] OT-P0-003 | Acting as a persona fails closed | Identity verification runs through agent-manager's live endpoint; an unverifiable or unreachable caller cannot act as any persona, and the refusal is recorded rather than downgraded to a weaker evidence grade
- [x] OT-P0-004 | Persona ACL governs who may act | Each persona declares which humans may act as it and which may only propose actions for approval; the ACL is evaluated server-side and is the one permission concept the existing scope machinery does not already provide
- [x] OT-P0-005 | A controlled email address per persona | A persona owns an address it can both send from and read, so verification mail is retrievable without an agent ever touching the operator's personal inbox
- [x] OT-P0-006 | One OTP retrieval seam, many adapters | Exactly one typed contract admits a one-time code into the system; email, an SMS provider, and device-control reading a leased phone are all ordinary adapters satisfying it, with no privileged path
- [x] OT-P0-007 | Handoff is a typed, resumable state and never an error | A step no machine may take produces a checkpoint naming exactly what a human must do, with every other field already filled in, and the flow resumes at that checkpoint once the human is done
- [x] OT-P0-008 | Identity documents are bound, never stored | Documents live in document-manager under its sensitivity class and custody journal; this scenario stores only which persona owns which document and never holds the bytes
- [x] OT-P0-009 | Document release requires a named handoff | A document is released only into a pre-declared handoff, evaluated server-side; there is no agent-readable read path for a passport or an incorporation record under any scope
- [x] OT-P0-010 | Personal and business personas are distinct kinds | Each kind carries its own identifier set and neither may borrow the other's documents, addresses, or legal basis
- [x] OT-P0-011 | Every persona action is attributable and append-only | Acting as, releasing, retrieving a code, opening and completing a handoff are all journaled with the verified run and the authorising human; no verb rewrites or deletes a journal row
- [x] OT-P0-012 | One typed resolution contract for consumers | A consumer such as treasury or a checkout driver resolves a persona through exactly one call that returns what it is entitled to see and nothing more, so entitlement is decided here rather than at each call site
- [x] OT-P0-013 | A persona is selected, never inferred | No default persona and no silent fallback; an outbound action names the persona it acts as or it is refused, because a wrong guess about identity is unrecoverable

### 🟠 P1 – Should have post-launch

- [x] OT-P1-001 | Credential linkage register | Every account created as a persona is recorded with its site, login seam, and recovery path, which is the prerequisite for retiring a persona without orphaning what it created
- [x] OT-P1-002 | Address book per persona | Billing and shipping addresses held per persona and releasable into a named handoff or resolution, under the same entitlement rule as documents
- [x] OT-P1-003 | Obligation registry | What each persona has signed up for, its renewal date, and its cancel path; the identity half of a recurring commitment whose money half belongs to treasury
- [x] OT-P1-004 | Handoff delivery through notification-hub | When notification-hub is available a handoff reaches the operator on the right device; when it is absent the built-in queue still works, so the relay is an enhancement and never a dependency
- [x] OT-P1-005 | Persona health and staleness | A persona surfaces what is expiring or broken — a document past its validity date, a mailbox that no longer authenticates, an unreachable OTP route — before a flow discovers it mid-enrolment
- [x] OT-P1-006 | Act-as grants held in prompt-manager | Which team members may act as which persona is ordinary member configuration in prompt-manager, keeping organisational policy where teams already live rather than duplicating a roster here
- [x] OT-P1-007 | Signed identity attestation | Emit a token carrying the full delegation chain for a counterparty to verify, aligned to the DIF KYA-OS direction rather than inventing a private claim shape
- [x] OT-P1-008 | Enrolment preparation as a first-class flow | An enrolment assembles every field a target requires, reports exactly which remain human-only, and presents them as one handoff instead of many

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Phone number provisioning | A persona holds a real number rather than borrowing the operator's, which removes the last shared artifact between a persona and its owner
- [ ] OT-P2-002 | Persona rotation and retirement | Retire or rotate a persona without orphaning the accounts it created, driven by the credential-linkage register rather than by memory
- [ ] OT-P2-003 | Paid human handoff | Route a blocked step to a human as a paid task when the operator is not the right person to complete it, the model Payman built a business on
- [ ] OT-P2-004 | Counterparty attestation exchange | Accept and verify another party's agent-identity attestation, making Know Your Agent bidirectional rather than something Vrooli only asserts
- [ ] OT-P2-005 | Cross-instance personas | A persona usable from another Vrooli instance, relayed through vrooli-bridge, without copying its documents or credentials across the boundary
- [ ] OT-P2-006 | Escalation routing for unsolvable challenges | A CAPTCHA or biometric prompt routes to the right human by policy rather than to whoever happens to be watching, and is never attempted by a machine

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**: Go API on Connect-RPC with proto-first contracts, a typed Go CLI built on cli-core primitives, and a React + TypeScript + Vite operator console on the `vrooli-default` design kit. No deviation from the template's stack is warranted; this scenario's difficulty is in its contracts, not its runtime.

- **Data + storage expectations**: SQLite for personas, bindings, ACL entries, handoff state, obligations, and the append-only action journal. No shared resource is needed and none should be added. Three classes of data deliberately live elsewhere and are referenced rather than copied: identity **documents** in `document-manager` under its sensitivity class and custody journal; **credentials** (mailbox passwords, API tokens, card details) in `secrets-manager`; and **run identity** in `agent-manager`. This scenario stores bindings and policy, never the sensitive payload itself, which is what keeps its blast radius small enough to be worth self-hosting.

- **Integration strategy**: One new `Meta` key (`persona_id`) and one scope namespace (`persona.act-as:<id>`) in `agent-manager` — the same shape `RUNTIME_ATTRIBUTION.md` already documents as a pending workshop for `team_id`/`member_id`. Both flow through the existing `IntersectScopes` and `Attenuate` machinery unchanged, so persona permissions inherit the child-can-never-widen guarantee for free. `prompt-manager` holds the grants (which team members may act as which persona) as ordinary member configuration. `device-control`'s existing lease-and-bounded-action pattern is the phone OTP adapter and needs no extension. `notification-hub` is an optional handoff relay, never a dependency.

- **Non-goals / guardrails**: Not an identity provider — `scenario-authenticator` owns credentials, sessions, and JWKS, and this scenario never issues an access token. No document bytes, ever; a binding plus a release verb is the whole surface. No credential storage. **No solving or circumventing a human-verification wall** — no CAPTCHA solving, no biometric spoofing, no synthetic identity — because the wall is a product feature and its evasion is exactly what gets accounts banned. No persona may represent a person who has not authorised it, which is enforced by the ACL rather than by convention. No KYC-vendor integration and no compliance badges as headline features. No collaborative editing and no per-persona dynamic schema.

## 🤝 Dependencies & Launch Plan

- **Required resources**: None beyond SQLite. This holds even at P2 — every heavier need is satisfied by an existing scenario rather than a new resource.

- **Scenario dependencies**: `agent-manager` (**required**) for the signed run claims the persona binds into, and for the verification call; the dependency is hard on the act-as path and the scenario fails closed when it is unreachable. `document-manager` (**required at P0-006/007**) as the only store for identity documents. `secrets-manager` (**required**) for mailbox and provider credentials. `prompt-manager` (**P1**) to hold act-as grants as member configuration. `device-control` (**optional**) as one OTP adapter among several. `notification-hub` (**optional**) as a handoff relay — the built-in queue works without it. `browser-automation-studio` (**optional**) as the flow driver that consumes a persona. `treasury` (**consumer, not dependency**) resolves a persona before a mandate is executed; the direction is one-way and this scenario has no knowledge of money.

- **Operational risks**: *Prompt injection into a handoff* — a merchant page that talks an agent into requesting a document release it does not need; mitigated by requiring release to name a pre-declared handoff, evaluated server-side. *Persona sprawl* — many half-configured personas is worse than one complete one, so creation requires a declared legal basis rather than defaulting. *Silent account orphaning* — retiring a persona can strand accounts created under it, which is why the credential-linkage register (P1) precedes rotation (P2). *Mailbox as a single point of compromise* — the controlled address is a master key to every account the persona created, so it is treated as a credential of the highest class, not as convenience infrastructure. *Verification-authority outage* — agent-manager unreachable means no act-as, by design; the availability cost is accepted deliberately.

- **Launch sequencing**: (1) Persona object, legal basis, and the append-only journal — the spine everything else hangs from. (2) The ACL, because a persona with no answer to "who may act as this" is not safe to bind to anything. (3) The `agent-manager` `Meta` key and scope namespace, landed as its own small change outside this scenario, since it is the join that makes attribution real. (4) Controlled email, then the OTP seam with the email adapter first — the adapter that needs no external provider. (5) Handoff as a typed state, with the built-in queue before any relay. (6) Document binding and release-to-handoff against `document-manager`. (7) Only then the P1 registers — credential linkage, addresses, obligations — because each records the results of flows that must already work.

## 🎨 UX & Branding

- **Look and feel**: The `vrooli-default` design kit, unmodified in its binding contract. Density is deliberately low: this is a scenario an operator visits rarely and under mild pressure — something is blocked and waiting on them — so the console optimises for *unambiguous state at a glance* over information density. The dominant surface is the handoff queue, which should read like an inbox with a clear "what you must do" line per item, not like a dashboard. A persona detail page reads as a record, not a form: what is configured, what is missing, and what it has been used for.

- **Accessibility bar**: WCAG 2.2 AA, with three commitments the design kit does not automatically supply. First, no state is conveyed by colour alone — a blocked handoff carries an icon and a text label as well as a status colour, because the entire product is about the difference between "waiting", "expired", and "refused". Second, every handoff action is reachable and completable by keyboard alone, since a handoff arriving on a phone at an awkward moment is the normal case, not the edge case. Third, one-time codes and their expiry are announced to assistive technology via a live region, because a code that has silently expired while being read is an accessibility failure with a real cost. Accessibility primitives from the template — `role`, `aria-*`, `data-testid` selectors — are preserved as durable seams.

- **Voice and messaging**: Plain, precise, and never coy about limits. The scenario's honesty is its product: when a wall cannot be crossed, the copy says exactly which wall and exactly what the human must do, never "something went wrong". Refusals are stated as decisions with reasons ("this document can only be released into a named handoff"), not as errors. No anthropomorphising of the persona — it is a record the operator owns, not a character.

- **Branding hooks**: The seeded PWA install surface (`site.webmanifest`, `sw.js`, maskable icons, relative install asset URLs, safe-area tokens) stays valid; generic icons are replaced when product branding exists. The handoff queue is the natural candidate for a future push surface, but push is treated as a product decision rather than a template obligation and is not assumed at P0.

## 📎 Appendix

**Why this is a scenario and not an extension of `agent-manager`.** Three reasons, each sufficient on its own. *Size*: `agent-manager` is 639 Go files across 52 internal packages; persona lifecycle, an OTP seam, address books, obligations, and a handoff state machine would not improve it. *Consumers*: agent-manager's identity is read by write seams proving attribution, while a persona is read by outbound flows — checkout drivers, treasury, handoff delivery — with almost no overlap. *Failure semantics*: an expired run token means "spawn again"; a broken persona means an account somewhere in the world is orphaned.

**The three tiers of outbound action.** Purchases and enrolments fail in three distinguishable ways, and the scenario's scope is set by them. *Tier A — machine-native*: the counterparty expects a machine, and no persona is needed. *Tier B — card-shaped checkout*: needs an address, an email, and often a code; fully automatable behind a gate once this scenario exists. *Tier C — identity-bound enrolment*: needs a government photo ID, a D-U-N-S number, or a biometric, and up to two weeks of human review. Tier C is not an automation target and never will be; modelling it as a resumable handoff is the honest answer and, per the funded market, also the product.

**Market position.** Know Your Agent is a funded category with a standards track — Vouched raised a $17M Series A on a KYA suite and donated its framework to the Decentralized Identity Foundation, where it became KYA-OS; Skyfire is building agent identity as its core product; Sumsub frames KYA as an agent whose activity is explicitly authorised by a real human, here and now. Every shipping product is hosted. This is the scenario that asks an operator to store a passport scan, which makes self-hosting a positioning wedge rather than a preference. The nearest adjacent idea worth borrowing is Payman's marketplace for agents paying humans to do what they cannot — the paid-handoff target at P2 is that idea applied to a wall rather than to a task.

**Monetization posture.** Free permanently: the persona object, the ACL, the journal, handoff state, and the email OTP adapter — all deterministic, local, no marginal cost, and all things a self-hoster could run with their own keys, which the ecosystem's stated posture says must never be gated. Metered: adapters with a real per-use cost, specifically SMS/phone-number provisioning where a provider charges per message or per number. Gated: the cross-device handoff relay and hosted mailbox provisioning, which buy convenience and integrated access rather than capability. Bundle placement is operator-curated canon and is not decided here.

**Relationship to `treasury`.** `treasury` holds what may be spent; this scenario holds who is spending. Neither contains the other, and the direction is one-way: treasury resolves a persona before executing a mandate, and this scenario has no knowledge of budgets, rails, or money. A persona is useful without treasury — signups, trials, and support correspondence need no payment at all — which is what makes it independently deployable.
