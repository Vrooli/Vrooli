# Go To Market — Money Ledger

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

- **Audience:** operators whose money arrives through more than one place and who have at
  least one revenue source with no API — freelancers, contractors, solo founders, small
  landlords, market and craft sellers, households run deliberately. The disqualifying
  trait is having a bookkeeper; that person already has this problem solved.
- **Positioning:** *the only ledger that mixes automatic and hand-entered sources and
  still tells you which is which.* Not the most integrations, not the prettiest charts,
  not automated bookkeeping — and **not merely "local-first," which is table stakes in
  this segment rather than a differentiator.**
- **Main claim:** every figure carries its basis, and a figure that cannot be computed
  says which source is missing instead of showing a number.
- **Proof needed:** a demonstrable adapter outage in which the product shows a named gap
  where a competitor shows a stale figure or a zero. This is the single most important
  asset to produce, and it is a screen recording, not a paragraph.
- **Product-side recording:** [`outage-demo-20260816.mp4`](../internal/evidence/outage-demo-20260816.mp4)
  captures the Offer Desk board before the outage, while Money Ledger is stopped, and
  after recovery. It proves the product-side named-unavailability behaviour; it is not
  represented as a competitor comparison because no runnable alternative is part of
  this repository.

## Where it sits against the alternatives

Positioning is only meaningful against what the user does today. Scanned 2026-08-13;
this table is a snapshot and its `Status` is `operator-asserted` until `market-validator`
captures each comp through a `validation-inbox/*` entry.

| Alternative | Examples | What it does better | Where it fails the target user |
|---|---|---|---|
| Spreadsheet | — | Total flexibility, zero learning curve, already open | No provenance — a typed number and an imported one are indistinguishable a month later. Every figure is a stale snapshot the moment it is pasted. |
| Aggregator-first consumer app | Monarch, Copilot, Quicken Simplifi | Effortless when connections work; strong forecasting and categorisation | Bank connections break routinely and the failure is usually silent. Sources with no API are second-class. Business and personal money get mixed or need two subscriptions. |
| **Local-first, no-credential app** | **SenticMoney** | **Already occupies our intended posture: no bank login, local storage, runway for irregular income** | **Avoids the aggregator problem by having no aggregators — the user types everything. There is no automatic source, so there is no need to distinguish one from a manual entry.** |
| Self-hosted open source | Actual Budget, Firefly III | Free, private, mature, good UX; Firefly has a strong rules engine | Budgeting-method-first (Actual enforces envelopes) or general-purpose; neither models source trustworthiness, and both assume a user willing to self-host. |
| Plain-text accounting | Beancount + Fava, hledger, Ledger | Genuinely auditable, excellent for investing, strong web UI in Fava | Demands a double-entry mental model and a text-file workflow. Correctness is enforced at the file level, not surfaced as per-figure trust. |
| Small-business accounting | QuickBooks, Xero, Zoho Books | Accountant-ready, complete, well-integrated | Accrual machinery a cash-basis operator will never use, at a price that assumes a bookkeeper. |
| Nothing | — | Free | The operator cannot answer "how long do I have." |

**The revised read.** Local-first and no-bank-credentials are no longer differentiating —
SenticMoney sells exactly that, and Actual and Firefly give it away. The remaining gap is
narrower and more defensible: **every alternative is either fully automatic or fully
manual, so none of them needs a concept of how much a given figure can be trusted.** We
admit both through one contract, which is the only reason `basis` has to exist. Lead with
the mixed-source case, not with privacy.

The secondary gap is unchanged and still real: **honest degradation.** Every
aggregator-first alternative is at its worst precisely when a source is unavailable, which
for this user is routine rather than exceptional.

## Market rationale for the P2 expansion targets

The scan on 2026-08-13 produced four expansions that are now **operational targets**
(`OT-P2-006`…`OT-P2-009` in [`../../PRD.md`](../../PRD.md), with requirements in
`requirements/04-expansion/`). This table holds the *market* reasoning behind each; the
target holds the commitment and the requirement holds the contract.

None may be built before the P0 journal and contract are green — they are P2 because the
sequencing in the PRD is load-bearing, not because they are speculative.

| Target | Requirement | Why it surfaced | Why it fits without straining the model | Sequencing trigger |
|---|---|---|---|---|
| `OT-P2-006` recurring / expected events | `RCR-001` | The most consistently present feature across every aggregator-first competitor; forward-looking cash flow is table stakes there | An expected event is an ordinary event with basis `projected`, so every existing provenance guarantee applies to it unchanged. No second kind of truth. | `position` is green and runway is computed from real data. |
| `OT-P2-007` rule-based categorisation | `CAT-001` | Firefly III's most-praised capability, and the main labour saving every incumbent advertises | Rules set a category and are structurally forbidden from writing an amount. Model assistance is a proposal with basis `derived`, never a silent write. | `OT-P1-006` tax categorisation begins. |
| `OT-P2-008` event attachments | `ATT-001` | SenticMoney charges for receipt capture at exactly the tier we would compete with | It is the evidence layer under the categorised export already promised by `OT-P1-006`; an export without evidence is the weaker half. | Same trigger as above. |
| `OT-P2-009` per-currency reporting | `CUR-001` | A real and unserved gap for the international freelancer | Every amount already carries a currency. Report per currency, **never convert** — a converted total's basis would depend on a rate and date the scenario does not own. | A real user holds money in more than one currency. |

