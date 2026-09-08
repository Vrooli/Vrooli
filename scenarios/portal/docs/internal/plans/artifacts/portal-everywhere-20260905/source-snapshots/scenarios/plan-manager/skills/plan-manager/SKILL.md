---
name: "plan-manager"
description: "Choose and resume planning operations, consume validation receipts, coordinate reviewed families, and capture outcome-linked usage evidence."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["plans", "execution", "learning"]
  status: "active"
  revision: 1
  createdAt: "2026-09-05T00:00:00Z"
  updatedAt: "2026-09-05T00:00:00Z"
  learning:
    scope: "plan-manager-usage"
    capture: "every attempt"
  requires:
    scenarios: ["plan-manager", "vrooli-memory", "test-genie", "agent-manager"]
    commands: ["plan-manager", "vrooli-memory", "prompt-manager skill read"]
  origin: {kind: "authored"}
---
## Tools focus: Plan Manager

Plan Manager owns plan state, scope, family claims and the reviewed frontier.
Test Genie owns validation admission, work and receipts. Agent Manager owns child
execution and supervision. Use this entry point for ordinary operation; use
`plan-manager-improve` when the task is to improve the capability.
Read `prompt-manager skill read vrooli-memory` once for the learning contract.

### Before acting

Recall the operation and comparable context in `plan-manager-usage`. Record the
advice IDs applied or rejected and the decision each changed; record `no_match`
or `unavailable` when appropriate. Advice cannot override current typed state.
Retain a task ID across retries, one attempt ID and ordinal per actual attempt,
and observed request/start timestamps before orientation. The context names
operation mode and relevant tool/policy versions; put plan/run IDs in evidence.

### Choose one operation

Read the rows in order; take the first applicable next step.

| Situation | Next step |
|---|---|
| Operation or flags are unknown | Read `plan-manager <group> help`. **[S1]** |
| Need a new plan or a material plan revision | Read `implementation-plan-authoring`; preserve its authoring judgment and use the owner wizard. **[S0]** |
| Need to resume an existing plan/execution | Run `plan-manager exec continue <plan-or-execution>`. Read its current action before issuing another operation. **[S1]** |
| Need to inspect current execution without advancing | Run `plan-manager exec context <execution>`. **[S1]** |
| The predicted change boundary or implementation is wrong | Read `implementation-plan-execution`; apply its divergence decision before changing scope. **[S0]** |
| Current action calls for validation | Run `plan-manager validate start` with the owner-documented subject and policy. Retain the returned operation/receipt identity. **[S1]** |
| Validation work is pending | Use the receipt owner's documented wait once; cancellation of a wait does not abort work. **[S1]** |
| Need to consume completed validation | Run `plan-manager validate sync` for the existing operation. An unavailable or insufficient receipt is not a pass. **[S1]** |
| Need to split, review, or schedule related plans | Read `plan-family-orchestration`. **[S0]** |
| A reviewed frontier needs child execution/supervision | Read `agent-manager-plan-family-supervision`. Supply the typed family and child identities. **[S0]** |
| Need to determine completion | Run `plan-manager exec complete <execution>`; honor its evidence verdict. **[S1]** |

A successful read proves only that the read succeeded. The attempt outcome refers
to the selected operation; it must not imply the whole plan is complete.
Source freshness follows relevant content and declared inputs. Commit movement
alone does not require recapture. Missing behavioral prior is handled by the
explicit validation policy; it is not an instruction to start certification.

### In-use settings and recovery

| Evidence | Action |
|---|---|
| Revision conflict | Re-read owner state and reconsider; do not replay a stale mutation. |
| Validation receipt unavailable | Retain its identity and reason; follow the owner action. |
| Relevant scope expands | Record the boundary through Plan Manager so validation follows. |
| Graph review no longer matches | Recompute and review the exact new revision before launch. |
| A repeated recovery sequence is necessary | Capture the sequence and outcome; route stable composition to a program and missing invariants to the scenario. |

### After acting, always

Use `vrooli-memory learning record --scope plan-manager-usage --attempt '<Attempt JSON>'`
once per actual attempt. The shared Memory skill and
`packages/proto/schemas/vrooli-memory/v1/learning/learning.proto` own its shape.
Include task/attempt IDs, ordinal, timestamps, operation, context, evidence,
recall status and advice decisions. Capture observed `firstActionAt` and
`toolRoundTrips`; omit unobserved values. Preserve failures with a stable
fingerprint, unavailable outcomes, and unresolved attempts. Mark fixtures `test`.
Only evidenced operation outcomes are `verified_success`. Advice verdicts remain
unknown until assessed; task success alone does not establish advice benefit.
On capture failure retain the payload and retry with its unchanged ID/body.
Do not also write an ordinary journal task-record. Code changes use a separate
work-record. Never copy transcript bodies, credentials or private content.

### Troubleshooting & Edge Cases

- A plan says draft but has execution evidence: inspect execution state and last
  activity. A declaration label does not prove abandonment or completion.
- Baseline recovery conflicts with the receipt policy: preserve the contradiction
  as an owner finding; do not invent a second lifecycle or weaken the required gate.
- A required producer is unavailable: preserve unknown evidence and continue only
  the work the current policy permits.
- No measured learning gain: use `plan-manager-improve` with comparable windows.
