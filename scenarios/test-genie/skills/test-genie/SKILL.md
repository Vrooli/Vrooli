---
name: "test-genie"
description: "Operate Test Genie through durable validation intents and receipts: create or attach once, read producer-owned evidence, wait once, and preserve unavailable as unknown."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["test-genie","validation","intent","receipt","wait"]
  icon: "check-circle"
  status: "active"
  revision: 1
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["test-genie"]
    commands: ["vrooli scenario test", "test-genie runs wait", "test-genie validation"]
  origin: {kind: "authored"}
---
## Tools focus: Test Genie Validation Intent

Use Test Genie as the owner of validation admission, execution, evidence, and
durable receipts. Submit one intent or scenario test, attach to its returned
handle, and make decisions from the producer-owned terminal evidence.

Required reading: `path:docs/TESTING.md` and
`path:scenarios/test-genie/docs/configuration/tunable-levers.md`.

### Scope

In scope: selecting intent, creating or attaching to work, reading admission
and receipt explanations, waiting once, aborting explicitly, and citing durable
evidence. Out of scope: client polling, private retry loops, reconstructing a
receipt, treating cancel as abort, or editing producer state.

### Decision table

| State | Action |
|---|---|
| An ordinary change needs validation | Follow `path:docs/TESTING.md` scope policy. Run `test-genie.iterate` with scenario, request_id and relevant phases; retain the receipt and printed wait command. Omitted phases select the owner's history-informed quick profile. |
| A scenario suite must run directly | Run `vrooli scenario test <scenario> --phases <relevant-phases>` once; retain its run id. |
| A typed cross-owner validation intent exists | Run `test-genie validation create --intent-file <file>`; accept attach/dedup. |
| Work is nonterminal | Run the documented wait command once. Do not poll. |
| Admission is retryable | Read `validation explain`; honor its retry hint without resubmitting in a loop. |
| Receipt is unavailable | Report unknown with the failed producer read. Do not infer pass or fail. |
| Operator requests termination | Use the explicit abort operation; cancelling the client only detaches. |

### Output expectations

Handoff names the run or receipt id, revision, terminal disposition, producer
evidence locators, and any typed degradation. Test the desired behavior, not a
current defect. Use `run test-genie.validation-digest` only for a bounded fleet
summary; inspect the named receipt before acting on one failure.

### Troubleshooting & Edge Cases

- `404 unimplemented`: the running Test Genie is stale; restart it through the
  scenario lifecycle. Do not fall back to an undocumented endpoint.
- Saturated admission: retain the first intent and follow the explanation.
- Lost terminal output: read the durable receipt or run by id.
