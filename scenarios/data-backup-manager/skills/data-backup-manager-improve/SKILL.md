---
name: "data-backup-manager-improve"
description: "Improve Data Backup Manager's protection coverage, restore proof, and destination repairability against approved targets, without loosening the evidence."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["data-backup-manager", "improve", "coverage", "restore-proof", "setpoint", "repairability"]
  icon: "database"
  status: "active"
  revision: 1
  createdAt: "2026-11-19T00:00:00Z"
  updatedAt: "2026-11-19T00:00:00Z"
  requires:
    scenarios: ["data-backup-manager", "prompt-manager", "test-genie"]
    commands:
      - "data-backup-manager coverage report"
      - "data-backup-manager drills list"
      - "vrooli scenario test data-backup-manager"
      - "prompt-manager skill-set validate"
  origin:
    kind: "authored"
---

## Practice focus: Data Backup Manager Improvement

Improve the plant that decides whether scenario-owned state survives: protection coverage, verified recovery points, and destination health. Select the smallest intervention that moves an approved target, and never treat a healthier reading as proof that data is recoverable.

Required reading:
- `path:docs/agent-system/SKILL_AUTHORING.md` §"Scenario skill sets" — the roles, the learning spine, and the step rungs.
- `path:docs/agent-system/SCENARIO_DEVELOPMENT.md` — authority, target artifacts, and completion.
- `prompt-manager skill read improvement-do-and-dont` — anti-gaming patterns D1, D2, and D3, cited by id below.
- `path:scenarios/program-runtime/docs/guides/program-contracts.md` §"The envelope" and §"Setpoint-read rows" — the board's output shape and reason vocabulary.
- `path:scenarios/data-backup-manager/docs/internal/INVARIANTS.md` — write admission, protection, and repair rules.
- `path:scenarios/data-backup-manager/docs/internal/PROBLEMS.md` — the capability gaps that have no sensor.

### 1. Focus and scope

The plant is the protection chain: a registered target, an enabled plan, at least one writable destination, a successful run, and a verified restore. The plant is not the number of destination rows or the number of snapshots.

This skill never edits a backup repository, never formats a disk, and never grants itself authority to repair a host. It reads owner evidence, selects a route, and hands the repair to the owning method. Five scenarios depend on Data Backup Manager (`infrastructure-manager`, `money-ledger`, `offer-desk`, `research-desk`, `secrets-manager`), and the `infra-health` team reads it as a coverage source, so a degraded protection chain is a fleet finding, not a local preference.

### 2. Setpoint

Readings are dated observations. A row with no approved band stays `undecided`; the current reading never becomes the target.

| Row | Sensor | Target | Reading | In band |
|---|---|---|---|---|
| `unprotected-targets` | `data-backup-manager coverage report --json` → `summary.unprotected_target_count` | `0`; a target with no successful copy is the issue (protection model, `INVARIANTS.md`) | `30 of 248` (2026-11-19) | no |
| `single-copy-targets` | same → `summary.single_copy_target_count` | undecided; one device is a warning, and the operator has not approved a zero-target | `197` (2026-11-19) | undecided |
| `protected-targets` | same → `summary.protected_target_count` | undecided; more is better, but a target is not a policy | `21` (2026-11-19) | undecided |
| `verified-restores` | `data-backup-manager drills list --json` | each critical target has a verified restore within its plan's drill cadence | `3 verified / 100 recorded` (2026-11-19); the list is plan-scoped, not target-scoped | undecided; measurement scope is narrower than the target |
| `unrepairable-destinations` | `data-backup-manager destinations list --json` + `destinations repair-options` | `0` unwaived destinations on a filesystem this host cannot repair | `3` NTFS destinations on one host (2026-11-19) | no |
| `write-admission-blocks` | run receipts with `destination_inaccessible` and `readiness unknown` | `0` blocked runs where a bounded probe passes | `0` after 2026-11-18; the 2026-11-18 run `07bcfa91` cleared the prior block | yes |
| `test-genie` | `vrooli scenario test data-backup-manager unit contracts storage api proto quality --json` | all listed phases pass | `20261119-003531-88f85142` passed (2026-11-19) | yes |
| `skill-set` | `prompt-manager skill-set validate data-backup-manager` | usage and improve present, no suppression-shaped waiver | usage present, improve authored 2026-11-19 | yes after declaration |
| `setpoint-read` | `.vrooli/program-runtime/setpoint-read` | the board runs as a governed read | absent | `pending_telemetry`; setup gap |
| `golden-corpus` | `evals/*.json` | at least one approved corpus with a floor | absent | `pending`; setup gap |
| `usage-learning` | `vrooli-memory` scope | a declared usage scope with a facet vocabulary | no scope declared | `pending`; setup gap |

### 3. Sensors

| Sensor | Command | Validity rule |
|---|---|---|
| Coverage | `data-backup-manager coverage report --json` | Derived live from enabled plans, run receipts, and verified restores. A reading is valid only when the API is reachable and the report is not truncated. Protection counts by distinct storage device, never by destination row. |
| Drills | `data-backup-manager drills list --json` | Persisted evidence. Failed drills include historical runs; filter by plan and date before comparing. A verified drill proves one recovery point, not every destination copy. |
| Destinations | `data-backup-manager destinations list --json` and `destinations readiness` | Identity and filesystem are recorded evidence; readiness is a current observation. A filesystem that cannot be repaired here is a policy row, not a performance reading. |
| Health | `GET /health` on the API | Current posture. Last-run failures clear after a successful run; do not treat a historical count as the current rate. |
| Test Genie | `vrooli scenario test data-backup-manager <phases> --json` | The run receipt is the evidence. A pass on a stale artifact is not a pass; check the run id and the phase list. |
| Skill set | `prompt-manager skill-set validate data-backup-manager` | Structural conformance only. It does not prove skill quality or executed behavior. |
| Measures | `measures-health validate scenario data-backup-manager` | Declaration coverage. Warnings about undeclared substrate are setup debt, not protection evidence. |

