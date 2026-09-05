---
name: "knowledge-observatory-improve"
description: "Improve agent knowledge-task outcomes using retrieval evidence, documentation health, and comparable memory cohorts; route recurring friction into skills, programs, or owning scenarios."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["documentation", "knowledge", "self-improvement"]
  icon: "gauge"
  status: "active"
  revision: 1
  createdAt: "2026-09-05T00:00:00Z"
  updatedAt: "2026-09-05T00:00:00Z"
  requires:
    scenarios: ["knowledge-observatory", "program-runtime", "search-hub", "vrooli-memory", "agent-manager"]
    commands: ["program-runtime library run", "program-runtime bindings condition", "search-hub evals", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Practice focus: Knowledge Observatory Improve

Required reading: `prompt-manager skill read knowledge-observatory`,
`prompt-manager skill read improvement-do-and-dont`, and
`prompt-manager skill read scenario-work-ladder`.

### 1. Focus and scope

Regulate source-grounded retrieval and documentation maintenance. Improve outcomes
and effort for comparable real tasks. KO owns its source/index operations and
skills/programs. Other scenarios retain their data and policy authority. Plan
Manager owns artifact placement/closeout; Memory owns learning; Search Hub owns
federation and evaluation records. No git checkout/reset/clean or bulk retirement.

### 2. Setpoint

Run `knowledge-observatory.setpoint-read` with a bounded base_path and, when
available, an immutable eval_run_id. Copy readings/reasons without estimation.

| Row | Sensor | Target | Initial observation |
|---|---|---|---|
| index-availability | knowledge-base status | available = true | Re-read each cycle. |
| reference-health | knowledge-base health, links/refs in base_path | zero findings | Re-read; preserve prior findings when narrowing scope. |
| binding-condition | program-runtime bindings condition | pending comparable baseline | No outcome band inferred from binding count. |
| retrieval-evaluation | search-hub evals show-run | pending comparable baseline | 18/23 met in provider_direct run b6398396-ec2c-4012-8d75-f67fbf377e41, 2026-09-05; one historical baseline, not current state. |
| usage-learning | knowledge-observatory.learning-read | pending comparable baseline | read_elsewhere:knowledge-observatory.learning-read |
| external-friction | agent-manager.friction-digest | pending comparable baseline | read_elsewhere:agent-manager.friction-digest |

Learning rows cover failure recurrence, completion effort, advice outcomes,
first-action latency, agent round trips and workflow reuse. Targets/in_band remain
null until at least two comparable operator windows justify a recorded band.
Program-share, file count, and lint pass rate cannot substitute for task outcomes.

### 3. Sensors

Use the setpoint program and `knowledge-observatory.learning-read` with fixed
from/to, operation, and context_key. Compare the same cohort and selectors in
before/after windows; keep test observations excluded and denominators visible.
Read `program-runtime bindings condition --scenario knowledge-observatory` and
run `agent-manager.friction-digest` with scenario=knowledge-observatory and
window_days=7. External failure evidence outranks self-reported success.

### 4. Golden corpora

The provider-owned suite is `knowledge-observatory.docs.starter` in
`path:scenarios/knowledge-observatory/.vrooli/search.json`. Run with
`search-hub evals run knowledge-observatory.docs.starter --tier provider_direct`;
measure federated separately. Select its immutable run in setpoint-read.
The descriptor's recall_target is 0.8; it is a retrieval target, not an empirically
derived task-success floor. Preserve reviewed expected IDs, negatives, and query
scope. Source-revision/authority/OS tests and program failure fixtures complement
retrieval grading; neither claims end-to-end answer correctness.

Derive any new empirical floor from comparable configurations, corpus revisions,
case counts, and non-degraded runs. A regression against an established comparable
floor takes priority over optimization; never lower it to fit a bad run. A stale
or degraded run and a changed denominator cannot establish improvement.

### 5. Routes

| Evidence | Route | Remeasure |
|---|---|---|
| Source revision/path differs across modes | KO work ladder W3: repair identity or source evidence. | identity regression tests and retrieval cases |
| Applicability/authority is lost in chunks or federation | KO W3 for provider metadata; Search Hub owner if transport discards declared metadata. | metadata contract tests and live provider/federation reads |
| Conflicting or outdated accepted source content | Source owner's reviewed correction; do not decide policy by rank. | representative questions and source comparison |
| Repeated expensive search/read/reference sequence | KO program repair; if a missing invariant causes it, KO scenario primitive first. | same learning cohort effort and outcomes |
| Agent ignores status/scope despite correct evidence | Usage skill judgment repair. | repeated failure fingerprint and advice verdicts |
| Supplemental artifacts leak into docs | Plan Manager owner: artifact placement and closeout guidance/operations. | owner evidence of corrected placement |
| Binding unavailable | Distinguish missing contract (W1) from transient runtime (W3 after three cycles). | binding-condition |
| All comparable outcome rows in band | Keep collecting; remove obsolete workarounds after lower-layer fixes ship. | next comparable window |

### 6. Anti-gaming

Apply improvement-do-and-dont D1 (weakened tests), D2 (deleted issue ledger), D3
(suppressed findings), and its skeptic test. Do not declare evidence accepted from
advice popularity, remove failing corpus cases, widen search expectations, bury
known conflicts, count synthetic attempts as operator work, or equate search hits
with verified task success. Lower layers should simplify the layers above them.

### 7. Evidence

One cycle work-record via `vrooli-memory journal note --kind work-record` names
trigger, changed layer/owner, before/after sensor readings, immutable run IDs,
comparison selectors, limits, and outcome. Usage attempts remain in the existing
learning projection, not a second journal. File out-of-owner defects through
report-bug. After three consecutive unreachable cycles, record the owner failure
in the existing problems document.

### 8. Stop rules

Missing bindings route W1 immediately; missing telemetry routes to its owner.
Transient unreachable sensors remain unknown; wait one cycle without inventing
readings. read_elsewhere rows route to that program. Unreliable or over-budget
readings remain unbanded. Refuse to lower a floor for a smaller/easier denominator.
A required grant goes through the session grant path. Two comparable cycles in
band justify proposing closeout, not autonomously closing another owner's work.
Cycle wall-clock ceiling: 30 minutes; unresolved findings remain explicit.
