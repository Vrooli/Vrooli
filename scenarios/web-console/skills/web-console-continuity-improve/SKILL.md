---
name: "web-console-continuity-improve"
description: "Interpret typed continuity integrity, receipt, reconciliation, and search evidence and route defects to the owning layer."
license: "MIT"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["web-console", "continuity", "improvement", "integrity"]
  status: "active"
---

# Web Console continuity improvement

Read evidence; do not mutate storage directly. Use the typed continuity
commands and bounded reports to classify the defect:

- integrity drift, missing catalog records, search coverage, lifecycle state,
  receipts, or deletion propagation belong to Web Console;
- global ranking, cross-run import, or source privacy mapping belongs to Agent
  Manager;
- provider routing or federated ranking belongs to Search Hub.

For Web Console findings, capture the generation, counts, operation status,
bounded error class, and affected lifecycle state without recording transcript
text, raw queries, secrets, or filesystem paths. A reconciliation apply must
be preceded by a dry-run and backup evidence. Re-run the same operation and
confirm it is a replay with zero additional mutations. Route recurring
multi-command diagnosis to a governed program when one exists; this skill
provides judgment only and never implements a private repair workflow.
For recurring bounded audits, run
`web-console.continuity-integrity-audit` through Program Runtime. Treat
`unavailable` as unknown, and use the returned orphan classification to decide
whether to inspect a dry-run manifest. Applying reconciliation always remains
an explicit operator command with a manifest and idempotency key.
