---
name: "readiness-storefront-copy-current"
description: "Evaluate the deployment readiness check: Storefront copy describes this version"
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
# Storefront copy describes this version

Read `docs/scenario-qa/methods/readiness/storefront-copy-current.md` and the checklist entry before evaluating.

Return one signal for checklist item `storefront-copy-current` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.