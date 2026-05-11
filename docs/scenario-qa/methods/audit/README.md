# Scenario QA Audit Techniques — Plan of Record

This folder is the **strategic canon** for the audit lenses the `quality-auditor` member applies during structural quality audits. One paired doc + skill per technique.

These docs answer: *what is this audit lens, when does it apply, when does it backfire, what's the failure mode the qa-contrarian watches for?*

They do **not** answer: *step-by-step procedure for applying it.* That belongs in the paired skill (`scenarios/prompt-manager/store/skills/packs/core/<technique-slug>/SKILL.md`).

## Why a separate folder for techniques

Deep structural audit is not one workflow. Different audits answer different questions: `screaming-architecture-audit` asks "does the structure express the domain"; `seam-discovery-and-enforcement` asks "are variation points isolated"; `code-cleanup` asks "is dead code accumulating." These are not stages of one process; they are independent lenses that the auditor rotates through.

Keeping each lens in its own canonical home — and pairing it with a focused skill — makes it possible to:

- Reason about each lens's applicability without scrolling through a mega-skill that branches on lens.
- Compress each skill independently as Vrooli's substrate (CLIs, lint integrations, audit-specific tooling) absorbs more of the work. A unified `quality-audit` mega-skill would compress worse — same argument the marketing team uses for one skill per post type.
- Surface graduation candidates from observed patterns: when a recurring kind of audit doesn't fit any registered lens, that approach graduates to a new entry here.

This mirrors the per-entity guidance in [`docs/agent-system/TEAM_DOCS_PATTERNS.md`](../../../agent-system/TEAM_DOCS_PATTERNS.md): each lens is its own entity. It also mirrors `path:docs/marketing/methods/post-techniques/`: cross-cutting techniques get one canonical home and are referenced by the consumer (here, `quality-auditor/RESPONSIBILITIES.md` Available Skills table; the rotation order is in `team.json`'s `skillRotation`).

## Doc + paired skill discipline

Every technique ships as `doc + paired skill`. This is a hard rule, not a recommendation. Neither half is optional, and neither half replaces the other. The doc holds *reasoning*; the skill holds *procedure*. A doc with no skill is a stale shrine. A skill with no doc is brittle.

Enforced by the canon coherence test at `scenarios/prompt-manager/test/agent_system_canon_test.sh`: every `<slug>.md` (excluding `README.md`) in this folder must have a matching `scenarios/prompt-manager/store/skills/packs/core/<slug>/SKILL.md`, and every skill tagged `audit-technique` must have a matching PoR doc here.

This registry was created on 2026-05-03 specifically to close the `skillless canon` smell across the seven audit lenses already in active rotation: the procedure side existed for each, but the strategic-canon side did not — so the qa-contrarian had no operator-curated home to challenge an audit's applicability or conclusions against.

## Lifecycle

Each technique has a status:

- **v0** — Strategic canon documented, but the technique is **not yet active**. The PoR doc exists; the paired skill is missing or incomplete; the quality-auditor must not include it in the rotation yet.
- **v1** — Active. Four activation requirements:
  1. Skill is authored at `path:scenarios/prompt-manager/store/skills/packs/core/<slug>/`.
  2. Skill cites this technique's PoR doc as required reading.
  3. PoR doc Status line bumped to `v1`.
  4. `quality-auditor/RESPONSIBILITIES.md` Available Skills table references the technique, AND `team.json`'s `skillRotation` includes the slug.

Compression happens per-technique once active: each skill compresses independently as Vrooli's substrate absorbs more of the work (e.g., a future `vrooli audit deps` CLI would compress most of `code-cleanup` into a thin orchestration skill).

## Files in this folder

All seven entries below pair with skills that already exist; this registry was authored to give them a strategic-canon home.

| File | Status | Applies to |
|------|--------|-----------|
| [`screaming-architecture-audit.md`](screaming-architecture-audit.md) | v1 (paired with existing skill, 2026-05-03) | Scenarios where structure may not express domain (god modules, scattered features, opaque names). |
| [`boundary-of-responsibility-enforcement.md`](boundary-of-responsibility-enforcement.md) | v1 (paired with existing skill, 2026-05-03) | Scenarios where presentation/coordination/domain/integration concerns may have bled across boundaries. |
| [`seam-discovery-and-enforcement.md`](seam-discovery-and-enforcement.md) | v1 (paired with existing skill, 2026-05-03) | Scenarios with hard-to-test or hard-to-substitute behavior at variation points. |
| [`invariant-discovery-and-enforcement.md`](invariant-discovery-and-enforcement.md) | v1 (paired with existing skill, 2026-05-03) | Scenarios where critical conditions ("must always be true") are implicit and unenforced. |
| [`cognitive-load-reduction.md`](cognitive-load-reduction.md) | v1 (paired with existing skill, 2026-05-03) | Scenarios that are hard to read, navigate, or reason about. |
| [`decision-boundary-extraction.md`](decision-boundary-extraction.md) | v1 (paired with existing skill, 2026-05-03) | Scenarios with deeply nested, scattered, or duplicated decision logic. |
| [`code-cleanup.md`](code-cleanup.md) | v1 (paired with existing skill, 2026-05-03) | Scenarios accumulating dead code, deprecated implementations, forwarding shims, stale TODOs. |

## Adding a technique

New techniques enter the registry via a `meta-self-improvement` decision filed on `meta-optimization`. The decision must include:

1. **Why a new technique?** What audit pattern recurred in `quality-audit/<scenario-id>/<skill-id>` knowledge entries (or in `challenge-report/*` from the qa-contrarian) that doesn't fit any registered lens.
2. **Strategic-canon doc draft.** A `<slug>.md` for this folder following the structure of the existing entries: definition, when-applies, when-backfires, qa-contrarian failure modes, paired-skill citation.
3. **Skill draft.** A `scenarios/prompt-manager/store/skills/packs/core/<slug>/{skill.json,SKILL.md}` with `tags: ["audit-technique"]` and required-reading citing the PoR doc.
4. **Activation step.** Update `quality-auditor/RESPONSIBILITIES.md` Available Skills table AND add the slug to `team.json`'s `skillRotation` array.

Future candidates the audit log will surface: `performance-audit`, `security-audit`, `deprecation-audit`, `accessibility-audit`, `observability-audit`. Each enters via the same decision flow.

## Cross-references

- [`../../README.md`](../../README.md) — scenario-qa team plan-of-record overview.
- [`../investigation/README.md`](../investigation/README.md) — sister registry for the bug-investigator's techniques; same lifecycle and discipline.
- [`docs/marketing/methods/post-techniques/README.md`](../../../marketing/post-techniques/README.md) — gold-standard reference this folder replicates.
- [`docs/agent-system/PROMOTION_LADDER.md`](../../../agent-system/PROMOTION_LADDER.md) — how prose techniques mature into CLI-backed ones.
- [`scenarios/prompt-manager/store/teams/scenario-qa/team.json`](../../../scenarios/prompt-manager/store/teams/scenario-qa/team.json) — runtime contract; `skillRotation` array on `quality-auditor` enumerates the active subset of this registry.
