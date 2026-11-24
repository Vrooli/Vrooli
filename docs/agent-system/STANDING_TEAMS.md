# Standing teams

A standing team is a Prompt Manager team for a recurring duty that outlives any outcome ([RULES.md](RULES.md), construct table). It is justified only while it has a manager above it (R5) and a detector proven by a planted fault (R6). Everything here serves those two tests and R10: tooling that exists to make the team faster at its outcome. Finite delivery teams are retired; use a research topic.

## The loop card

A standing team's entry file opens with the five parts from RULES.md. For a standing team they mean:

| Part | In a standing team |
|---|---|
| Setpoint | The outcome the duty keeps true, its next gate and the pace (R3). The obligation list below is what makes the outcome measurable: declared by the capability owner, never by the team's own instrument. |
| Sensors | One instrument (the board): the one scenario members read for "what is the state of what I own, and what next". Each reading names the canary that proved it (R6). |
| Actuators | What the team may change: work items, skills, plans. The instrument surfaces and ranks; it never decides. |
| Manager | The loop above the team (steward or research manager). A team's own contrarian can notice a deviation; only the manager restructures it (R5). |
| Tooling | The team's skills, programs and checks, each naming the friction it removed (R10). |

A team is justified by a distinct plant, a distinct clock and a distinct failure mode, never by a new objective. A member separation is justified only by a control, a clock or a judgment boundary. A member that exists to hand state to the next member collapses once gates enforce the order.

## Setpoint: obligations, cells, four modes

An **obligation** is a responsibility derived from an objective ("produce publishable marketing content"); an **operation** is countable ("verify a claim"). A **cell** is one obligation crossed with one mode: Answer (can we know it), Validate (can we check it), Guide (is there a skill for it), Act (can an agent invoke it). Only operations can be counted, so a denominator built from obligations alone never rises above `sketch` confidence. Every denominator is owned by the capability owner and carries a confidence (`authoritative`, `partial` or `sketch`) and a rationale. Coverage, condition (does existing supply still work) and empirical (what actually hurts) stay separate numbers, and an unmeasured axis is never reported healthy. Worked example: `scenarios/meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md`.

## The instrument: six invariants (the detector rules, R6)

1. **One address.** A member starts at the board, not at the tools. Members read different rows of one surface.
2. **The setpoint is owned elsewhere.** An observer that writes its own reference model is confirming itself.
3. **Three axes, never merged.** Coverage, condition and empirical stay separate.
4. **Honest by construction.** Every ratio carries its denominator confidence. An unreachable owner reads `UNAVAILABLE` with a reason, never `0%` and never a dropped row. Numerators are computed live and never stored.
5. **Surfaces, does not decide.** Ranking and confidence only.
6. **Prose cites, never restates.** Team files name no owner, scenario or current number; they cite the board. This is what makes a team get smaller (R7), and it holds only if the prose the instrument replaced is deleted.

An unbounded-output team (what content should exist has no defensible total) keeps a production ledger instead of a coverage board: queue state, staleness against a window and outcome evidence, with no percentages.

**Degradation.** The board degrades legibly. Every member declares its fallback: if the instrument is unavailable, say so in the continuity record and use the manual path; never silently skip the board (R6). The instrument is never the only sensor watching itself: another loop watches its condition.

## Skills: one per capability, shrinking to a pointer

Skills index capabilities, not scenarios. A capability matures Guide, then Validate, then Answer; Act is a peer. A skill that has collapsed to a thin pointer is the success state. If mean skill length for capabilities with live Validate and Act cells does not fall each cycle, the scenario shipped and the prose was never deleted (R8). Skills stay per lane: a skill is a judgment method, and handing every member every skill dissolves the contrarian's separation.

Procedure for reading and acting on the instrument belongs in the instrument scenario's usage skill ([SKILL_AUTHORING.md](SKILL_AUTHORING.md#the-three-roles)), not in member files. The same paragraph in two member files is procedure that belongs to the instrument. A member keeps its lane's judgment, its fallback and its stop conditions.

## Reporting what moved

A team returns the outcome it moved, with proof, or says nothing moved (R2). A contrarian holds its team against this list, and any member reports a deviation once, as friction, to the manager, who decides the restructure:

- **No instrument**: none declared and no dated gap marker.
- **More than one address**: member files send you to several scenarios to learn one team's state.
- **State in prose**: a status, lifecycle or counter you must hand-edit to keep true.
- **An unenforceable rule**: "must" with no gate behind it.
- **A long skill over a thin scenario**: prose carries what a validator or command should.
- **An instrument owning its denominator** (invariant 2) or **an unwatched instrument** (the degradation paragraph).
- **A stale obligation**: the objective moved and the list was never re-derived.
- **Unowned procedure**: the same instrument-reading steps in two member files.
- **A pipeline-stage member** that only hands state to the next member.
- **A silent capability**: counted and green but read by nobody.
