# Director Swarm Plan of Record

This folder is the plan of record for portfolio direction, initiative sequencing, and outcome discipline. It is owned by the `director-swarm` team in prompt-manager and curated through approved operator decisions.

The local contract is [`manifest.json`](manifest.json), which instantiates the shared plan-of-record shape from [`docs/agent-system/team-plan-of-record.manifest.json`](../agent-system/team-plan-of-record.manifest.json).

## Start here for agents

Use this README first, then choose the module that matches the work:

| Question | Start with |
|---|---|
| How does the director-swarm team operate end to end? | [`operating/OPERATING_MODEL.md`](operating/OPERATING_MODEL.md) |
| Which work should matter most, and why? | [`strategy/PORTFOLIO_PHILOSOPHY.md`](strategy/PORTFOLIO_PHILOSOPHY.md) |
| What sequence of initiatives are we steering toward? | [`strategy/ROADMAP.md`](strategy/ROADMAP.md) |
| What outcome categories should the Command Center eventually close? | [`evidence/OUTCOMES_CHARTER.md`](evidence/OUTCOMES_CHARTER.md) |
| Does a proposed direction drift from the operator's north star? | [`../../VISION.md`](../../VISION.md) |
| Does a proposal depend on how Vrooli technically works? | [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) |

## Folder map

| Folder | Purpose |
|---|---|
| [`operating/`](operating/README.md) | Team operating contract and validation commands. |
| [`strategy/`](strategy/README.md) | Portfolio philosophy and thematic roadmap canon. |
| [`evidence/`](evidence/README.md) | Outcome framing and Command Center gap-closure evidence. |
| [`governance/`](governance/editing.md) | Editing authority, adoption validation, and changelog. |

## Editing rules

- Agents never write to plan-of-record canon directly during heartbeat work.
- Proposed changes go through `director-swarm` decisions and are applied only after approval.
- Use the most specific module: operating contract under `operating/`, directional portfolio truth under `strategy/`, and outcome or metric framing under `evidence/`.
- Swarm Manager remains authoritative for live initiative status, dependencies, backlog items, and execution state.

Decision-context detail lives in [`governance/editing.md`](governance/editing.md).

## Cross-references

- [`docs/agent-system/PLAN_OF_RECORD_STRUCTURE.md`](../agent-system/PLAN_OF_RECORD_STRUCTURE.md) - shared PoR architecture and extension rules.
- [`../../VISION.md`](../../VISION.md) - operator-authored manifesto and north star.
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) - canonical technical reference for how Vrooli works.
- [`../monetization/`](../monetization/README.md) - monetization strategy and revenue truth consumed by director-swarm.
- [`../meta-optimization/`](../meta-optimization/README.md) - self-improvement loop that feeds director-level capability gaps.

## Future PoR work

- Add `interfaces/` only if the team needs standalone input/output/consumer contracts beyond the operating model.
- Promote stable Command Center metrics into [`evidence/OUTCOMES_CHARTER.md`](evidence/OUTCOMES_CHARTER.md) once the dashboard surfaces are live.
- Add PoR manifest validation once prompt-manager consumes `manifest.json`.
- Split roadmap themes into separate strategy packages only if one-theme-per-file editing becomes safer than the current compact roadmap.
