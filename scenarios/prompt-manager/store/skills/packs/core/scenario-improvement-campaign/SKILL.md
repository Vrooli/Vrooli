---
name: "scenario-improvement-campaign"
description: "Implement successive improvements toward an approved scenario contract or explicit maturity target within one engagement, using scenario improve judgment and provider evidence rather than per-repair backlog approvals."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["scenario", "development", "maturity", "self-improvement"]
  icon: "list-checks"
  status: "active"
  targetToolId: "run-agent"
  revision: 5
  createdAt: "2026-05-29T00:00:00Z"
  updatedAt: "2026-09-10T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "test-genie"]
    commands: ["prompt-manager skill read", "test-genie runs", "vrooli scenario test"]
  origin:
    kind: "authored"
---
## Practice focus: Scenario Improvement Campaign

Implement and verify successive improvements inside the caller's authorized
engagement. Let the scenario improve skill choose the concern; let owner evidence
establish the result. One coherent repair needs no campaign machinery of its own.
The engagement always belongs to a canonical Plan Manager work package; the
campaign is an adaptive execution strategy over that plan, not a planless
development lifecycle.

Required reading:
- `path:docs/agent-system/SCENARIO_DEVELOPMENT.md` — target, authority, and completion.
- `path:docs/TESTING.md` — proportionate validation and durable waits.
- `prompt-manager skill read improvement-do-and-dont` — anti-gaming rules.

Load `<scenario>-improve` for scenario-contract work. Load `scenario-maturity-ladder`
for an explicit maturity review. Load implementation-plan execution when working
an actual plan; it is not a prerequisite for every repair.

When Swarm supplies an ordinary adaptive plan item, use its accepted canonical
plan, item limits and execution checkpoint. The Swarm execution owner admits
the declared workflow and retains its correlation; Agent Manager owns the run
journal and grant accounting. This skill selects the next in-scope repair and
reports evidence. Independently reviewed routine phase boundaries continue
under the original approval. Target or grant amendments remain operator decisions.
Existing retained `contract-development` engagements keep their own owner;
never add that second lifecycle to an ordinary adaptive item.
Do not create a second approval for an ordinary repair or treat a workflow's
terminal result as the operator's final acceptance.

### 1. Resolve the engagement

Identify the approved target, work identity, permitted changes, effects, budget,
and acceptance policy. An explicit operator instruction may supply these before
typed mandate admission exists; do not claim an owner receipt. Workflow-launched
agents retain their actual grants, including fixed slice boundaries.

| Requested outcome | Evidence and stopping condition |
| --- | --- |
| Develop against a scenario contract. | Applicable evidence for every required outcome under the approved policy. |
| Reach a named maturity target. | The provider-owned target and evidence, not a copied ladder or local score. |
| Fix a bounded defect. | Use the owning repair method directly; do not expand it into a campaign. |
| Explore within a bounded investment. | Return findings and remaining work at the agreed limit, not a full-development claim. |

A profile prioritizes work; it cannot redefine completion. Do not default to zero
findings or a comprehensive green suite for ordinary development.

### 2. Establish target and evidence

Read approved artifact contents and existing scenario problem evidence. Use
declared sensor programs, owner reports, and relevant Test Genie findings. Keep
observed behavior distinct from desired-state documentation.

Use `skill-set-authoring` when setup is absent and setup work is authorized.
A missing instrument is work to perform, not another item by default. An undecided
product target still requires an operator decision.

Separate required outcomes from diagnostic ranking signals using the approved
contract. An unavailable optional friction report does not prevent a known,
authorized repair. An unavailable required measurement remains an unmet obligation.

Use `scenario-work-ladder` for upstream disagreements or layer selection. A known
local defect can take its scoped route without a complete readiness audit.

### 3. Implement one coherent intervention

Entry: an unmet outcome and a permitted intervention are identified.

1. Select the next unmet outcome using scenario priorities and evidence validity.
2. State the expected result and the observation that can falsify it.
3. Check the change against the active grant.
4. Load the method that owns the repair.
5. Change implementation, tests, and affected documentation within scope.
6. Run focused checks that distinguish success from the observed failure.
7. Checkpoint the result and choose the next intervention from the new evidence.

Exit: retain a supported result or a rejected hypothesis with its evidence.
When the observation falsifies the hypothesis, revise it before another attempt.

Use `scientific-debugging` when the cause is uncertain. Repair sensors, programs,
resources, and shared packages at their owners. Do not create private dependency
workarounds to fit an inaccurate scope estimate. Request amendment when the actual
grant excludes the needed repair.

Use a subordinate plan when it helps or the mandate requires it. Do not create
another Swarm approval for each implementation choice or rewrite an active plan
each cycle. Use its owner's log and revision rules.

### 4. Verify and checkpoint

Use `vrooli scenario test <scenario> --phases <relevant-phases>`, selecting actual
owner phase names. Follow `docs/TESTING.md` for pending work; wait on its identity
instead of polling or replaying it.

Record evidence references, valid before/after readings, changed areas, remaining
obligations, decisions, pending operations, and known budget use. Use the active
owner's log and shared Memory contract. Keep output with its producer and avoid
duplicate automatic capture or copied finding ledgers.

Architecture Cartographer may supply optional ranking or an existing campaign
projection. It is not the work authority; its unavailability does not require
another tracker. Discover its current contract before using that path.

Retain broad advisory failures. A credible failure of the promised outcome remains
unfinished. Missing historical evidence never justifies replacing shared-worktree
contents. Do not change a floor or suppress a finding to manufacture success.

### 5. Continue or return

| Evidence and authority | Next action |
| --- | --- |
| In-scope outcomes remain and useful work is possible. | Continue; do not request approval for each repair. |
| A specific action needs another target, grant, or budget. | Request amendment; continue independent authorized work. |
| Owner operation pending. | Attach to its wait and preserve identity. |
| Budget exhausted or authority revoked. | Stop new effects and return an honest checkpoint. |
| Required outcomes have applicable evidence. | Return the assessment for the configured final disposition. |

Swarm closeout and Git/publication effects require their actual authority and owner
operations. A harness success signal cannot override an unmet product obligation.

### 6. Troubleshooting & Edge Cases

| Situation | Response |
| --- | --- |
| Old improve guidance requires every repair to be filed. | Apply the active mandate and repair obsolete guidance within scope rather than duplicating work. |
| Required sensor is unavailable. | Repair the evidence path within authority; retain unknown status until verified. |
| A provider finding appears wrong. | Test expected provider behavior and repair an authorized rule defect instead of suppressing the finding. |
| Restart or compaction loses context. | Reload the target and owner checkpoint; do not infer prior success from a summary. |
| No valid authority can be recovered. | Preserve work and request it before further effects. |
| An optimization changes acceptance semantics. | Preserve the target and request amendment. |

### 7. Output expectations

| Anti-pattern | Why it fails | Response |
| --- | --- | --- |
| Convert every repair into another approval request. | Delegated development becomes work intake. | Keep in-scope repairs under the current engagement. |
| Optimize a diagnostic score as if it were acceptance. | The agent changes the destination. | Evaluate the approved required outcomes. |

Report delivered outcomes with evidence, limitations, unmet obligations, approach
changes, and the next action. Completed investigation, filed work, or a passing
wrapper is not a completed development mandate.
