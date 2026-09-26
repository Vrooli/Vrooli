---
name: "scenario-to-cloud"
description: "Use Scenario-to-cloud to preview, admit, observe, update, recover, and retire a target-bound deployment through one durable operation path."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["deployment", "vps", "target", "operation", "recovery"]
  icon: "cloud"
  status: "active"
  revision: 1
  createdAt: "2026-09-11T00:00:00Z"
  updatedAt: "2026-09-11T00:00:00Z"
  requires:
    scenarios: ["scenario-to-cloud", "vrooli-memory", "prompt-manager"]
    commands: ["scenario-to-cloud", "vrooli-memory journal note", "prompt-manager skill read"]
  origin:
    kind: "authored"
---

## Tools focus: Scenario-to-cloud

Use Scenario-to-cloud for target-bound deployment lifecycle work. Preview the
material change first, admit one durable operation with a stable request key,
and read the target owner's receipt. The API, CLI, and UI are adapters over the
same operation identity; a caller timeout only detaches the observer.

### 1. Scope

In scope: selecting an enrolled target, compiling a plan, admitting deployment
or update operations, observing target state, and coordinating backup, restore,
credential, and retirement owner actions.

Out of scope: raw SSH execution, copying secrets into argv, treating desired
state as observed state, or declaring success from transport status alone.

### 2. Decision tree

```
I need to change or inspect a deployment
├─ Is the scenario closure, target binding, and authorization complete?
│  ├─ no → follow the typed onboarding handoff and stop before effects [S1]
│  └─ yes
├─ Is the reviewed plan digest current?
│  ├─ no → compile a new plan and review its material diff [S1]
│  └─ yes
├─ Is the request mutating?
│  ├─ no → read target-observed state and its freshness standing [S1]
│  └─ yes → admit one durable operation with an idempotency key [S1]
├─ Did the observer disconnect?
│  ├─ yes → reattach using the operation id; do not replay the effect [S1]
│  └─ no
├─ Is the target outcome known?
│  ├─ no → reconcile through the target owner before another action [S1]
│  └─ yes → accept only the owner receipt whose identity and fence match [S1]
```

Transport choice is explicit. A revoked Bridge binding cannot silently fall back
to SSH. Unsupported platforms, missing credentials, stale enrollment generations,
and incompatible recovery schemas are refusals with a typed next action.

### 3. Verification

Before an effect, verify the reviewed plan digest, candidate identity, target
generation, authorization scope, and recovery posture. After an effect, verify
the target-produced receipt, observed release/configuration identity, health
freshness, and operation standing. For recovery, accept only a target-bound
restore receipt with logical-data invariants and measured RTO/RPO.

### 4. After acting

Journal the operation ID, target generation, plan digest, receipt standing, and
next owner action. Never journal secret values, private key paths, or credential
material.

### 5. In-use settings

| Symptom | Setting move |
|---|---|
| Preview says input is missing | Follow the returned onboarding owner handoff; journal the missing typed input. |
| The request timed out | Reattach to the durable operation and journal the observed standing. |
| The target binding is stale | Re-enroll or refresh the binding through Bridge; do not change transport implicitly. |
| Health disagrees with desired state | Preserve observed facts and route reconciliation to the target owner. |

Promotion note: repeated multi-step operator workflows should become a governed
program or typed Action. Until that owner contract exists, keep this skill as the
judgment layer and use machine-readable plan and receipt output.
