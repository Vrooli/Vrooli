---
name: "skill-set-authoring"
description: "Determine owed and requested scenario skill roles, inventory real evidence and programs, author through per-role guides, and declare the selected skill set. Optional feature roles are not fleet obligations."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["meta"]
  tags: ["skill", "authoring", "skill-set", "scenario", "self-improvement", "meta-optimization"]
  icon: "layers"
  status: "active"
  revision: 9
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-08T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "measures-health", "program-runtime"]
    commands: ["prompt-manager skill read", "prompt-manager skill list", "measures-health validate scenario", "business-health matrix show", "program-runtime bindings condition", "program-runtime library search", "cli-health search query", "vrooli scenario status", "prompt-manager skill-set validate"]
  origin:
    kind: "authored"
---
## Meta focus: Skill Set Authoring

For one scenario, select the owed or requested roles and inventory their actual dependencies.
Author through the per-role guides and declare the result in `.vrooli/service.json`.
Use `prompt-manager skill-set validate <scenario>` for declaration findings and
`skill-validation` for content judgment. The installed validator checks declared
roles, source presence, basic metadata markers, and waiver fields. It does not
prove every applicability trigger, program reference, sensor, or skill decision.

Required reading:
- `path:docs/agent-system/SCENARIO_DEVELOPMENT.md` — artifact placement and authority-aware setup.
- `path:docs/agent-system/SKILL_AUTHORING.md` §"Scenario skill sets" — the roles, their owed-when triggers, step rungs, the learning spine, programs as steps. Cited, never restated here.

Load the per-role guide in Phase 3 only for a role selected for authoring.
An inventory-only request stops after reporting; it does not authorize Phase 4.

### 1. Scope

**In scope:** one scenario per run; owed roles; sensor and program inventory; role authoring through the guides; the `skills` declaration; dated waivers; routing missing setup under the active authority.

**Out of scope:** skill-quality judgment (`skill-validation`) and conditioning cost (`skill-improvement-suggestions`). Delegate program, measure, and product repairs to their owning methods within the same authorized engagement. This setup guide alone grants no such effects. The `skills` declaration is its only service-manifest edit.

### 2. Governance surface

This skill owns role applicability and setup inventory. Distinguish the fleet
minimum from roles explicitly requested by the operator. An uncertain trigger
remains an assessment limitation; a waiver records an intentional deferral, not
a substitute for investigating applicability. `skill-set-read` reports declarations;
the per-role guides own content, and the caller owns mutation authority.

### 3. Process

#### Phase 1: Classify the scenario

**Entry:** the scenario name and `scenarios/<scenario>/` exist.

1. Read `scenarios/<scenario>/.vrooli/service.json` and `scenarios/<scenario>/cli/manifest.json` (present or absent).
2. Inspect declared dependents in scenario `dependencies.scenarios` and explicit
   dependency statements in maintained team contracts. Count distinct owners;
   cite the source for each. Do not invent a team dependency field or add binding
   invocation counts to an owner count. Use `program-runtime bindings condition`
   only as exercise evidence. If owner inventory is incomplete, report that limit.
3. Read `path:docs/concepts/RECURSIVE_SELF_IMPROVEMENT.md` §2 for the projection owners.
4. Walk the trigger table:

| Question | If YES | If NO |
|---|---|---|
| Does `cli/manifest.json` exist? | usage is owed | usage is not owed; record why the scenario has no CLI |
| Is the scenario a projection owner? | improve is owed | continue |
| Do two or more distinct scenarios or teams depend on it? | improve is owed | no fleet obligation is established by this trigger; retain any inventory uncertainty |
| Did the operator request an improve role or scenario-contract development? | select improve even when the fleet minimum does not require it | author only selected or owed roles |
| Does a feature pattern pass `capability-extraction`'s reuse test? | optional feature is eligible; select it only when it adds distinct value within scope | no feature role |

**Exit:** a table separating role obligation (`owed`, `not owed`, `unresolved`),
selection for this task, and source evidence. Optional does not mean mandatory.

**Artifacts:** applicability evidence in the existing work record; dated waiver
reasons only for roles intentionally deferred under the caller's authority.

#### Phase 2: Inventory sensors and programs

**Entry:** at least one role is owed or explicitly selected.

