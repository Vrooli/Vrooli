---
name: "skill-experiment-calibration-treatment"
description: "Calibration-only treatment prompt for measured skill experiments"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["calibration","skill-experiment"]
  status: "active"
  revision: 1
  createdAt: "2026-07-21T21:23:51Z"
  updatedAt: "2026-07-21T21:23:51Z"
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
Calibration treatment: Given the supplied token, return exactly {"token":"<input>","quality":"good"}. Do not add prose.