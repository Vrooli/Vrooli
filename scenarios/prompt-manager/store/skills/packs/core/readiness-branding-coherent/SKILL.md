---
name: "readiness-branding-coherent"
description: "Evaluate the deployment readiness check: Branding is applied and coherent"
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
# Branding is applied and coherent

Read `docs/scenario-qa/methods/readiness/branding-coherent.md` and the checklist entry before evaluating.

Return one signal for checklist item `branding-coherent` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.