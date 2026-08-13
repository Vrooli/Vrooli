# Go To Market — Offer Desk

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Posture: no independent go-to-market

This scenario has **no standalone launch motion**, and that is a decision rather than a
gap. [`MONETIZATION.md`](MONETIZATION.md) records why: standalone demand is unvalidated,
the value is highest as a component, and Money Ledger is the marketed surface.

Offer Desk reaches users **through Money Ledger's surfaces**. Its go-to-market work is
therefore to make specific ledger claims possible, not to acquire its own audience.

Revisit this posture only when the `MONETIZATION.md` revisit trigger fires — all three of:
Money Ledger has paying users, the actuals join has run against real data for a quarter,
and three external operators have described the forgotten-condition pain unprompted.

## Audience And Positioning

- **Audience (internal):** the Vrooli `monetization` team — the only committed user.
- **Audience (through the ledger):** operators who already use Money Ledger and reach the
  question it cannot answer alone: *which of the things I sell is actually earning?*
- **Positioning, when it is expressed at all:** the half of the picture the ledger cannot
  know. Never positioned as a catalog tool, a CRM, or a product-management surface.
- **Main claim:** an offer waiting on a condition stops being something a human has to
  remember.
- **Proof needed:** a trigger that fires on its own and surfaces an offer the operator had
  genuinely forgotten. One real instance is worth more than any description, and Vrooli's
  own catalog will produce it — `trigger-met` has never once been reached by any record in
  the source documents because nothing evaluated a trigger.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| In-product expansion (via Money Ledger) | The only active channel. Offer Desk data appears inside ledger surfaces; the user never buys it separately. | The actuals join (`OT-P1-002`) and the ranked board (`OT-P1-003`) | Ledger users engaging with per-offer earned-versus-intended |
| Bundle membership | Ships inside a bundle containing Money Ledger. | Catalog membership decided by this scenario once it owns the catalog | Attach rate |
| OSS discovery | Passive only — the repo is public, but no acquisition effort is spent. | None | Unprompted interest is itself the P2 demand signal |
| Community content | **Not activated.** Content effort belongs to the ledger's thesis. | n/a | n/a |
| App stores / Web SEO | **Not activated.** No standalone app is planned. | n/a | n/a |

## Launch Motion

1. Prove the scaffold at runtime (Gate 0 — outstanding, see `PROBLEMS.md`).
2. Build `catalog` with lifecycle enforcement, then `gates`.
3. Pause the `monetization` team, run the verified import, confirm per-source-file counts,
   then delete the source files. Order is absolute (`OT-P0-006`).
4. Let scheduled evaluation run against the real catalog until a trigger fires unassisted.
   **That event is the proof asset.**
5. Build the actuals join and the board — the first work that is visible to a ledger user.
6. Do not pursue independent positioning before the revisit trigger.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "The condition is declared once and evaluates itself." | Internal; ledger users | A real unassisted trigger firing (step 4) | pending-evidence |
| "An illegal transition is refused, not discouraged." | Internal | `OT-P0-002` — enforcement in the API | ready-on-build |
| "Agents may propose; only you may promote." | Operators wary of agent autonomy | `OT-P0-005` — operator-role permission | ready-on-build |
| "Active and earning nothing is a state you can see." | Ledger users | The actuals join plus the board | pending-evidence |

Nothing marked `pending-evidence` may be published.

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Import the real catalog and run evaluation | Internal | ≥1 trigger fires that the operator had forgotten | The core capability is proven or it is not |
| Surface earned-versus-intended to ledger users | In-product | Engagement with the per-offer view | Justifies further investment in the join |
| Count unprompted external interest | OSS discovery | 3 distinct operators describing the pain | Fires part (c) of the standalone revisit trigger |

## Risks to the posture

- **Component work has no natural forcing function.** Without a launch date, the actuals
  join can drift indefinitely. Mitigation: it is `OT-P1-002`, sequenced, not "someday".
- **The graph invites scope creep toward CRM.** Deals, contacts, and pipelines are all one
  small step away and all wrong. `DOMAINS.md` excludes them explicitly.
- **Deferral can be misread as abandonment.** This document and `MONETIZATION.md` both
  state the deferral with a reason and a trigger so a future reader can tell a decision
  from neglect.

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
- Money Ledger's go-to-market: `path:scenarios/money-ledger/docs/business/GO-TO-MARKET.md`
