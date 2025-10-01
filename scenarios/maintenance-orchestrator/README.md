# Maintenance Orchestrator

**Control Plane for System Maintenance Operations**

## Overview

The Maintenance Orchestrator provides centralized control over all maintenance-related scenarios in Vrooli. It ensures maintenance operations never run wild by starting all scenarios in an inactive state and providing a unified interface to selectively enable them.

Think of it as "mission control" for system maintenance - you decide exactly what runs and when.

## Key Features

- **Auto-discovery** of all maintenance scenarios via tags
- **Safe by default** - All maintenance starts inactive
- **Preset configurations** for common maintenance patterns
- **Real-time dashboard** showing maintenance status
- **Resource monitoring** to prevent overload
- **Calendar integration** (optional) for scheduled maintenance

## Quick Start

```bash
# Start the orchestrator
vrooli scenario run maintenance-orchestrator
# Or using Make:
cd scenarios/maintenance-orchestrator && make run

# Access the UI dashboard (port shown in startup logs)
# The UI will display at http://localhost:371XX

# List all maintenance scenarios
maintenance-orchestrator list

# Activate a specific scenario
maintenance-orchestrator activate code-smell

# Apply a preset
maintenance-orchestrator preset apply full

# Check status
maintenance-orchestrator status --json
```

## Default Presets

### 1. Full Maintenance
Activates all maintenance scenarios - use with caution!

### 2. Security
Security-related maintenance scenarios

### 3. Performance
Optimization and monitoring scenarios:
- performance-monitor
- cache-optimizer
- query-analyzer

### 4. Off Hours
Heavy maintenance operations:
- database-vacuum
- log-rotation
- backup-manager

### 5. Minimal
Essential maintenance only:
- health-checker
- log-cleanup

## Web UI

Access the dashboard at `http://localhost:3250` (or configured port).

Features:
- Visual status indicators for each scenario
- One-click preset activation
- Activity log with timestamps
- Resource usage graphs
- Quick toggle switches

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/v1/scenarios | List all maintenance scenarios |
| POST | /api/v1/scenarios/{id}/activate | Activate a scenario |
| POST | /api/v1/scenarios/{id}/deactivate | Deactivate a scenario |
| GET | /api/v1/presets | List available presets |
| POST | /api/v1/presets/{id}/apply | Apply a preset |
| GET | /api/v1/status | Get current system status |

## How It Works

1. **Discovery Phase**: On startup, scans all scenarios for `"maintenance"` tag in their service.json
2. **Registration**: Each maintenance scenario registers its control endpoints
3. **State Management**: Maintains in-memory state (always starts inactive)
4. **Control Loop**: Monitors and manages scenario states based on your commands
5. **Safety Checks**: Prevents conflicts and resource exhaustion

## Integration with Other Scenarios

### Making a Scenario Maintenance-Compatible

Add to your scenario's `.vrooli/service.json`:
```json
{
  "tags": ["maintenance"],
  "maintenance": {
    "canDeactivate": true,
    "defaultState": "inactive"
  }
}
```

Update your API status endpoint to include:
```json
{
  "health": "healthy",
  "maintenanceState": "active|inactive",
  "canToggle": true
}
```

### Calendar Integration

When the calendar scenario is available, you can schedule maintenance windows:
```bash
# Schedule security maintenance for Sunday 2 AM
maintenance-orchestrator schedule create \
  --preset security-only \
  --cron "0 2 * * 0" \
  --duration 2h
```

## Configuration

### Environment Variables
- `MAINTENANCE_PORT`: API port (default: 3250)
- `DISCOVERY_INTERVAL`: How often to scan for new scenarios (default: 60s)
- `DEFAULT_TIMEOUT`: Auto-deactivate timer in minutes (default: disabled)
- `ENABLE_CALENDAR`: Enable calendar integration (default: false)

### Custom Presets

Create via CLI:
```bash
# Save current state as preset
maintenance-orchestrator preset create my-preset

# Or via JSON
maintenance-orchestrator preset import preset.json
```

## Architecture

```
┌─────────────────────────────────────┐
│     Maintenance Orchestrator        │
├─────────────────────────────────────┤
│  Discovery Engine                   │
│  ├─ Scans scenarios/                │
│  └─ Filters by "maintenance" tag    │
├─────────────────────────────────────┤
│  State Manager (In-Memory)          │
│  ├─ Scenario States                 │
│  ├─ Active Presets                  │
│  └─ Activity Log                    │
├─────────────────────────────────────┤
│  Control Interface                  │
│  ├─ REST API                        │
│  ├─ CLI Commands                    │
│  └─ Web Dashboard                   │
├─────────────────────────────────────┤
│  Integration Layer                  │
│  ├─ Calendar (optional)             │
│  ├─ Monitoring                      │
│  └─ Notifications                   │
└─────────────────────────────────────┘
```

## Safety Features

1. **Confirmation Dialogs**: Bulk operations require confirmation
2. **Resource Limits**: Prevents activating too many heavy scenarios
3. **Auto-deactivate**: Optional timers to prevent forgotten maintenance
4. **Conflict Detection**: Warns about incompatible scenarios
5. **Emergency Stop**: `maintenance-orchestrator stop-all` for immediate shutdown

## Troubleshooting

### Scenario Not Discovered
- Check scenario has `"maintenance"` tag in service.json
- Verify scenario is running
- Check discovery logs: `maintenance-orchestrator logs --filter discovery`

### Cannot Activate Scenario  
- Ensure scenario has proper status endpoint
- Check scenario supports maintenance state
- Verify no conflicts with other active scenarios

### State Not Persisting
- This is by design - all scenarios start inactive
- Use presets to quickly restore desired state
- Consider calendar integration for automation

## Best Practices

1. **Start Small**: Activate one scenario at a time initially
2. **Monitor Resources**: Watch CPU/memory when activating multiple scenarios
3. **Use Presets**: Create presets for common patterns
4. **Schedule Wisely**: Run heavy maintenance during off-hours
5. **Document Dependencies**: Note which scenarios conflict

## Contributing

To add maintenance orchestration support to your scenario:

1. Add the `"maintenance"` tag
2. Implement state control in your API
3. Ensure clean shutdown behavior
4. Test with the orchestrator
5. Document any special requirements

## Related Scenarios

- **calendar**: Schedule maintenance windows
- **system-monitor**: Track resource usage during maintenance
- **alert-manager**: Get notified of maintenance events
- **maintenance-reporter**: Generate maintenance reports

---

*Part of the Vrooli self-improving intelligence system - where every capability becomes permanent.*