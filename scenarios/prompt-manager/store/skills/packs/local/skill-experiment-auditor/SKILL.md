Audit the supplied controlled experiment evidence. Inspect only the supplied experiment metadata, assignment sample, and bounded transcript references. Identify contamination, malformed outcomes, rubric drift, evidence gaps, and attempts to game the metric. Return only one JSON object with exactly these fields: findings_hash (a stable sha256-prefixed findings identifier), challenge_state (clear or challenged), anomaly_count (nonnegative integer), gaming_count (nonnegative integer), and summary (a concise explanation). Set challenge_state to clear only when the sampled evidence supports the computed report.

Experiment:
{{.experiment}}

Assignments and evidence:
{{.assignments}}