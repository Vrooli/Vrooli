---
name: "scenario-improvement-campaign"
description: "Drive a scenario to an operator-selected maturity target with provider-owned Test Genie evidence and an optional profile-ranked Architecture Cartographer campaign."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools","audits"]
  tags: ["skill","audit-technique"]
  icon: "list-checks"
  status: "active"
  targetToolId: "run-agent"
  revision: 2
  createdAt: "2026-05-29T00:00:00Z"
  updatedAt: "2026-07-26T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "swarm-manager", "test-genie"]
    commands: ["prompt-manager discover", "prompt-manager skill", "prompt-manager skill read", "swarm-manager", "test-genie", "test-genie execute", "test-genie runs"]
  origin:
    kind: "authored"
---
## Tools focus: Scenario Improvement Campaign

Drive `{{TARGET}}` toward an operator-selected maturity target. Use Test Genie's current, provider-owned phase standing as the evidence source. Use an Architecture Cartographer campaign to preserve and rank multi-pass work. Do not replace either system's judgment with a second score or a copied finding list.

Required reading:

- `prompt-manager skill read scenario-maturity-ladder scenario-readiness-review improvement-do-and-dont`
- `docs/TESTING.md`
- `scenarios/test-genie/docs/concepts/phase-capability-contract.md`

Read when the campaign will create or update Swarm Manager work:

- `prompt-manager skill read swarm-manager-work-authoring implementation-plan-authoring`

## 1. Choose the campaign posture

A profile orders work. A target defines the stopping condition. Keep these decisions separate.

| Operator intent | Target | Profile | Escalation rule |
|---|---|---|---|
| Restore a runnable scenario | `suite-green` | `fast` | Stop when the fresh comprehensive suite passes. |
| Remove release blockers | `zero-blockers` | `balanced` | Work security and dependency blockers before lower-risk debt. |
| Modernize a drifted scenario | named maturity target or `zero-blockers`, then `suite-green` | `long-term` | Work structural root causes before symptom cleanup. |
| Limit investment | bounded item count or time budget | `balanced` | Preserve the campaign and record the next ranked item. |

Do not use `zero-findings` as the default target. Select it only when the operator accepts the cost of resolving or explicitly classifying advisory debt.

## 2. Read current evidence before opening work

Read the latest phase standing first:

```bash
test-genie runs findings --scenario {{TARGET}}
```

Treat the provider's North Star, priority focus, next action, blocking codes, and freshness as authoritative. Test Genie renders those fields. It does not invent them. Do not infer a cross-phase prerequisite that the evidence does not show.

Use this work table:

| Evidence state | Action |
|---|---|
| Fresh run has one bounded failed phase | Fix that phase as one coherent remediation slice. Do not create a campaign by default. |
| Fresh run has several failed phases or a large actionable finding set | Open or resume a campaign. |
| Fresh run has a security or dependency blocker | Address that blocker before ordinary campaign ranking. |
| Evidence is historical, degraded, or missing | Run a fresh applicable suite before claiming a maturity state. |
| Provider reports unavailable | Record the unavailable evidence. Do not treat it as pass or as a scenario defect. |

## 3. Open or resume the durable tracker

Use the campaign tracker only when the work exceeds one responsible pass. A campaign ingests a machine-readable Test Genie audit because Architecture Cartographer needs stable finding identities. This is an explicit exception to human-first output; the `--json` object is the documented input contract of `campaign create` and `campaign reaudit`.

```bash
campaign_audit_path="$(mktemp -t {{TARGET}}-campaign-audit.XXXXXX.json)"
test-genie execute {{TARGET}} --preset comprehensive --json > "$campaign_audit_path"
architecture-cartographer campaign create {{TARGET}} --name "<operator goal>" --from-audit "$campaign_audit_path"
```

Use a previously open campaign when it has the same scenario and target. Do not open competing campaigns for the same remediation scope.

Select the next worklist with the chosen profile:

```bash
architecture-cartographer campaign next "<campaign-id>" --profile long-term
```

`fast` is the shortest path to a green suite. `balanced` prioritizes regressions, cycles, and severity. `long-term` prioritizes structural root causes before symptoms. A newly observed regression leads every profile.

## 4. Work one coherent remediation slice

For the selected item or phase:

1. Read the provider-recommended remediation topics from `test-genie runs findings`.
2. Load the provider skill that owns the change type.
3. Group findings that share one cause and one change boundary.
4. Fix behavior, contract/configuration, tests, and durable scenario documentation together.
5. Run the smallest provider phase and behavioral tests that prove the slice.
6. Do not suppress a finding, weaken a test, lower a threshold, or delete a debt ledger to improve a score.

