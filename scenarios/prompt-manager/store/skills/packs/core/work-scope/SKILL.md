---
name: "work-scope"
description: "Session constraints for one kind of change: feature, bugfix, refactor, architecture, platform or audit. Pick the row the task names; state which one you chose."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["scope","constraints"]
  modes: ["scope"]
  status: "active"
  revision: 1
  origin:
    kind: "authored"
---
# Work scope

Scope bounds what a session may change; it never lowers the pace (R3). Choose the mode the task names, say so in your first update, and stay inside it.

| Mode | May change | Must not |
|---|---|---|
| feature | the new behavior and its tests | unrelated refactors, behavior outside the request |
| bugfix | the fault's cause and a regression test | new features, drive-by cleanup |
| refactor | structure, names, duplication, types | observable behavior, public contracts, new dependencies, new features; existing tests pass unmodified |
| architecture | module layout, seams, abstractions, wiring, tests that follow the new structure | new features, business-logic changes, lower coverage |
| platform (shared packages, templates, contracts) | fixes, additive capability, tests plus a downstream compatibility set, deprecation with a migration path | breaking exported APIs, flags or formats; new external dependencies; mass edits across scenarios |
| audit | reading, measuring, read-only tools, findings with evidence, severity and a recommendation | any source, config or test change; fixing what you find; commits |

Defaults (the task may override):

- Code is greenfield: change interfaces and proto types cleanly, with no legacy shims. Platform work is brownfield: compatibility first.
- Large structural change goes in reviewable steps.
- Other agents may be editing the same tree. Leave changes outside your target alone.

Done means: behavior is as the mode allows, tests pass, coverage is not lower, and nothing outside the target changed.
