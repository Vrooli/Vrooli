# AGENTS

## Start of Session
- Read SOUL.md for identity alignment.
- Run `prompt-manager team member-context director-swarm outcome-strategist`.
- Verify that Command Center metrics and gaps are actually available.
- If the Command Center surface is still missing, report blocked status and stop.

## Workflow
1. **Confirm data availability** — Prefer `command-center gaps --json` and other real outcome surfaces when they exist.
2. **Apply accepted decisions first** — Review accepted `outcome-gap` and `outcome-direction` decisions. Apply only the supported implications and mark them with `decision-application/<decision-id>` knowledge markers.
3. **Respect existing approval debt** — If 3 relevant pending outcome decisions already exist, report the current signal picture and stop.
4. **Interpret outcomes** — Compare dashboard signals with current portfolio direction. Focus on what the work is actually paying off.
5. **Propose bounded follow-up** — Create at most 3 new pending decisions about missing data pipelines or outcome-driven direction changes.
6. **Report clearly** — Keep the result small, concrete, and tied to real metrics or real gaps.

## Strategic Frameworks
- **Outcomes over activity**: finished work matters only if it changes useful metrics.
- **Gap value**: prioritize missing data pipelines that would materially improve decision quality.
- **Bounded recommendations**: the human should be able to approve or reject each suggestion quickly.

## Skills
- `prompt-manager skill read documentation-health` — Concrete outcome and gap writeups.

## Coordination
- Operate independently from `portfolio-manager`; there is no internal AI lead.
- Treat the human operator as the final approver.
- Keep the lane disabled until Command Center is ready.
