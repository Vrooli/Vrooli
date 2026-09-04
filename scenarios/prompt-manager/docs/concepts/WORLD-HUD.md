# World HUD

The HUD answers "what is my swarm doing right now" without a click, and puts
every world action one click away. It reads only the simulation's read model
(`buildView`) and calls only `data/actions.ts`; any new signal is added to the
sim view first. With the canvas hidden (2D mode) every action still works.

## Surfaces

| Surface | Where | What it shows | What it does |
|---|---|---|---|
| Summary strip | top centre | Running, Gathering, Idle, Failed counts and the next heartbeat (team, T-mm:ss) | Each count toggles a filter |
| Swarm panel | bottom left | Filters (search, team, only failed), the team list with member state counts, the event ticker | Team click focuses the room; hover highlights it; ticker click focuses the actor |
| Agent card | bottom right (docked in 2D mode) | Name, team, state and how long, last run with a link to `/runs/:id`, error, skill count, last message | Run now, Stop, Acknowledge (Failed only), Open editor, Follow |
| Settings | the overlay gear | Scene, quality profile and auto toggle, time of day, camera home, diagnostics, and a dev-only Levers tab | Choices persist through `WorldService.SetWorldConfig` |
| 2D mode | replaces the canvas | Every actor as a row grouped by team, with state and time-in-state | Row click focuses; the agent card docks below |

## What each signal means

| Actor state | Meaning | Where the actor is |
|---|---|---|
| Idle | no run, no heartbeat due | the commons: resting, wandering, sitting at the campfire or chatting |
| Heading to desk | a run just started | walking to its desk |
| Working | an agent-manager run is active | at its desk, spinner ring above it |
| Failed | the last run failed | at its desk, red marker; stays until acknowledged, the next run, or `failedAckSeconds` |
| Heading to table / Gathered | the team's heartbeat is within `gatherLeadSeconds` | walking to, then sitting at, the team table with an amber marker |
| Chatting | idle socialising with a neighbour | facing its partner in the commons |

Events in the ticker are server signals only: run started, finished, failed
(with the error), heartbeat scheduled or cancelled, agent message, failure
acknowledged. State transitions and arrivals are not shown.

The strip's feed badge reports how signals arrive: `stream` (WorldService
stream), `polling` (5 s fallback when the stream is silent or failing),
`connecting`, or `stopped`.

## Actions

| Action | Calls | Availability |
|---|---|---|
| Run now | `HeartbeatService.TriggerHeartbeat` | members of a team that are not already running |
| Stop | `HeartbeatService.StopRunning` | members with an active run |
| Acknowledge | local `failed.acknowledged` signal | Failed actors |
| Open editor | route `/agents/:id` | always |
| Follow | camera follows the actor while it walks | when an actor is focused |
| Home | camera back to the hero pose | always |

Failures of a request are shown on the card in place; nothing is retried
silently.

## Keyboard

| Key | Effect |
|---|---|
| Esc | close the card, stop following, return home |
| ← → | orbit around the diorama (clamped) |
| ↑ ↓ | tilt (clamped) |
| + / − | dolly in / out |
| Tab | moves through every control; each has a visible focus ring |

Deep links: `/world?focus=<agentId>` opens on an actor;
`?scene=`, `?profile=`, `?period=`, `?intro=0`, `?diag=1` pin the settings.
When automatic quality changes profile, a dismissible notice names the new
profile and the diagnostics overlay retains the measured FPS and bound that
caused the verdict. Clustered room labels use a stronger stroke and a static
above-prop height so geometry cannot hide them.

## 2D mode

2D mode is used when WebGL 2 is unavailable, below 768 px, or when the
operator toggles it (persisted). The HUD test suite runs the same action
tests against the docked card in 2D mode and checks the HUD with axe at
desktop and narrow widths.
