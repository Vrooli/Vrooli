---
name: "vrooli-onboarding"
description: "Operate Vrooli Onboarding through capability selection, control-plane-owned apply, composed readiness, explicit degradation, and typed handoff."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["onboarding","setup","readiness","degradation","handoff"]
  icon: "rocket"
  status: "active"
  revision: 1
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["vrooli-onboarding"]
    commands: ["vrooli-onboarding readiness", "vrooli-onboarding operator"]
  origin: {kind: "authored"}
---
## Tools focus: Vrooli Onboarding

Use Vrooli Onboarding to record capability intent, delegate host changes to
the control plane, read composed readiness, acknowledge only the exact optional
degraded set, and produce a typed local or remote handoff.

Required reading: `path:scenarios/vrooli-onboarding/docs/WIZARD_FLOW.md`.

### Scope

In scope: selection, closure review, apply, credentials, readiness, degraded
acknowledgement, completion, and remote handoff. Out of scope: private host
repair, bypassing required blockers, visual redesign, or treating a completion
marker as readiness.

| Readiness | Action |
|---|---|
| `ready` | Complete and retain the readiness evidence. |
| `missing` | Apply the named remediation through the owning control-plane operation. |
| `degraded`, required blocker present | Do not complete. Resolve the blocker. |
| `degraded`, optional set only | Show the set; acknowledge its exact digest only with operator intent. |
| `unsupported` or catalog unavailable | Preserve the typed degradation and supported alternatives. |

### Output expectations

Handoff includes selection, closure, target, readiness status, blockers,
degraded digest/acknowledgement, and the next command. No evidence program is
declared because Onboarding currently exposes no Program Runtime bindings.

### Troubleshooting & Edge Cases

- Marker/readiness conflict: readiness wins; report configuration pending.
- Remote failure: retain the remote step and readiness evidence; do not claim local success.
- Host remediation request: route to the control plane, never add it here.
