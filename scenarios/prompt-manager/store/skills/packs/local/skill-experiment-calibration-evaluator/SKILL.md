---
name: "skill-experiment-calibration-evaluator"
description: "Independent evaluator prompt for measured skill experiments"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["calibration","skill-experiment","evaluator"]
  status: "active"
  revision: 3
  createdAt: "2026-07-21T21:23:51Z"
  updatedAt: "2026-07-21T21:43:41Z"
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
Independent calibration evaluator for the fixed alpha fixture: inspect the treatment result below. Return exactly {"verdict":"pass"} only if it is valid JSON with token exactly alpha and quality exactly good; otherwise return {"verdict":"fail"}. Do not add prose.

Treatment result:
{{.treatment}}