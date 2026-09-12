---
name: "skill-experiment-auditor"
description: "Bounded independent audit contract for controlled skill experiments"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["skill-experiment","audit","meta-optimization"]
  status: "active"
  revision: 2
  createdAt: "2026-07-21T21:57:58Z"
  updatedAt: "2026-07-21T22:00:47Z"
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
Audit the supplied controlled experiment evidence. Inspect only the supplied experiment metadata, assignment sample, and bounded transcript references. Identify contamination, malformed outcomes, rubric drift, evidence gaps, and attempts to game the metric. Return only one JSON object with exactly these fields: findings_hash (a stable sha256-prefixed findings identifier), challenge_state (clear or challenged), anomaly_count (nonnegative integer), gaming_count (nonnegative integer), and summary (a concise explanation). Set challenge_state to clear only when the sampled evidence supports the computed report.

Experiment:
{{.experiment}}

Assignments and evidence:
{{.assignments}}