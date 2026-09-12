---
name: "scenario-to-desktop"
description: "Build and inspect Electron desktop applications through the desktop ramp; select deployment modes, preserve target-bound evidence, and route release approval to Deployment Manager."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["desktop", "electron", "packaging", "deployment", "evidence"]
  icon: "monitor"
  status: "active"
  revision: 52
  createdAt: "2026-01-31T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["scenario-to-desktop", "deployment-manager", "program-runtime", "vrooli-memory"]
    commands: ["scenario-to-desktop", "program-runtime library run", "vrooli scenario", "vrooli-memory"]
  learning:
    scope: "scenario-to-desktop-usage"
    capture: "every attempt"
  origin:
    kind: "authored"
---
## Tools focus: Scenario to Desktop

Use the desktop ramp to generate, package, and inspect Electron applications.
Keep package creation, native runtime evidence, and release approval separate.

In scope: bundled and external-server desktop modes, pipeline diagnosis,
retained evidence, signing configuration, and authorized publication handoff.
Scenario repair belongs to `prompt-manager skill read scenario-to-desktop-improve`.
Deployment Manager owns release approval. The workflow provider owns semantic
journeys. `delivery-ramp-go` owns shared matrix and disposition rules.

Required reading:
- `path:scenarios/scenario-to-desktop/docs/OVERVIEW.md`
- `path:docs/reference/scenario-to-desktop-evidence-and-tier-contract.md`
- `prompt-manager skill read program-runtime`
- `prompt-manager skill read vrooli-memory`

### Before acting

Recall the packaged scenario, platform, and operation from
`scenario-to-desktop-usage`. Treat recalled workarounds as evidence to check
against the current command contract. Memory mechanics belong to `vrooli-memory`.

Before choosing the leaf, retain the IDs of advice applied or explicitly rejected
and the decision each changed. Record `no_match` when no applicable advice exists,
or `unavailable` when recall fails. Read-only program success does not establish
success of the surrounding build or release task.

Use a stable task ID across retries and a new attempt ID and ordinal per attempt.
Keep a task start timestamp and attempt start timestamp. The comparison context
names the target/profile, mode, and relevant tool/policy versions; change it when
those conditions change. Artifact/run IDs remain evidence references.

### Choose the operation

Read rows in order. Take the first row that matches the current task.
Use command help for inputs; do not copy flags from historical work records.

| Situation | One step |
|---|---|
| The command or its inputs are unknown | Run `scenario-to-desktop help`, or the selected command's `--help` when its name is known. **[S1]** |
| Need to choose a deployment mode | Read `path:scenarios/scenario-to-desktop/docs/concepts/deployment-modes.md`. Choose bundled for a verified local dependency plan; choose proxy for an explicitly configured external server. Resolve missing mode requirements before building. **[S0]** |
| Need to inspect templates | Run `scenario-to-desktop templates`. **[S1]** |
| Need a build with an agreed mode and target platforms | Run `scenario-to-desktop pipeline run` for the selected scenario with explicit `--platforms` and `--deployment-mode`. Preserve the returned pipeline ID. **[S1]** |
| Need to diagnose or inspect a pipeline | Run `scenario-to-desktop.pipeline-inspect`. Supply the exact pipeline ID when continuing work; omit it only when the task is to inspect the newest pipeline. **[S3]** |
| Need to locate retained desktop evidence | Run `scenario-to-desktop.evidence-inventory`. **[S3]** |
| Need the latest journey's assertions after locating its evidence | Run `scenario-to-desktop evidence journey <scenario>`. This command selects the latest journey; it cannot establish the identity of an arbitrary candidate release. **[S1]** |
| Need a validation matrix, target inventory, or profile capability check | Use the scenario's validation UI and `path:scenarios/scenario-to-desktop/docs/guides/interactive-desktop.md`. Governed matrix CLI bindings are not yet available; do not invent program calls. **[S0]** |
| Need signing-tool availability | Run `scenario-to-desktop signing prerequisites`. **[S1]** |
| Need to validate an existing signing configuration | Run `scenario-to-desktop signing validate <scenario>`. **[S1]** |
| Need to configure signing | Run `scenario-to-desktop signing set` with the selected configuration per its help. Resolve credentials through the credential authority. **[S1]** |
| Need an existing installer locally | Run `scenario-to-desktop download <scenario> <platform>`. **[S1]** |
| Need to ingest a supplied telemetry file | Run `scenario-to-desktop telemetry ingest` with the scenario and file selectors from its help. **[S1]** |
| Need a telemetry summary | Run `scenario-to-desktop telemetry summary <scenario>`. **[S1]** |
| Need to approve, publish, observe, or recover a release | Read `prompt-manager skill read deployment-manager`. Carry the exact artifact, target, channel, and review identity into its workflow. **[S0]** |

