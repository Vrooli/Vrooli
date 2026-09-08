# World HUD

The HUD answers "what is my swarm doing right now" without a click, and puts
every world action one click away. It reads only the simulation's read model
(`buildView`) and routes heartbeat actions through `data/actions.ts`. Member conversations use
`data/conversations.ts`, with Agent Manager as transcript authority. New world
signals are added to the sim view first.

## Surfaces

| Surface | Where | What it shows | What it does |
|---|---|---|---|
| Summary strip | top centre | Running, Gathering, Idle, Failed counts and the next heartbeat (team, T-mm:ss) | Each count toggles a filter |
| Swarm panel | bottom left | Filters (search, team, only failed), the team list with member state counts, the event ticker | Team click focuses the room; hover highlights it; ticker click focuses the actor |
| Conversation bubble | anchored near the selected agent | greeting, operator messages, agent replies and send status | Send, continue the run, open full conversation, close |
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
| Esc | dismiss the active overlay; in walking mode, release mouse capture, then return to Explore |
| ← → | orbit in Explore; look left/right while walking |
| ↑ ↓ | tilt in Explore; look up/down while walking |
| WASD | pan in Explore; walk in first/third person |
| Shift / Space | run / jump while walking |
| + / − | dolly in / out in Explore |
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


## Camera continuity

The live world remembers its camera in this browser, separately for each scene
and seed. Reloading restores Explore eye/target/zoom or the walking mode, body
position, yaw, pitch, third-person distance, and previous Explore view. A restored
view skips the establishing intro. Returning from walking to Explore retains the
view from before walking, including after a reload. Home resets the view normally.

The camera record is separate from shared WorldService preferences. Writes are
bounded during movement and flushed on page hide, visibility change, and rig
unmount. Invalid/unavailable browser storage falls back to ordinary navigation.
Restored bodies are checked against current terrain/geometry and relocate to nearby
clear ground if needed; gravity settles positions whose old support disappeared.
Mouse capture is never restored automatically. Explicit agent deep links, imported
recipes, and synthetic capture worlds keep their requested framing.

## Member conversations

Clicking an agent opens a greeting immediately in Explore, reveals its enclosure
cutaway, and holds it in place during the conversation. In walking modes,
the agent must reach a nearby conversation position and stop before the bubble
opens. Selection alone does not launch a model. Sending the first message calls
`HeartbeatService.CreateRun` with a `conversation` body; the server validates the
agent/team membership, assembles reference context without HEARTBEAT.md, and
resolves the member's declared execution profile. Unassigned personas use their
agent context without inheriting team-member attribution.

Further messages use the run's existing continuation action. The composer
releases mouse capture and contains keyboard events. Escape closes the bubble
and returns focus to the canvas. Synthetic preview agents show the greeting but
cannot send real messages. An overhead “New message” label marks an assistant
reply until its conversation is visible and read.

The browser saves up to 24 member/run pointers and read receipts, and retains at
most 200 messages per conversation in memory. Polling requests at most four
conversations at once, skips hidden documents, and stops for drained terminal
runs; selected conversations continue refreshing. Agent Manager retains the
full transcript, accessible through the bubble's link. This browser-local index
is not cross-device conversation discovery. The initial send retains its UUID
across failures/reloads; the server reuses its task and run on retry rather than
starting duplicate work. A pending first message is restored to the composer.
