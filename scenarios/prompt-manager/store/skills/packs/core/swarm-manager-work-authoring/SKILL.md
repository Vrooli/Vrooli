---
name: "swarm-manager-work-authoring"
description: "How to write backlog item, goal, and milestone text so it stays true as the code moves: outcome statement plus Gherkin done-condition."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["swarm-manager","backlog","goals","writing"]
  status: "active"
  revision: 2
  createdAt: "2026-07-25T00:00:00Z"
  updatedAt: "2026-09-11T12:00:00Z"
  requires:
    scenarios: ["vrooli"]
    commands: ["vrooli scenario"]
  origin:
    kind: "authored"
---
# Work Authoring

## Purpose

How to write a backlog item, a goal, and a milestone so the text stays true as
the code moves. All skills that create or edit work should defer to this
document for the shape of `title`, `description`, and acceptance criteria.

**Scope boundaries:**
- **In scope**: what the text must say, how work is shaped, and how the live
  backlog contract is used

## Writing Standard (outcome statement + Gherkin done-condition)

The schema enforces that `description` is non-empty; it cannot enforce that the
description is still true next month. That is what this section is for.

**The rule, in one line: a description states a condition an agent can check as
true or false today, without knowing when the item was written.**

Everything else here follows from that rule.

## Runtime item and CLI contract

Backlog items are read from the runtime `{{ITEM_FOLDER}}`; use the Swarm
Manager CLI for read-only context and typed mutations rather than hand-editing
project state. The durable item contract is `spec.json`, with optional
`handoff/`, `research/`, and `archive/` evidence directories. Keep generated
execution context under `handoff/`, keep research evidence supplemental, and
use `plan_ref` as the canonical implementation-plan reference.

## Scenario development work packages

When the outcome is material scenario development, shape one work package with
these parts: the goal states why the world should change; the backlog item
states the bounded outcome; Plan Manager owns the canonical implementation plan;
and Swarm owns operator approval, effects, budget, checkpoint, and final
disposition. A plan reference is required before development authority is
approved. The plan may point to a reviewed Tech Tree Designer revision when a
design graph helps, but that revision does not replace the plan.

Use the same work package for a phased plan and for an adaptive mandate. The
plan's shape says what the work is; the execution mode (`sliced` or `goal`)
says how it runs, and any shape runs under either mode. A mandate-shaped plan
lets `scenario-improvement-campaign` make several repairs under one approved
grant. Do not create a separate planless item for ordinary scenario
development. Include the target scenario's usage and improve skills, expected
evidence, acceptance boundary, an operator note, and a proposed execution mode
so the later plan author and operator have the information needed to review it.

### Plan-backed item recipe

Send only the fields the CLI accepts today; unknown fields fail fast. `kind`
is one of `idea`, `research`, `fix`, `execute`, `chore`.

```bash
swarm-manager backlog create --data '{
  "kind": "execute",
  "name": "<name>",
  "title": "<end state, present tense>",
  "description": "<outcome>\n\nDone when:\n- <Gherkin or command>\n\nOperator note\n<intent, reminders, authority to fix shared packages with a record>",
  "plan_ref": {"provider": "plan-manager", "plan_id": "<plan uuid>", "slug": "<plan slug>", "role": "execution_spec"},
  "execution_strategy": "phased-plan-drain",
  "execution_limits": {"max_slices": 8, "max_tokens": 4000000, "max_wall_seconds": 28800, "max_turns": 400, "max_charge_micro_usd": 20000000, "max_children": 16, "max_node_attempts": 64, "max_retries": 4},
  "continuation": "until-allowance",
  "scope_policy": "extend-with-record",
  "acceptance_allow": ["scenarios/<scenario>/**", "packages/<shared-package>/**", "packages/proto/schemas/<scenario>/**"],
  "acceptance_deny": ["scenarios/<scenario>/data/**"]
}'
swarm-manager backlog update --kind execute --name <name> --data '{
  "execution_strategy": "adaptive-improvement",
  "continuation": "until-allowance",
  "scope_policy": "extend-with-record"
}'
```

| Field | Values today | Meaning |
|---|---|---|
| `execution_strategy` | `phased-plan-drain`, `adaptive-improvement`, `goal-session` | Today's spelling of the execution mode: sliced over a phased plan, sliced over a mandate-shaped plan, and the interim workflow-based goal mode (superseded). Target field: `execution_mode` (`sliced` \| `goal`). |
| `execution_limits` | `max_slices`, `max_tokens`, `max_wall_seconds`, `max_turns`, `max_charge_micro_usd`, `max_children`, `max_node_attempts`, `max_retries` | One aggregate allowance across resumes. `max_slices` applies to sliced mode only. |
| `continuation` | `manual`, `until-allowance` | Resume policy for involuntary interruptions only. A verdict is final under both. |
| `scope_policy` | `fixed`, `extend-with-record` | Whether the worker may append allow globs with `plan-manager exec boundary-extend` and a reason. |
| `acceptance_allow` / `acceptance_deny` | globs | Narrow to the target scenario, its shared packages, protos and docs. Deny still refuses after an extension. |
| `operator_note` | target field | Free text that reaches the agent verbatim. Until it exists, put it in the description under an "Operator note" heading. |

