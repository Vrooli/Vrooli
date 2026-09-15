---
name: "readiness-update-policy-set"
description: "Evaluate the deployment readiness check: Update policy is declared"
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
# Update policy is declared

Read `docs/scenario-qa/methods/readiness/update-policy-set.md` and the checklist entry before evaluating.

Return one signal for checklist item `update-policy-set` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.