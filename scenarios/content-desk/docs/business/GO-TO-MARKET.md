# Go To Market — Content Desk

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Verdict — deferred, not not-applicable

**Revised 2026-07-28 (D-014).** An earlier revision said this scenario has no
go-to-market motion because it is not sold. [`MONETIZATION.md`](MONETIZATION.md)
now records it as a direct-product candidate, so the honest status is
**deferred pending preconditions**, not absent.

Nothing external is scheduled. No asset is produced, no channel is opened, and
no claim is tested until the preconditions in `MONETIZATION.md` are met and the
kill signal has come back negative.

There is also a pleasing recursion worth stating rather than discovering: **the
first external campaign for this scenario would be drafted, verified, and
recorded inside this scenario.** That is the strongest available demonstration
and the strictest available test — if the desk cannot substantiate the claims in
its own launch post, it has failed its own thesis.

## Audience And Positioning

- **Audience:** teams whose marketing makes checkable claims and whose audience checks them — developer-tools and infrastructure companies first, then anyone publishing AI-assisted content at volume.
- **Positioning:** not a content tool. The category is claim substantiation, currently served at enterprise price points and enterprise weight.
- **Main claim:** *"Know which of your published posts are no longer true."*
- **Proof needed:** a real contamination report — a published post flagged because a claim it cited was re-checked and had changed. One true positive is worth more than any feature list, and it cannot be faked.

The claim above is the only one worth leading with. "Editorial workflow",
"content ledger", and "approval gates" all describe the product accurately and
sell nothing, because every competitor asserts them without meaning anything by
them.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Dev-log / builder-in-public | The build story is the pitch — a verification gate is more credible when you watch it catch something. | Dev-log posts; a screenshot of a real blocked approval. | Replies from people who recognise the problem, not compliments on the tooling. |
| Technical writing on claim rot | The idea travels further than the product; "your published claims decay" is a durable observation. | One essay; the contamination report as evidence. | Inbound from teams describing the pain unprompted. |
| Vrooli bundle | Reaches existing users with no new acquisition cost. | Bundle catalogue entry. | Attach rate among existing subscribers. |
| Direct outreach to devtool marketing teams | The segment most likely to feel the pain acutely. | A working demo on their own public claims. | Willingness to run it against their own site. |

None of these are open. They are recorded so a future decision starts from
options rather than a blank page.

## Launch Motion

1. Dogfood internally until the kill signal resolves.
2. Produce a true-positive contamination report.
3. Prove doctrine portability — a second, non-Vrooli post-type set seeded end to end.
4. Decide evidence execution (D-016) and tenancy (D-015 note in `MONETIZATION.md`).
5. Only then open a channel.

Steps 1–3 are internal and cost nothing beyond work already planned. Step 4 is
where external ambition first becomes expensive, which is why it sits behind
three cheaper filters.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Know which of your published posts are no longer true." | All segments | Contamination report | Untested |
| "Your marketing claims decay silently." | Devtool marketing | Claim re-verification history | Untested |
| "Substantiation without the enterprise suite." | Regulated-adjacent | Audit trail, per-claim evidence | Untested, and must not be used near regulated-advice framing without review |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Publish the build story as it happens | Dev-log | Recognition of the problem in replies | Continue or stop investing in the external story |
| Run the desk against a non-Vrooli claim set | Internal | Seeds cleanly with no code change | Proves or disproves doctrine portability |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging, mechanism, preconditions, kill signal
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-014, D-015, D-016
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
