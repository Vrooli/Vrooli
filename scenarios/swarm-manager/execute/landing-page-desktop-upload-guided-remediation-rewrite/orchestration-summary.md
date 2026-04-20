# Meta-Orchestrator Summary

## Source

Follow-up to `execute/deployment-manager-lpbs-desktop-release-orchestration`. The skill is currently a thin DM wrapper with a plain-text decision table. With typed errors and preflight in place, the skill can walk operators through every prerequisite with concrete fix commands.

## Codebase Inspection Already Done

- `scenarios/prompt-manager/store/skills/packs/core/landing-page-desktop-upload/SKILL.md` — current version: 161 lines, thin DM wrapper. Has a Decision Table (section 5) mapping symptoms to actions, but actions are prose, not commands an agent can auto-run.
- `scenarios/prompt-manager/store/skills/packs/core/landing-page-deploy-setup/SKILL.md` — the skill operators are pointed to when LPBS prerequisites are missing. Good candidate for cross-skill remediation links.
- Meta-orchestrator skill itself demonstrates the interactive-walkthrough pattern we want: the agent walks the user through decisions in small batches, presenting options and waiting for approval.

## Decisions Made

- Step 1 becomes `releases preflight`. Step 2 becomes gap-walkthrough. Step 3 becomes `releases start`. Step 4 becomes result surfacing.
- Agents must ALWAYS ask before running remediation commands. Do not auto-execute destructive operations.
- Skill includes a remediation table keyed by error code, not by symptom string.

## Unresolved Questions Deferred To Workshop

- **Where does the remediation table live?** Embedded in SKILL.md (self-contained, duplicates error metadata) vs. computed from DM's typed errors response (single source of truth, but agent has to fetch/parse). Recommend embedded for discoverability, with a "regenerate from DM" linting check in CI.
- **How chatty should the walkthrough be?** Explicit one-gate-at-a-time confirmation vs. "here are all 3 gates that failed, approve running all fixes in sequence?". Workshop should define the default and make it overridable.
- **What about mid-flight failures?** After `releases start` fails at publish or verify, the skill also needs to present remediation. Should post-start remediation share the same table as pre-start, or be a separate section? Recommend same table, since the error codes overlap (readiness_* could appear if LPBS changes state mid-release).
- **Fallback section**: the current 'DM unavailable' section is verbose. With guided remediation, can it collapse to a single-line pointer ("If `deployment-manager status` fails, check DM logs and restart")? Workshop decides.

## Dependency Notes

Depends on BOTH:
- `execute/deployment-manager-release-typed-remediation-errors` — skill consumes the typed error codes
- `execute/deployment-manager-release-preflight-endpoint` — Step 1 is a preflight call

Both must land before this item can complete, because the skill references commands and response shapes that don't exist until then.

## Greenfield Constraint

The current skill body gets replaced, not incrementally patched. The multi-stage CLI relay already living in the fallback section can shrink dramatically or be removed entirely depending on workshop's call on fallback verbosity.
