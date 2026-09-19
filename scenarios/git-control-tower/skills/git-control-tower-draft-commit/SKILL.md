---
name: "git-control-tower-draft-commit"
description: "Draft an editable commit message for an exact Git Control Tower subject without committing or changing staging."
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["git-control-tower", "advisory", "commit-draft"]
  status: "active"
  revision: 1
  requires:
    scenarios: ["git-control-tower"]
    commands: []
  origin: {kind: "authored"}
---

# Draft a commit message

Use only the exact subject and its evidence bundle. Return an editable subject
and body, supported trailer suggestions, omitted-file disclosure, and
uncertainty notes. Preserve operator text and trailer order on regeneration.

This workflow never stages files, runs precommit, creates a commit, rewrites
history, or treats a trailer as authorship proof. A human must review and
choose any later repository action through the authenticated control path.
