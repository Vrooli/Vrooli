# Flows

## Purpose Of This Document

Record the lifecycle of an onboarding decision: how it is made, committed,
applied, verified, and revised.

## Flow inventory

| Flow | Trigger | Ends when |
|---|---|---|
| First run | No completion marker | Apply succeeds and validation is ready or acknowledged degraded |
| Re-entry | Completion marker present | The revised decision is committed and re-applied |
| Credential provisioning | An unconfigured declared descriptor | The authority reports the value configured |
| Host consent | A selected manifest declares a tool or safeguard | The operator opts in or declines |
| Remote onboarding | vrooli-bridge reaches a host | That host reports a green readiness report |
| Degraded continue | Optional items are not ready | The acknowledgement is recorded |

## The decision lifecycle

Every decision follows the same path, whichever surface makes it.

```mermaid
sequenceDiagram
  autonumber
  participant S as Surface (UI · CLI · API)
  participant A as Onboarding API
  participant O as internal/operatorstate
  participant F as operator-state.json

  S->>A: read step data
  A->>O: resolve effective values
  O->>F: load
  F-->>O: stored document
  O-->>A: manifest defaults overlaid with operator choices
  A-->>S: step read model

  S->>O: patch (only the changed fields)
  O->>F: load
  O->>O: merge patch into loaded document
  O->>O: validate against operator-state.schema.json
  alt valid
    O->>F: atomic write, secret permissions
    O-->>S: committed
  else invalid
    O-->>S: reject, naming the failing path
    Note over F: unchanged
  end
```

Two properties fall out of this shape:

- A field the writing binary does not model is loaded, merged around, and
  written back unchanged. Trust posture and the core-set authority survive a
  wizard save because nothing in the write path enumerates the document.
- Two surfaces patching disjoint fields at the same time both succeed, because
  the merge happens under the lock rather than in the caller.

## Intent becoming host state

Recording a choice changes nothing on the host. Apply is the transition.

```mermaid
stateDiagram-v2
  [*] --> Recorded: operator commits a choice
  Recorded --> Planned: apply plans items in dependency order
  Planned --> Applying
  Applying --> Applied: every item succeeded
  Applying --> PartiallyApplied: an item failed
  PartiallyApplied --> Planned: operator fixes and re-applies
  Applied --> Verified: validation probes pass
  Applied --> Degraded: optional probes fail, acknowledged
  Verified --> Recorded: operator revises a decision
  Degraded --> Recorded: operator revises a decision
  Verified --> [*]
  Degraded --> [*]
```

`Applied` is not `Verified`. An item can install cleanly and still fail its
probe — a tool present but not on `PATH`, a safeguard written but not loaded, a
resource enabled but not reachable. Keeping the two states distinct is what lets
validation be a real gate rather than an echo of apply.

## Re-entry

Re-entry reads the completion marker and the committed selection, then resumes
at the first unsatisfied step rather than the first step. Adding one scenario
later does not mean re-answering every question, and no committed decision
becomes read-only after the first pass.

Navigation is disposable; decisions are not. Closing the browser mid-flow loses
the step pointer, never a committed choice.

## Remote onboarding

```mermaid
flowchart LR
  Op(["Operator or agent"]) --> B["vrooli-bridge<br/>reaches and holds the connection"]
  B --> R["Remote host"]
  R --> W["vrooli-onboarding<br/>non-interactive surface"]
  W --> RS[("that host's<br/>operator-state.json")]
  W --> RH["that host"]
  W -- "readiness report<br/>+ exit code" --> B
  B --> Op

  classDef store fill:none,stroke-dasharray:4 3
  class RS store
```

Bridge owns reaching the machine; onboarding owns deciding what runs there. The
boundary is the declarative selection document plus a machine-readable exit
code, so automation can branch on a real readiness result instead of parsing
prose.

## Failure handling

| Failure | Behaviour |
|---|---|
| Manifest catalog unavailable on this tier | Typed degraded state naming the missing catalog; other steps continue |
| Credential authority unreachable | Status reports `unsupported` with the backend condition and its fix, not `unconfigured` |
| Host has no graphical session | The encrypted file store is offered as the path forward, not reported as an error |
| Safeguard config fails its schema | Rejected at the write boundary with the failing path; nothing is written |
| One apply item fails | Its dependants are skipped and named; independent items continue; the run is partially applied |
| Required probe fails | Validation blocks and names the remediation |
| Optional probe fails | Degraded continue is offered and the acknowledgement is recorded |

## Deferred flows

Connector authorization and connection binding belong to integration-hub.
Onboarding declares the deferral and simulates nothing, because state nothing
can honour is worse than an empty step.

## Cross-References

- [Architecture](ARCHITECTURE.md) · [Data](DATA.md) · [Domains](DOMAINS.md)
- [Wizard flow](../WIZARD_FLOW.md)
- [Runbook](../operations/RUNBOOK.md)
