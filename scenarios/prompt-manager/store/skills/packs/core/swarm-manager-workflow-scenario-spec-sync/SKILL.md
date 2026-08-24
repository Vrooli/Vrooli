---
name: "swarm-manager-workflow-scenario-spec-sync"
description: "Typed prompt contract for scenario specification synchronization."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["swarm-manager","agent-manager","workflow","prompt-contract"]
  status: "active"
  revision: 1
  createdAt: "2026-07-18T03:05:30Z"
  updatedAt: "2026-07-18T03:05:30Z"
  modes: ["contract"]
  requires:
    scenarios: []
    commands: []
  origin:
    kind: "authored"
---
# Scenario Spec Sync Workflow

Synchronize the named scenario's specification with its preset or custom target, keeping every preserve path intact. Verify the result before you report it.

## Outcome work table

| Observable end state | Outcome |
| --- | --- |
| The specification is synchronized, every preserve path is intact and verified, the scenario's checks pass, and a read-only search of the repository confirmed nothing outside the scenario depends on its live directory. Name the verification and the dependency search in `summary`. | `complete` |
| Anything else: an unverified sync, a missing or altered preserve path, an unexpected repository state, a failing check, or a dependency on the live directory you could not rule out. State what is wrong in `summary`. | `needs_attention` |

Warning: when this workflow completes, Swarm archives the scenario and then deletes its directory. The `complete` row's conditions are the archive-safety check — any doubt about any of them is `needs_attention`.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.entity}}` | Scenario identity. |
| `{{.snapshot}}` | The scenario name, the preset-or-custom target, and the preserve paths. Preserve paths are inviolable. |

## Boundary

Write only inside the named scenario's directory; reading elsewhere in the repository is permitted for the dependency search. Do not archive or delete anything yourself — Swarm owns that step. Do not touch preserve paths except to verify them.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
