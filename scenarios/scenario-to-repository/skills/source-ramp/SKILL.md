---
name: "source-ramp"
description: "Prepare, verify, and hand off a reproducible source distribution without Git or remote mutation."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["source-distribution", "closure", "artifact", "human-handoff"]
  status: "active"
  revision: 1
  requires:
    scenarios: ["scenario-to-repository", "scenario-dependency-analyzer"]
    commands: ["source analyze", "source assemble", "source verify"]
  origin: {kind: "authored"}
---

## Scope

Use this skill for one-way source export preparation, exact artifact
verification, clean-input diagnosis, update preview, and human publication
handoff. The monorepo remains canonical. Never initialize, clone, checkout,
commit, push, force-push, or mutate a Git repository as part of this workflow.

## Order

Analyze the SDA-owned closure, capture a stable source digest, validate the
versioned recipe and publishability policy, assemble deterministic `tar_gzip`,
verify the archive and manifest, then prepare a human-only handoff. Preserve
unknown, unavailable, refused, and failed states; do not green-skip missing
runtime requirements.

## Evidence

Carry scenario, source digest, closure digest, policy digest, recipe digest,
archive digest, verification receipt, and destination preconditions together.
If any identity changes, require a new review. Actual publication requires a
human-authorized external owner and destination read-back.
