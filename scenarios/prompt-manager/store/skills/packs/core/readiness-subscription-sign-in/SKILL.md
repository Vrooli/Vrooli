---
name: "readiness-subscription-sign-in"
description: "Evaluate the deployment readiness check: Subscription sign-in works end to end"
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
# Subscription sign-in works end to end

Read `docs/scenario-qa/methods/readiness/subscription-sign-in.md` and the checklist entry before evaluating.

Return one signal for checklist item `subscription-sign-in` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.