## Deliberately not building

| Rejected | Why |
|---|---|
| Envelope / zero-based budgeting | YNAB owns the method and Actual enforces it well. It is a *planning* discipline; this product is a *record*. Adopting it would put us in a feature war we would lose on a battlefield we did not choose. |
| Investment performance analysis | Beancount is materially better at it, and it pulls toward valuation, FX, and cost-basis machinery the PRD already excludes. `OT-P2-003` holds valuation accounts to a deliberate minimum. |
| Automated bookkeeping or categorisation as a headline | Every incumbent claims it. Claiming it invites comparison on integration count, which is the axis we cannot win and do not want to. |

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| OSS discovery | The local-first, no-cloud, no-bank-credentials posture is itself the pitch to a technical audience that distrusts aggregators. | Public repo, honest README, the outage demo | Stars/forks are vanity; the signal is issues describing the honesty failure in other tools |
| Community content | Long-form on the specific thesis — why a silent zero is worse than an error — reaches people who have been burned. | One strong written piece, the outage recording | Unprompted replies describing the same failure |
| In-product expansion | Reaches operators already inside a Vrooli bundle who have a financial question. | Bundle membership via Offer Desk | Attach rate once bundles have users |
| App stores | Broadest reach for a local-first personal-finance app. | Packaged app, store assets, scenario-specific icons (asset now exists) | Deferred until the outage demo exists |
| Web SEO | High-intent search on "why does my finance app show zero" style queries. | Landing page, content | Deferred — needs the written piece first |

Channels are listed with a posture, not activated. Activation is Offer Desk's job once it
owns the channel registry.

## Launch Motion

1. Prove the scaffold at runtime (Gate 0 — still outstanding, see `PROBLEMS.md`).
2. Build `books` + `journal`, then `ingest` with manual and file adapters.
3. **Run Vrooli's own finances on it for a full month**, including at least one real
   adapter outage. No external claim before this.
4. Produce the comparative outage demo: the same moment rendered by this product and
   by an alternative, side by side. The product-side recording above is the durable
   first half; the alternative capture remains pending until an actual alternative
   environment is available.
5. Decide bundle membership through Offer Desk rather than asserting it here.
6. Only then: channel assets, pricing research, and a landing surface.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Automatic where it can be, typed where it has to be — and it always tells you which." | All. **The lead message.** | The mixed-source contract (`OT-P0-004`, `OT-P0-005`) | ready-on-build |
| "It tells you when it doesn't know." | All | The outage demo | pending-evidence |
| "Your cash sale is a first-class citizen, not a workaround." | Sources-without-an-API operators | Manual and file adapters are ordinary adapters by design (`OT-P0-006`) | ready-on-build |
| "Personal and business in one system without mixing them." | Solo operators | The books model (`OT-P0-001`) | ready-on-build |
| "We run our own business on it." | All | Dogfooding, once step 3 is complete | pending-evidence |
| "We never ask for your bank password." | Privacy-motivated, aggregator-burned | PRD non-goal: no direct bank credential storage, ever | **demoted** — true, but SenticMoney, Actual and Firefly all say it. Supporting detail, never the headline. |

Nothing here may be published while marked `pending-evidence`. The team's own honesty
conventions apply to its own marketing claims.

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Publish the honesty thesis as a written piece | Community content | ≥3 unprompted replies describing the same failure in another tool | Confirms the wedge is felt, not just reasoned |
| Show the outage demo to 5 target operators | Direct | ≥3 identify the gap without being told what to look for | If they miss it, the UI is not communicating the product |
| Price sensitivity against captured comps | Direct | Comps captured first | Blocked on `market-validator`; do not guess |

## Risks to the positioning

- **The honesty wedge may be invisible until it fails.** A user comparing products on a
  good day sees fewer features and no benefit. Mitigation: lead with the failure case;
  the demo is the pitch.
- **The privacy lane is crowded and we are late to it.** SenticMoney sells our intended
  posture today, and Actual and Firefly give it away. Mitigation: the mixed-source claim
  above, which none of them can make. If the mixed-source claim also fails to
  differentiate, the direct-product hypothesis should be dropped rather than repositioned
  a third time — the internal role stands on its own.
- **A competitor could add a provenance field.** It is not technically hard. What is hard
  is retrofitting it through a product built on the assumption that every figure came from
  the same kind of place. Mitigation: none needed yet, but do not treat the moat as deep.
- **"Local-first" reads as "more work" to non-technical users.** Mitigation: demoted to a
  supporting message outside the OSS channel.
- **Dogfooding evidence is not demand evidence.** Stated in `MONETIZATION.md` and repeated
  here because it is the most likely self-deception on this scenario.

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
