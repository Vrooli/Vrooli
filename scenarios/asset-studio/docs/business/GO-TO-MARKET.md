# Go To Market — Asset Studio

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

- Audience: solo operators and small teams running more than one brand or
  persona, who already generate marketing media and have felt it drift.
- Positioning: **not "generate images and video" — every tool does that.**
  *The same character, six months later.* An identity is a versioned record,
  every artifact records exactly what produced it, and nothing ships that failed
  a consistency check.
- Main claim: an artifact can be explained and re-made. Provenance is complete
  enough to regenerate from (`ASSET-P1-010`), and that is testable rather than
  rhetorical.
- Proof needed: one identity rendered across several artifacts over time, shown
  side by side, with the provenance chain visible. That artifact does not exist
  and cannot be faked — which is the point.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Dev-log on X and blog | The build itself is the story — an identity registry with immutable versions is an unusual answer to a problem most people solve with a prompt library. | A `dev-log` draft through `content-desk`, with claims verified. | Replies describing drift as a problem they have. Not follower counts. |
| Scenario spotlight | Once the loop produces published output, the before/after of a drifting persona versus a versioned one is the demo. | A `scenario-spotlight` draft; the side-by-side artifact above. | Someone asking to use it. |
| AI-builder communities | The category is crowded, so the interesting claim is the gate, not the generation. | A written argument, not a product page. | Substantive disagreement about whether the gate is worth the friction — that is more informative than agreement. |
| Enterprise DAM adjacency | Rejected for now. | n/a | The buyer profile, price point, and compliance surface are a different product. |

## Launch Motion

1. Build the P0 slice: one identity, one still image, released through the gate.
2. Consume that artifact from a `content-desk` draft, proving the reference seam.
3. Run the loop long enough to produce the consistency evidence — the same
   identity across several artifacts, with provenance.
4. **Only then** decide whether this is internal capability, direct product, or
   bundle component. `MONETIZATION.md` records the hypothesis; the decision is
   not due until step 3 produces evidence.
5. Add channel-specific assets after the role is clear, using this scenario to
   produce them — which is also the strongest possible dogfooding signal.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "The same character, six months later." | Operators running personas | None yet — needs the longitudinal artifact from step 3. | drafted, unproven |
| "Every asset can tell you exactly how it was made." | Builders, provenance-minded | `ASSET-P0-007` plus `ASSET-P1-010` regeneration. Testable. | drafted, unproven |
| "Nothing ships that failed the consistency check." | Brand-conscious operators | `ASSET-P0-010`. Depends entirely on the gate working. | **blocked on the kill signal** |
| "Faster AI-UGC production" | — | Not our claim. The funded tools in this category win on volume and speed; competing there would be a losing framing. | rejected |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Publish the build story as a dev-log | X, blog | Any substantive reply describing drift as a lived problem. | Weak signal either way — informs framing, not the product decision. |
| Show the longitudinal consistency artifact | X, blog | Someone asks how to run it. | The first real demand signal. Until then, external interest is unmeasured, not absent. |
| Ask three operators whether they would pay to prevent drift | Direct | Two of three describe a current manual workaround. | Nobody has been asked. This is the cheapest experiment here and it has not been run. |

**Sequencing note.** Every experiment above depends on `content-desk` being able
to publish, which depends on paired post-type skills and an activated channel —
none of which this scenario controls. Marketing this before the internal loop
runs would also violate the claim gate the sibling scenario exists to enforce,
which would be an unusually visible way to undermine both.

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — the unvalidated conformance comparison this positioning rests on
