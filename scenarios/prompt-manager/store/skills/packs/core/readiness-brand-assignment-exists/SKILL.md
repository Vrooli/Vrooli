---
name: "readiness-brand-assignment-exists"
description: "Evaluate the deployment readiness check: Brand assignment exists"
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
# Brand assignment exists

Read `docs/scenario-qa/methods/readiness/brand-assignment-exists.md` and the checklist entry before evaluating.

Return one signal for checklist item `brand-assignment-exists` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.