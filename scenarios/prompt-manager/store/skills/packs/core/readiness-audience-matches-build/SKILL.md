---
name: "readiness-audience-matches-build"
description: "Evaluate the deployment readiness check: Declared audience matches the build"
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
# Declared audience matches the build

Read `docs/scenario-qa/methods/readiness/audience-matches-build.md` and the checklist entry before evaluating.

Return one signal for checklist item `audience-matches-build` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.