### Shape

```
<Outcome. One or two sentences naming the end state in the present tense.>

Done when:
- Given <context>, when <action>, then <observable result>.
- <or a command to run and the result that counts as passing>
```

Write the `Done when` bullets in Gherkin, per `e2e-testing` §4. Write any
procedural text in STE-100, per `docs/agent-system/SKILL_AUTHORING.md`
§"Universal quality bars". Both standards are already canon; this section only
says where they apply.

### The three content classes

Sort every sentence you are about to write by how fast it decays, then put it
where it belongs:

| Class | Decays | Goes in |
|---|---|---|
| **Outcome / done-condition** — what becomes true | only when the goal changes | `description` |
| **Evidence / provenance** — what you observed, when, and where | fast; true as of a date | `research/summary.md`, `finding_ref`, `note` |
| **Prescribed solution** — the commands or edits you predict will work | fastest | usually nowhere |

The prescribed-solution class is the one to be strict about. It does not merely
go stale, it **forecloses**: a frozen hypothesis in the description outranks
fresh evidence for no reason other than being written down. Name the outcome and
let the executing agent find the route. Prescribe a specific route only when the
route *is* the requirement (an operator-mandated approach, a compatibility
constraint), and then say why it is mandatory.

### Rules

- **Never store what a command regenerates.** `vrooli scenario test <name>`
  already prints what is failing. Copying that output into a description creates
  a second copy that can drift, and the copy is the one that misleads. Reference
  the check; do not transcribe its result.
- **No provenance in the description.** Inbox IDs, run IDs, prior-art search
  results, and "investigation found…" narration go in `research/` or `note`.
  `created_by` already records authorship.
- **Prefer a check that runs over a claim to trust.** "`go test ./...` exits 0
  from `api/`" beats "the module metadata is correct."
- **Scope belongs in `acceptance_allow`, not in prose.** Do not narrate file
  paths the item is allowed to touch; that field already carries them, and it is
  machine-validated — a path that stops existing marks the item stale.
- **The title names the end state, not the activity.** "API module builds and
  tests run" over "Reconcile API Go module metadata".
- **One outcome per item.** Two unrelated done-conditions are two items.

### Generated and recurring items

Items filed by an automated loop (QA readiness sweeps, autofiler suggestions,
scheduled audits) must bind to a **phase and rung**, never to an individual
finding. A finding is a snapshot; a rung is re-measured. See
`scenario-maturity-ladder` for R0–R4.

```
Scenario <name> reaches R0 on <phase>: build green, no blocker or error findings.

Done when:
- When `vrooli scenario test <name>` runs, then the <phase> phase reports no
  blocker or error findings.
```

This shape absorbs validation-rule changes for free. If we add a check next
month, the item covers it; if we retire one, the item stops demanding it.
A per-finding item cannot do either, and goes stale the moment the rule moves.

### Anti-patterns

- **The investigation transcript.** Root cause, prior-art results, and a command
  list pasted into `description` because the source record was closed. Put the
  evidence in `research/`; the description keeps the outcome.
- **The dated claim.** "Currently 47 warnings" — true once, misleading forever,
  and it invites an agent to chase the number rather than the condition.
- **The solution as the title.** "Run `go mod tidy`" is a step. The item is
  "the API module's tests run."
- **The absent subject.** "Fix the failures" with no way to enumerate them. Name
  the command whose output defines the set.

## Writing Standard: goals and milestones

A goal is an operator intent statement — the outcome, not the route. The same
time-independence rule applies, with two additions that are specific to goals:

- **State the change in the world, not the work.** "Operators can deploy any
  scenario to a desktop without hand-editing config" is a goal. "Finish the
  desktop packaging work" is a status update wearing a goal's clothes.
- **Do not name the items.** A goal's scope is derived truth — its targets plus
  their prerequisite closure. Listing member items in the intent text creates a
  second, hand-maintained copy of the graph, and it is always the stale one.

A milestone carries acceptance criteria as its definition of done. Write those
in Gherkin, verified against repository evidence — the same standard as an
item's `Done when`, at the milestone's grain.

### The three layers say different things

| Layer | Answers | Carries |
|---|---|---|
| Goal | why — the change in the world | intent, priority, targets |
| Milestone | how you would prove it | acceptance criteria |
| Item | what someone does | outcome + `Done when` |

**A goal states what becomes true. A milestone states how you would prove it.
An item is one thing you would do.** If a milestone's text reads like its
goal's text, the milestone is not doing its job — that is the shape to fix,
not a style preference.

### How many milestones

Every goal needs at least one: acceptance criteria live only on a milestone, so
a goal without one can never be reviewed or closed out. Past that, count the
points where you would stop and check delivery **before letting the rest
proceed**. Not phases, not themes — verification checkpoints.

A candidate split is real only if both hold:

- You can write its acceptance criteria without restating the goal's outcome.
- It could come back `delivered` while other parts of the goal are still open.

If a split fails either test, do not split. One milestone is the right answer
for most goals. Never copy the goal's title or description into it; title the
milestone for the evidence ("Bridge verified on real hardware"), not for its
parent.
