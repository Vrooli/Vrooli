---
name: "readiness-enforcement-paths-gate"
description: "Evaluate the deployment readiness check: Enforcement paths actually gate"
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
# Enforcement paths actually gate

Read `docs/scenario-qa/methods/readiness/enforcement-paths-gate.md` and the checklist entry before evaluating.

Return one signal for checklist item `enforcement-paths-gate` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.