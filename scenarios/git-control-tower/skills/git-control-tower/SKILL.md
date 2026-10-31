---
name: "git-control-tower"
description: "Operate Git Control Tower as the owner of immutable regression snapshots, collections, diffs, and source-evidence standing."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["git-control-tower","baseline","collection","diff","evidence"]
  icon: "git-compare"
  status: "active"
  revision: 1
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["git-control-tower"]
    commands: ["git-control-tower baseline", "git-control-tower collection"]
  origin: {kind: "authored"}
---
## Tools focus: Git Control Tower Regression Evidence

Use Git Control Tower as the authority for immutable before-state snapshots,
multi-root collections, diffs, and operation standing. Retain producer handles
and cite their evidence; do not rebuild baseline truth from Git commands.

Required reading: `path:scenarios/git-control-tower/docs` and
`path:docs/TESTING.md`.

### Scope

In scope: selecting a snapshot or collection, capture, explicit extension,
diff start, one wait, standing reads, and evidence handoff. Out of scope:
client polling, mutable replacement of an immutable anchor, silent dirty-tree
exclusion, or private baseline storage.

| Need | Operation |
|---|---|
| One content root | Use `baseline snapshot`; preserve its id and identity. |
| Several declared roots | Use `collection capture`; preserve child handles. |
| Compare current state | Start one baseline or collection diff and attach once. |
| Scope is incomplete | Estimate first; extend explicitly and retain the extension evidence. |
| Standing is unavailable | Report unknown with the producer error. |

### Output expectations

Handoff names anchor id, source identity, operation id, terminal standing,
diff evidence, dirty/ignored policy, and degradations. No governed program is
declared: baseline and collection operations are not yet run-eligible bindings.

### Troubleshooting & Edge Cases

- A cancelled client detaches; read standing before deciding whether work ended.
- A dirty anchor is valid only when its policy and identity record that state.
- An unbound program request must stop at the binding gap; do not shell out from Python.
