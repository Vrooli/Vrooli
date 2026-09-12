# Instrument Model

**Status:** contract canon for this scenario. Derived from `path:docs/agent-system/TARGET_MODEL.md`; that document is the authority, this one states how Command Center satisfies it.

## What this scenario is

Command Center is `team:director-swarm`'s **instrument**: the one scenario the team reads to answer *what is the state of the world I own, and what should I do next?*

The team's own record already says so, and already records that it is not yet true:

```json
// prompt-manager/store/teams/director-swarm/team.json
"instrument": {
  "status": "partial",
  "archetype": "production-ledger",
  "coversScenarios": ["swarm-manager", "command-center"],
  "gapMarker": "2026-08-13 — two addresses. swarm-manager holds the portfolio
    state, so the never-own-your-own-denominator rule disqualifies it from
    grading that portfolio; command-center is the only candidate that owns
    none of what it would observe. Blocked on deciding whether the objective
    join moves out of `prompt-manager graph objectives` or is read from
    there as a transmitter."
}
```

Three things follow directly.

1. **The deviation is "two addresses."** A member currently has to know whether Swarm Manager or Command Center holds the answer, which breaks invariant 1. This scenario becoming the one address is how the marker closes.
2. **Command Center is the right candidate for a stated reason** — it owns none of what it would observe. Swarm Manager holds the portfolio state, so grading that portfolio from inside it would be an observer confirming its own reference model.
3. **The blocker has a canonical answer.** The objective join stays where it is and is read as a *transmitter* — a standardised read verb on a shared bus — because invariant 2 forbids the instrument from absorbing the reference model it measures against. Moving the join into this scenario would trade one deviation for a worse one.

## Naming

`TARGET_MODEL.md` is explicit about vocabulary, and this scenario uses it:

| Term | Meaning here |
|---|---|
| **Instrument** | The architectural role. Command Center, as a whole. |
| **Board** | The runtime surface members read. The six rooms and the ranked surface. |
| **Observer** | Why the instrument may report uncertainty about denominators it does not own. |
| **Capability owner** | A scenario on the plant side that self-instruments and exposes its own space. |

Do not call this a "manager scenario." Twenty scenarios are named `*-manager` and nineteen of them do not play this role.

## The six invariants, and how each is satisfied

| # | Invariant | How Command Center satisfies it |
|---|---|---|
| 1 | **One address.** Members read different rows of one surface, never different surfaces. | The board, the describe endpoint and the CLI carry the same reads. Swarm Manager remains a *source*, not a second address. `CC-P0-012` |
| 2 | **The setpoint is owned elsewhere.** The instrument never authors the denominators it measures against. | The objective set is read through `prompt-manager graph objectives` as a transmitter. The setpoint is a checked-in file this scenario parses and validates but has no path to write. `CC-P0-005`, `CC-P0-013` |
| 3 | **Three axes, never merged.** Coverage, condition and empirical stay separate numbers. | All three are independent payload fields with closed vocabularies, and no aggregate merges them. Condition is carried as `trust` at reading granularity; empirical is carried as the prediction-ledger verdict riding on each metric. See [COVERAGE-MODEL.md](COVERAGE-MODEL.md). `CC-P0-002`, `CC-P0-014` |
| 4 | **Honest by construction.** Every ratio carries denominator-confidence; an unreachable owner is `UNAVAILABLE` with a reason, never `0%`; numerators are computed live and never stored. | No reading is persisted in the P0 set, so a stale board is structurally impossible. Source outage degrades legibly and never mutates coverage. `CC-P0-009` |
| 5 | **Surfaces, does not decide.** | No write path of any kind. Gap findings are ranked and emitted; `outcome-strategist` proposes and the operator approves. `CC-P0-013` |
| 6 | **Prose cites, never restates.** Team documents name no owner, no scenario and no current number — they cite the board. | The outcomes charter's hand-maintained sensor-map table becomes a citation of this instrument's live surface. This is the invariant that makes the team smaller. |

Invariant 6 is the one that compounds. Without an instrument, a team's reading load grows as *scenarios × members*: every new scenario costs a tool skill plus an edit to every member file that might use it. Through an instrument, a new scenario is one row in a denominator the board already reads and the team-side edit count is zero. That property holds **only if the prose the instrument replaced is actually deleted** — adding a board on top of the existing hand-maintained tables gives you both, and both is worse than either.

## Archetype: production ledger, not coverage board

Director Swarm's declared archetype is `production-ledger`. The target model distinguishes the two:

| Archetype | When it fits | What the board returns |
|---|---|---|
| Coverage board | Supply is bounded and a denominator can be honestly authored | Ranked gaps with a ratio and a confidence per mode |
| **Production ledger** | Output is unbounded — "what outcomes should exist" has no defensible total | Queue state, staleness against a window, outcome evidence. **No percentages.** |

This is a live constraint on the board's design, not a footnote. *Do not force a coverage ratio onto a production team: a denominator nobody can defend is worse than an honest ledger.*

Concretely, for this board:

- **A composite health score is forbidden.** An aggregate "system health: 82" is a percentage over a denominator nobody authored. Where a room needs a headline figure, it uses ledger state: a count, a queue depth, a staleness age, or a count of outcome categories currently unmeasurable.
- **Coverage counts are still legitimate** — "4 of 12 outcome categories measurable" is a count with a stated denominator the objective set authors, not an invented completeness ratio.
- **Ring gauges and progress arcs need care.** A gauge implies a bounded scale. Use them only where the bound is authored (a setpoint bar), never to render a synthesised score.

The archetype belongs to the team, not to this scenario. If Director Swarm's record later declares `coverage-board`, this constraint relaxes — and that is a team decision recorded in `team.json`, not a design change made here.

## The degradation contract

A team that depends on an instrument is not fragile if three things hold, and this instrument must provide all three:

1. **The board degrades legibly.** An unreadable source becomes a visible availability entry with a stated reason — never a dropped row, never a zero.
2. **It stores no numerators**, so it is either fresh or honestly unavailable.
3. **Every member declares its fallback.** The board makes the good path cheap; it never makes the manual path illegal. Member files carry the shape: *if the instrument is unavailable, say so in the continuity record and fall back to the manual path; never silently skip the board.*

A fourth rule binds the instrument itself: **an instrument may not be the only sensor watching itself.** Command Center's own condition is watched by another loop — infra-health, through `infrastructure-manager`. This scenario reports its own blind spots (`CC-P0-011`) but must not be the only thing that notices when it is down.

## What this instrument is not

- **Not the universal outcome surface.** Terminal objectives in personal-life domains are measured by the scenario that serves them, per the objectives evidence-routing table. An outcome that fits no room is a finding against the outcomes charter, not an outcome that does not count.
- **Not a controller.** Instrumentation encodes this in tag letters: `FT` is a flow transmitter, `FIC` is a flow indicating *controller*, and the `C` is what confers the right to act. This scenario has no `C`.
- **Not the portfolio's owner.** Swarm Manager holds goals, backlog and execution evidence. This board grades outcomes; it does not hold work.

## Cross-references

- [COVERAGE-MODEL.md](COVERAGE-MODEL.md) — the two honesty axes and the closed vocabularies
- [OUTCOME-TAXONOMY.md](OUTCOME-TAXONOMY.md) — why the room list is derived rather than authored here
- [SOURCE-MAP.md](SOURCE-MAP.md) — the fleet's instrument maturity and what it means for each room
- [PROVENANCE-MODEL.md](PROVENANCE-MODEL.md) — how a reading's honesty is rendered
- `path:docs/agent-system/TARGET_MODEL.md` — the authority for everything above
