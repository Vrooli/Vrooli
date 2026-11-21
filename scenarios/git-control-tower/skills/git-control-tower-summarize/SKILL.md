---
name: "git-control-tower-summarize"
description: "Summarize one immutable Git Control Tower ChangeSubject from bounded evidence without inventing intent or readiness."
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["git-control-tower", "advisory", "summary", "evidence"]
  status: "active"
  revision: 1
  requires:
    scenarios: ["git-control-tower"]
    commands: []
  origin: {kind: "authored"}
---

# Summarize changes

Resolve exactly one `ChangeSubject`, capture its immutable digest, and produce
short behavioral claims linked to evidence references. Keep selected current
changes distinct from staged changes. Never stage, checkout, fetch, execute
hooks, or infer missing intent.

Required output: subject identity, included and omitted coverage, claims with
supporting evidence, unknowns, validation standing, and a freshness warning.

Stop when the source changes during capture, a required object is unavailable,
or a claim has no evidence. Report `partial`, `unavailable`, or `refused`; do
not convert missing evidence to a successful summary.
