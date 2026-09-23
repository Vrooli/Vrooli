# Go To Market — Nutrition Planner

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

The working product name is **Nooch** (formerly Daily; decision D-042). Scenario id is `nutrition-planner`.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

Channels are **hypotheses**, not activated programs. Activation is Offer
Desk's job once it owns the channel registry. Nothing here may be published
as though it were proven.

## Audience And Positioning

- **Starting user:** the originating vegan high-protein, low-effort user
  (spec §2.1). This user is not a market segment to test on; they are the
  first person whose real routine the product must fit before anything
  external is claimed.
- **Broader audience:** people who want to be told what to eat. The
  disqualifying trait is wanting to research and optimize every meal
  themselves — that person already has the problem solved by a spreadsheet
  or by doing it for fun.
- **Positioning:** **dinner, decided** — and the rest of the day with it.
  Affordable variety without turning life into a bookkeeping task, in an app
  that feels like a beautifully designed cookbook in a warm kitchen. Not the most recipes, not the most
  integrations, not a clinical nutrition authority, and not a calorie
  tracker. The product gives one concrete, explainable recommendation at
  the moment the decision is made.
- **Main claim:** a returning user sees the next meal immediately, starts
  guided preparation in one action, compares alternatives without writing a
  reason, and saves a name-only idea without entering nutrition. Every
  number is explained, scoped to what is known, and never invented.
- **Proof needed:** a real week run end to end by a real person — a fresh
  setup, one accepted plan, a shopping list with honest package counts, and
  recorded actual intake — showing that the unknown/partial states are
  honest and the low-effort path actually gets used. This is a working
  session, not a paragraph.

**Where it sits against the alternatives** (a positioning snapshot, not
captured market evidence):

| Alternative | What it does better | Where it fails this user |
|---|---|---|
| Notes app / spreadsheet | Zero learning curve, total flexibility | No planning, no nutrition arithmetic, no honesty about unknowns; the user still decides everything. |
| Recipe website + manual list | Huge variety, familiar | Still a search-and-decide task every week; no cost, effort, or rule awareness. |
| Macro-tracking app | Precise logging, large food database | Optimizes recording, not deciding; demands daily bookkeeping the user will not sustain. |
| Meal-kit delivery | Removes shopping and planning | Costs more per meal, limited to the kit's menu and diet support, and does not use food on hand. |
| Ordering out | Immediate, zero effort | The expensive default this product exists to displace. |

**Visual quality is the adoption wedge.** Meal planners are a crowded,
look-alike category; people decide within seconds whether an app is pleasant
enough to open every evening. The v2.0 redesign makes reference-quality
visuals a P0 target (`OT-P0-021`): photoreal meal scenes, an editorial serif,
Light and Evening appearances, and a phone experience composed for the kitchen
and the store. Beauty earns the daily open; the honest planner earns the trust.
Marketing lines such as "Dinner, decided." live in marketing copy only — the
app itself uses functional copy without slogans (R29.3).

The defensible part is not "local-first" or "private" by itself; those are
table stakes for a self-hosted segment. The defensible part is the
combination of a **deterministic, explainable planner** with **honest
incomplete data** and a **low-effort capture path** — a product that will
say "amount not set" and "based on the values available" rather than
invent a complete-looking number.

**Deliberately not claimed.**

