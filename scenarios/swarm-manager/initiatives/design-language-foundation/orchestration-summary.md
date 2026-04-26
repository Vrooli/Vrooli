# Design-Language Standard + Component Library + Generation Guardrails — Orchestration Summary

## Strategic Rationale

Vrooli scenarios drift visually because there is no shared design language. The pattern matches the voice-canon failure caught the same day: `docs/marketing/STRATEGY.md` had real voice canon but no enforcement, so agents drifted and the operator was unaware the canon existed. Design has no canon at all yet — the failure mode is even more severe.

Compounding this, the existing image-generation initiatives (`ai-image-generation-foundation`, `decision-question-visuals`, `decision-visual-grounding-propagation`) cover BAS-grounded mockups for *existing* scenarios but have no style anchor for *new* scenarios. Without a canon as input, AI-generated mockups for new-scenario decisions will diverge. The mockup-generation work is "well-structured at the planning level" (per 2026-04-25 audit) but cannot deliver consistent new-scenario output until this initiative provides Phase 1 output.

This initiative is the operator-declared top priority of three infrastructure initiatives surfaced at the 2026-04-25 vision walk. The other two — BAS-readiness, hosted-Vrooli-deployment-research — are tracked separately. The walk explicitly framed all three as upstream of bundle-headliner credibility (web-console-readiness, GCT cluster).

Origin: 2026-04-25 vision walk (third instance of morning-vision-walk skill). Operator surfaced design-language gap during open floor; image-gen connection caught mid-walk during Phase 8 audit.

## Cross-Item Decisions

- **Fold react-component-library into this initiative as the implementation arm.** The scenario already exists at v0.0.1 with full structure (api/bas/cli/ui/PRD.md) but has no driving initiative. Per "option B" decision on 2026-04-25 walk: design-language defines the canon; react-component-library builds the React rendering of that canon. One initiative, two artifacts. The alternative (parallel initiatives) was rejected because it produces "design says X, components say Y, no enforcement" drift.

- **Generation-time enforcement is non-negotiable.** Documentation-without-enforcement is the explicit failure mode this initiative repairs. Voice-linter (idea/marketing-voice-linter, filed 2026-04-25) is the precedent — the *design-linter* sibling within this initiative is the second instance of the same substrate. Both pre-flight gate agent generation against canonical rules; both surface violations before drafts hit any human-or-decision queue.

- **The agent-generation-guardrails substrate generalizes beyond design.** Voice + design + code-style + accessibility + branding all want the same enforcement gate. This initiative ships the design instance of it; the substrate itself becomes a reusable capability. Per operator's `feedback_duplicate_before_extract.md` rule, the guardrails substrate is *not* extracted as a shared scenario today — it's instantiated twice (voice-linter, design-linter), allowed to mature, then extracted later as a dedicated scenario when a third instance lights up.

- **Cross-edge to ai-image-generation-foundation, not absorption.** The existing image-gen initiatives stay where they are. Phase 2 of this initiative wires the canon into the existing `execute/ui-mockup-generation-skill` item via a new `depends_on` edge. That is the *only* cross-initiative edge. Absorbing image-gen into this initiative was considered and rejected — image-gen has its own coherent shape (vendor research, key plumbing, CLI helper, schema work) that does not benefit from being merged.

- **Scenario-launcher integration is in scope.** When a new scenario is created, the operator picks a canon variant; the chosen canon lands at the per-scenario canonical path. This must happen in scenario-launcher's template-instantiation flow, not as a manual post-create step. Otherwise the canon is documented but unenforced at creation time — same drift failure mode as voice.

- **Brand-manager is upstream-eventual, not blocking.** Visual identity primitives (colors, typography, logos) eventually live in brand-manager's structured storage. The design canon inherits from that store when it exists. Until brand-manager scenario ships, the canon files carry their own copy. Cross-edge to add later via canon-versioning, not blocking Phase 1.

## Sequencing Notes

The intended phase order (workshop will refine — this is not prescriptive):

1. **Phase 1 — Research + canon authoring.** Evaluate Google's open-source design.md standard (announced 2026-04 timeframe; verify current shape). Design 2-3 default canon variants. Land them at the canonical project path. Document the canonical per-scenario path that knowledge-observatory will enforce.

2. **Phase 2 — Image-gen integration (closes new-scenario mockup gap).** Wire chosen canon into `ui-mockup-generation-skill` (existing item, ai-image-generation-foundation initiative). Add explicit `depends_on` edge. Without this, image-gen for new scenarios continues to drift.

3. **Phase 3 — React-component-library implementation.** Component primitives that conform to and render the canon. Existing scenario (v0.0.1) becomes the build target. Design-language drives its development priorities now.

4. **Phase 4 — Generation-time linter (design-linter).** Sibling to marketing-voice-linter. Two layers: deterministic (rule-based blocks for canonical violations) + LLM-judged (does this match the canon's principles). CLI surface; pre-flight gate inside any agent UI/component generation flow.

5. **Phase 5 — Knowledge-observatory enforcement + scenario-launcher integration.** Knowledge-observatory verifies per-scenario design.md exists at canonical path. Scenario-launcher templates include the canon at create time. Both close drift loops.

## Pattern Note for Future Guardrails Initiatives

This is the second instance of the agent-generation-guardrails substrate (voice-linter is the first; design-linter is this initiative's Phase 4). Per operator's `feedback_duplicate_before_extract.md` rule: do not extract a shared "agent-generation-guardrails" scenario or skill yet. Let voice-linter and design-linter mature independently; if a third instance lights up (code-style? accessibility? branding?), the pattern is mature enough to extract as a dedicated scenario.

When extraction happens, the shape will likely be: a `generation-guardrails` scenario or skill that takes (canon-path, generated-output) and returns (pass | violations[]) — with deterministic and LLM-judged layers as plug-ins. Until then, voice-linter and design-linter each instantiate the pattern internally.

## Open Workshop Questions

- Where does the canonical project-level path live? `docs/design/` vs `scenarios/scenario-launcher/templates/design/` vs root-level. Workshop pick.
- How are per-scenario overrides expressed? Inheritance from project canon with diff-style override, or full copy? Versioning matters once brand-manager ships.
- What is the linter's pass/fail threshold for the LLM-judged layer? Voice-linter is precedent — match its threshold and surface format unless workshop finds a reason to diverge.
- How does react-component-library's existing UI_IMPLEMENTATION.md reconcile with the canon? Existing implementation may already encode design choices that the canon will override — need a migration path, not a hard reset.
