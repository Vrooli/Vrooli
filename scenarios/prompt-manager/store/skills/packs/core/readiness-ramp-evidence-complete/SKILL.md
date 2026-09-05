---
name: "readiness-ramp-evidence-complete"
description: "Evaluate the deployment readiness check: Ramp evidence exists for every target"
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
# Ramp evidence exists for every target

Read `docs/scenario-qa/methods/readiness/ramp-evidence-complete.md` and the checklist entry before evaluating.

Return one signal for checklist item `ramp-evidence-complete` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.