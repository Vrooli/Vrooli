---
name: "readiness-storefront-registered"
description: "Evaluate the deployment readiness check: Storefront application is registered"
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
# Storefront application is registered

Read `docs/scenario-qa/methods/readiness/storefront-registered.md` and the checklist entry before evaluating.

Return one signal for checklist item `storefront-registered` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.