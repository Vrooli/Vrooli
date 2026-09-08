---
name: "browser-automation-studio"
description: "Run browser tasks with exact workflow selection, bounded navigation, validated workflow promotion and repair, and outcome-linked learning."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["automation", "workflow", "learning"]
  icon: "play"
  status: "active"
  revision: 53
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["browser-automation-studio", "program-runtime", "vrooli-memory"]
    commands: ["browser-automation-studio", "program-runtime", "vrooli-memory"]
  learning:
    scope: "bas-usage"
    capture: "every attempt"
  origin:
    kind: "authored"
---
## Tools focus: Browser Automation Studio

Use BAS for captures, typed browser workflows, and bounded intent navigation.
BAS owns execution and version history. The usage skill chooses the route;
`browser-automation-studio-improve` regulates BAS itself.
Scenario acceptance cases under `bas/` belong to `e2e-testing`.
Roles and step rungs follow `path:docs/agent-system/SKILL_AUTHORING.md`.

### Choose one operation

Read rows in order; the first matching row is the next step.

| Situation | One step |
|---|---|
| Operation or inputs are unknown | Read `browser-automation-studio <group> help`. **[S1]** |
| Need one page without interaction | Run `browser-automation-studio capture` with the requested URL and capture kind. **[S1]** |
| Have the authorized workflow UUID and exact revision | Run `browser-automation-studio.do-task` with task, workflow_id, version and declared parameters. **[S3]** |
| Need to find a reusable flow | Run `browser-automation-studio.find-flows` with task and scenario when applicable. **[S3]** |
| Search returned a candidate | Read `browser-automation-studio workflows get <workflow-id>`; verify project, definition, revision, parameters, and authority before selecting it. Relevance alone does not authorize execution. **[S1]** |
| Have a new typed candidate with an explicit outcome assertion | Run `browser-automation-studio.author-flow` with flow, project_id and name. It validates, executes once, and saves only a completed candidate. **[S3]** |
| A saved workflow failed and a repair candidate is ready | Run `browser-automation-studio.author-flow` with flow, workflow_id and expected_version. Existing assertions, graph topology, metadata and settings must remain intact. **[S3]** |
| No reusable flow exists and an authorized browser session is available | Run `browser-automation-studio.navigate-intent` with session, prompt, selected model and max_steps. **[S3]** |
| Need a navigator before intent navigation | Read `browser-automation-studio vision-navigation list-navigators`; use its current route/model contract. **[S1]** |
| Need to diagnose the failed operation | Read `browser-automation-studio executions timeline <execution-id>`. Preserve the failed step and evidence for the repair candidate. **[S1]** |
| Need to change the acceptance contract itself | Route through `scenario-work-ladder` for the owner scenario; an automated repair cannot weaken checks. **[S0]** |

Reuse a matched flow instead of reconstructing it. For a successful unfamiliar
path likely to recur, draft a parameterized typed candidate with explicit
preconditions and outcome assertions, then run author-flow. A navigation success
does not automatically establish a replayable workflow. Retain its navigation ID
as provenance when recording the authoring attempt.

### Program contracts

Use these declared programs for reusable browser work and its bounded evidence:

| Program | Purpose | Required inputs |
|---|---|---|
| `browser-automation-studio.author-flow` | Validate and save a typed workflow candidate | `flow`, `project_id`, `name` |
| `browser-automation-studio.do-task` | Execute an authorized workflow revision with assertions | `task`, `workflow_id`, `version` |
| `browser-automation-studio.find-flows` | Find reusable flows for a task | `task`, optional `scenario` |
| `browser-automation-studio.learning-read` | Read comparable browser-task learning outcomes | `operation`, `context_key` |
| `browser-automation-studio.navigate-intent` | Navigate an authorized session toward an intent | `session`, `prompt`, `selected_model`, `max_steps` |
| `browser-automation-studio.setpoint-read` | Read the BAS improvement board | optional window inputs |
| `browser-automation-studio.smoke-flow` | Run a bounded smoke workflow and report evidence | `workflow_id`, `version` |

Invoke one with `program-runtime library run <name> --input key=value`; preserve
the execution and workflow revision as evidence.

Repair starts with the failed execution and selected version, then changes the
smallest cause (for example a navigation selector or wait). The old version
remains available. Do not replace a failed flow until the candidate passes.
If the expected revision changed, read the current definition and reconsider
the candidate; do not overwrite another author's work.

### In-use settings

| Symptom | Allowed move | Record |
|---|---|---|
| Authentication expired | Inspect `session-profiles list`; refresh through the site's authorized login flow | Profile and resulting execution |
| Content has not become ready | Select an explicit readiness condition through capture/workflow inputs | Condition and observed result |
| Known workflow is available | Supply its UUID and revision directly to do-task | Reused workflow and saved orientation effort |
| Navigation reaches its budget | Preserve partial progress; narrow the next authorized attempt | Failure fingerprint and unresolved outcome |

### Verification and authority

Use the existing workflow labels, routed test storage, and execution gates.
A test fixture must not drive an arbitrary production workflow discovered by name.
Check that explicit assertions cover the user's actual request; a completed
workflow proves only its asserted behavior. Unknown or stale evidence is not success.
Keep browser sessions and service lifecycle under their owning CLI operations.
Single commands remain CLI leaves because wrapping them adds no useful composition.

### Learning and program use

Use `bas-usage` for comparable task evidence. Keep target, profile, and
relevant version contexts distinct; remembered selectors and endpoints require
current verification.

The declared do-task, author-flow, and smoke-flow programs use automatic learning.
Inspect their outcome and delivery receipt; nested calls share the parent attempt.
For direct operations without automatic learning, use the manual path in
`prompt-manager skill read vrooli-memory`. That skill owns attempt fields,
advice decisions, measurements, and capture recovery. Never record device or
browser contents, credentials, or private URLs in memory.

Read `prompt-manager skill read program-runtime` when invocation or result
handling is unfamiliar. The selected program contract owns inputs and statuses.
Use Memory's receipt recovery for pending capture; repeating a domain operation
does not repair capture. Domain success requires the assertions described above.

### Troubleshooting & Edge Cases

| Result | Next action |
|---|---|
| selection_required | Select the exact authorized workflow UUID/revision or a candidate/session; nothing was executed |
| version_conflict or repair baseline changed | Re-read the current definition and revise against that baseline |
| validation_failed or preserved assertion refusal | Correct the candidate; changing acceptance requires owner contract work |
| execution_pending / still_running | Retain execution ID; use the owner's documented observation path without rerunning |
| selector_not_found, timeout, auth_required | Inspect the failed timeline entry and relevant session before one repair attempt |
| no_grant / not_run_eligible | Follow the runtime's exact grant/refusal path under existing task authority |
| scenario_unreachable | Read lifecycle status; retain an unavailable outcome |
| memory_unavailable | Complete the permitted operation and retain its uncaptured record |
| No repeated-use improvement | Compare learning-read cohorts and route through the improve skill |

The do-task program delegates capture to runtime metadata.
Its outcome remains unknown for completed workflows or navigation alone.
Only a child that explicitly verifies assertions and supplies evidence may report
verified_success. The learning-read program delegates comparison to
`lib.vrooli_memory.compare_outcomes` and retains BAS's seven metric rows.
It preserves full cohort identities, resolved comparison windows, reliability,
and the shared partial, unavailable, or refused status. Incomplete timelines,
missing assertion results, and unrecognized execution states remain unknown.
Promote stable orchestration to programs and durable policy to BAS; remove
superseded workaround prose when the owner gains the operation.
