# Temporal Flows

This file indexes time-dependent behavior in Test Genie. The linked workflow
owners and tests define behavior; this index records the flow boundary and
remaining confidence gaps.

## Flow Index

| Flow ID | Domain | Risk | Model status | Source of truth | Tests | Remaining gaps |
|---|---|---|---|---|---|---|
| `test-genie.execution-stream` | UI execution stream | Medium: a server stream can end before a run result arrives, or an obsolete request can finish late. | The hook owns named `idle`, `streaming`, `completed`, and `error` states. A single failure transition records the error and aborts the request. Completion is separate from the test verdict. | `path:scenarios/test-genie/ui/src/hooks/useExecutionStream.ts` | `path:scenarios/test-genie/ui/src/hooks/useExecutionStream.test.tsx` covers success, server/request errors, premature EOF, reset, replacement, terminal event order, and callback failure. | No declarative transition matrix; add one if the event/state surface expands. |

## Unmodeled Candidates

| Candidate | Why it may be temporal | Current risk | Recommended next step |
|---|---|---|---|
| None identified in the execution stream hook after tracing its terminal paths and cancellation tests. | The hook has one owner and explicit terminal states; no additional lifecycle was found outside it. | Low | Revisit when a new stream event or cancellation mode is added. |

## Declarative & Formal Spec Status

No declarative or formal spec is adopted for this UI stream. The local status type,
transition owner, and behavior tests are the current source of confidence.
