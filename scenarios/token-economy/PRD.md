# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario token-economy`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Adds the permanent capability to run a real economy with money that is not real. Tokens are minted under a declared authority, granted to holders with rules that survive the grant, earned through adapters any scenario can satisfy, and redeemed against a catalog the minter controls. Balance is a query over an append-only journal, never a stored assertion. The scenario is deliberately local-value-only: nothing here converts to money, and that constraint is what makes a multi-holder economy safe to run on one machine.

- **Primary users / verticals**: Households sharing one Vrooli instance — an adult who mints and a child who earns and redeems — as the first and defining case. Beyond that: teams running internal recognition or reward economies; classrooms and tutoring; communities and clubs allocating shared perks; game and simulation authors needing a governed in-world currency; and Vrooli itself, which uses the scenario as the zero-blast-radius rehearsal surface for the spend-authority model that `treasury` applies to real money.

- **Deployment surfaces**: Go API over Connect-RPC, a typed Go CLI, and a React operator console with two distinct authenticated audiences — the minter console and the holder view. Earning adapters reach the scenario programmatically; no adapter is privileged, and the operator-entry adapter is a first-class path rather than a fallback.

- **Value promise**: Every allowance and chore-reward product on the market is a bank product. Greenlight, BusyKid, FamZoo, Modak and Acorns Early all require linking a real bank account, issuing a real card to a child, and paying a monthly fee — and all of them can only ever pay out in dollars. This scenario is the opposite position: no bank, no card, no KYC, no per-child fee, and rewards that are whatever the minter says they are — screen time, a trip, a chore traded between siblings, a privilege that has no price. It also composes: any Vrooli scenario can become an earning surface through one contract, which no standalone allowance app can offer because no standalone app has an ecosystem behind it.

