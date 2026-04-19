# Requirements Registry

This directory maps Command Center's operational targets to technical requirements. It is consumed by `test-genie` and the `scenario-completeness-scoring` integration.

## Module Structure

| Folder | Priority | Description |
|---|---|---|
| `01-dashboard-aggregation/` | P0 | Core dashboard aggregation capability (registry + cache + upstream clients + handlers). |
| `02-mission-control-slice/` | P0 | Mission Control vertical slice (theme + real scene + live data + gap badges). |
| `03-ui-shell/` | P1 | UI shell: router, 6 themed routes, placeholder pages for the 5 non-Mission-Control dashboards. |

## Requirement ID Pattern

All requirements use the prefix `CC-` followed by a module abbreviation:

- `CC-AGG-*` — dashboard aggregation (API layer)
- `CC-MC-*` — Mission Control slice
- `CC-UI-*` — UI shell

## Test Tagging

Tag tests with `[REQ:ID]` to enable auto-sync. Validation entries on each requirement point to the test paths that cover them.

## Scope

This registry tracks only the **scaffold** items delivered by `execute/command-center-scenario-scaffold`. Follow-up execute items (`command-center-theming-engine`, `command-center-kiosk-ux`, `lpbs-command-center-dashboard-endpoints`) own their own requirement entries and may add new modules here.
