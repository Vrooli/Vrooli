---
name: "readiness-payment-stop-survivable"
description: "Evaluate the deployment readiness check: Stopping payment is survivable"
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
# Stopping payment is survivable

Read `docs/scenario-qa/methods/readiness/payment-stop-survivable.md` and the checklist entry before evaluating.

Return one signal for checklist item `payment-stop-survivable` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.