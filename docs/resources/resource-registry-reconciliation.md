# Resource Registry Reconciliation

This document reconciles implemented resource directories under `resources/` with registry entries under `.vrooli/resource-registry/`.

It is the concrete output of step 1 before Phase 0 inventory classification.

## Summary

- Resource directories: `80`
- Registry entries: `46`
- Scenario service files audited: `92`

Mismatch categories:

- `39` resources exist in `resources/` without a matching registry entry
- `5` registry entries exist without a matching `resources/` directory

## Step 2 Decision

`.vrooli/resource-registry/` is now treated as transitional metadata, not a canonical source of truth.

Canonical sources of truth are:

- `resources/<name>` for implemented resources
- root `.vrooli/service.json` for project-level enablement
- scenario `.vrooli/service.json` files for dependency usage
- future blueprint/archive stores for non-implemented resource knowledge

Implications:

- missing registry entries are not blockers for active repo correctness
- registry-only entries are not evidence of supported implementation
- Phase 0 should classify resources from implementation and usage signals, not from registry presence
- registry cleanup can be performed incrementally or replaced later by derived metadata

## Decision Rules

- `add-registry`
  The resource exists in `resources/` and still has active project/scenario usage. Keep it discoverable until registry becomes fully derived or removed.
- `phase0-review`
  The resource exists in `resources/` but has no current usage signal. Do not create registry entries yet; classify during Phase 0 as `keep`, `blueprint`, or `deprecate`.
- `remove-registry`
  The registry entry has no matching implementation and no active usage signal. Treat it as stale metadata and remove or archive it.
- `restore-or-blueprint`
  The registry entry has no matching implementation but still has scenario or ecosystem references. Decide whether to restore implementation, convert references, or preserve as blueprint knowledge.

## Resources Missing Registry Entries

| Resource | Root Enabled | Scenario Refs | Proposed Action | Rationale |
|---|---:|---:|---|---|
| `kafka` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `kokoro` | yes | 1 | `phase0-review` | Still referenced by `web-console`; keep review open until blueprint vs implemented status is decided. |
| `pihole` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `pybullet` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `restic` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `ros2` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `segment-anything` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `sqlite` | yes | 6 | `keep-implemented` | Enabled at project level and widely used by scenarios. |
| `su2` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `terraform` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `traccar` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `ultralytics-yolo` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `virustotal` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `wireguard` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `zigbee2mqtt` | yes | 0 | `blueprint` | Explicitly designated blueprint and disabled at project level. |
| `cncjs` | no | 0 | `phase0-review` | No current config usage signal. |
| `earthly` | no | 0 | `phase0-review` | No current config usage signal. |
| `eclipse-ditto` | no | 0 | `phase0-review` | No current config usage signal. |
| `elmer-fem` | no | 0 | `phase0-review` | No current config usage signal. |
| `esphome` | no | 0 | `phase0-review` | No current config usage signal. |
| `freecad` | no | 0 | `phase0-review` | No current config usage signal. |
| `gazebo` | no | 0 | `phase0-review` | No current config usage signal. |
| `geonode` | no | 0 | `phase0-review` | No current config usage signal. |
| `ggwave` | no | 0 | `phase0-review` | No current config usage signal. |
| `godot` | no | 0 | `phase0-review` | No current config usage signal. |
| `gridlabd` | no | 0 | `phase0-review` | No current config usage signal. |
| `lnbits` | no | 0 | `phase0-review` | No current config usage signal. |
| `mathlib` | no | 0 | `phase0-review` | No current config usage signal. |
| `matrix-synapse` | no | 0 | `phase0-review` | No current config usage signal. |
| `mcrcon` | no | 0 | `phase0-review` | No current config usage signal. |
| `meep` | no | 0 | `phase0-review` | No current config usage signal. |
| `mifos` | no | 0 | `phase0-review` | No current config usage signal. |
| `nsfw-detector` | no | 0 | `phase0-review` | No current config usage signal. |
| `octoprint` | no | 0 | `phase0-review` | No current config usage signal. |
| `open-data-cube` | no | 0 | `phase0-review` | No current config usage signal. |
| `openfoam` | no | 0 | `phase0-review` | No current config usage signal. |
| `papermc` | no | 0 | `phase0-review` | No current config usage signal. |
| `speaker-verification` | no | 0 | `phase0-review` | No current config usage signal. |
| `vnc` | no | 0 | `phase0-review` | No current config usage signal. |

## Registry Entries Missing Resource Directories

| Registry Entry | Root Enabled | Scenario Refs | Proposed Action | Rationale |
|---|---:|---:|---|---|
| `node-red` | no | 0 | `blueprint` | Explicitly designated blueprint. Live scenario manifest references were removed during Phase 0 validation. |
| `autogen-studio` | no | 0 | `remove-registry` | No implementation and no active usage signal. |
| `erpnext` | no | 0 | `remove-registry` | No implementation and no active usage signal in current scenario configs. |
| `langchain` | no | 0 | `remove-registry` | No implementation, no active scenario usage, and only disabled root config metadata remains. |
| `musicgen` | no | 0 | `remove-registry` | No implementation and no active scenario usage. |

## Root Config Concepts Without Implementation or Registry

These concepts are outside the registry-vs-implementation mismatch counts above, but still matter for Phase 0 because root `.vrooli/service.json` is part of the source-of-truth set.

| Concept | Root Enabled | Proposed Action | Rationale |
|---|---:|---|---|
| `parlant` | no | `blueprint` | Disabled root config concept with no implementation or registry entry; preserve as blueprint knowledge rather than supported integration metadata. |

## Immediate Follow-up Tasks

1. Carry the blueprint-designated resources above into Phase 0 as pre-decided blueprint candidates.
2. Carry the remaining `phase0-review` set into the Phase 0 inventory with a bias toward `blueprint` or `deprecate` unless stronger validation evidence appears.

## Suggested Next Step

Execute Phase 0 inventory/classification using implementation and usage evidence, not registry completeness.
