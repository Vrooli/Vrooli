---
name: "readiness-declared-features-reachable"
description: "Evaluate the deployment readiness check: Declared features are reachable"
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
# Declared features are reachable

Read `docs/scenario-qa/methods/readiness/declared-features-reachable.md` and the checklist entry before evaluating.

Return one signal for checklist item `declared-features-reachable` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.