| Not claimed | Why |
|---|---|
| Proven grocery savings | `CST-03` requires a named, comparable baseline before any savings claim. There is none. Planned versus actual spend may be shown as two labeled numbers, never as a savings result. |
| Allergy safety or clinical adequacy | `DOM-07` and spec §3.2 forbid it. Prefer "Fits your current rules" over "Safe for your allergy." |
| Global real-time price coverage | Prices are dated observations with conditions, not a universal market price (`CST-02`). |
| Autonomous ordering or supplement dosing | Explicit non-goals (spec §3.2). No checkout, no dose changes. |

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| OSS/self-host discovery | A self-hosted, no-custody, no-subscription personal tool appeals to a technical audience that distrusts health-data custody. | Public repo, honest README, QUICKSTART, a real one-week walkthrough | Issues describing the *decision* problem in other tools, not stars |
| Community content | Writing honestly about the hard part — decision fatigue and false nutrition precision — reaches people who have been burned by confident wrong numbers. | One strong written piece grounded in a real session | Unprompted replies describing the same failure |
| In-product / bundle expansion | Reaches people already inside a Vrooli lifestyle bundle with a recurring "what do I eat" question. | Bundle membership via Offer Desk (proposed, not committed) | Attach rate once bundles have users |
| Desktop / mobile packaging | A phone-first PWA is where the decision actually happens (in the kitchen, at the store). | Packaged app, store assets, brand-manager icons, install assets already scaffolded | Deferred until the outage-honest R1 loop is real |
| Search / SEO | High-intent queries such as "what to eat with what I have" or "meal plan for one person cheap." | Landing page, content | Deferred until the written piece exists |

## Launch Motion

1. Complete the v2.0 redesign: repair the foundation so the app works
   end to end, reach reference-quality fidelity against the approved mockups,
   and make R1 personally useful for the originating user, using fixtures only
   behind the explicit demo path
   ([`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md)).
2. **Run one real week on real data** — setup, accepted plan, shopping,
   cooking, recorded intake, an export — before any external claim. This is
   the dogfooding gate.
3. Exercise the honest-failure paths deliberately: provider unavailable,
   unknown nutrients, stale replan, import conflict. Record what the UI
   shows (see [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md)).
4. Decide bundle membership through Offer Desk rather than asserting it
   here.
5. Only then: channel assets, pricing research, and a landing surface.

No billing, paywall, or checkout is part of this motion. Billing is R3 and
requires separate authorization (`OT-P2-002`).

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Dinner, decided." | All. **The lead message.** | The Today hero showing one concrete next meal with Start cooking and Swap meal (`OT-P0-008`) | ready-on-build |
| "A cookbook that plans your week." | People who bounce off utilitarian trackers | Captures of Today, Week, Meals, and Cooking matching the approved mockups in both appearances (`OT-P0-021`) | pending-evidence |
| "Affordable variety without the bookkeeping." | All | Name-only capture, deterministic planning, optional feedback (`C02`, `C07`) | ready-on-build |
| "It tells you what it doesn't know." | Anyone burned by false nutrition precision | Explicit unknown/partial semantics and provenance (`NUT-03`, `ACT-040`) | pending-evidence |
| "Your food stays on your box." | Privacy-motivated self-hosters | Self-hosted SQLite runtime, no required external services (`SYS-03`, `SYS-04`) | ready-on-build |
| "Change the plan, not your rules." | Users with restrictions | Required exclusions cannot be traded against cost, effort, or variety (`OT-P0-006`, `PLN-02`) | ready-on-build |
| "We can show planned versus actual spend." | Cost-aware users | Separated allocated, checkout, and actual spend (`CST-01`, `CST-03`) | pending-evidence — never stated as savings |

Nothing here may be published while marked `pending-evidence`.

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| One real week run by the originating user, measured | Direct | Time-to-first-useful-plan is short enough that the user does not abandon setup; the plan is accepted with at most light swaps | If setup is abandoned, the low-effort thesis fails before any channel work |
| Returning-user retention over several weeks | Direct | The user keeps opening the app and accepting/reusing meals rather than reverting to the expensive default | Confirms the product displaces the default, not just explains it |
| Voluntary swap reasons collected | In-product | A meaningful share of swaps carry a reason (not "missing feedback") | Reasons teach the product; silence is genuinely unknown (`DOM-07` #13) |
| Planned versus actual spend, labeled separately | In-product | Both numbers are recorded for a real period | Never converted into a savings percentage without a comparable baseline |
| Written piece on decision fatigue / false precision | Community content | At least three unprompted replies describing the same failure | Confirms the problem is felt, not just reasoned |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — role, packaging, and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes and `OT-*` targets
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — what validation may measure
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — durable positioning and monetization choices
- [`../reference/product-specification.md`](../reference/product-specification.md) — sections 2, 3, 19, and 23
