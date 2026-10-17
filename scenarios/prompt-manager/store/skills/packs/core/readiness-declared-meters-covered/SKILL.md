---
name: "readiness-declared-meters-covered"
description: "Evaluate the deployment readiness check: Declared meters have limits and enforcement"
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
# Declared meters have limits and enforcement

Read `docs/scenario-qa/methods/readiness/declared-meters-covered.md` and the checklist entry before evaluating.

Return one signal for checklist item `declared-meters-covered` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.