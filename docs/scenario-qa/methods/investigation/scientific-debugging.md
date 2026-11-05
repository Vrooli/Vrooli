# Technique: Scientific Debugging

**Status:** v1 (paired with existing `scientific-debugging` skill, 2026-05-03). The skill predates this PoR doc; this entry closes the `skillless canon` smell by giving the technique a strategic-canon home.

## Definition

Apply the **scientific method to debugging**: generate falsifiable hypotheses, design experiments (tests) to validate them, and systematically narrow down to the root cause. The procedure produces regression tests and documented findings that prevent recurrence.

The practical process (orient, discriminate, repair and verify) lives in the paired skill. This document is the strategic-canon side: when the technique applies, when it backfires, and what the qa-contrarian watches for.

## When it applies

✅ **Cause is not immediately obvious.** The bug's surface symptom doesn't point at a single line or component; multiple plausible explanations exist. The technique tests explanations against observations that could disprove them.

✅ **Initial fixes didn't work or made things worse.** A symptom-level patch is degrading the situation; root-cause discipline is required.

✅ **Multiple interacting components.** The bug spans services, scenarios, or boundaries. A representative reproduction or other distinguishing evidence narrows the surface.

✅ **The bug must be explained, not just patched.** Other agents will read the `bug-investigation-report/<slug>` audit log and need to understand why the bug happened. A concise explanation in the existing investigation record supports this.

✅ **You want to prevent recurrence.** A meaningful regression test can capture the bug as a permanent guardrail.

✅ **Default for unclassified bugs.** The bug-report taxonomy's `defaultMethod` is `scientific-debugging` for every signal type — the bug-investigator starts here unless evidence points at a more specific technique.

## When it backfires

⚠️ **Typos or obvious one-line fixes.** The hypothesis-generation overhead is wasted ceremony when the cause is mechanically visible (e.g., a misspelled identifier in a stack trace).

⚠️ **Well-understood, documented error conditions.** If prior work establishes a known-good fix, the rest of the methodology short-circuits. Re-running the full process is duplicative.

⚠️ **Bugs where the fix is already known.** When the operator or a prior decision already specifies the fix, the investigator's job is to apply and verify, not to re-investigate.

⚠️ **Performance / security / data-corruption bugs.** Different methodologies dominate (profiling for performance, containment-first for security, backup/restore for corruption). `scientific-debugging`'s "Boundaries" section explicitly disclaims these.

⚠️ **Race conditions and timing-dependent bugs.** The "design a test" step is harder when the bug is non-deterministic. The technique still applies, but specialized tooling (e.g., chaos injection, trace-replay harnesses) is often a prerequisite. Future technique entries (e.g., `differential-trace`) will cover this more directly.

## What the qa-contrarian watches for

The `qa-contrarian` member challenges bug-investigation outcomes; for `scientific-debugging` specifically, watch for:

- **Unchallenged explanation.** Does the evidence distinguish the proposed cause from plausible alternatives? A single well-discriminated hypothesis can suffice.
- **Symptom fix.** Does the change address the failure mechanism at the owning boundary, including related components when warranted?
- **Missed prior work.** Was relevant existing evidence reused, and were unavailable sources treated as unknown?
- **Weak verification.** Does proportionate behavior validation cover the actual defect and consequential counterexamples? A pre-fix failing test is useful when practical, not mandatory in every case.
- **Excess ceremony.** Did investigation or documentation requirements delay an understood, authorized fix?

## Paired skill

`scenarios/prompt-manager/store/skills/packs/core/scientific-debugging/SKILL.md` — the executable spec. The skill is self-contained for routine use; consult this document for QA review.

## Cross-references

- [`README.md`](README.md) — registry overview, lifecycle rules, doc + paired skill discipline.
- [`../../taxonomies/bug-report/README.md`](../../taxonomies/bug-report/README.md) — bug-report taxonomy; this technique is the `defaultMethod` for every signal type.
- [`../../README.md`](../../README.md) — scenario-qa team plan-of-record overview.
- [`docs/agent-system/SKILL_AUTHORING.md`](../../../agent-system/SKILL_AUTHORING.md) — universal authoring quality bars.