- **The load-bearing decision**: a grant is a mandate. A token grant carrying spend rules — what it buys, from whom, until when — is structurally the same object as the signed, scoped, expiring grant that authorizes real spending in `treasury`. The two scenarios share one contract shape deliberately, so that maturing the policy model here matures it there, and so the eventual real-value adapter is a new rail rather than a rewrite.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | The token-type contract | Every token exists inside a declared type carrying a supply policy and a named minter authority; no token can be created outside one
- [ ] OT-P0-002 | The grant contract is mandate-shaped | Exactly one typed grant admits tokens to a holder, carrying holder, amount, rules, expiry and provenance, in the same shape treasury uses to authorize real spend
- [ ] OT-P0-003 | Rules are evaluated server-side | A redemption is validated against its grant inside the API; no client, holder surface or earning adapter ever holds the decision
- [ ] OT-P0-004 | Balance is a query, never an assertion | Every mint, grant, redemption, transfer and reversal is an append-only event, and a balance is computed from them rather than stored as truth
- [ ] OT-P0-005 | Minting authority is structurally separated from holding | The holder-facing service has no method to mint, grant, or alter rules; separation is a codegen-visible service boundary, not a runtime permission check
- [ ] OT-P0-006 | Multiple holders on one instance | Distinct holders with distinct balances and separately authenticated views; a holder can never read, spend or infer another holder's balance
- [ ] OT-P0-007 | Earning surfaces are ordinary adapters | One inbound contract admits earned tokens; a chore scenario, a habit tracker and an adult pressing a button all satisfy it with no privileged earner
- [ ] OT-P0-008 | Redemption against a minter-controlled catalog | What a token buys is declared by the minter as catalog entries with their own availability and approval posture, never hardcoded in the product
- [ ] OT-P0-009 | Idempotent settlement | A redemption retried with the same idempotency key is a successful no-op after first commit, making double-spend impossible under retry or partial failure
- [ ] OT-P0-010 | Reversal is an event, never a deletion | A mistaken mint, grant or redemption is corrected by a compensating event that preserves the original; no verb rewrites or removes history
- [ ] OT-P0-011 | Every event carries actor provenance | Each journal entry records who caused it and how that actor was verified, so an agent-initiated grant is distinguishable from an operator-initiated one
- [ ] OT-P0-012 | Holder-visible history | A holder can see exactly why their balance is what it is — every earn, grant, redemption and reversal affecting them, in their own view
- [ ] OT-P0-013 | Approval posture is per-catalog-entry | A catalog entry declares whether redeeming it settles immediately or waits on minter approval, and the queue is a first-class surface rather than a notification
- [ ] OT-P0-014 | No real value, enforced structurally | No redemption path produces money, no external transfer exists, and no token carries a price; the constraint is enforced by the absence of the capability, not by policy text

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Non-fungible tokens | Badges, one-off rewards and collectibles as a distinct token class sharing the mint, grant and journal spine rather than a parallel system
- [ ] OT-P1-002 | Rule programs | Declared conditions that mint or release automatically when satisfied — the smart-contract shape with no chain underneath and no arbitrary code execution
- [ ] OT-P1-003 | Scheduled and recurring grants | A weekly allowance is a standing grant with a next-issue date and a one-action cancel, not a reminder someone has to act on
- [ ] OT-P1-004 | Holder-to-holder transfer | Peer exchange within the instance, permitted, restricted or forbidden by minter policy per token type
- [ ] OT-P1-005 | Goals and reservations | A holder can reserve balance toward a catalog entry, making saving visible and making the reserved portion unspendable until released
- [ ] OT-P1-006 | Household console | A minter dashboard showing every holder, pending approvals and economy health, beside a holder view built for someone who is not an operator
- [ ] OT-P1-007 | Multiple token types in play | More than one economy on one instance with distinct rules, catalogs and holders, without either economy leaking into the other
- [ ] OT-P1-008 | Grant expiry and decay | A grant may expire or decay on a declared schedule, so unspent balance can be use-it-or-lose-it where the minter wants that pressure
- [ ] OT-P1-009 | A first real earning adapter | One concrete Vrooli scenario wired as an earning surface end to end, proving the adapter contract against something that is not a test double
- [ ] OT-P1-010 | Statement and export | A minter or holder can export the journal for a period in a stable machine-readable shape, with the same event semantics the API exposes

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Real-chain adapter | A chain-backed rail behind the same grant contract; the local rail stays first-class and crossing into real value re-opens the custody question deliberately
- [ ] OT-P2-002 | Cross-instance tokens | Token types recognized across Vrooli instances, relayed through vrooli-bridge rather than by direct peer trust
- [ ] OT-P2-003 | Holder marketplace | Listings and offers between holders, subject to minter policy, extending transfer into genuine price discovery within the economy
- [ ] OT-P2-004 | Shared mandate contract with treasury | The grant and mandate shapes extracted into one governed contract both scenarios consume, once both have shipped enough to know what is genuinely common
- [ ] OT-P2-005 | Delegated minter authority | A second adult may mint or approve within limits the primary minter sets, using the same one-way attenuation the ecosystem already applies to agent scopes
- [ ] OT-P2-006 | Non-transferable achievement tokens | Reputation-shaped tokens that can be earned and displayed but never transferred or redeemed, for recognition economies rather than reward economies
- [ ] OT-P2-007 | Economy analytics | Earning and redemption patterns over time, surfacing whether an economy is inflating, stalling, or rewarding what the minter intended

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**: The standard Vrooli full-stack scenario shape with no deviation — Go API behind Connect-RPC with proto-first contracts, a typed Go CLI built on cli-core primitives, and React + TypeScript + Vite for the console using the `vrooli-default` design kit and adopted `react-component-library` primitives. Every domain endpoint goes through a proto service; literal REST paths are rejected by codegen and the four `RESTReason` exceptions do not apply here because this scenario has no binary upload edge.

- **Data + storage expectations**: SQLite through the routed scenario storage seam, and no shared resource. The journal is append-only and is the only authority on balance; derived balances may be cached but are never the source of truth and must be reconstructible from events alone. Every mutation takes a caller-supplied idempotency key and holds a row lock for the duration of the commit, following the proven `landing-page-business-suite` credit-wallet pattern — a retried key is a successful no-op after first commit, and distinct keys remain independent. Volume is inherently low (a household, not a market), so SQLite is the correct terminal choice rather than a starting point; the one condition that would revisit it is the P2 real-chain adapter, which introduces concurrency this scenario does not otherwise face.

