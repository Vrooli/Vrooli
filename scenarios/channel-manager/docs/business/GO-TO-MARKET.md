# Go To Market — Channel Manager

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

**This scenario is never marketed on its own.** It reaches users as one component
of a marketing-workflow bundle, and [`MONETIZATION.md`](MONETIZATION.md) records
why: sold alone, the only available headline is warming, and that is the
positioning this project refuses. Inside a bundle whose headline is editorial and
production, it is a capability rather than a pitch.

- Audience (internal): the operator running Vrooli's brand and persona accounts,
  and the `marketing-crew` producer that needs to know whether an identity may
  carry a lane.
- Audience (external): Tier 1 `business` bundle subscribers who already came for
  the editorial and production workflow. Not a separately acquired segment.
- Positioning: *your accounts, operated on a schedule you control, with a record of
  everything done as them.* Not a growth tool, not a scheduler, and never a way to
  avoid platform enforcement.
- Main claim: withheld. This scenario has never operated a real account.
- Proof needed before any external claim: one identity graduated, posts shipped,
  and warming observations recorded (`CHANMGR-P1-006`).

### The copy constraint, stated so it survives a rewrite

The line is in the positioning, not the packaging. Any of the following would be
selling warming automation regardless of which bundle it ships in, and should be
refused at review:

- Framing warming as avoiding shadowbans, throttling, or platform detection.
- Leading with account count, or with scale of operation.
- Any survival, reach, or "accounts last longer" claim — these are also unevidenced
  and would fail `content-desk`'s claim gate on their own merits.

The defensible framing is operational: cadence you control, actions recorded,
health visible.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| None active. | No external channel is being pursued. The scenario is internal-first and its most marketable capability is deliberately not marketed. | n/a | An unprompted external request for the **audit-ledger** capability specifically — not for scheduling — would be the first real signal. |
| Dev-log mention (internal marketing) | The *build* is publishable even though the product is not. How account operations are modelled, and why warming defaults ship marked speculative, is builder-voice material. | A `content-desk` draft with claims verified against this repository. | Ordinary dev-log engagement. This is `content-desk`'s pipeline, not a channel for this scenario. |

Note the recursion, and keep it honest: this scenario is part of the machinery that
would publish about this scenario. Any such post is subject to the same claim gate
as anything else — and today the honest claim is "designed and documented," not
"working."

## Launch Motion

There is no external launch. The internal motion is:

1. Implement the P0 domains and remove the template reference domain.
2. Create one identity with attested preconditions and run a warming program
   end to end, manually.
3. Record observations against the program, converting D-002's speculative
   defaults into their first real measurement.
4. Graduate the identity, take a release from `content-desk`, and confirm the
   two-question boundary holds in practice.
5. Only then revisit whether any external story exists — and revisit
   `MONETIZATION.md`'s recommendation against selling warming automation as an
   explicit decision rather than by drift.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Accounts have an action history, not just a post history." | Internal, and the audit-ledger segments if ever pursued. | The action record model in `DATA.md`; nothing published yet. | hypothesis |
| "Warming defaults are hypotheses and say so." | Internal; useful builder-voice material. | Every descriptor's `provenance` block, `confidence: speculative`. | true today, and publishable |
| Anything about improved account survival or reach. | — | **None.** No account has been operated. | **blocked** — this is exactly the claim the gate should refuse |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Warming efficacy | Internal | Do graduated identities show materially better initial distribution than identities that posted immediately? | This is the **kill signal** in `MONETIZATION.md`. If warming shows no effect, the differentiated capability is not real and what remains is a commodity scheduler. Requires several identities and is slow by construction. |
| Audit-ledger demand | Inbound only | An unprompted request for action-history or accountability, not for scheduling. | Would reopen the external question. Not solicited. |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
