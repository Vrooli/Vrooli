---
name: "platform-scope"
description: "Session constraints for shared packages/platform code: compatibility-first, brownfield-safe, no breaking changes by default."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["scope","platform","constraints"]
  modes: ["scope"]
  status: "active"
  revision: 1
  createdAt: "2026-02-10T00:00:00Z"
  updatedAt: "2026-02-10T00:00:00Z"
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
# Platform Scope

Session constraints for work on shared platform code (for example `path:packages/*`, shared templates, shared contracts). These boundaries keep changes brownfield-safe and compatible with downstream scenarios.

## Session Boundaries

### ALLOWED
- Bugfixes and reliability improvements in shared packages
- Additive capabilities that do not break existing consumers
- Improving default human output contracts (clear next steps, deterministic exit codes)
- Adding tests proportional to blast radius (unit + a small downstream compat set)
- Updating documentation/changelogs for changed contracts
- Adding deprecation warnings and migration paths when needed

### NOT ALLOWED (unless explicitly requested)
- Breaking changes (removing/renaming exported APIs, flags, output fields, file formats)
- Introducing new external dependencies or toolchains
- Forcing scenarios to adopt a new language/framework
- Sweeping refactors unrelated to the reported issue
- Mass edits across many scenarios (prefer a compat set + targeted fixes)

## Quality Requirements

1. **Compatibility-first**: preserve existing behavior/contracts by default
2. **Minimal blast radius**: keep diffs tight; avoid “cleanup” unrelated to the change
3. **Contract clarity**: human-first output is the default contract; `--json` is optional
4. **Verification**: add/extend tests and run a downstream compat set

## Verification Checklist

Before completing any platform/package task:
- [ ] Change type classified (bugfix/additive/behavior change/breaking)
- [ ] No breaking changes unless explicitly approved (and if so: deprecation + migration note)
- [ ] Tests added/updated in the package
- [ ] A downstream compat set was run and passed
- [ ] Default output contracts remain human-first and action-guiding

