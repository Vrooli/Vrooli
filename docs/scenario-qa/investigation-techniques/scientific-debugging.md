# Technique: Scientific Debugging

**Status:** v1 (paired with existing `scientific-debugging` skill, 2026-05-03). The skill predates this PoR doc; this entry closes the `skillless canon` smell by giving the technique a strategic-canon home.

## Definition

Apply the **scientific method to debugging**: generate falsifiable hypotheses, design experiments (tests) to validate them, and systematically narrow down to the root cause. The procedure produces regression tests and documented findings that prevent recurrence.

The full procedure (Phase 0 prior-art check → Observe → Hypothesize → Test → Analyze → Fix → Verify) lives in the paired skill. This document is the strategic-canon side: when the technique applies, when it backfires, and what the qa-contrarian watches for.

## When it applies

✅ **Cause is not immediately obvious.** The bug's surface symptom doesn't point at a single line or component; multiple plausible explanations exist. The technique forces the investigator to enumerate hypotheses rather than fixate on the first one.

✅ **Initial fixes didn't work or made things worse.** A symptom-level patch is degrading the situation; root-cause discipline is required.

✅ **Multiple interacting components.** The bug spans services, scenarios, or boundaries. The technique's "minimal reproduction" exit criterion forces the investigator to narrow the surface.

✅ **The bug must be explained, not just patched.** Other agents will read the `bug-investigation-report/<slug>` audit log and need to understand why the bug happened. The technique's mandatory root-cause documentation supports this.

✅ **You want to prevent recurrence.** The mandatory regression test captures the bug as a permanent guardrail.

✅ **Default for unclassified bugs.** The bug-report taxonomy's `defaultMethod` is `scientific-debugging` for every signal type — the bug-investigator starts here unless evidence points at a more specific technique.

## When it backfires

⚠️ **Typos or obvious one-line fixes.** The hypothesis-generation overhead is wasted ceremony when the cause is mechanically visible (e.g., a misspelled identifier in a stack trace).

⚠️ **Well-understood, documented error conditions.** If Phase 0 (prior-art check) finds a known-good fix recipe, the rest of the methodology short-circuits. Re-running the full process is duplicative.

⚠️ **Bugs where the fix is already known.** When the operator or a prior decision already specifies the fix, the investigator's job is to apply and verify, not to re-investigate.

⚠️ **Performance / security / data-corruption bugs.** Different methodologies dominate (profiling for performance, containment-first for security, backup/restore for corruption). `scientific-debugging`'s "Boundaries" section explicitly disclaims these.

⚠️ **Race conditions and timing-dependent bugs.** The "design a test" step is harder when the bug is non-deterministic. The technique still applies, but specialized tooling (e.g., chaos injection, trace-replay harnesses) is often a prerequisite. Future technique entries (e.g., `differential-trace`) will cover this more directly.

## What the qa-contrarian watches for

The `qa-contrarian` member challenges bug-investigation outcomes; for `scientific-debugging` specifically, watch for:

- **Single-hypothesis fixation.** The investigator generated one plausible cause, ran one test, declared the bug solved. The discipline requires at least two hypotheses with falsification criteria; one-hypothesis investigations are confirmation bias dressed up as method.
- **Stopped at first plausible explanation.** Phase 4 (Analyze) is supposed to ask "could this cause other issues" and "could similar patterns exist elsewhere"; a too-fast close skips both. Challenge: was the root cause really identified, or just *a* cause that explains *this* observation?
- **Symptom-fix masquerading as root-cause-fix.** Phase 5's checklist demands the fix address root cause, not symptom; contrarian challenge: does the fix actually prevent the underlying mechanism, or does it just suppress this manifestation?
- **Skipped or perfunctory Phase 0.** If the prior-art check is a one-line "no prior art" without showing the queries, it's not a real check. Challenge: is the recurrence claim falsifiable from what the investigator actually ran?
- **Missing regression test.** Phase 5 mandates a failing test before the fix and the test passing after. A bug-investigation entry that omits this discipline is incomplete and may regress silently.
- **No similar-pattern check.** Phase 6 demands "could this bug exist elsewhere"; bug-investigations that skip this leave related defects unsurfaced.

## Paired skill

`scenarios/prompt-manager/store/skills/packs/core/scientific-debugging/SKILL.md` — the executable spec. Required reading: `docs/agent-system/SKILL_AUTHORING.md` and this PoR doc.

## Cross-references

- [`README.md`](README.md) — registry overview, lifecycle rules, doc + paired skill discipline.
- [`../BUG_REPORT_TAXONOMY.md`](../BUG_REPORT_TAXONOMY.md) — bug-report taxonomy; this technique is the `defaultMethod` for every signal type.
- [`../README.md`](../README.md) — scenario-qa team plan-of-record overview.
- [`docs/agent-system/SKILL_AUTHORING.md`](../../agent-system/SKILL_AUTHORING.md) — universal authoring quality bars.
