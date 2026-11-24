---
name: "scenario-skill-setup"
description: "Set up one scenario's skill roles: decide which roles are owed or requested, inventory real sensors and programs, author usage, improve and feature roles against an approved target, and declare the skill set. Optional roles are not fleet obligations."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["meta"]
  tags: ["skill", "authoring", "skill-set", "scenario", "improve", "setpoint", "meta-optimization"]
  icon: "layers"
  status: "active"
  revision: 10
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "measures-health", "program-runtime"]
    commands: ["prompt-manager skill read", "prompt-manager skill list", "measures-health validate scenario", "business-health matrix show", "program-runtime bindings condition", "program-runtime library search", "prompt-manager skill-set validate", "vrooli scenario test"]
  origin:
    kind: "authored"
---
## Scenario skill setup

For one scenario, select the owed or requested roles, inventory what they depend on, author them, and declare the result in `scenarios/<scenario>/.vrooli/service.json`. Roles, owed-when triggers, rungs and the learning spine are defined in `docs/agent-system/SKILL_AUTHORING.md` §"Scenario skill sets"; artifact placement and authority are in `docs/agent-system/SCENARIO_DEVELOPMENT.md`. Cite them, do not restate them (R7). An inventory-only request stops after reporting. Sensor implies no authority; program, measure and product repairs belong to their owning methods and need the caller's grant. `prompt-manager skill-set validate <scenario>` checks declarations only; `skill-validation` judges content.

### 1. Which roles are owed

| Question | Yes | No |
|---|---|---|
| `cli/manifest.json` exists? | usage owed | record why there is no CLI |
| Scenario is a projection owner (`docs/concepts/RECURSIVE_SELF_IMPROVEMENT.md` §2)? | improve owed | continue |
| Two or more distinct scenarios or teams declare a dependency on it? | improve owed | no obligation from this trigger; keep any inventory uncertainty |
| Operator asked for an improve role or contract development? | select improve | author only owed or selected roles |
| A feature pattern passes `capability-extraction`'s reuse test? | feature eligible if it adds distinct value | no feature role |

Count dependents from `dependencies.scenarios` and from team contracts, citing each source; binding invocation volume is not an owner count. Record obligation (`owed`, `not owed`, `unresolved`) separately from selection. A waiver records an authorized deferral with dated applicability evidence; it does not replace investigating the trigger or deliver the role.

### 2. Inventory what the roles consume

Scope the inventory to the selected roles; the full outcome and sensor inventory applies when improve is selected.

1. Measures: `measures-health validate scenario <scenario>`; declared measures and stateful domains neither covered nor waived.
2. Golden corpora: `scenarios/<scenario>/evals/` and any `*.primary.json` with a `floor`.
3. Programs: `scenarios/<scenario>/.vrooli/program-runtime/*.json` and `program-runtime library search "<operation>"`; record inputs, validity conditions, effects, budget. Any owner's program may be composed within the caller's scope.
4. Targets: `business-health matrix show <scenario> --format summary`. A target with no evidence source is a measurement gap, not a dropped target and not a Swarm goal.
5. Open entries in the scenario's `PROBLEMS.md`.

Separate an unreadable sensor from an absent one. File presence proves a declaration, not a measurement. A missing sensor or program: under a development mandate, load `measures-adoption` or `program-runtime`, set it up, return here; for skill-only setup, write honest `pending_telemetry` rows and record the prerequisite; for read-only assessment, report the gap and change nothing. Never invent a sensor to finish a skill.

### 3. Author each selected role

Run `skill-validation` on an existing role first; do not rewrite a skill whose findings are all Minor. Each role lives at `scenarios/<scenario>/skills/<skill-id>/SKILL.md` with `name` equal to the folder. Moving a skill out of a pack means deleting the pack copy and creating the scenario copy in the same change, with the same id (duplicate ids make the registry list fail); never move a skill in `store/skills/_base-pack.json`.

**Usage:** a tools skill (see `skill-authoring`) that is a decision tree whose leaves each carry a rung label and one step, a learning spine when `metadata.learning.scope` is set (memory mechanics belong to `vrooli-memory`), one `symptom -> setting move` table, and a promotion pass: a leaf whose command returns a deterministic pass or fail becomes an Action reference, and anything `--help` already says is deleted.

**Feature:** only from extraction-test evidence; author with `skill-authoring`.

**Improve:** judgment against an approved target the agent cannot redefine; it picks priorities, reads evidence and names repair routes. Execution inside a mandate belongs to `scenario-improvement-campaign`; `goal-loop` chooses development or observation. Use these sections in order, plus `Troubleshooting & Edge Cases`:

