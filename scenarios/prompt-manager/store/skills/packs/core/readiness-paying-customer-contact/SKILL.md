---
name: "readiness-paying-customer-contact"
description: "Evaluate the deployment readiness check: A paying customer has a contact path"
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
# A paying customer has a contact path

Read `docs/scenario-qa/methods/readiness/paying-customer-contact.md` and the checklist entry before evaluating.

Return one signal for checklist item `paying-customer-contact` with status `passed`, `failed`, `unavailable`, or `unknown`. Include source, run identifier when available, observed timestamp, evaluated commit, and concise evidence detail. Never report passed without attributable evidence.