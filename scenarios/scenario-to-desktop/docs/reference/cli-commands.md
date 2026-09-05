# CLI Interface Contract

> Relocated from PRD.md during documentation restructuring (2026-04-06).

## Overview

| Property | Value |
|----------|-------|
| Binary | `scenario-to-desktop` |
| Installation | Installed by the control plane from the declared Go module |

## Required Commands

### `status`

Show desktop generation system status.

**Flags**: `--json`, `--verbose`

### `help`

Display command help and usage.

**Flags**: `--all`, `--command <name>`

### `version`

Show CLI and API version information.

**Flags**: `--json`

## Custom Commands

### `generate`

Generate a desktop application for a scenario.

**API endpoint**: `POST /api/v1/desktop/generate`

**Arguments**:
| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `scenario_name` | string | yes | Name of the scenario to create desktop app for |

**Flags**:
| Flag | Description | Default |
|------|-------------|---------|
| `--framework` | Desktop framework (electron) | `electron` |
| `--template` | Application template type (basic, advanced, kiosk, multi_window) | `basic` |
| `--platforms` | Target platforms (win, mac, linux or 'all') | `all` |
| `--output` | Output directory for generated application | — |
| `--features` | Comma-separated list of features (tray, updater, menus) | — |

**Output**: Desktop build ID and installation paths.

### `build`

Build a desktop application project.

**API endpoint**: `POST /api/v1/desktop/build`

**Arguments**:
| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `desktop_path` | string | yes | Path to desktop application source directory |

**Flags**:
| Flag | Description |
|------|-------------|
| `--platforms` | Platforms to build for (win, mac, linux or 'all') |
| `--sign` | Code sign the applications (requires certificates) |
| `--publish` | Publish to configured distribution channels |

**Output**: Built application package paths.

### `test`

Test desktop application functionality.

**API endpoint**: `POST /api/v1/desktop/test`

**Arguments**:
| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `app_path` | string | yes | Path to desktop application to test |

**Flags**:
| Flag | Description |
|------|-------------|
| `--platforms` | Platforms to test on (current platform by default) |
| `--headless` | Run tests in headless mode |

**Output**: Test results and screenshots.

### `package`

Package desktop application for distribution.

**API endpoint**: `POST /api/v1/desktop/package`

**Arguments**:
| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `app_path` | string | yes | Path to built desktop application |

**Flags**:
| Flag | Description |
|------|-------------|
| `--store` | Target store (microsoft, mac, snap, all) |
| `--enterprise` | Create enterprise deployment packages |

**Output**: Package status and distribution URLs.
