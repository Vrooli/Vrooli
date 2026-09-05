---
name: "device-control"
description: "Resolve authorized devices, reuse exact saved flows, validate and persist repairs, and learn from measured task outcomes."
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
    scenarios: ["device-control", "program-runtime", "vrooli-memory"]
    commands: ["device-control", "program-runtime", "vrooli-memory"]
  learning:
    scope: "device-control-usage"
    capture: "every attempt"
  origin:
    kind: "authored"
---
## Tools focus: Device Control

Control owner-authorized devices through device-control. Resolve durable device
identity before acting; inventory order, a remembered IP, or a matching model
alone does not identify the user's target. This skill covers televisions,
phones, and other declared strategies. Repairing the capability belongs to
`device-control-improve`. Role/rung canon:
`path:docs/agent-system/SKILL_AUTHORING.md`.
Read `path:scenarios/device-control/docs/reference/capabilities.md` when the
selected task needs an unfamiliar modality.

### Choose one operation

Read rows in order; the first matching row is the next step.

| Situation | One step |
|---|---|
| Operation or inputs are unknown | Read `device-control <group> help`. **[S1]** |
| Device is not paired/onboarded | Run `device-control device connect` for its kind and follow the first unavailable rung. **[S1]** |
| Need the intended device and reusable task flows | Run `device-control.prepare-task` with exact device ID or unique name and context_key. **[S3]** |
| Multiple devices match | Ask the operator to identify the target; never select the first row. **[S0]** |
| Have an authorized saved flow and exact revision | Run `device-control.replay-flow` with device_id, context_key, flow_id, version and actor. **[S3]** |
| Need to inspect a candidate before reuse | Run `device-control flow get` with id and version. Verify its goal, device, context, preconditions and assertions. **[S1]** |
| Have a new typed flow with an outcome assertion | Run `device-control.author-flow` with flow, device_id, context_key and actor. **[S3]** |
| Have a repair candidate for a failed saved flow | Run `device-control.author-flow` with the same inputs plus flow_id and expected_version. **[S3]** |
| No reusable flow exists and bounded AI control is appropriate | Run `device-control agent start` with the requested goal and exact device, using current help. **[S1]** |
| Successful agent exploration should become reusable | Run `device-control agent promote <id>` to obtain the replay candidate; then export, add an explicit outcome assertion, and validate it through author-flow on the next step. **[S1]** |
| Need the promoted candidate definition | Run `device-control flow export <run-id>`. Export alone does not save or validate a reusable library revision. **[S1]** |
| Need a live state check | Run `device-control device state <id>`. **[S1]** |
| Saved wireless endpoint is stale | Run `device-control device reconnect <id>`; it verifies hardware identity. **[S1]** |
| Need to stop a leased operation | Run `device-control session kill <id>`. **[S1]** |

Select capabilities per device/transport. Android TV Remote can supply directional
and media input without a frame; never retry screenshot requests on a screenless
transport. Use typed state where available. Use semantic targeting first, visual
anchors next, and vision only when required and available.
A fast command acknowledgement does not prove the requested screen or media state.

A context_key names the task, app/profile and relevant versions. Reuse that key
across attempts under comparable conditions; change it when assumptions change.
The library retains immutable revisions across service restarts. Repairs preserve
assertions, authentication, redaction and transport policy and reject stale versions.
A terminal outcome assertion is required for promotion. Retain unsupported or
unavailable observation as an explicit verification gap.

### In-use settings

| Symptom | Allowed move | Record |
|---|---|---|
| A saved flow matches the current task/context | Supply its exact ID/revision | Reuse, result, and measured effort |
| A device needs wireless reconnect | Use the owner reconnect command | Verified device identity and result |
| Transport cannot observe frames | Use its typed state capabilities or select an authorized frame-bearing transport | Actual evidence scope |
| Device is locked | Use declared authentication profiles and the auth unlock operation | Profile reference and verified unlock outcome |
| AI loop reaches a bound | Retain its run and choose a narrower next attempt | Failure fingerprint; no silent budget increase |

