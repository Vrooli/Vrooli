# experience/ — UX Contract

This folder is the scenario's **experience contract**. It is the design-first sibling of `requirements/`: requirements say what the scenario does, `experience/` says what the board must communicate.

These specs were authored on 2026-09-01 **ahead of implementation**, deliberately. This board's whole reason to exist is that a number on a wall is believed, so the honesty obligations have to be fixed before pixels harden around them. A spec written after the fact describes what was built; these describe what must be true.

Run `experience-manager spec validate command-center --json` after edits.

## Depth

Page specs here are at **L3** — identity, route, purpose, target linkage, communication priorities, elements, claims, bindings and explicit states — with one L4 journey. Machine-tier claims are only added where the surface has a stable selector; everything else is `manual-review` until the implementation lands.

## The three rules every surface inherits

Every page in this contract inherits these, and a page that breaks one is wrong regardless of what its own spec says:

1. **No figure appears without the qualifier that makes it honest.** A value never renders alone. Live shows its freshness, cached its age, in-reach what is needed, missing its owner and the days it has stood.
2. **No unreadable source is ever allowed to look healthy.** A source outage becomes a visible availability entry with its reason. It never becomes a zero, a dropped row, or a change in coverage.
3. **No surface offers a control the scenario has no right to perform.** There is no write path. A button that files work, writes a setpoint, or mutates an upstream would be the instrument becoming a controller, rendered in pixels.

A fourth rule is specific to this board because it runs unattended in a room:

4. **A room never renders blank.** A mounted scene that draws nothing is a failure, not a pass. Every scene surface falls back to a composed still frame.

## Surfaces

| Spec | What it is |
|---|---|
| `pages/room.json` | The generic room surface. Six rooms share one spec because the composition varies and the honesty contract does not — a per-room spec would duplicate the obligations six times and let them drift. |
| `pages/focus.json` | The one ranked surface: what is worth building next, ordered, each entry naming its owner. |
| `pages/open-loop.json` | The self-report: every missing cell and unregistered outcome, dated and ageing, including this instrument's own blind spots. |
| `pages/control-bar.json` | The hidden control layer — revealed on input, 64px targets, every command acknowledged. |
| `journeys/find-next-sensor.json` | The vision-walk journey: board → focus → the owning team, without a write action anywhere in it. |

## What is deliberately not here

- **No settings page.** Configuration is URL parameters and checked-in files. A settings surface on an unattended display is an attack surface and a maintenance burden with no reader.
- **No per-metric detail page.** Inspecting a reading reveals its provenance in place. Navigating away from the board to understand a number defeats a board.
- **No edit affordance of any kind**, anywhere, for any reason.
