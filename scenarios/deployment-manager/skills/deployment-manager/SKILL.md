---
name: "deployment-manager"
description: "Use Deployment Manager to assess, prepare, release, observe, and recover Vrooli deployments with commit-scoped evidence and explicit safety boundaries."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["deployment-manager", "deployment", "readiness", "release"]
  icon: "rocket"
  status: "active"
  revision: 3
  createdAt: "2026-09-03T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  learning:
    scope: "deployment-manager-usage"
    capture: "every attempt"
  requires:
    scenarios: ["deployment-manager", "vrooli-memory"]
    commands: ["deployment-manager", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Tools focus: Deployment Manager

Use Deployment Manager as the source of truth for release preparation, readiness,
promotion, observation, and recovery. Keep deployment decisions tied to an exact
scenario commit and artifact; never treat a fitness score or package build as
release approval.

Required reading:
- `path:scenarios/deployment-manager/docs/DEPLOYMENT-GUIDE.md`
- `prompt-manager skill read vrooli-memory`

### Before acting

Recall prior task records from scope `deployment-manager-usage` for the scenario,
target, and operation. Use `search-hub` first when the task also needs design,
failure, or deployment-history context. A recalled workaround is evidence, not
authority to bypass a gate.

Before choosing the leaf, retain the IDs of advice applied or explicitly rejected
and the decision each changed. Record `no_match` when no applicable advice exists,
or `unavailable` when recall fails. Read-only program success does not establish
success of the surrounding build or release task.

Use a stable task ID across retries and a new attempt ID and ordinal per attempt.
Keep a task start timestamp and attempt start timestamp. The comparison context
names the target/profile, mode, and relevant tool/policy versions; change it when
those conditions change. Artifact/run IDs remain evidence references.

### Choose the leaf

| Situation | Step |
|---|---|
| Need to learn the supported surface | Run `deployment-manager help`, then the selected command's help. Do not infer flags from this skill. **[S1]** |
| Need to prepare or resume readiness | Run `deployment-manager.readiness-review` with the exact scenario, profile, commit, artifact digest, target, and channel. It prepares one review and reconciles only unresolved Swarm milestones. Branch on `status` and `errors[0].class`. **[S3]** |
| Need a non-mutating release decision | Run `deployment-manager.release-preflight` with the immutable review key. `signals.ready` is true only for an approved exact review. **[S3]** |
| Need to inspect or reverify a release | Use the `releases` command group. Confirm the returned commit and artifact identity before interpreting its evidence. **[S1]** |
| Need to prepare or start deployment | Use the profile, validation, and deployment command groups in that order. Stop when the release gate refuses; do not substitute fitness, packaging, or an open readiness goal for approval. **[S1]** |
| Need to diagnose a running deployment | Run `deployment-manager.release-observe` with the review key; preserve its bounded evidence references and comparison mode. **[S3]** |
| Need rollback or forward repair | Run `deployment-manager.release-recover` in its default read mode. Execute mode must name an owner binding and requires the runtime grant, explicit confirmation, exact target identity, and dry-run reference; otherwise refusal is correct. **[S3]** |
| A single low-level producer needs to report a signal | Use the typed producer/test seam. Never make a human or agent assemble that signal file as the primary readiness workflow. **[S1]** |

### Readiness judgment

- Storage follows `prompt-manager skill read storage-steer`. Whether real users or
  a prior production deployment exist determines the migration strategy.
- A release with no schema change needs no migration. A production schema change
  needs a complete ordered delta proven against representative predecessor data;
  do not require an arbitrary count of migration files.
- Tests must be current and attributable to the candidate. Compare behavioral
  results and requirements evidence with the last actual deployed release. Treat
  raw coverage as a supporting trend, not proof of correctness.
- Gherkin applies to acceptance criteria and behavioral/e2e descriptions, per
  `prompt-manager skill read writing-standards`.
- The artifact tested must be the artifact promoted. A matching source commit
  alone is insufficient when build inputs or signing differ.

### In-use settings

| Symptom | Allowed move | Journal |
|---|---|---|
| Human output omits identifiers needed for evidence | Select the command's JSON format when supported | Why structured output was required and the identifiers read |
| A bounded wait expires while server-owned work continues | Use the owning command's documented wait/status path once | Run id and terminal or still-running state |
| A signal producer is unavailable | Record `unavailable`; continue only if the readiness policy classifies it advisory | Producer, required/advisory class, and resulting gate state |

### Safety and ownership

- Manage scenario processes only through the Vrooli lifecycle.
- Never bypass a release refusal by editing release state, readiness rows, or
  deployment profiles directly.
- Never hand-edit dependency approvals or run raw package-manager installation.
- Programs may compose declared bindings; Deployment Manager remains the owner of
  release state, migration policy, recovery invariants, and promotion decisions.
- A repeated program workaround is a missing-capability signal. Route it to
  `scenario-work-ladder`; do not harden a shadow implementation.

### After acting, always

Use `vrooli-memory learning record --scope deployment-manager-usage --attempt '<Attempt JSON>'`
for one outcome-linked `task-record` per attempt. The shared `vrooli-memory`
skill and `path:packages/proto/schemas/vrooli-memory/v1/learning/learning.proto`
own the record shape. This replaces the ordinary task-record append; do not write
both. Retry capture with the same ID and unchanged payload after transport failure.

Include the chosen leaf, task and attempt identities, comparison context,
observed timestamps, evidence references, and every applied/rejected advice ID.
Use `verified_success` only for the selected operation's evidenced outcome.
Keep `failed`, `unavailable`, and `unknown` distinct. Failed attempts name a stable
failure fingerprint. Unverified advice verdicts remain `unknown`; a successful
operation alone does not prove its recalled advice helped. Mark fixtures `test`.

Use a separate `binding-note` for an incorrect callable contract. Memory failure
does not change the operation's outcome: retain the uncaptured record in the work
reference and report capture unavailable. Never include logs, media, private
endpoints, or credentials. Follow the shared curation rules for confirmed or
superseded advice.

### Troubleshooting & Edge Cases

| Symptom | First check | Response |
|---|---|---|
| Readiness goal is duplicated or milestones cannot be written | Read the canonical goal name returned by Swarm Manager | Stop and report the integration defect; do not create another logical goal |
| Verdict passes but promotion refuses | Compare approved commit, candidate commit, goal closure, and waiver scope | Refresh stale evidence or close the exact goal; never relabel the commit |
| No actual predecessor can be established | Release history and declared deployment target | Classify the release as greenfield; do not invent a migration baseline |
| Existing data needs a schema change | Storage maturity and predecessor schema | Read `storage-steer`; prove the selected migration tier on a copy before release |
| A useful repeated route has no program | Task records for the same input/join shape | File the program contract candidate; keep using the explicit safe leaves meanwhile |

The four recurring usage paths are governed programs. A single deterministic
operation remains in the typed Deployment Manager CLI; do not duplicate it in
skill prose or another program.
