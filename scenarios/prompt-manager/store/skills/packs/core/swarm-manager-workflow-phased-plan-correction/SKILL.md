---
name: "swarm-manager-workflow-phased-plan-correction"
description: "Typed continuation prompt for independent-review correction."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["swarm-manager","agent-manager","workflow","prompt-contract"]
  status: "active"
  revision: 1
  createdAt: "2026-07-18T03:05:26Z"
  updatedAt: "2026-07-18T03:05:26Z"
  modes: ["contract"]
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
# Phased Plan Slice Correction

Correct your latest slice result in this same conversation. Resolve every defect the review note names, then re-verify. You may update progress only for the bound Plan Manager execution after validation passes. Do not mutate plan content or backlog records. Return one complete replacement result under the same outcome work table as the original slice — not a delta, and not a new slice. A correction turn never sets `correctionRequired` again: if defects remain that you cannot fix here, return `blocked` and name them. Do not rely on this conversation for anything beyond this correction.

<review_note>{{.review_note}}</review_note>
