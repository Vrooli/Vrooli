---
name: "readiness-signal"
description: "Evaluate one deployment readiness check and return its signal: passed, failed, unavailable or unknown, with attributable evidence."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["readiness-check", "deployment-manager"]
  status: "active"
  revision: 1
  origin:
    kind: "authored"
---
# Readiness signal

The remediation topic names one checklist item id. Read its entry in `scenarios/deployment-manager/api/readiness/readiness-policy.json` (acceptance, producer, freshness) and its method doc `docs/scenario-qa/methods/readiness/<item-id>.md` before evaluating.

Return one signal for that item:

| Field | Value |
|---|---|
| status | `passed`, `failed`, `unavailable` or `unknown` |
| source | the producer or command that observed it |
| run id | when one exists |
| observed at | timestamp of the observation |
| commit | the evaluated commit |
| detail | concise evidence |

Never report `passed` without attributable evidence. A producer that did not answer is `unavailable`, never `passed` or a quiet skip (R6, R2).
