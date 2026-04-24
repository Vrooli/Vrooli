# TOOLS

## Tool Access
`prompt-manager skill read <skill-id>`

## Primary Skills
- **swarm-manager-backlog-tools** — Initiative and backlog inspection commands.
- **swarm-manager-recommendations** — Approval-gated backlog proposal authoring.
- **documentation-health** — Keep decisions, proposals, and handoffs concrete and readable.

## Primary Surfaces
- `swarm-manager overview`
- `swarm-manager initiatives list`
- `swarm-manager initiatives list --scenario <csv>` — initiatives whose member items target any of the named scenarios.
- `swarm-manager initiatives get --name <initiative>`
- `swarm-manager initiatives context --name <initiative>` — single-call neighborhood for one initiative (members, upstream, downstream).
- `swarm-manager initiatives context --scenario <name>` — single-call coverage rollup for a scenario (initiatives targeting it, orphan items targeting it, combined completion rollup).
- `swarm-manager backlog list --scenario <csv>` — every backlog item targeting the named scenario(s).
- `swarm-manager stats summary`
- `prompt-manager team decision-list director-swarm --status=<status> --context=<context>`
- `prompt-manager team knowledge-list director-swarm --topic=decision-application/<decision-id>`

## Scenario Coverage Enumeration
Before proposing any initiative whose name or intent is scoped to a specific scenario (`<scenario>-readiness`, `<scenario>-launch-prep`, `<scenario>-paid-cutover`, etc.), enumerate what already exists:

1. `swarm-manager initiatives context --scenario <scenario>` — one call that returns every initiative targeting the scenario, every orphan item targeting the scenario, and the combined rollup.
2. If the combined output is non-empty, the proposal must be framed as an **umbrella initiative** with `depends_on` edges to each existing initiative, and its own items scoped to the *gaps* in coverage (including adoption of any orphan items).
3. If the output is empty, a greenfield proposal is warranted — note in the decision notes that no prior coverage was found.

Never propose a parallel scenario-scoped initiative when existing coverage overlaps the same scenario — the human has to do the reconciliation work, and the signal gets lost.

## Usage Rules
- Apply accepted decisions before creating new ones.
- Stop early when there are already 3 unresolved relevant pending decisions.
- Do not create more than 3 new decisions in one run.
- Do not deploy teams or create backlog items without approval; frame them as proposals when approval is missing.
- For any scenario-scoped initiative proposal, run the Scenario Coverage Enumeration above first. A proposal that did not check coverage is invalid.
