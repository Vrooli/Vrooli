---
name: "vrooli-onboarding-improve"
description: "Regulate onboarding completion, readiness, degradation, handoff, drop-off, and confusing-step evidence without folding visual redesign into automation."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["onboarding","improve","drop-off","readiness","handoff"]
  icon: "gauge"
  status: "active"
  revision: 1
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["vrooli-onboarding", "agent-manager"]
    commands: ["vrooli-onboarding readiness", "agent-manager"]
  origin: {kind: "authored"}
---
## Practice focus: Vrooli Onboarding Improve

Regulate onboarding from typed step, readiness, degradation, and handoff
evidence. Separate workflow correctness from visual redesign.

Required reading: `prompt-manager skill read improvement-do-and-dont`,
`prompt-manager skill read scenario-work-ladder`, and
`prompt-manager skill read vrooli-onboarding`.

### Scope and setpoint

In scope: step drop-off, confusing-step reports, apply/readiness disagreement,
degraded acknowledgement, completion truth, and remote handoff. Out of scope:
visual redesign or host repair inside Onboarding.

| Row | Desired band | Route |
|---|---|---|
| completion-truth | 100% of completion claims have ready or acknowledged-optional readiness | Fix the completion invariant. |
| blocker-remediation | Every blocker names one owning remediation | Fix the readiness contract or control-plane operation. |
| step-drop-off | Every abandoned session names its last durable step | Fix evidence capture before interpretation. |
| confusing-step | Zero recurring fingerprints for one step | Tighten the usage skill or typed response. |
| handoff-truth | Every remote outcome carries remote step and readiness evidence | Fix the handoff contract. |

Read bounded onboarding evidence and Agent Manager friction. Apply one change
to the highest safety or completion gap and repeat the same reading. The
evidence digest/setpoint programs remain deferred until Onboarding publishes
bounded typed bindings. Capture visual findings for a separate UX plan.

### Output expectations

Report each row as `in_band`, `out_of_band`, or `unavailable`. Name the evidence
locator, owner, actuator, before/after reading, and any separate UX finding.

### Troubleshooting & Edge Cases

- No bindings: file the obligation; do not scrape CLI output in a program.
- Sparse sessions: enforce completion truth without claiming a drop-off trend.
