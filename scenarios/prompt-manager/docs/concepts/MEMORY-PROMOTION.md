# Memory Promotion

Memory promotion is the process for moving raw agent discoveries into the right durable form.

Status: adopted for Action-aware meta-optimization. This document defines the ontology prompt-manager uses when teams analyze typed knowledge, run lessons, and other accumulated observations.

## Core Ontology

```text
Continue -> handoff
Observe  -> typed knowledge and friction evidence
  Propose  -> Swarm Manager work
Operate  -> team working state
```

Promotion starts after observation. Knowledge log entries hold structured evidence, snapshots, findings, concrete friction signals, unresolved patterns, workarounds, and rough lessons that are not ready for durable structure. Plan-of-record docs hold accepted durable truth.

Team working state is different: it is team-local operational memory such as task boards, ledgers, registers, rolling audits, append-only logs, and operator input files. It is not automatically canonical outside the team.

The promotion destination ontology is:

```text
Knowledge = structured observation/evidence memory
Plan of Record = accepted durable truth
Skill = reusable judgment/process guidance
Action = typed executable operation
CLI = implementation of behavior
Backlog = unbuilt or broken behavior
Gated work = reviewable, evidence-backed request to change a durable surface
```

The short form:

```text
If it says what is true -> Plan of Record.
If it says how to decide -> Skill.
If it says what to run -> Action.
If it says how it works -> CLI implementation.
If it says what is missing -> Backlog/capability-work.
If it is evidence or a measurement -> Knowledge.
If it is unverified, recurring, or a workaround -> typed knowledge under the most specific topic.
If it is missing, broken, confusing, slow, undocumented, or harder than expected -> friction observation.
```

This keeps prompt-manager from turning every useful note into prose. It also prevents deterministic operations from staying trapped inside agent instructions after the system can execute them directly.

## Entity Responsibilities

| Form | Responsibility | Typical owner |
|------|----------------|---------------|
| Knowledge | Structured observations, evidence, measurements, snapshots, findings, concrete friction signals | Any agent or team |
| Plan of Record | Accepted durable truth, policy, architecture, canonical context | Operator or curator |
| Skill | Judgment, decision process, methodology, reusable guidance | Skill authoring / optimization lane |
| Action | Typed execution wrapper over one Vrooli-controlled CLI command | Prompt-manager Action registry |
| CLI | Implementation, branching, retries, validation, operational behavior | Scenario/resource/project owner |
| Backlog | Missing or broken behavior that needs implementation | Swarm-manager / owning scenario |
| Gated work | Reviewable proposal to change a durable surface | Owning team/member |

## Promotion Decision Tree

For each typed knowledge entry:

1. Is it still true and useful?
   - No: retire, archive, or mark obsolete.
   - Unsure: keep it as typed knowledge and add a verification task.
   - Yes: continue.

2. Is it a fact, policy, constraint, architecture decision, or durable context?
   - Yes: promote to the Plan of Record.

3. Is it guidance for how an agent should think, choose, evaluate, or synthesize?
   - Yes: create or update a skill.

4. Is it an exact repeatable operation with stable inputs and outputs that maps to one Vrooli-controlled CLI command?
   - Yes: create or update an Action.

5. Is it an operation, but the proper CLI does not exist yet?
   - Yes: create a backlog item or `capability-work` for the owning scenario/resource/project.
   - If discoverable execution is the intended future interface, note the future Action that should wrap the CLI once it exists.

6. Is it deterministic but multi-step?
   - Move the workflow into a CLI command that owns the steps.
   - Then expose one Action over that command.

7. Is it partly judgment and partly execution?
   - Split it:
     - Skill for judgment
     - Action for deterministic execution
     - CLI/backlog for implementation
     - Plan of Record for durable truth discovered along the way

## Plan of Record vs Skill

Use this distinction:

```text
Plan of Record answers: What is true here?
Skill answers: How should an agent approach this class of work?
```

Examples:

| Typed knowledge item | Promotion |
|---------------|-----------|
| "Meta-optimization runs in approval mode because edits affect other teams." | Plan of Record |
| "Before proposing a meta-layer change, capture a baseline and expected delta." | Skill |
| "Actions wrap exactly one Vrooli-controlled CLI command." | Plan of Record |
| "If an operation requires conditional logic, move it into the CLI and keep the Action thin." | Skill or Action-authoring rule |
| "Run `swarm-manager backlog list --json` to check work pressure." | Action candidate |
| "There is no scenario UI screenshot command yet." | Backlog/capability-work |

## Workaround Handling

Workarounds deserve special care because they often imply multiple promotions.

```text
Typed workaround evidence
  -> Is it still needed?
    -> No: retire the entry
    -> Yes: why does it exist?
```

Then classify:

- Missing CLI capability -> backlog item or `capability-work`
- Existing CLI is awkward -> CLI improvement backlog item
- Existing CLI is good but undiscoverable -> Action candidate
- Agents keep forgetting the right approach -> skill update
- Workaround revealed a durable rule -> Plan of Record update

A single workaround can produce more than one output, but the curator should usually promote the highest-leverage next step first and leave breadcrumbs for later work.

## Friction Handling

Friction is an Observe subcase:

```text
Something expected was missing, broken, confusing, slow, undocumented, misplaced, unstable, or harder than it should have been.
```

Route friction through the lightest useful durable form:

- Minor or one-off friction goes in the final handoff when it helps the next run.
- Concrete system friction, workarounds, inefficiencies, or process leaks are filed through `report-friction` to `friction-inbox/<scope>/<slug>`.
- The friction curator routes accepted entries to `friction-report/<scope>/<YYYY-MM-DD>/<slug>` and records routing in `friction-triage-record/<YYYY-MM-DD>`.
- Missing or broken capability that blocks work becomes one evidence-backed Swarm Manager backlog item.

Good friction notes include:

```markdown
Expected: ...
Actual: ...
Surface: ...
Workaround: ...
Impact: ...
Recurrence: one-off | recurring | unknown
```

## Promotion Pipeline

Recommended flow:

```text
Typed knowledge entry
  -> classifier applies promotion decision tree
  -> Swarm Manager work records target form and rationale
  -> operator dispositions the work
  -> owning lane implements the dispositioned work
  -> obsolete typed evidence is retired or linked to the permanent form
```

Agents should not silently rewrite durable memory. The promotion path should produce reviewable Swarm Manager work when the change affects shared team behavior, prompt-manager entities, scenario implementation, or operator-facing policy.

## Adoption in Meta-Optimization

The meta-optimization team is the natural owner of this loop:

- `skill-optimizer` identifies skill sections that can collapse into Actions.
- `debt-curator` promotes stable typed evidence patterns into permanent structure.
- `run-introspector` detects repeated manual operations in real agent runs.
- `toolchain-validator` raises gaps when missing tools block reproducible validation.
- `meta-contrarian` challenges premature or unsafe promotions.

The expected end state is not "all skills become Actions." The expected end state is that every piece of durable knowledge lives in the cheapest reliable form that preserves its meaning.
