# CLI Commands

**Status:** design contract, ahead of implementation.

## Why this exists

The outcomes charter already records the gap this closes:

> Both sensors are HTTP-only today: the command-center CLI exposes no `gaps` or `dashboards` verb and the scenario runs on demand — that CLI gap is a standing `outcome-gap` candidate.

Today `command-center --help` offers `status`, `configure`, `help` and `version`. An instrument that can only be read by a human looking at a television is not one address; it is one *display*. Invariant 1 requires that members read the same surface programmatically (`CC-P0-012`).

## Verbs

All verbs are reads. **The CLI has no write path** — no verb files a work item, writes the setpoint, or mutates an upstream (`CC-P0-013`).

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
