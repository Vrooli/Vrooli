---
name: "readiness-non-subscriber-gate-honest"
description: "Evaluate the deployment readiness check: The gate is honest to a paying customer"
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
# The gate is honest to a paying customer

Read `docs/scenario-qa/methods/readiness/non-subscriber-gate-honest.md` and the checklist entry before evaluating.

Return one signal for checklist item `non-subscriber-gate-honest` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.