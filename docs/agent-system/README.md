# Agent system

Start with [RULES.md](RULES.md): the frame (cascading control, the loop card), ten rules (R1-R10), the bright lines and the choice of construct. Every section below follows from a rule or holds a fact with no other home. A section that cannot cite a rule is deleted (R7, R10).

## Files

| File | Holds | Rules |
|---|---|---|
| [RULES.md](RULES.md) | The rules. Operator-owned. | all |
| [LEVELS.md](LEVELS.md) | Home files at each level, inbox, decision packets, spawn messages | R4, R5, R9 |
| [LONG_EFFORT_DESIGN.md](LONG_EFFORT_DESIGN.md) | Sustained goals: closure, briefs, tiers, verified Codex settings, baselines | R2, R3, R5 |
| [SKILL_AUTHORING.md](SKILL_AUTHORING.md) | Skill quality bars, promotion ladder, scenario skill sets | R7, R10 |
| [SCENARIO_DEVELOPMENT.md](SCENARIO_DEVELOPMENT.md) | Plan-backed grants, work shapes, authority classes | R1, R3, R4 |
| [STANDING_TEAMS.md](STANDING_TEAMS.md) | Loop card for a Prompt Manager standing team | R6, R10 |
| [EFFORT_SUPERVISION.md](EFFORT_SUPERVISION.md) | Retirement stub: where the finite-delivery supervision facts went, and the authority list with no other home. Delete once no team record cites it. | R5 |
| [team-plan-of-record.manifest.json](team-plan-of-record.manifest.json) | Manifest contract for team plan-of-record folders, read by Prompt Manager | n/a |

Prompt Manager's topic, intake and attribution docs live with their code in [scenarios/prompt-manager/docs/internal/](../../scenarios/prompt-manager/docs/internal/). Reference-scenario and template docs live in [scenarios/meta-optimization-manager/docs/](../../scenarios/meta-optimization-manager/docs/). The work entry point is [.vrooli/work/README.md](../../.vrooli/work/README.md).

## Who may edit

`RULES.md` needs the operator. Everything else is edited by whichever manager owns the area (R5), within its grant, and is deleted if it cannot cite a rule. Skills and team files link here; they do not restate (R7). for them.

## Where does this paragraph go?

| It says | Home |
|---|---|
| What is true now: mission, state, decisions, inbox | The agent's home folder (R4, [LEVELS.md](LEVELS.md)) |
| A rule for every agent | `RULES.md` (operator only) |
| How to decide, a judgment a tool cannot hold | A skill |
| A repeated multi-call procedure | A governed program |
| One deterministic command | A CLI verb, or an action wrapping it |
| State or invariants | A scenario |
| Unverified or one-off | The home's inbox, then delete |

Skills follow [the promotion ladder](SKILL_AUTHORING.md#promotion-ladder): judgment becomes a program, a program becomes a scenario operation, and the prose it replaced is deleted (R7, R8).

## Terms beyond RULES.md

- **Action**: a discoverable wrapper (`prompt-manager discover`) around exactly one deterministic CLI command, with inputs, outputs, validation and `runEligible`.
- **CLI**: the command surface for Vrooli operations (`vrooli`, resource and scenario CLIs). Scenarios own the rules and state; CLIs are thin adapters. Cross-operation composition belongs in a program.
- **Knowledge entry**: a team-scoped record under a topic prefix in Source Ledger. Raw observations land in an inbox prefix, and the draining member retags or deletes them.
- **Objective**: what the operator wants the system to be for, in `docs/director-swarm/strategy/OBJECTIVES.md`. Only the operator writes objectives. A goal is measurable, time-boxed and agent-proposed; it serves one objective.
- **Plan of record**: durable operator-curated truth under `docs/<domain>/`. It is not a state home: live state lives in the agent's home (R4).