| # | Section | Rule |
|---|---|---|
| 1 | Focus and scope | Name the plant (what is regulated) and what the skill never touches |
| 2 | Setpoint | Each required outcome links to named rows, sensor commands or gaps, approved bands and dated readings. A row is not a Swarm goal. |
| 3 | Sensors | Commands plus validity and comparability rules; an independent measurement outranks a self-report |
| 4 | Golden corpora | Suites, approved floors, sample rules; preserve holdouts; a failed protected floor takes priority and only the target owner changes it |
| 5 | Routes | Problems, sensor rows and scope mapped to the owning repair method and the expected evidence change |
| 6 | Anti-gaming | Cite D1 to D3 and the skeptic test from `improvement-do-and-dont` by id; add this scenario's own gaming moves (waiver on a needed domain, editing a registry to match a claim, widening a floor, deleting a failing case) |
| 7 | Evidence | Target identity, valid before and after evidence, unmet obligations, next action; reuse automatic capture, no parallel ledger |
| 8 | Stop rules | Missing or unreliable required evidence stays unmet; honor budget, cancellation and amendment limits; completion needs every required outcome |

Setpoint rules. A band references approved intent: never the current reading or an invented trend (a detector that tracks the thing it watches cannot see a fault, R6); leave the target undecided until it is approved. Use one owner-derived reading per row and keep composite acceptance in its owning evaluator. Readings are dated observations. The recursive row ("share of targets with a conformant skill set") belongs only to scenarios that install capability elsewhere.

Routes rule. Read two sources: sensor rows and the problems ledger. A wanted capability (data that is persisted and unreadable, an absent corpus) has no target and no sensor, so it can only live in the ledger; do not promote it into the Setpoint table. A silent ledger is unknown, not clean. Choose the smallest intervention that advances an approved outcome without breaching a protected floor; in-cycle curation the scenario already exposes is allowed with its grant, a missing sensor or binding goes to `measures-adoption` or the binding owner, a contract or implementation defect goes to the method chosen through `scenario-work-ladder`, and a dependency defect goes to its owner unless the grant covers it. Sensor evidence never authorizes a cross-owner edit.

`setpoint-read` program, when program edits are authorized: read-only, no inference or delegation, composes only children whose effects and budgets fit; the row shape, reason vocabulary and statuses are owned by `scenarios/program-runtime/docs/guides/program-contracts.md` (§"Setpoint-read rows"), so copy none of it. Keep required rows even when their sensor or band is missing. Validate with `vrooli scenario test <scenario> --phases programs`, with fixtures for valid evidence, missing targets, permanent gaps and failed children, and expected statuses taken from the fixture inputs, never from today's outage.

### 4. Declare

Write the `skills` block (schema `.vrooli/schemas/service.schema.json` `skillSetDeclaration`) only after the skills exist:

```json
"skills": {
  "usage":   { "source": "skills/<scenario>/SKILL.md", "programs": ["<scenario>.<name>"] },
  "improve": { "source": "skills/<scenario>-improve/SKILL.md", "programs": ["<scenario>.setpoint-read"] },
  "feature": [], "learning": { "scope": "<scenario>-usage" }, "waivers": {}
}
```

`programs` lists only programs that exist as contract files or library entries (declaration validation does not resolve names; check yourself). `learning.scope` appears only when the usage skill declares `metadata.learning`. A waiver is `waivers.<role>` with a `reason` of at least twenty characters naming the trigger evidence, and `declared_at`.

Done: each authored role passes `skill-validation` with no Critical or Major finding; `prompt-manager skill list` shows every declared role under pack `scenario`; `skill-set validate` has no findings. Report declaration, semantic and executed-behavior readiness separately; an all-pending improve role preserves a target and is not a runnable loop. Update the scenario's entry point to link the roles; do not append a second onboarding checklist.

### Troubleshooting & Edge Cases

| Symptom | Cause | Fix |
|---|---|---|
| `skill list` errors after adding a scenario skill | frontmatter `name` differs from the folder, or the id collides with a pack skill | rename the folder or id; remove the duplicate |
| Every improve row is `pending_telemetry` | measurements absent | preserve the target with pending rows, report the setup as non-executable, route repair by authority |
| A row's command is a local CLI (`binding.kind: local`) | no governed binding | row is `unavailable` with reason `no_governed_binding`; route to the binding owner |
| A corpus has no `floor` | target not approved | propose a floor with comparable evidence; keep the target undecided |
| Two route rows for one reading | the divergence probe will flag it | merge or add a discriminating predicate |
