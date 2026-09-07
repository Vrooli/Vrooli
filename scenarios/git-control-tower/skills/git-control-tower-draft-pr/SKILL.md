---
name: "git-control-tower-draft-pr"
description: "Compose a reviewer-oriented pull-request draft from immutable base/head evidence and explicit work references."
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["git-control-tower", "advisory", "pull-request", "draft"]
  status: "active"
  revision: 1
  requires:
    scenarios: ["git-control-tower"]
    commands: []
  origin: {kind: "authored"}
---

# Draft a pull request

Require host instance, provider repository identity, immutable base, immutable
head, and subject digest. Separate observed behavior, validation results,
unknowns, and audience-facing narrative. Preserve unavailable checks as
unavailable and call out stale heads.

Drafting does not authenticate a provider, post comments, create a PR, push a
branch, select a branch, fetch, or publish release text. Publication requires
a separate bounded human capability and an explicit handoff.
