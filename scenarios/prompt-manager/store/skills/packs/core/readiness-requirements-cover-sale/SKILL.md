---
name: "readiness-requirements-cover-sale"
description: "Evaluate the deployment readiness check: Requirements cover what is sold"
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
# Requirements cover what is sold

Read `docs/scenario-qa/methods/readiness/requirements-cover-sale.md` and the checklist entry before evaluating.

Return one signal for checklist item `requirements-cover-sale` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.