# Command Center

`team:director-swarm`'s **instrument** — the one address the team reads to answer *is the work we are doing producing results, and which sensor is worth building next?*

It renders that answer as a **board**: full-bleed rooms, one per outcome category, designed to run unattended on a wall panel and to be equally correct on a desktop browser, a phone, and a gamepad-controlled TV.

> **Current implementation:** Command Center is the Director Swarm's operator and agent observability board. It reads typed board/room data, reports integration health, exposes scenario CLI/API entry points, and saves editable dashboard settings. Writes are limited to its own board configuration; operational producer data stays read-only. The old 2026-04 kiosk-aggregator warning is superseded. Open source, runtime, accessibility, and validation gaps remain tracked in [docs/internal/PROBLEMS.md](docs/internal/PROBLEMS.md); read [docs/START-HERE.md](docs/START-HERE.md) before changing a surface.

## What makes it an instrument rather than a dashboard

- **It reads a space it does not own.** The objective set is read through `prompt-manager graph objectives` as a transmitter; the setpoint is a checked-in file it parses and never writes. An observer that authors its own reference model is confirming itself.
- **Two honesty axes, never merged.** Every reading carries *coverage* — is there a sensor at all — and *trust* — is this reading good right now. A source outage changes trust and never changes coverage.
- **It always shows a figure.** Where a pipeline does not exist, the board renders an authored, reviewed, stamped sample in a material that says so from ten feet away. Never an empty slot, never a generated number.
- **It counts what it cannot see.** Missing cells and unregistered outcomes are dated and age visibly, including this instrument's own blind spots.
- **It surfaces and ranks operational data; it never decides or changes producer data.** Local board settings are editable through the Settings page and persisted as dashboard configuration. A sensor grant gives no authority over the system it observes.

## Documentation

| Start here | |
|---|---|
| [docs/START-HERE.md](docs/START-HERE.md) | Reading order, and the three things most likely to be got wrong |
| [PRD.md](PRD.md) | Operational targets |
| [DESIGN.md](DESIGN.md) | The `vrooli-command-display` design contract governing every visual surface |

| Concepts | |
|---|---|
| [Instrument model](docs/concepts/INSTRUMENT-MODEL.md) | The six invariants, the production-ledger archetype, the degradation contract |
| [Coverage model](docs/concepts/COVERAGE-MODEL.md) | The two honesty axes and their closed vocabularies |
| [Provenance model](docs/concepts/PROVENANCE-MODEL.md) | The four inks, and why sample values are authored rather than generated |
| [Outcome taxonomy](docs/concepts/OUTCOME-TAXONOMY.md) | Why the rooms are derived data, and the rules that keep them migratable |
| [Source map](docs/concepts/SOURCE-MAP.md) | The fleet's declared instruments, and what each room inherits |
| [Data](docs/concepts/DATA.md) · [Architecture](docs/concepts/ARCHITECTURE.md) · [UI architecture](docs/concepts/UI-ARCHITECTURE.md) | Shapes, layers, and the board |

| Reference and memory | |
|---|---|
| [API endpoints](docs/reference/api-endpoints.md) · [CLI commands](docs/reference/cli-commands.md) | The read surfaces |
| [Decisions](docs/internal/DECISIONS.md) · [Problems](docs/internal/PROBLEMS.md) · [Progress](docs/internal/PROGRESS.md) | Why, what is broken, what changed |
| [experience/](experience/) | What each surface must communicate, authored ahead of implementation |

## Quick start

```bash
make setup                              # build API + install UI deps + build UI bundle
vrooli scenario start command-center    # ports assigned by the lifecycle manager
make status                             # running ports / PIDs
make logs                               # tail combined logs
make test                               # Go unit + vitest + BAS cases
make stop
```

Never run the binaries directly and never hardcode a port — the lifecycle manager allocates from the ranges in `.vrooli/service.json` and exposes them as `API_PORT` and `UI_PORT`.

## Canon this scenario implements but does not own

- `path:docs/agent-system/STANDING_TEAMS.md` — the instrument contract
- `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md` — the outcome categories and the gap-closure loop
- `path:docs/director-swarm/operating/OPERATING_MODEL.md` — the portfolio loop these readings close
- `path:docs/director-swarm/strategy/OBJECTIVES.md` — what every category traces upward to
