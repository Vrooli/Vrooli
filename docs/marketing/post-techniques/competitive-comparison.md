# Technique: Competitive Comparison

**Status:** v1. Authored 2026-04-28 (walk #5 divergence #4) extracting the framing pattern observed in viral OSS-framework marketing (e.g., the jcode-vs-Claude-Code-and-Codex post). Cross-cuts `dev-log`, `scenario-spotlight`, and the planned `oss-framework` post type.

## Rule

Frame a Vrooli scenario, capability, or the framework itself **against a named alternative the audience already knows**, leading with concrete, verifiable differentiators. The reader's mental model already contains the alternative; comparison anchors our claim against that model rather than asking the reader to build a new one from scratch.

The comparison must pass the contrarian gate: every named-alternative claim must be cited, every multiplier must be measured against a named benchmark, and the differentiator must be genuine — not a strawman.

## Why it matters

Reader attention is finite. A post that says "Vrooli does X, Y, Z" asks the reader to build a fresh mental category. A post that says "Vrooli does X better than Tool-The-Reader-Already-Uses, here's how" piggybacks on an existing category and converts faster.

The technique is also a forcing function for clarity: comparison surfaces what's *actually* different about our tool versus alternatives. If after drafting a comparison post the differentiator looks weak, that's a signal to either pick a different angle or invest in a real differentiator before posting.

The corresponding contrarian failure modes are **uncited-multiplier**, **unfair-comparison**, and **comparison-without-genuine-differentiator**. All three are covered below.

## When it applies

- ✅ `oss-framework` posts (planned post type) — comparing Vrooli as a framework against named OSS or commercial agentic-app substrates.
- ✅ `scenario-spotlight` posts — when the scenario competes with a specific named tool the audience knows (e.g., LPBS vs. a specific landing-page builder; web-console vs. a specific browser-IDE).
- ✅ `dev-log` posts — occasionally, when a milestone is best understood as catching up to or surpassing a named alternative.
- ➖ Avoid when there is no widely-known direct alternative; manufactured competitors waste the post.
- ❌ Never when the differentiator is fabricated, partial, or dependent on conditions the reader won't have.

## Sub-pattern: hyperbolic-but-verifiable multipliers

Viral comparison posts often lead with a bold multiplier ("20x more memory efficient", "63x faster spawn", "10x cheaper"). The pattern works *only* when the multiplier is:

1. **Measured.** A specific benchmark with a specific methodology that produced the number. Not "feels faster"; not "much more efficient."
2. **Cited.** The post (or a one-click link) shows the benchmark, the methodology, and the conditions under which the multiplier holds.
3. **Honest about scope.** "20x faster on cold-start spawning of N parallel agents on hardware X" is honest. "20x faster than Claude Code" without qualification is overclaim.
4. **Attached to the differentiator the post is actually selling.** Don't lead with a 20x on a metric the audience doesn't care about.

The contrarian's `feature_claims=measured` honesty flag is the gate. A multiplier without a citation lands as `feature_claims=overclaimed` and the contrarian rejects.

When Vrooli has a genuine multiplier (e.g., "agent-built scenarios ship N× faster than human-built equivalents on benchmark X"), this sub-pattern is the way to lead. When we don't, leading with a multiplier is dishonest and converts short-term while burning long-term.

## How to apply it correctly

1. **Pick one named alternative.** Comparing against a basket of three tools dilutes the framing. One alternative, one differentiator (or one cluster of related differentiators), one post.
2. **State the comparison early.** The hook or first line names the alternative and the axis of comparison. Don't bury it.
3. **Be specific on the differentiator.** Not "more flexible" — *what* flexibility, measured *how*. Not "better at X" — what does "better" mean operationally.
4. **Acknowledge what the alternative does well.** A comparison that paints the alternative as wholly inferior reads as hostile and invites pushback. Acknowledging the alternative's strengths on dimensions other than ours signals that the comparison is calibrated, not motivated reasoning.
5. **Lead with evidence, not vibes.** Show the benchmark, the demo, the screenshot, the diff. The body delivers the evidence the hook promises.
6. **Honest about caveats.** Conditions where the alternative still wins, or where our differentiator doesn't yet apply, get named in the post — not buried.

## Contrarian failure modes

| Failure mode | What it looks like | What to check |
|---|---|---|
| **Uncited multiplier** | "20x faster than Claude Code" with no benchmark linked or described. | Reject unless a one-click-reachable benchmark with methodology and conditions is provided. |
| **Unfair comparison** | Comparing our optimized path against the alternative's worst-case configuration; comparing our v3 against their v1; comparing our recent benchmark against their old one. | Verify the alternative's configuration matches what its typical user would deploy; verify version parity; verify benchmark recency. |
| **Comparison without genuine differentiator** | The post asserts "we're better at X" but X is a tie or marginal in actual measurement. | Ask: would a fair-minded reader who used both tools agree that we're meaningfully ahead on X? If no, the differentiator is manufactured; pick a real one or skip the post. |
| **Strawman alternative** | The named alternative is described in a way its actual users would not recognize; capabilities are misstated to make ours look better by contrast. | Verify each capability claim about the alternative against its current docs / readme / changelog. |
| **Hostile tone** | The comparison reads as attack rather than calibration; the alternative's authors / users feel insulted. | Reject; reword to acknowledge what the alternative does well before pivoting to our differentiator. |
| **Stale comparison** | The benchmark / capability claim was true 6 months ago but the alternative has since improved. | Time-stamp the comparison; require freshness check on republish. |

Honesty flags the publisher must attach to a competitive-comparison draft (extending the per-type honesty-flag schema):

- `comparison_basis=measured | partial | aspirational` — `measured` requires a cited benchmark or explicit feature-by-feature check; `partial` means some claims are measured and others are reasoned; `aspirational` is the upper bound and requires explicit hedging in the copy.
- `alternative_version_pinned=YES | NO` — the named alternative's version (and approximate date) the comparison was made against. NO is a reject reason for any non-trivial comparison.

## What's allowed

- ✅ Direct named comparison with cited benchmark and acknowledged caveats.
- ✅ "Vrooli does X that Tool-Y can't do today, here's the demo" with a one-click demo link and a clear scope statement.
- ✅ Multipliers that are measured, cited, and scope-honest.
- ✅ Comparisons that acknowledge the alternative's strengths on dimensions other than ours.

## What's prohibited

- ❌ Multipliers without citations.
- ❌ Comparisons against strawman versions of the alternative.
- ❌ Comparing against a basket of three tools in one post.
- ❌ Hostile or mocking framing of the alternative.
- ❌ Stale comparisons republished without a freshness check.

## Cross-references

- `STRATEGY.md` — voice canon (builder-not-marketer; honest about struggles applies inversely here as honest about caveats).
- `post-types/scenario-spotlight.md` — Comparison variant in the conversion-rate-friendly variants section; this technique is its canonical home.
- `post-techniques/recommendation-framing.md` — companion when the comparison is third-party-attributed.
- `post-techniques/hook-vs-body-asymmetry.md` — comparisons should lead the hook with the differentiator and let the body deliver the evidence.
- `post-techniques/intro-on-first-mention.md` — when the named alternative may be unfamiliar to part of the audience, apply on top.
