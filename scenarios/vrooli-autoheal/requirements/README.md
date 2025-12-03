# Requirements Registry

This directory contains the technical requirements for vrooli-autoheal, organized by operational target groupings from the PRD.

## Module Layout

| Folder | PRD Refs | Description |
|--------|----------|-------------|
| `01-cli-interface/` | OT-P0-001, OT-P0-002, OT-P0-010 | CLI commands (tick, loop, status) |
| `02-platform-detection/` | OT-P0-003 | Cross-platform detection and capabilities |
| `03-health-registry/` | OT-P0-004, OT-P1-008 | Health check registration and execution |
| `04-core-bootstrap/` | OT-P0-005 | Bootstrap DB, resources, and scenarios |
| `05-resource-monitoring/` | OT-P0-006 | Resource health checks and auto-restart |
| `06-scenario-monitoring/` | OT-P0-007 | Scenario health checks and auto-restart |
| `07-os-watchdog/` | OT-P0-008 | OS-level service installation |
| `08-persistence/` | OT-P0-009, OT-P1-006 | Health result storage and history |
| `09-infrastructure-checks/` | OT-P1-001 to OT-P1-005, OT-P1-009 | Network, disk, RDP, Docker, cloudflared |
| `10-web-dashboard/` | OT-P1-007 | React UI for health visualization |
| `11-future-features/` | OT-P2-* | P2 expansion ideas |

## Naming Pattern

- **Requirement IDs**: `[MODULE]-[FEATURE]-[NUMBER]` (e.g., `CLI-TICK-001`, `PLAT-DETECT-002`)
- **Module prefixes**:
  - `CLI-` - CLI interface requirements
  - `PLAT-` - Platform detection requirements
  - `REG-` - Health registry requirements
  - `BOOT-` - Bootstrap requirements
  - `RES-` - Resource monitoring requirements
  - `SCEN-` - Scenario monitoring requirements
  - `WATCH-` - OS watchdog requirements
  - `PERSIST-` - Persistence requirements
  - `INFRA-` - Infrastructure check requirements
  - `UI-` - Web dashboard requirements
  - `FUTURE-` - Future feature requirements

## Test Tagging

Tag tests with `[REQ:ID]` so auto-sync can update status:

```go
// [REQ:CLI-TICK-001] Test tick command execution
func TestTickCommand(t *testing.T) {
    // ...
}
```

## Running Tests

```bash
# Full phased test suite
cd test && ./run-tests.sh

# Via Makefile
make test
```

## Reference

See `docs/testing/guides/requirement-tracking-quick-start.md` for schema details.
