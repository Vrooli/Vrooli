---
name: "readiness-platform-assets-set"
description: "Evaluate the deployment readiness check: Per-platform assets are set"
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
# Per-platform assets are set

Read `docs/scenario-qa/methods/readiness/platform-assets-set.md` and the checklist entry before evaluating.

Return one signal for checklist item `platform-assets-set` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.