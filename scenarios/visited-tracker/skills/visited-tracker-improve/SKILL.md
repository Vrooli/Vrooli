---
name: "visited-tracker-improve"
description: "Regulate Visited Tracker claims, attention ranking, storage, skills and programs using measured evidence and explicit telemetry gaps."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["visited-tracker", "attention", "composition"]
  icon: "route"
  status: "active"
  revision: 3
  createdAt: "2026-09-06T00:00:00Z"
  updatedAt: "2026-09-06T21:00:00Z"
  requires: {"scenarios": ["visited-tracker", "program-runtime", "vrooli-memory", "prompt-manager"], "commands": ["visited-tracker attention preview", "program-runtime library run", "prompt-manager skill read"]}
  origin: {"kind": "authored"}
---
## Practice focus: Improve Visited Tracker

Regulate Visited Tracker's attention and ownership behavior. Keep caller priorities outside this scenario. Read `path:docs/agent-system/TARGET_MODEL.md` §2 and `prompt-manager skill read improvement-do-and-dont`. Follow `scenario-work-ladder` before changing code. Skill roles and program contracts are defined in `path:docs/agent-system/SKILL_AUTHORING.md` and `path:scenarios/program-runtime/docs/guides/program-contracts.md`.

### Setpoint and sensors

Run `visited-tracker.setpoint-read` once per cycle. It reuses Program Runtime's governed binding-condition sensor. The observation on 2026-09-06 reported four dormant bindings and zero degraded bindings; dormant does not establish correctness. Re-read every cycle.

| Row | Sensor | Target | Missing-evidence route |
|---|---|---|---|
| binding-condition | program-runtime bindings condition for visited-tracker | Zero degraded bindings, complete bounded reading | Unavailable or empty/truncated readings have no band verdict. |
| claim-correctness | Preview integrity schema 1: current active-claim identity and revision ownership | Zero current violations | Missing/unknown check schema or inconsistent counts are unavailable; guarded stale-completion rejection is not a violation. |
| attention-fairness | `visited-tracker/attention/preview` queue-age metadata for an authorized campaign | Oldest eligible age <= 86400 seconds | Omit campaign_id and the row remains unavailable; this observes queue age and does not prove starvation freedom. |
| durable-storage | Preview process-scoped completed write observations | Zero consecutive failures, with a sample no older than 300 seconds and directory-sync support | No samples, stale samples, unsupported durability, and missing telemetry have no band verdict. |

These observations require a purpose-scoped campaign_id; preview never writes to manufacture a storage sample. Integrity covers currently active claims in that campaign, not historical completion correctness. Storage counters cover all campaigns in one API process and reset when its epoch changes. A failed replacement can already have renamed the file before directory sync failed; do not interpret failures as proof that no state changed. A subsequent successful write clears the consecutive-failure reading but retains the failed total within that epoch. Never restart the API merely to clear its metrics. Compare epoch, sample age and counts before drawing a trend; no write history is unknown, not healthy.

### Golden corpora

There is no declared floor-bearing evaluation corpus. `path:scenarios/visited-tracker/api/attention_test.go` supplies regression cases for concurrency, retries, expiry, revision changes, renames, and read-only preview. Run scenario tests through Test Genie. Add a failure case before fixing a newly discovered invariant. Derive a future floor from comparable measurements; do not invent one.

### Curation and routes

| Evidence | Move | Re-read |
|---|---|---|
| A binding is degraded | Locate its owning implementation with scenario-work-ladder W3. | binding-condition |
| Missing binding for a required operation | Route W1 to the owning scenario. | Binding catalog and execution |
| Claim overlap or stale review credit | Preserve the campaign, claim, revision, and rejection evidence; route W3. | Original failing regression |
| Ranking repeatedly hides required work | Route the scheduling invariant and caller exploration policy; preserve the neglected case. | Same queue-age observation once instrumented |
| A program copies claim recovery or hashing | Move the invariant into the API, then simplify the consumer. | Consumer success and failure fixtures |
| Repeated judgment is unclear | Edit the usage skill through skill-set-authoring; keep CLI flags in help. | Skill divergence review |
| Defect owned elsewhere | Use report-bug against that owner. | Owner's evidence |

There is no sanctioned bulk reset, claim-history deletion, or score rebasing curation move. A new campaign partitions future work; it does not erase the previous ledger. Compose other scenarios' programs when their contracts supply the needed evidence. Validate both child success and incomplete/failure paths.

### Anti-gaming

Apply D1, D2, and D3 from improvement-do-and-dont. Do not count visits as completed review. Do not widen a band to the observed failure rate, remove an expired-claim assertion, reset campaign history to improve coverage, or mark missing telemetry healthy. File presence and a successful parent submission do not prove child behavior.

### Evidence and stop rules

Leave one `vrooli-memory journal note --kind work-record` with trigger, approach, before/after readings from the same sensor, and outcome. Preserve program IDs and claim/revision evidence.

Route no_governed_binding and pending_telemetry on the first read. For scenario_unreachable, record unknown and wait one cycle; after three consecutive cycles, record the problem and route W3. Follow read_elsewhere by running its named program. Do not band unreliable readings. Do not lower an evaluation floor without comparable runs. Use the governed grant path when a move requires a grant. After two cycles in band, propose close-out; missing or out-of-scope correctness telemetry still prevents a correctness claim. Limit each cycle to five minutes and one attempted repair per row.

### Troubleshooting & Edge Cases

A clean setpoint board proves only its explicitly measured scopes. Current claim integrity and recent storage write success are not historical correctness certificates. Structural skill-set validation proves declarations; review skill judgment separately. Run program fixtures to test executable behavior. If a contract fixture cannot reach its dependency, retain the failed proof and diagnose reachability through the lifecycle.