For property-capable adapters, use a terminal `property-assert` step with
`arguments: {"name":"volume","equals":20}` for an exact observed state check.
It reads through the device adapter and needs no screenshot; an unsupported
property remains a capability gap. Choose the expected value from the task.

### Verification and authority

Programs call the scenario's lease-owned execution. Never drive raw ADB or remote
control protocols, manufacture a lease, expose renderer control, or place secrets
in flows or memory. Credentials remain owned by the credential authority.
A promoted agent candidate still needs replay validation and an outcome assertion
before it enters the durable library. Never label a dry-run as successful actuation.
For physical validation, use an explicitly requested device task; general code
validation uses fakes and replay fixtures without sending commands to a household TV.

### Before acting

Read `prompt-manager skill read vrooli-memory` and
`prompt-manager skill read program-runtime` once when their contracts are unknown.
Recall the task and exact target from device-control-usage. Record applied or rejected advice
IDs and the decision each changed; retrieval alone is not advice use.
Keep device/site/profile/tool-version contexts distinct. A remembered endpoint
or selector is a hint to verify, never current authority.

Retain the user-request timestamp, task ID, attempt ID, ordinal, and attempt
start before orientation. Measure time to the first useful action when observed.
Count outer-agent tool round trips and visual reasoning calls only when observable;
omit unknown values rather than estimating zero. Keep one task ID across retries.

### Program invocation and results

Run an existing program with
`program-runtime library run PROGRAM --input key=value`.
For structured values, quote the complete input argument, for example
`--input 'project_id=UUID,flow={"nodes":[],"edges":[]}'` (replace the empty
definition with the actual candidate). Never build a scratch session for a registered program. The sibling JSON contract owns all inputs.
Read `status`, then `errors[0].class`, then owner outcome and evidence IDs.
A successful read is not successful execution. A run that is still active is
`unknown` until its owner returns a terminal result. No speculative retries.

### After acting, always

Write one `vrooli-memory learning record --scope device-control-usage --attempt '<Attempt JSON>'`
after the selected operation ends. The shared Memory skill and
`path:packages/proto/schemas/vrooli-memory/v1/learning/learning.proto` own the
record shape. Structured JSON is necessary for evidence linkage and comparison
fields; do not also append a journal task-record.

Include task/attempt identities and timestamps, exact comparison context,
operation, outcome evidence, applied/rejected advice, and caller provenance.
When observed, include `firstActionAt`, `toolRoundTrips`,
`visualReasoningCalls`, and `reusedWorkflow`.
Only use `verified_success` for the requested outcome established by owner
evidence. Keep failed, unavailable and unknown separate; failed attempts need a
stable failure fingerprint. A failed repair does not disappear when its retry
passes. Mark fixtures `test`; changing the label cannot establish operator benefit.

On capture transport failure, retain the payload and retry with the same ID and
unchanged body. Report capture unavailable without changing the task outcome.
Use separate binding-note/work-record entries for callable defects or code work.
Follow shared curation for confirmed advice and supersede contradicted guidance.
Never record screen bytes, credentials, private URLs, or log bodies.

### Troubleshooting & Edge Cases

| Result | Next action |
|---|---|
| device_selection_required | Select an exact ID or resolve ambiguous naming with the operator |
| capability_gap | Follow the owner's missing capability and next action; no actuation occurred |
| identity_mismatch | Stop using the selected flow; verify device and comparison context |
| flow_failed | Retain run evidence and failed step; create one repair candidate |
| flow version conflict | Re-read current revision before changing the candidate |
| promotion refused | Preserve the refusal; add real outcome proof or deterministic targeting and replay |
| scenario_unreachable | Read lifecycle status; retain the unknown observation |
| no_grant / not_run_eligible | Use the exact runtime grant path under existing task authority |
| No improvement across uses | Read setpoint-read and compare equivalent learning windows |

Recurring discovery, selection and execution joins belong in programs.
Device identity, leases, replay validation and recovery guarantees remain in the
scenario. Simplify the faster layers after the owner absorbs a workaround.
