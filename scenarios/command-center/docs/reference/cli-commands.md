# CLI Commands

**Status:** verified implementation contract.

## Why this exists

The outcomes charter already records the gap this closes:

> Both sensors are HTTP-only today: the command-center CLI exposes no `gaps` or `dashboards` verb and the scenario runs on demand — that CLI gap is a standing `outcome-gap` candidate.

Today `command-center --help` offers the board, room, focus, open-loop, capability, gap, and integration inspection surfaces, plus explicit integration refresh and action commands. An instrument that can only be read by a human looking at a television is not one address; it is one *display*. Invariant 1 requires that members read the same surface programmatically (`CC-P0-012`).

## Verbs

Read commands never mutate business state. `integration-action` is the sole operational mutation command and requires `--confirm`; it can invoke only a lifecycle action returned for a declared scenario dependency.

### `command-center board`

The derived board shape: rooms, the outcome denominator with its confidence, and per-source availability.

```
command-center board --json
```

### `command-center room <id>`

Composed readings for one room, each with both honesty axes and its source's instrument state.

```
command-center room forge --json
command-center room ledger --samples hide --json
```

| Flag | Values | Effect |
|---|---|---|
| `--samples` | `hide` \| `mark` \| `full` | Whether illustrative readings are withheld, marked, or returned plain. Default `mark`. |

### `command-center focus`

The one ranked surface: what is worth building next, ordered, each entry naming its owner and the reason it ranks where it does. This is the verb `outcome-strategist` reads at the vision walk.

```
command-center focus --json
command-center focus --kind no-instrument --json
```

| Flag | Values |
|---|---|
| `--kind` | `untrusted-reading` \| `source-unavailable` \| `no-instrument` \| `no-pipeline` \| `unregistered-outcome` |
| `--owner` | A team id, to filter to one team's findings |

### `command-center open-loop`

Every `MISSING` cell and `UNREGISTERED` outcome, dated and aged, including this scenario's own blind spots.

```
command-center open-loop --json
command-center open-loop --older-than 90d --json
```

### `command-center gaps`

Compatibility alias for `open-loop` filtered to non-`NOW` coverage grouped by room, matching the charter's named sensor.

### `command-center describe`

The full projection of this instrument's sensor space, from the shared capabilities module. The programmatic answer to "what can this instrument see, and what can it not."

### `command-center integrations`

Lists declared integrations with lifecycle, feature, origin, criticality, and action metadata. The command uses the generated Connect client and shared protobuf contract.

```
command-center integrations --json
```

### `command-center integrations-refresh`

Refreshes the integration snapshot through the typed Command Center service.

```
command-center integrations-refresh --json
```

### `command-center integration-action <id> <action> --confirm`

Runs one eligible, allowlisted lifecycle action for a declared scenario dependency. It cannot accept arbitrary commands or mutate upstream business state.

```
command-center integration-action swarm-manager scenario_restart --confirm --json
```

## Global options

Standard `cli-core` globals apply and are placed before the command:

```
command-center [--api-base <url>] [--instance <name>] [--node <name>] [--auto-start] [--no-color] <command>
```

Note that `--node` is advertised by every scenario CLI and is not honoured by all of them; for cross-node reads prefer an explicit relay call.

## Output contract

- `--json` emits machine-readable output on every verb, and is the form agents should use.
- Human output states provenance inline. A value never prints without its coverage and trust, because a number copied out of a terminal into a document is exactly where an illustrative figure becomes a claim.
- A sample value prints with its `basis` string attached (`CC-P0-003`).
- An unreadable source prints as an availability line with its reason, never as a zero or an omitted row.
- Exit codes: `0` on a successful read even when sources degraded — degradation is data, not failure. Non-zero only when the registry or setpoint could not be parsed, or the API could not be reached at all.

## Cross-references

- [api-endpoints.md](api-endpoints.md) — the surfaces these verbs read
- [../concepts/INSTRUMENT-MODEL.md](../concepts/INSTRUMENT-MODEL.md) § invariant 1 — why the CLI is not optional


## Morning vision walk

Command Center owns the `command-center`, `command-center-vision-walk-prep`,
`morning-vision-walk`, and `command-center-improve` skills. Resolve them through
`prompt-manager skill read <id>`. The stable interactive skill id is preserved.

`command-center walk read --limit 40 --json` returns bounded authoritative board
readings with coverage, trust, empirical result, source observation time and TTL.
Authored sample values are excluded. This read does not start a walk.

`program-runtime library run command-center.vision-walk-prep --input limit=3 --json`
collects twelve phase-aligned evidence sections and independently reads the latest
checkpoint. The declared runner provides a 64 KiB output tier; the program returns
one complete JSON envelope below 60,000 bytes. Inspect `program.stdout` as JSON,
then its status, errors and `signals`. Partial evidence remains explicit. Prep
still supplements one owner CLI read: `prompt-manager team heartbeat-fleet-health
--json`; run the prep skill to synthesize and publish a durable briefing.

`program-runtime library run command-center.setpoint-read --json` reads external
binding health and learning reliability. Missing usefulness telemetry has no
numeric target. The old promoted `morning-vision-walk-prep` version 5 is only a
compatibility summary; new callers use the canonical declared program.

Checkpoint events use Source Ledger scope `team:director-swarm`, kind
`walk-checkpoint`, with JSON fields `walk_id`, `state`, and for an active event,
`resume_phase` and exact `content`. Read newest-first with an exact kind filter.
Completion and abandonment events prevent an older active checkpoint from
resurfacing. See `morning-vision-walk` for the user-controlled resume protocol.


### Owned publication, continuity, and comparison

Preparation v2 accepts `channel=operator|test` (default operator). It selects the
oldest pending work and the latest completed work within declared limits. Source
and row bounds remain visible. `signals.changes` names the prior briefing entry
and reports newly observed, changed, refreshed, and not-observed identities.
A refreshed observation timestamp alone is not a changed outcome. Missing/capped
rows never establish resolution. Read one exact owner item when a decision needs
more evidence instead of increasing every source limit.

- `command-center walk state --json`: latest briefing and checkpoint, independently.
- `command-center walk publish --request-key <attempt> --expected-previous-id <briefing-id-or-empty> --program-id <id> --envelope-json <json> --briefing <prose> --fleet-health-json <json> --json`: validate required evidence and freshness, conditionally append, and verify the durable receipt.
- `command-center walk checkpoint --request-key <transition> --expected-previous-id <checkpoint-id-or-empty> --walk-id <id> --state active --resume-phase <phase> --content <exact-text> --json`: validate the transition and return a verified receipt. Terminal states are `completed` and `abandoned`; omit active content and resume phase for those states.
- `program-runtime library run command-center.learning-read --input operation=vision-walk-prep`: effort and recurrence, preserving unavailable samples. Supply `from`, `to`, and `context_key` for comparable cohorts.

All three owner commands accept `--channel test` for isolated rehearsals. Test
records cannot become operator state. Preserve the exact request key and payload
when retrying an uncertain write; changing content under the same key is a
conflict. A different predecessor is a conflict too. Reread and resolve intent
rather than changing the key to bypass it. Owner validation checks supplied
structure and continuity; the prep skill still verifies source interpretation
and decision relevance. Publication does not approve or execute a decision.
