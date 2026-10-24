---
name: "readiness-suite-state-known"
description: "Evaluate the deployment readiness check: Suite state is known"
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
# Suite state is known

Read `docs/scenario-qa/methods/readiness/suite-state-known.md` and the checklist entry before evaluating.

Return one signal for checklist item `suite-state-known` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.