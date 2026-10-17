---
name: "readiness-bundle-price-present"
description: "Evaluate the deployment readiness check: A current bundle price exists"
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
# A current bundle price exists

Read `docs/scenario-qa/methods/readiness/bundle-price-present.md` and the checklist entry before evaluating.

Return one signal for checklist item `bundle-price-present` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.