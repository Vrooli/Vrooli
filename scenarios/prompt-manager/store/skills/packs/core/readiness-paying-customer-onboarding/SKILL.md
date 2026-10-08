---
name: "readiness-paying-customer-onboarding"
description: "Evaluate the deployment readiness check: Onboarding is usable by a paying customer"
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
# Onboarding is usable by a paying customer

Read `docs/scenario-qa/methods/readiness/paying-customer-onboarding.md` and the checklist entry before evaluating.

Return one signal for checklist item `paying-customer-onboarding` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.