Invoke an S3 leaf with `program-runtime library run <program> --input key=value`.
Its sibling JSON contract owns inputs and output vocabulary. Branch on `status`
and `errors[0].class`. `ok` means the inspection completed; read the owner's
pipeline status or capture metadata before deciding the next operation.

These programs compose recurring diagnosis and evidence reads. Single operations
remain CLI leaves because the CLI already owns their execution and next steps.
Do not wrap each command in a new program.

### In-use settings

| Symptom | Allowed setting move | Journal |
|---|---|---|
| The default platform set includes an unavailable target | Select the requested platforms with `pipeline run --platforms`; retain omitted required platforms as evidence gaps | Selected platforms and unresolved obligations |
| Need to continue a specific build | Supply `pipeline_id` to `pipeline-inspect`; never substitute the newest pipeline | Requested and returned pipeline IDs |
| Human output omits identifiers needed for evidence linkage | Use that command's `--json` format and verify the expected fields are present | Why structured output was necessary and which references were used |
| Build returns before completion | Preserve its ID and use the pipeline UI for continued observation; `pipeline-inspect` is one snapshot, not a polling loop | ID and observed state; current CLI has no wait flag |

### Verification and authority

Use the canonical evidence contract for the complete platform/profile matrix.
A capture count, checksum field, package build, or successful window launch is
not proof of native behavior, communication, offline support, or release approval.
Require evidence tied to the selected artifact, target, journey, and environment.
Keep unavailable and unsupported cells visible with their missing capability.

Manage host state through `vrooli`; manage dependencies through Scenario Dependency
Analyzer. Use routed test storage for mutating journeys. Never start scenario
binaries directly, expose renderer control remotely, or log credential values.

Use publication or recovery only within the task's existing authority and the
owning release gate. Do not delete build output merely to obtain a fresh-looking
result. Select `--clean` only when replacing that output is part of the task.

### After acting, always

Use `vrooli-memory learning record --scope scenario-to-desktop-usage --attempt '<Attempt JSON>'`
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
| `unavailable` / `scenario_unreachable` | `vrooli scenario status scenario-to-desktop` | Use lifecycle remediation within the task's authority; keep the read unknown |
| `failed` / `invalid_input` | Selected program contract | Correct the selector; no work has started |
| `failed` / `pipeline_not_found` | `scenario-to-desktop pipeline list` | Select an existing pipeline or start an authorized build; inspection never creates one |
| `failed` / `identity_mismatch` | Requested scenario and pipeline/capture IDs | Stop using the result; report the owner defect with references |
| `partial` / `snapshot_changed` | Capture activity in the UI | Preserve the inconsistent read; obtain a later snapshot after activity stops |
| `partial` after one read fails | `errors[].where` and class | Use only the returned evidence; record the missing read without estimating |
| `refused` | Exact runtime refusal | Follow the session grant path for already authorized work; never bypass run eligibility |
| `failed` / `invalid_response`, `binding_error`, or `kernel_runtime` | Program ID and binding descriptor | Route the binding or program defect to the owning improvement work |
| Pipeline owner status is failed but program status is `ok` | `scenario-to-desktop pipeline status <pipeline-id>` | Diagnose the named stage; the program only succeeded at reading it |
| Evidence exists only for another build or an emulator | Matrix artifact and target identities | Retain its actual scope; obtain native evidence before a native support claim |
| Signing or trust authority is absent | `signing prerequisites` and Deployment Manager review | Preserve the release refusal; do not convert development-local evidence into production approval |
| A program needs matrix orchestration or a server-owned pipeline wait | Existing binding work in `docs/internal/PROGRESS.md` | Reuse the obligation; add the typed owner operation before authoring orchestration |

Promote repeated compatibility, recovery, or evidence-policy workarounds to the
owning scenario. Then remove the superseded program logic and skill prose.
