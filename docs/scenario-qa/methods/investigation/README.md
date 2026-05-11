# Scenario QA Investigation Techniques — Plan of Record

This folder is the **strategic canon** for techniques the `bug-investigator` member applies when draining `bug-inbox/*`. One paired doc + skill per technique.

These docs answer: *what is this technique, when does it apply, when does it backfire, what's the failure mode the qa-contrarian watches for?*

They do **not** answer: *step-by-step procedure for applying it.* That belongs in the paired skill (`scenarios/prompt-manager/store/skills/packs/core/<technique-slug>/SKILL.md`).

## Why a separate folder for techniques

Bug investigation is not one monolithic workflow. Different bugs benefit from different methodologies — `scientific-debugging` is the right shape when the cause is unknown and falsifiable hypotheses must be generated; future entries like `bisect-debugging` are the right shape when a regression's root cause hides somewhere in a known git range. Keeping each technique in its own canonical home — and pairing it with a focused skill — makes it possible to:

- Reason about each technique's applicability in one place rather than across a mega-skill's branching.
- Compress each skill independently as Vrooli's substrate (CLIs, debug tooling, reproduction harnesses) absorbs more of the work. A unified `bug-investigate` mega-skill that branches on technique would compress worse — same argument the marketing team uses for one skill per post type.
- Surface graduation candidates from observed patterns: when the bug-investigator's audit log (`bug-investigation-report/<slug>` entries) shows a recurring approach that doesn't fit any registered technique, that approach graduates to a new entry here.

This mirrors the per-entity guidance in [`docs/agent-system/TEAM_DOCS_PATTERNS.md`](../../../agent-system/TEAM_DOCS_PATTERNS.md): each technique is its own entity. It also mirrors `path:docs/marketing/post-techniques/`: cross-cutting techniques get one canonical home and are referenced by the consumer (here, `bug-investigator/RESPONSIBILITIES.md` Available Skills table).

## Doc + paired skill discipline

Every technique ships as `doc + paired skill`. This is a hard rule, not a recommendation. Neither half is optional, and neither half replaces the other. The doc holds *reasoning*; the skill holds *procedure*. A doc with no skill is a stale shrine. A skill with no doc is brittle.

Enforced by the canon coherence test at `scenarios/prompt-manager/test/agent_system_canon_test.sh`: every `<slug>.md` (excluding `README.md`) in this folder must have a matching `scenarios/prompt-manager/store/skills/packs/core/<slug>/SKILL.md`, and every skill tagged `investigation-technique` must have a matching PoR doc here.

## Lifecycle

Each technique has a status:

- **v0** — Strategic canon documented, but the technique is **not yet active**. The PoR doc exists; the paired skill is missing or incomplete; the bug-investigator must not apply it yet.
- **v1** — Active. Four activation requirements:
  1. Skill is authored at `path:scenarios/prompt-manager/store/skills/packs/core/<slug>/`.
  2. Skill cites this technique's PoR doc as required reading.
  3. PoR doc Status line bumped to `v1`.
  4. `bug-investigator/RESPONSIBILITIES.md` references the skill in its Available Skills table.

Compression happens per-technique once active: each skill compresses independently as Vrooli's substrate absorbs more of the work (e.g., a future `vrooli debug bisect` CLI would compress most of `bisect-debugging` into a thin orchestration skill).

## Files in this folder

| File | Status | Applies to |
|------|--------|-----------|
| [`scientific-debugging.md`](scientific-debugging.md) | v1 (paired with existing `scientific-debugging` skill, 2026-05-03) | All bug-inbox signal types where the cause is not immediately obvious; the default technique (`taxonomies/bug-report/taxonomy.json` defaults every signal type to it). |

## Adding a technique

New techniques enter the registry via a `meta-self-improvement` decision filed on `meta-optimization`. The decision must include:

1. **Why a new technique?** What pattern recurred in `bug-investigation-report/<slug>` audit entries that doesn't fit any registered technique.
2. **Strategic-canon doc draft.** A `<slug>.md` for this folder following the structure of `scientific-debugging.md`: definition, when-applies, when-backfires, qa-contrarian failure modes, paired-skill citation.
3. **Skill draft.** A `scenarios/prompt-manager/store/skills/packs/core/<slug>/{skill.json,SKILL.md}` with `tags: ["investigation-technique"]` and required-reading citing the PoR doc.
4. **Activation step.** Update `bug-investigator/RESPONSIBILITIES.md` Available Skills table to include the new technique.

The bug-investigator surfaces graduation candidates in its heartbeat output as `meta-self-improvement` proposals. Operator approval activates the technique.

## Cross-references

- [`../../README.md`](../../README.md) — scenario-qa team plan-of-record overview.
- [`../../taxonomies/bug-report/README.md`](../../taxonomies/bug-report/README.md) — bug-report taxonomy; each signal type's `defaultMethod` resolves to a technique here.
- [`../audit/README.md`](../audit/README.md) — sister registry for `quality-auditor`'s audit lenses; same lifecycle and discipline.
- [`docs/marketing/post-techniques/README.md`](../../../marketing/post-techniques/README.md) — gold-standard reference this folder replicates.
- [`docs/agent-system/PROMOTION_LADDER.md`](../../../agent-system/PROMOTION_LADDER.md) — how prose techniques mature into CLI-backed ones.
