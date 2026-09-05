# Requirements Registry

This directory maps Command Center's operational targets to technical requirements. It is consumed by `test-genie` and the `scenario-completeness-scoring` integration.

Modules are grouped by priority so a module maps one-to-one onto a PRD priority band, matching the shape used by the fleet's other team instruments.

## Module Structure

| Folder | Priority | Description |
|---|---|---|
| `01-must-ship/` | P0 | The instrument itself: readings that carry values, the two honesty axes, authored samples, the setpoint read, the derived board shape, the ranked surface, the open-loop self-report, and one address. |
| `02-post-launch/` | P1 | The board: the provenance rendering contract, composed rooms in both orientations, the ambient shell, one intent vocabulary, audience modes, and the capability ladder. |
| `03-future/` | P2 | Prediction binding, reading history, and the two source bindings this scenario cannot satisfy until the teams behind them declare an instrument. |

## Requirement ID Pattern

Requirements use `CC-<priority>-<nnn>`:

- `CC-P0-*` — instrument core
- `CC-P1-*` — board and kiosk surfaces
- `CC-P2-*` — future and blocked-on-upstream

Ids are stable. A retired requirement keeps its id and is marked retired rather than being renumbered, because tests carry `[REQ:ID]` tags and prediction blocks may reference the target it traced to.

## Superseded modules

The 2026-04 registry grouped by feature area (`01-dashboard-aggregation`, `02-mission-control-slice`, `03-ui-shell`) with ids `CC-AGG-*`, `CC-MC-*` and `CC-UI-*`. Those modules described the read-only kiosk aggregator and were retired on 2026-09-01 with the PRD that generated them. The aggregation and cache behaviour they covered survives inside `CC-P0-001`, `CC-P0-004` and `CC-P0-009`, restated against the honesty contract rather than against the endpoint shape.

## Auto-sync

`auto_sync_enabled: true` in `index.json` means requirement statuses are updated automatically rather than by hand. Tests carry `[REQ:ID]` tags in their names or comments; requirement sync reads those tags, updates each `validation[].status`, rolls the requirement `status` up from them, and the PRD Control Tower flips the matching operational-target checkbox in `PRD.md`.

Two consequences worth knowing:

- **Do not hand-edit a status to make a target look done.** Sync will overwrite it, and in the interval it is a false claim in a registry whose whole job is to prevent those.
- **A `planned` validation with no test file is honest, not broken.** The refs below name tests that do not exist yet, which is why their type is `planned` rather than `test` — only code-typed refs (`test`, `unit`, `integration`, `code`, `automation`) are path-checked. When a test lands, change the type to `test` and let sync own the status from then on.

## Validation status vocabulary

Each `validation` entry carries `status`:

- `not_implemented` — the test does not exist yet
- `failing` — the test exists and does not pass
- `implemented` — the test exists and passes

A requirement is only `complete` when every validation entry is `implemented`. A requirement with no validation entry is not complete regardless of what the code does — an unverified claim is the thing this registry exists to prevent.