Scope this inventory to the selected roles. Usage/feature-only work inspects the
operations, programs, learning, and outcome checks those roles consume. It does not
require a scenario-wide setpoint board or corpus audit. Apply the full outcome and
sensor inventory below when improve is selected. Report other owed-role gaps
without silently expanding the task into their implementation.

1. Measures: `measures-health validate scenario <scenario>`; list the declared measures and the stateful domains that are neither covered nor waived.
2. Golden corpora: `ls scenarios/<scenario>/evals/` and any `*.primary.json` with a `floor` field.
3. Check available shared diagnostics: `program-runtime bindings condition --scenario <scenario> --window-seconds 604800` and `agent-manager.friction-digest`. Read their contracts and validity rules. They may lack samples or be unavailable; diagnostic availability is not universal and does not create an acceptance gate.
4. Programs: inspect `scenarios/<scenario>/.vrooli/program-runtime/*.json` and use `program-runtime library search "<operation>"` for reusable operations across owners. Record each candidate's inputs, output validity conditions, effects, and budget. The owning scenario of a program does not restrict which usage or improve programs may call it; composition follows `program-contracts.md` and the caller's authorized scope.
5. PRD operational targets: `business-health matrix show <scenario> --format summary` supplies target and requirement linkage. A target without an evidence source remains an outcome with a measurement gap; do not drop it or create a Swarm goal for it.
6. Problems ledger: open entries in `scenarios/<scenario>/docs/PROBLEMS.md` or `docs/internal/PROBLEMS.md`.

**Exit:** an evidence inventory linking outcomes to source, command, validity rule,
observed result or gap, and repair owner; plus a program inventory. Separate an
existing but unreadable sensor from an absent sensor. File presence proves a
declaration, not a measurement. Approved target absence is a separate decision gap.

**Artifacts:** the inventories, handed to the per-role guides. Do not write a sensor the inventory does not contain.

**Missing setup disposition.** Keep every measurement gap visible in the inventory.

| Active authority | Next step |
| --- | --- |
| Development mandate includes the missing sensor or program. | Load `measures-adoption` or `program-runtime`, perform the setup, and return to this inventory. Keep the existing work identity. |
| Documentation/skill setup only. | Author honest pending rows and record the implementation prerequisite for the same work. Do not claim executable readiness. |
| Read-only assessment. | Report the gap without creating work or changing declarations. |
| Separate work is required and filing is authorized. | Use `swarm-manager-work-authoring`; missing measurement alone is not a bug report. |

An all-pending improve role can preserve a target contract. It is not a runnable
closed loop. Report declaration, semantic, and executed-behavior readiness separately.

#### Phase 3: Author or repair each selected role

**Entry:** the inventories exist.

| Role | Guide | What this skill hands the guide |
|---|---|---|
| usage | `skill-authoring-tools` §7 | the program inventory, the memory scope if one is declared, the commands whose output contracts are deterministic (promotion candidates) |
| improve | `improve-skill-authoring` | approved outcome references, evidence and program inventories, corpora, problems, and applicability evidence |
| feature | `skill-authoring-tools` or `skill-authoring-practice` | the extraction-test evidence |

Repair rule: when the role exists, run `prompt-manager skill read skill-validation` on it first and hand its findings to the guide. Do not rewrite a skill whose findings are all Minor.

Location rule: every role lives at `scenarios/<scenario>/skills/<skill-id>/SKILL.md` with frontmatter `name` equal to the folder and the full Vrooli metadata block. A role that currently lives in a prompt-manager pack moves by deleting the pack copy and creating the scenario copy in the same change; the id does not change, so `promptRef` and Action references keep resolving. Confirm the id is not in `store/skills/_base-pack.json` before moving, because scenario skills are not projected into native harness directories.

**Exit:** each authored role passes `skill-validation`, including its divergence
probe, with no Critical or Major finding. Report deferred roles separately;
do not imply that they were authored or tested.

#### Phase 4: Declare

**Entry:** roles authored.

Write the `skills` block in `scenarios/<scenario>/.vrooli/service.json`:

```json
"skills": {
  "usage":   { "source": "skills/<scenario>/SKILL.md", "programs": ["<scenario>.<name>"] },
  "improve": { "source": "skills/<scenario>-improve/SKILL.md", "programs": ["<scenario>.setpoint-read"] },
  "feature": [],
  "learning": { "scope": "<scenario>-usage" },
  "waivers": {}
}
```

