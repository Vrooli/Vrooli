# Start Here

Command Center is `team:director-swarm`'s **instrument**: the one address the team reads to answer *is the work we are doing producing results, and which sensor is worth building next?*

It renders that answer as a **board** — six full-bleed rooms designed to run unattended on a wall, a desk, a phone, or a gamepad-controlled TV.

> **Read this first if you are about to change something.** The documentation in this scenario was rewritten on 2026-09-01 to a design the code does not yet implement. The code still implements the 2026-04 read-only kiosk aggregator. [internal/PROBLEMS.md](internal/PROBLEMS.md) lists every divergence, dated, with the requirement that closes it.

## Read in this order

| # | Document | Why |
|---|---|---|
| 1 | [concepts/INSTRUMENT-MODEL.md](concepts/INSTRUMENT-MODEL.md) | What this scenario *is*, the six invariants it must satisfy, and why its archetype forbids composite scores. Everything else follows from this. |
| 2 | [concepts/COVERAGE-MODEL.md](concepts/COVERAGE-MODEL.md) | The two honesty axes. Every number on the board answers both before it is allowed on screen. |
| 3 | [concepts/PROVENANCE-MODEL.md](concepts/PROVENANCE-MODEL.md) | How a reading's honesty becomes visible from ten feet away — the ink system, and why sample values are authored rather than generated. |
| 4 | [concepts/OUTCOME-TAXONOMY.md](concepts/OUTCOME-TAXONOMY.md) | Why the rooms are derived data rather than routes, and the rules that keep the taxonomy migratable. |
| 5 | [concepts/SOURCE-MAP.md](concepts/SOURCE-MAP.md) | The fleet this board reads, and why a room cannot be more instrumented than the team behind it. |
| 6 | [concepts/ARCHITECTURE.md](concepts/ARCHITECTURE.md) · [concepts/DATA.md](concepts/DATA.md) · [concepts/UI-ARCHITECTURE.md](concepts/UI-ARCHITECTURE.md) | How it is built, what travels on the wire, and how the board is composed. |
| 7 | [internal/DECISIONS.md](internal/DECISIONS.md) | Why the design is what it is. Read before proposing a change to any of the above. |

`DESIGN.md` in the scenario root is the display design contract (`vrooli-command-display`) and governs every visual surface. `PRD.md` holds the operational targets; `requirements/` traces them to verifiable requirements.

## The three things most likely to be got wrong

1. **Coverage and trust never merge.** "Is there a sensor" and "is this reading good" are independent fields with closed vocabularies. A source outage changes trust; it must never change coverage. Collapsing them into one status is the defect the previous model shipped.
2. **A sample value may never originate from an upstream.** Illustrative numbers are checked-in, reviewed, stamped registry data. Generating a plausible number at runtime is indistinguishable from a bug and is forbidden.
3. **There is no write path.** No work-item filing, no setpoint writes, no upstream mutation. Sensor implies no authority; an instrument that decided would be a controller with a bad boundary.

## Running it

```bash
make setup                              # build API + install UI deps + build UI bundle
vrooli scenario start command-center    # ports are assigned by the lifecycle manager
make status                             # running ports / PIDs
make logs                               # tail combined logs
make test                               # Go unit + vitest + BAS cases
make stop
```

Never run the binaries directly and never hardcode a port — the lifecycle manager allocates from the ranges in `.vrooli/service.json` and exposes them as `API_PORT` and `UI_PORT`.

## Where the canon lives

This scenario implements contracts it does not own. When this documentation and one of these disagree, these win:

- `path:docs/agent-system/TARGET_MODEL.md` — the instrument contract
- `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md` — the outcome categories and the gap-closure loop
- `path:docs/director-swarm/operating/OPERATING_MODEL.md` — the portfolio loop this board's readings close
- `path:docs/director-swarm/strategy/OBJECTIVES.md` — what every category must trace upward to
- `prompt-manager/store/teams/*/team.json` — the live instrument declarations this board reads