- **Integration strategy**: Inbound earning is a single typed contract with no privileged satisfier, mirroring how `money-ledger` admits every money source through one adapter shape. Holder authentication resolves through `scenario-authenticator` rather than a private credential store. Actor provenance on journal events resolves through the shared `packages/cli-core` identity verifier and agent-manager's signed run claims, so an agent-initiated grant is distinguishable from an operator-initiated one without this scenario inventing an attribution model. Redemption approval is owned in-scenario as a first-class queue; `notification-hub` is an optional relay that improves reach and is never a dependency. The scenario reads no other scenario's data and publishes no findings.

- **Non-goals / guardrails**: No real money, ever, in P0 and P1 — no price on a token, no conversion path, no external transfer, and no integration with any payment rail. That is enforced by the capability being absent rather than by a policy check. No arbitrary code execution in rule programs; conditions are declared and evaluated by the engine, never supplied as scripts. No chain, no wallet, no key custody until the P2 adapter is deliberately opened. No tax, accounting or financial-position computation — that is `money-ledger`'s domain and this scenario's tokens are not money. No parental-control surface beyond the economy itself: this does not manage screen time, devices, or a child's account elsewhere, it only records that such a thing was redeemed. No cross-instance trust without `vrooli-bridge`.

## 🤝 Dependencies & Launch Plan

- **Required resources**: None beyond SQLite for P0 and P1. No shared database, no cache, no vector store, no chain node. The scenario is deliberately runnable on a laptop with nothing else started, because a household economy that requires infrastructure is not a household economy.

- **Scenario dependencies**: `scenario-authenticator` for holder identity, so that a child's view is genuinely authenticated rather than a client-side role flag — this is the one hard dependency. `notification-hub` is optional and relays approval requests when present. `react-component-library` supplies governed UI primitives by adoption. `agent-manager` and `packages/cli-core` supply verified actor provenance for journal events where the caller is an agent. `treasury` is a **contract sibling, not a dependency**: the two scenarios share a grant/mandate shape by design and neither calls the other, and this scenario must remain fully functional with `treasury` absent.

- **Operational risks**: The largest is **contract drift against `treasury`** — if the grant shape here and the mandate shape there diverge before the P2 unification, the ecosystem maintains two policy engines and the shared-contract target becomes a rewrite. Mitigation is to author the grant contract against the treasury mandate shape from the first commit and record every intentional divergence in `docs/internal/DECISIONS.md`. Second: **journal integrity under partial failure** — a redemption that debits without recording, or records without debiting, corrupts an append-only store that has no repair verb by design; the mitigation is that debit and event are one transaction with an idempotency key, tested against induced failure rather than assumed. Third: **the multi-holder boundary** — a holder reading another holder's balance is the failure that ends trust in a household product, so it is an authorization test at the repository layer and not only at the handler. Fourth: **the no-real-value constraint eroding** — every future feature request will push toward a price or a payout; the guardrail is that the capability is absent from the service surface, so adding it requires a visible contract change. Fifth: **market hypothesis risk** — the value promise is reasoned from a competitive scan, not validated with a real household, and P1 should not be built before it is.

- **Launch sequencing**: (1) Mint and the token-type contract, because nothing exists without a declared type. (2) The journal and balance-as-query, before any verb that moves a token, so no code path ever learns to trust a stored balance. (3) Grants with server-side rule evaluation and the holder/minter service split, which together are the security spine. (4) Holders and authenticated views, which is when the multi-holder boundary becomes testable. (5) The earning adapter contract with operator entry as its first satisfier. (6) Catalog and redemption with the per-entry approval posture, completing the earn-to-redeem loop. (7) Reversal and provenance, hardening the journal. (8) Only then the console, because the two audiences cannot be designed before the loop they present is real. P1 opens after one real household has run the P0 loop for a period; the market hypothesis is checked there and not in a document.

## 🎨 UX & Branding