Rules:
- `programs` lists only programs that exist as `.vrooli/program-runtime/<name>.json` or as library entries (`program-runtime library search "<name>"`). A name that resolves to neither is a declaration defect; verify it against the contract index, because declaration validation does not resolve program names.
- `learning.scope` is present only when the usage skill declares `metadata.learning`. A scenario whose own ledger is the memory sets `learning.ledger` instead.
- A waiver is `waivers.<role>` with `reason` (at least twenty characters, naming the trigger evidence) and `declared_at`. Preserve the owed obligation and explain the authorized deferral; a waiver is not delivered capability. Flag suppression when a waiver hides that obligation. The validator checks fields, while this review checks applicability and authority.
- The schema is `path:.vrooli/schemas/service.schema.json` `skillSetDeclaration`; validate the file before finishing.

**Exit:** `prompt-manager skill list` shows every declared role under pack `scenario` with no registry error; `prompt-manager skill-set validate <scenario>` has no declaration findings; the program-reference inventory and per-role review pass. A clean structural result alone does not certify skill quality.

**Artifacts:** the declaration, validation evidence, remaining setup gaps, and
the active work owner's record. Update the existing scenario entry point to link
the roles and their target/evidence owners. Reconcile contradictory setup or
readiness instructions there; do not append a second onboarding checklist.
Follow `vrooli-memory` for learning capture; do not duplicate automatic capture.

### 4. Convergence patterns

Two agents given the same scenario evidence and role selections must agree on
the declaration. Record optional selection decisions rather than presenting them
as fleet requirements. The places that decide it:

| Decision | Deterministic source |
|---|---|
| usage owed | existence of `cli/manifest.json` |
| improve owed | projection ownership or evidence of at least two distinct dependent scenarios/teams |
| improve selected | owed-role repair or explicit setup/development request |
| feature selected | extraction evidence plus a recorded in-scope selection decision |
| a sensor is declared / measured | owner definition / a successful applicable read; report these separately |
| a program is real | its contract file exists or `library search` returns it |
| a waiver is honest | dated applicability evidence and authorized deferral; the owed role remains undelivered |

### 5. Anti-patterns

| Anti-pattern | Why it fails | Instead |
|---|---|---|
| Inventing a sensor to complete an improve skill | The loop regulates prose rather than evidence | Record `pending_telemetry` and use Phase 2's authority-aware disposition |
| Authoring a feature skill from one agent's habit | Sprawl; the pattern has not proven reuse | Fail the extraction test, note the candidate in the work record |
| Leaving both a pack copy and a scenario copy of a moved skill | Duplicate ids make the whole registry list fail | Delete and create in one change; run `prompt-manager skill list` |
| Writing the declaration before the skills exist | Validator reports drift on every role | Declare in Phase 4 only |
| Restating role definitions in the authored skill | Canon drift | Cite `SKILL_AUTHORING.md` §"Scenario skill sets" |

### 6. Output expectations

You may: create or update files under `scenarios/<scenario>/skills/`, the `skills` block of `scenarios/<scenario>/.vrooli/service.json`, and delete a pack copy of a skill you moved.

You must: cite canon for roles and rungs; use the per-role guides; verify cited commands; retain every pending measurement; distinguish declaration readiness from executable readiness.

You must not: infer implementation authority from setup instructions, mark an all-pending loop executable, or move a skill in the base pack.

### 7. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `prompt-manager skill list` errors after creating a scenario skill | Frontmatter `name` differs from the folder, or the id collides with a pack skill | `ls scenarios/<scenario>/skills/`; `prompt-manager skill list \| grep <id>` | Rename the folder or the id; remove the duplicate |
| The scenario has a manifest but no CLI binary builds | usage is still owed; the skill documents the contract, not the binary | `cli-health search query "<scenario>"` | Author usage; note the build state in its Troubleshooting section |
| Few declared dependents but many binding invocations | Invocation count is not a dependent-owner count | inspect owner references and condition validity | Retain applicability uncertainty; do not infer caller identity from volume |
| The scenario owns a projection only in prose | The space doc exists but no `space --projection` command | `ls scenarios/<scenario>/docs/spaces/` | improve is owed; its setpoint records the coverage row as `pending_telemetry` |
| `measures-health` is stopped | Sensor inventory cannot be graded | `vrooli scenario status measures-health` | Start through the lifecycle only when runtime changes are authorized; otherwise report unavailable validation. A manifest read is inventory, not probe evidence. |
