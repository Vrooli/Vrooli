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

### Optional managed renderer profiles for bundle staging

The `pipeline run` command supports `--tool-artifact-root`,
`--renderer-tool-profiles`, and `--artifact-trust-mode`. The renderer selector
is finite: `blender`, `ffmpeg-video`, or both. It must point at a verified
release-v1 directory containing the matching renderer roles and pinned Linux
amd64 trees. For local private-bundle work, use `development-local`; that mode
is unsigned and cannot be promoted. This path does not provision tools and does
not change the normal `host-required` tool declaration.

To validate only package staging, keep the pipeline limited to its bundle stage:

```bash
scenario-to-desktop pipeline run device-studio \
  --stages bundle --platforms linux --location-mode staging \
  --tool-artifact-root "$RELEASE_DIR" \
  --renderer-tool-profiles blender,ffmpeg-video \
  --artifact-trust-mode development-local
```

The bundle stage must resolve a valid Electron bundle manifest for the target
scenario. Its default is `platforms/electron/bundle/bundle.json` in the
scenario; staging mode resolves a manifest under the pipeline staging output.
The stage may ask the configured manifest generator to create it. The pipeline
verifies the release directory and each selected tree before copying packages
to fixed bundle paths. Successful staging proves package integrity only, not a
standalone-app launch, rendering workflow, license grant, or production release.

## Agent-driven validation

The matrix surface is the provider-neutral, UI-free route for desktop
validation:

| Command | Purpose |
|---|---|
| `scenario-to-desktop targets list` | List target id, node name, OS, architecture, capabilities, availability, reason, missing capability, and next action. Add `--json` for the endpoint-shaped inventory. |
| `scenario-to-desktop matrix create <scenario> --artifact-digest <sha256:...> --platform <os> --journey <id> --profile <name>` | Create a matrix from ergonomic selectors. The platform selector resolves a target by target id, display name, or OS; a missing normalized platform produces an explicit unavailable target. `--selection-file` remains available for complete selections and overrides/augments these fields. |
| `scenario-to-desktop matrix start <run-id>` | Start a queued matrix. |
| `scenario-to-desktop matrix wait <run-id>` | Block once until the matrix is terminal. |
| `scenario-to-desktop matrix show <run-id>` | Read one matrix. |
| `scenario-to-desktop matrix list` | List durable matrices. |
| `scenario-to-desktop matrix compare <run-id> <prior-run-id>` | Compare two durable matrices. |
| `scenario-to-desktop validation run --scenario <scenario> --artifact-digest <sha256:...> --artifact-path <path> --platform <os> --journey <id> --profile <name>` | Create, start, and wait for a validation matrix in one command. Use `--selection-file <file>` for a complete native selection; otherwise the ergonomic flags resolve the target and construct the required artifact, journey, and profile fields. |
| `scenario-to-desktop validation status <run-id>` | Read one validation run. |

All commands support `--json`; default output is human-readable. A matrix
selection file is the escape hatch for multiple targets, per-target artifacts,
and richer journey metadata. The CLI does not expose a separate retry loop or
polling interval: waiting is server-owned and blocks once.

Selecting an absent platform never becomes an implicit skip: the matrix records
an unavailable cell with the missing platform as its reason. An unknown target
identifier remains a command error.

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