Shared diagnostics: `program-runtime bindings condition --scenario data-backup-manager --window-seconds 604800` reports every binding as `DORMANT` because no invocation ledger exists yet (2026-11-19). Treat it as unavailable evidence, not as a zero rate.

### 4. Golden corpora

No `evals/` corpus or approved floor exists (2026-11-19). The recovery evidence is the drill and the restore verification, which are runtime receipts, not a frozen corpus. Do not invent a corpus to fill this section; record the setup gap and route it to `measures-adoption` or a Test Genie fixture under the active authority.

### 5. Routes

| Finding | Route | Evidence of repair |
|---|---|---|
| A registered target is unprotected | Bind it to an enabled plan with a healthy destination (`coverage accept-defaults`, plan create/update). | `unprotected-targets` falls; the target gains a successful run. |
| A destination is on a filesystem this host cannot repair | `destinations repair-options`, then `.private/scripts/repair-drive.sh` under `safe-operator-script`; or record a dated operator acceptance with `--allow-unrepairable`. | `repair=metadata` and the destination is ext4; or a dated waiver names the risk. |
| A verified restore is missing | Schedule or run a drill for the target's plan; verify the remaining destination copies. | A `DRILL_STATUS_VERIFIED` receipt and a coverage verified count. |
| A write is blocked although the drive is writable | Inspect the write verdict and the probe; repair `destinationreadiness` or the run gate that regressed. | A run reaches capture on the same destination. |
| A listed Test Genie phase fails | Select the owning rung through `scenario-work-ladder`; the phase names the provider. | A new run id passes the same phase. |
| A role or declaration is missing | `skill-set-authoring` for the role; `skill-authoring-*` for content. | `prompt-manager skill-set validate` reports both roles. |
| `setpoint-read` is absent | `program-runtime` under the active grant; compose only owner reads. | A contract and fixture exist; the board returns the declared rows. |
| The usage skill names a leaf that repeats unchanged | Promote the leaf to a CLI command or Action per `PROMOTION_LADDER.md`; record the candidate in the scenario's problem ledger. | The command exists and the skill cites it. |

### 6. Anti-gaming

- **D1** — Do not loosen or delete a tagged test to make a phase pass.
- **D2** — Do not delete a known-issue ledger entry to make the scenario look clean.
- **D3** — Do not suppress a finding, a failed drill, or an unreadable destination from the reading.

DBM-specific moves:
- Counting destination rows as devices, or counting a second destination on the same drive as a second copy.
- Treating `snapshot list` as a restore proof.
- Calling `ntfsfix` exit 0 a repair.
- Waiving `unrepairable-destinations` without naming the filesystem, the host, and the accepted risk.
- Deleting exception records instead of resolving the underlying unreadable entry.
- Lowering the protection threshold from two devices to one to raise the protected count.

The skeptic test applies: if the only change is a reading, and no recovery point, destination, or plan changed, the work did not improve protection.

### 7. Evidence

Preserve the target identity, the reading before and after, the run or drill receipt, the destination change, and any unmet obligation. Read and write `path:scenarios/data-backup-manager/docs/internal/PROBLEMS.md` and `PROGRESS.md` for durable findings, and use `vrooli-memory journal note --kind work-record` for the searchable work record. Reuse automatic capture; do not create a parallel ledger.

A reading is not a work item. If a gap has no sensor and no target, keep it in the problem ledger; do not invent a setpoint row.

### 8. Stop rules

A required outcome is unmet when its evidence is missing, unreliable, or older than its cadence. Permanent gaps stay visible and are routed within authority; they do not become passing rows. Completion needs every required outcome and its validity rule, not a fixed number of cycles.

The board is not executable yet because `setpoint-read` is absent. This role may select and route work, but it must not claim an executable closed loop until the program exists and its fixtures pass. Route that setup through `program-runtime` under the active grant.

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Every row is `pending_telemetry` | No board program exists | `.vrooli/program-runtime/setpoint-read` | Keep the rows and record the setup gap; do not substitute prose for a reading. |
| Coverage and drill counts disagree | Coverage requires every enabled destination copy to be verified; drills are plan-scoped | `coverage report --json` and `drills list` | Compare the same scope; do not average the two. |
| A destination is reported protected but the drive is NTFS | Two devices hold copies; protection is device-counted | `destinations list` and `coverage report` | Protection and repairability are separate rows; migrate the NTFS destination and keep the protection count honest. |
| A failed drill count dominates the reading | Historical failures from before a fix | Drill timestamps and plan ids | Filter to the current cadence; do not delete the failures. |
| `skill-set validate` passes but the role is thin | The validator checks structure, not quality | Run `skill-validation` on the role | Repair content through the per-role guide; do not add filler. |
| A run succeeds and health still says failed | Health uses the last failed run per target | The target's last success and run id | Confirm the success is newer; do not edit health state. |
