---
name: "skill-set-authoring"
description: "Derive the skills a scenario owes (usage, feature, improve), inventory the sensors and programs those skills can cite, orchestrate authoring through the per-role guides, and write the service.json skills declaration. Refuses to invent a sensor."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["meta"]
  tags: ["skill", "authoring", "skill-set", "scenario", "self-improvement", "meta-optimization"]
  icon: "layers"
  status: "active"
  revision: 4
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T23:00:00Z"
  requires:
    scenarios: ["prompt-manager", "measures-health", "program-runtime"]
    commands: ["prompt-manager skill read", "prompt-manager skill list", "measures-health validate scenario", "business-health matrix show", "program-runtime bindings condition", "program-runtime library search", "program-runtime programs submit", "swarm-manager backlog create", "cli-health search query", "vrooli scenario status", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Meta focus: Skill Set Authoring

Given one scenario, decide which skill roles it owes, find every sensor and program its skills may cite, author or repair the roles through the per-role guides, and declare the result in the scenario's `.vrooli/service.json`. The output is a skill set that `prompt-manager.skill-set-read` reports as present and that `skill-validation` can pass. Today that program reads five rows (registered ids, usage present, improve present, set token size, read counts); waiver grading, `programs[]` resolution, sensor reality, and dialect checks are planned for a validator command that does not exist yet.

Required reading:
- `path:docs/agent-system/SKILL_AUTHORING.md` §"Scenario skill sets" — the roles, their owed-when triggers, step rungs, the learning spine, programs as steps. Cited, never restated here.
- `prompt-manager skill read skill-authoring-tools` — authors the usage role.
- `prompt-manager skill read improve-skill-authoring` — authors the improve role and renders its sensor-read program.

### 1. Scope

**In scope:** one scenario per run; deciding owed roles; the sensor and program inventory; writing or repairing the role skills through the guides; the `skills` block in `.vrooli/service.json`; waivers with dated reasons; filing a backlog item for every missing sensor (§3 Phase 2 "Filing recipe").

**Out of scope:** judging skill quality (`skill-validation`); conditioning cost (`skill-improvement-suggestions`); writing programs the skills name (`program-runtime` skill and `path:scenarios/program-runtime/docs/guides/program-contracts.md`); changing the scenario's code, manifest, or measures; moving skills between packs for scenarios outside this run.

### 2. Governance surface

This skill owns one decision: **which roles a scenario owes and whether each is present, waived, or missing.** The triggers are canon; this skill applies them. Where the trigger is ambiguous (a dependents count near the threshold, a projection ownership that is declared in prose only), this skill records the ambiguity in the declaration's waiver reason and chooses the owed reading. Authority ends at the declaration: `skill-set-read` reports it, the per-role guides author the content.

### 3. Process

#### Phase 1: Classify the scenario

**Entry:** the scenario name and `scenarios/<scenario>/` exist.

1. Read `scenarios/<scenario>/.vrooli/service.json` and `scenarios/<scenario>/cli/manifest.json` (present or absent).
2. Count dependents. There are two sources, and only the first is a count of scenarios:
   - (a) **Declared dependents** — scenarios whose `.vrooli/service.json` names this scenario under `dependencies.scenarios` (an object keyed by scenario name): `jq -r '.dependencies.scenarios // {} | keys[]' scenarios/*/.vrooli/service.json | grep -cx <scenario>`. This is the dependents count the trigger table reads.
   - (b) **Observed callers** — `program-runtime bindings condition --scenario <scenario> --window-seconds 2592000 --json` reports exercise counts per binding over thirty days. The invocation ledger does not attribute a call to the calling scenario, so (b) confirms that the scenario is called; it does not count callers. Record it as corroboration; never add it to (a).
   Canon (`SKILL_AUTHORING.md` §"The three roles") names no other source. `team.json` has no scenario-dependency field and is not a source.
3. Read `path:docs/concepts/RECURSIVE_SELF_IMPROVEMENT.md` §2 for the projection owners.
4. Walk the trigger table:

| Question | If YES | If NO |
|---|---|---|
| Does `cli/manifest.json` exist? | usage is owed | usage is not owed; record why the scenario has no CLI |
| Is the scenario a projection owner? | improve is owed | continue |
| Is the declared dependents count (source (a)) two or more? | improve is owed | improve is waivable with the count from (a) and the reading from (b) as the reason |
| Does a feature pattern appear in two or more agents' instructions (`capability-extraction` §Extraction Test)? | that feature is owed | no feature role |

**Exit:** a table of roles with `owed` / `not owed` and the evidence line for each.

**Artifacts:** the trigger table, kept for the declaration's `waivers.<role>.reason`.

#### Phase 2: Inventory sensors and programs

**Entry:** at least one role is owed.

1. Measures: `measures-health validate scenario <scenario>`; list the declared measures and the stateful domains that are neither covered nor waived.
2. Golden corpora: `ls scenarios/<scenario>/evals/` and any `*.primary.json` with a `floor` field.
3. Condition and friction sensors that exist for every scenario: `program-runtime bindings condition --scenario <scenario> --window-seconds 604800` for its bindings; the `agent-manager.friction-digest` program (inputs `scenario`, `window_days`, default 7) for recurring friction on its commands.
4. Programs: `ls scenarios/<scenario>/.vrooli/program-runtime/*.json`; `program-runtime library search "<scenario>"` for library-owned entries.
5. PRD operational targets: `business-health matrix show <scenario> --format summary` lists every `OT-*` with its requirement and validation linkage; a target with no sensor in steps 1 to 3 is a `pending_telemetry` row, not a goal.
6. Problems ledger: open entries in `scenarios/<scenario>/docs/PROBLEMS.md` or `docs/internal/PROBLEMS.md`.

**Exit:** a sensor inventory with three columns — sensor, command, status (`measured`, `pending_telemetry`) — and a program inventory.

**Artifacts:** the inventories, handed to the per-role guides. Do not write a sensor the inventory does not contain.

**Filing recipe.** Every `pending_telemetry` row with a stateful domain behind it becomes one backlog item:

```
swarm-manager backlog create --data '{"kind":"chore","name":"<scenario>-measure-<domain>","title":"Declare measure <domain>.<name> for <scenario>","description":"<which improve row needs it>"}' --json
```

`--json` because the created `kind/name` is read from the response and quoted in the work record. The backlog item's free-text field is `description` (`scenarios/swarm-manager/api/internal/backlog/types.go`); there is no `context` field. `report-bug` is not the channel: its signal types are defects, and a missing measure is not one. The worker that picks the item up reads `prompt-manager skill read measures-adoption` and cites it in the work record. When swarm-manager is stopped (`vrooli scenario status swarm-manager`), record the item text in this run's work record and file it on the next run.

#### Phase 3: Author or repair each owed role

**Entry:** the inventories exist.

| Role | Guide | What this skill hands the guide |
|---|---|---|
| usage | `skill-authoring-tools` §7 | the program inventory, the memory scope if one is declared, the commands whose output contracts are deterministic (promotion candidates) |
| improve | `improve-skill-authoring` | the sensor inventory, the corpora, the problems ledger, the dependents count from Phase 1 step 2 (a) |
| feature | `skill-authoring-tools` or `skill-authoring-practice` | the extraction-test evidence |

Repair rule: when the role exists, run `prompt-manager skill read skill-validation` on it first and hand its findings to the guide. Do not rewrite a skill whose findings are all Minor.

Location rule: every role lives at `scenarios/<scenario>/skills/<skill-id>/SKILL.md` with frontmatter `name` equal to the folder and the full Vrooli metadata block. A role that currently lives in a prompt-manager pack moves by deleting the pack copy and creating the scenario copy in the same change; the id does not change, so `promptRef` and Action references keep resolving. Confirm the id is not in `store/skills/_base-pack.json` before moving, because scenario skills are not projected into native harness directories.

**Exit:** each owed role passes `skill-validation`, including the divergence probe (`skill-validation` §3.3), with no Critical or Major finding. A confirmed C4 (`SKILL_AUTHORING.md` §"Conditioning defect patterns") is Major and blocks the exit until the author makes the decision the probe exposed.

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
- `programs` lists only programs that exist as `.vrooli/program-runtime/<name>.json` or as library entries (`program-runtime library search "<name>"`). A name that resolves to neither is a declaration defect; check it by hand, because `skill-set-read` does not resolve program names yet.
- `learning.scope` is present only when the usage skill declares `metadata.learning`. A scenario whose own ledger is the memory sets `learning.ledger` instead.
- A waiver is `waivers.<role>` with `reason` (at least twenty characters, naming the trigger evidence) and `declared_at`. A waiver on a role the trigger says is owed is recorded, and this skill states in its report and its work record that the waiver is suspect; the planned validator will grade it.
- The schema is `path:.vrooli/schemas/service.schema.json` `skillSetDeclaration`; validate the file before finishing.

**Exit:** `prompt-manager skill list` shows every declared role under pack `scenario` with no registry error, and the declaration validates.

**Artifacts:** the declaration; one `vrooli-memory journal note "skill-set: <scenario> <roles authored or repaired>" --kind work-record --trigger "<why>" --approach "<roles and guides>" --evidence "<pending_telemetry sensors>" --outcome "<items filed>"`.

### 4. Convergence patterns

Two agents running this skill on the same scenario must produce the same declaration. The places that decide it:

| Decision | Deterministic source |
|---|---|
| usage owed | existence of `cli/manifest.json` |
| improve owed | projection ownership from canon, or the declared dependents count (Phase 1 step 2 (a)) |
| a sensor is real | it appears in the measures manifest, an evals file with a floor, a `bindings condition` row, or a friction digest |
| a program is real | its contract file exists or `library search` returns it |
| a waiver is honest | its reason quotes the trigger evidence |

### 5. Anti-patterns

| Anti-pattern | Why it fails | Instead |
|---|---|---|
| Inventing a sensor to complete an improve skill | The skill's setpoint reads as measured when it is prose; the loop regulates nothing | Record `pending_telemetry`, file with the Phase 2 recipe |
| Authoring a feature skill from one agent's habit | Sprawl; the pattern has not proven reuse | Fail the extraction test, note the candidate in the work record |
| Leaving both a pack copy and a scenario copy of a moved skill | Duplicate ids make the whole registry list fail | Delete and create in one change; run `prompt-manager skill list` |
| Writing the declaration before the skills exist | Validator reports drift on every role | Declare in Phase 4 only |
| Restating role definitions in the authored skill | Canon drift | Cite `SKILL_AUTHORING.md` §"Scenario skill sets" |

### 6. Output expectations

You may: create or update files under `scenarios/<scenario>/skills/`, the `skills` block of `scenarios/<scenario>/.vrooli/service.json`, and delete a pack copy of a skill you moved.

You must: cite canon for roles and rungs; hand authoring to the per-role guides; keep every cited command real (`skill-validation` §3.6); file one backlog item (Phase 2 recipe) for every `pending_telemetry` sensor with a stateful domain; leave a work record.

You must not: change scenario code, manifests, measures, or programs; author an improve skill whose setpoint has no measured row; move a skill that is in the base pack.

### 7. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `prompt-manager skill list` errors after creating a scenario skill | Frontmatter `name` differs from the folder, or the id collides with a pack skill | `ls scenarios/<scenario>/skills/`; `prompt-manager skill list \| grep <id>` | Rename the folder or the id; remove the duplicate |
| The scenario has a manifest but no CLI binary builds | usage is still owed; the skill documents the contract, not the binary | `cli-health search query "<scenario>"` | Author usage; note the build state in its Troubleshooting section |
| Source (a) counts zero dependents but source (b) shows exercised bindings | Callers are agents or operators, not scenarios | the `bindings condition` rows | The trigger reads (a); quote both in the waiver reason if waiving |
| The scenario owns a projection only in prose | The space doc exists but no `space --projection` command | `ls scenarios/<scenario>/docs/spaces/` | improve is owed; its setpoint records the coverage row as `pending_telemetry` |
| `measures-health` is stopped | Sensor inventory cannot be graded | `vrooli scenario status measures-health` | Start it through the lifecycle; do not read the manifest by hand as a substitute for the probe |