- **Look and feel**: Two audiences in one product, and the design must not pretend otherwise. The **minter console** is an operational surface in the `vrooli-default` Operational Console language — dense, tabular, evidence-first, built for someone reviewing a queue and auditing a journal. The **holder view** is the opposite: sparse, large-target, few-decisions, legible to a child or to anyone who is not an operator, and it never shows the machinery. They share tokens, type scale and status semantics so the product reads as one thing, but they do not share density or information architecture. Balance is presented as a consequence of visible history rather than as a number that simply appears, because the whole point of the journal is that a holder can see why.

- **Accessibility bar**: WCAG 2.2 AA as the floor, taken seriously rather than asserted. The holder view carries the harder commitment: it must be operable by a young or low-literacy user, which means icon-plus-label rather than icon alone, no reliance on color to convey state, touch targets at the upper end of the guidance rather than the minimum, and every earn/redeem action reachable and understandable by keyboard and by screen reader. Status color follows the design kit's semantics and always pairs with text or shape. Motion respects `prefers-reduced-motion`, and no celebratory animation is ever the only signal that something succeeded. Locale switching stays Settings-owned per the shell contract, and every string flows through the i18n wiring — a reward economy is culturally specific and hardcoded English would make it un-adoptable.

- **Voice and messaging**: Plain, concrete and non-condescending in both views. A holder is told what they have, what it buys, and what they did to get it — never gamified into pressure, never scolded for a balance. Refusals explain the rule that refused them and what would change it, because an economy whose denials are opaque teaches nothing. The minter surface is factual and unsentimental: it reports state, names what needs a decision, and does not editorialize about a holder's behavior. Nothing in the product implies that tokens are money or convert to it; the copy is deliberate about this, because that is the constraint the scenario is built on.

- **Branding hooks**: The token type is the brand surface — a minter names it, gives it a symbol and a color, and that identity carries through both views, exports and notifications. The product itself stays neutral so the household's own economy is the thing with personality. `brand-manager` tokens are referenced rather than redefined.

- **PWA install surface**: The seeded web app manifest, service worker, maskable icons, relative install asset URLs and safe-area tokens stay valid and are genuinely load-bearing here — the holder view is a phone surface first, and installability is what makes a household economy something a child actually opens. Generic icons are replaced when product branding exists. Dynamic caching is tuned only if offline balance viewing becomes a real requirement; shortcuts, share targets and push flows are deferred product decisions rather than template obligations.

## 📎 Appendix

**Why this scenario exists at all.** It was scoped during a workshop on giving Vrooli agents the ability to spend and earn real money. Three capabilities came out of that work: `treasury` (authority over real spend and earn), `persona` (the identity an agent transacts as), and this scenario. This one is deliberately last in the build order, because the grant contract it reuses should be settled in `treasury` first.

**Why local-only is a feature and not a limitation.** Every rule worth having on real money — a cap, an expiry, a merchant restriction, an approval gate, a one-way narrowing to a delegate — can be modeled, shipped and broken here at zero cost. A policy engine that has only ever run against real money has never been safely wrong. This scenario is where the ecosystem's spend-authority semantics get to be wrong cheaply, which is a genuine engineering asset and not a consolation.

**Competitive scan (2026-08).** Greenlight leads on parental control depth (store restrictions, category limits, reporting). BusyKid ties earning to completed chores with parent approval and is the cheapest paid option at roughly $4/month flat per family. FamZoo runs a virtual family-bank IOU model, which is the closest existing product to this one in mechanism though not in deployment. Modak is the free debit-card option. Acorns Early absorbed GoHenry's US customer base in late 2025, retaining the card and chore features. Every one of them is a regulated bank product paying out in dollars. None can express a reward that has no price, and none composes with anything.

**The rejected framings**, recorded so a later reader can tell a decision from an omission: a real cryptocurrency first (rejected — key custody and irreversible loss before the semantics are proven), a generic points library without holders (rejected — the multi-holder boundary is the hard part and skipping it defers the real work), and folding this into `treasury` as a rail (rejected — a household economy has its own users, its own UX and its own product thesis, and burying it would make it invisible to the people it is for).