Use the scenario maturity rung to size the work. At R0 and R1, make precise safety and runnability changes. At R2 and above, make the architectural or feature change that closes the real gap. Do not substitute cosmetic cleanup for an unsatisfied higher-rung obligation.

## 5. Reconcile evidence, not intention

After a slice passes its targeted verification, mark the implementation result in the tracker:

```bash
architecture-cartographer campaign resolve "<campaign-id>" --finding "<afid>" --note "<root cause, change, and targeted verification>"
```

Then capture a fresh comparable audit and reconcile it:

```bash
campaign_audit_path="$(mktemp -t {{TARGET}}-campaign-reaudit.XXXXXX.json)"
test-genie execute {{TARGET}} --preset comprehensive --json > "$campaign_audit_path"
architecture-cartographer campaign reaudit "<campaign-id>" --from-audit "$campaign_audit_path"
architecture-cartographer campaign status "<campaign-id>"
```

The fresh audit decides the standing: absent findings validate, persistent findings remain open, and new or returned findings are regressions. A `resolve` note is not proof that a finding is gone.

At each re-audit, re-check the selected target. Continue with the current profile unless a blocker, regression, or verified dependency changes the order.

## 6. Create work only at the right scope

Use the campaign as the work tracker. Use Swarm Manager work only when the operator wants durable execution ownership or the scope exceeds a single pass.

| Scope | Create | Required content |
|---|---|---|
| One phase with bounded remediation | One backlog item | Current evidence trigger, outcome statement, and Gherkin done-condition. |
| Several related phases or cross-cutting structural debt | One goal with milestone-backed work | Target maturity state, fresh-evidence acceptance, and a campaign reference. |
| Repeated fleet pattern | Separate capability/tool improvement work | Shared cause, affected scenarios, and the required tool contract. |

Do not encode transient counts, copied findings, or a historical pass result as the completion condition. Reference the run evidence and require a fresh Test Genie result. Load `swarm-manager-work-authoring` before writing a backlog item, goal, or milestone.

## 7. Stop deliberately

| Condition | Required action |
|---|---|
| Selected target is met on fresh evidence | Close the campaign and report the final run evidence. |
| Only explicitly accepted advisory/blocker residue remains | Keep the campaign open or close it with the residue and owner recorded. Do not claim the target is met. |
| The operator's budget is reached | Preserve the campaign. Report the current standing and next ranked item. |
| A new lower-rung gate appears | Return to that gate before advancing higher-rung work. |

Use `scenario-readiness-review` between material iterations to assess coherence and commit readiness. The campaign does not authorize commits, pushes, or source changes by itself.

## 8. Troubleshooting & Edge Cases

| Symptom | First check | Required response |
|---|---|---|
| Test Genie run is in progress | Read the re-attach command from the run output. | Use one server-owned wait. Do not poll or start a competing run. |
| Test Genie evidence is unavailable or degraded | `test-genie runs findings --scenario {{TARGET}}` | Record the degraded state. Do not infer PASS, maturity, or a completed fix. |
| Architecture Cartographer is stopped or its campaign command fails | `architecture-cartographer campaign --help` and its scenario status | Do not fabricate campaign state. Continue only with fresh Test Genie evidence. If sustained work is needed, ask the operator to create governed Swarm Manager work or restore the tracker. |
| Campaign creation cannot ingest the audit | Confirm `campaign create --help` and retain the failing command output. | Do not hand-convert findings into fake stable IDs. File or propose a tool-contract improvement when the failure recurs. |
| A provider finding looks wrong | Read the provider remediation topic and inspect the source. | Scope the provider rule with a regression test if it is a false positive. Do not suppress the scenario finding. |
| A repeated manual step appears across campaigns | Check `prompt-manager discover "<operation>" --type all`. | Promote a stable deterministic operation to a CLI or Action instead of expanding this skill. |

## 9. Output expectations

You may create or update a campaign, remediation work, scenario code, tests, and durable scenario documentation when the operator authorized the work.

You must:

- state the selected target, profile, evidence run, and current standing;
- use provider-owned maturity output rather than duplicating phase ladders;
- re-audit after each material remediation slice;
- preserve unresolved work and degraded evidence honestly; and
- use current work-authoring and planning guidance for Swarm Manager artifacts.

You must not:

- create a campaign for every isolated warning;
- mark a finding validated without a fresh comparable audit;
- treat tracker unavailability as a successful campaign; or
- weaken validation to make the selected target appear met.
