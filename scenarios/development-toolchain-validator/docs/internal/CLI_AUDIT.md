# CLI Audit - development-toolchain-validator

## Last Updated
2026-03-11

## Audit Summary

This CLI follows the cli-core patterns per CLI Steer skill guidance.

## Current State

- [x] Go-based CLI exists (`cli/app.go`)
- [x] Uses cli-core package (`github.com/vrooli/cli-core/cliapp`)
- [x] Cross-platform installers present (`install.sh`, `install.ps1`)
- [x] All API endpoints have CLI commands

## API Coverage

| API Endpoint | CLI Command | Status |
|--------------|-------------|--------|
| GET /health | `status` | ✅ |
| GET /api/v1/references | `reference list` | ✅ |
| GET /api/v1/references/{id} | `reference get <id>` | ✅ |
| GET /api/v1/references/by-slug/{slug} | `reference get <slug>` | ✅ |
| POST /api/v1/references | `reference create` | ✅ |
| PATCH /api/v1/references/{id} | `reference update <id>` | ✅ |
| DELETE /api/v1/references/{id} | `reference delete <id>` | ✅ |

## CLI Architecture

### Command Groups

1. **Health** - API connectivity
   - `status` - Check API health and readiness

2. **References** - Reference scenario management
   - `reference list` - List all references with optional template filter
   - `reference get <id|slug>` - Get reference by ID or slug
   - `reference create` - Create new reference (requires --slug, --name, --template, --path)
   - `reference update <id>` - Update existing reference
   - `reference delete <id>` - Delete reference

3. **Configuration** - CLI configuration
   - `configure` - Set API base URL and tokens

### HTTP Helper Pattern

The App struct includes typed HTTP helper methods that centralize:
- Path construction with `/api/v1` prefix
- JSON marshaling/unmarshaling
- Error handling

```go
func (a *App) get(path string, result interface{}) error
func (a *App) post(path string, payload interface{}, result interface{}) error
func (a *App) patch(path string, payload interface{}, result interface{}) error
func (a *App) delete(path string) error
```

### Output Modes

All commands support two output modes per CLI Steer skill:
- **Human mode** (default): Formatted, readable output
- **JSON mode** (`--json`): Machine-readable for scripting

## Dry-Run Support

The API supports dry-run via `X-Dry-Run: true` header on all mutating endpoints:
- POST /api/v1/references (Create)
- PATCH /api/v1/references/{id} (Update)
- DELETE /api/v1/references/{id} (Delete)

CLI commands will gain `--dry-run` flag support automatically through cli-core's global flag handling.

## Issues Found

None - CLI is now compliant with CLI Steer skill patterns.

## Previous Issues (Resolved)

1. **Missing reference CRUD commands** - FIXED: Added full reference command group
2. **No HTTP helpers** - FIXED: Added get/post/patch/delete helper methods
3. **Only status command existed** - FIXED: Full API parity achieved

## Test Coverage

Tests in `cli/app_test.go` cover:
- App initialization
- API path construction
- Response parsing for all types
- Command routing and validation
- Utility functions (truncate)

## Priority Fixes (Completed)

1. ✅ Implemented reference command group with all CRUD operations
2. ✅ Added HTTP helper methods for consistent API interaction
3. ✅ Added --json flag support for all commands
4. ✅ Added comprehensive tests for CLI commands

## Future Commands (When Domains Implemented)

| Future API Endpoint | Planned CLI Command |
|---------------------|---------------------|
| GET/POST /connections | `connection list`, `connection create` |
| POST /connections/{id}/expectations | `expectation add` |
| POST /validate/{reference} | `validate <reference>` |
| GET /drift/{reference} | `drift check <reference>` |

These commands will be added when the corresponding API domains are implemented.
