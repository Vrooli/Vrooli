---
name: "git-control-tower-draft-release"
description: "Draft audience-specific release notes from source-backed change and validation evidence without claiming publication."
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["git-control-tower", "advisory", "release-notes", "draft"]
  status: "active"
  revision: 1
  requires:
    scenarios: ["git-control-tower"]
    commands: []
  origin: {kind: "authored"}
---

# Draft release notes

Separate source facts, validation facts, known limitations, and audience
wording. Do not infer rollout dates, readiness, host publication, or user
impact that is absent from evidence. Keep omitted and failed evidence visible.

Return editable text and an exact source/evidence manifest. This workflow is
draft-only: it cannot tag, commit, push, call a host file-write API, or publish
a release.
