---
name: "device-control"
description: "Resolve logical devices, execute typed semantic volume operations, reuse exact saved flows, and learn from measured task outcomes."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["automation", "workflow", "learning"]
  icon: "play"
  status: "active"
  revision: 55
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-06T00:00:00Z"
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
| Need one device task with identity resolution and outcome capture | Run `device-control.do-task` with device, context_key, actor, and a selected flow/revision or candidate. **[S4]** |
| Need a volume change on a device | Run `program-runtime library run device-control.volume --input device=<logical-id-or-unique-name> --input actor=<actor> --input goal='turn down by 50 percent' --input confirm=true`. The service resolves the selected device's declared volume operations, owns the lease, one recovery retry, and honest verification. `confirm=true` is required for this governed write and is never sent to the device. **[S3]** |
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

For volume, `by 50 percent` means a relative reduction
(`fraction_of_current=-0.5`), while `set to 50%` means an absolute normalized
setpoint (`0.5`). Android TV Remote accepts relative volume keys, not numeric
absolute values. When its live state identifies an ARC/eARC amplifier route, the
planner uses a writable Cast `MASTER` setter and verifies the resulting Remote
readback; directional Remote keys are retained as the plain-speaker fallback.
Same-host transport candidates are diagnostic evidence and never a durable merge
without owner confirmation. A confirmed merge retains both transport profiles
across restart and makes the route reusable.

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

### Learning and program use

Use `device-control-usage` for comparable task evidence. Keep target, profile, and
relevant version contexts distinct; remembered selectors and endpoints require
current verification.

The declared `device-control.volume`, do-task, author-flow, and replay-flow programs use automatic learning.
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
