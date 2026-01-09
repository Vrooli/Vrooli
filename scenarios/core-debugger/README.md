# Core Debugger

## Purpose
Professional monitoring and debugging system for Vrooli's core infrastructure. Tracks health of critical components (CLI, orchestrator, resource manager, setup scripts) and provides automated issue detection, workaround suggestions, and fix generation.

## Key Features
- **Zero Dependencies**: Only requires claude-code resource - works even when everything else is broken
- **File-Based Storage**: Git-trackable issue and workaround database
- **Component Health Monitoring**: Real-time health checks of core systems
- **Workaround Database**: Known issues with proven solutions
- **Intelligent Analysis**: Claude-powered issue analysis and fix generation
- **Status Dashboard**: Professional status page showing system health

## Why This Matters
Core Debugger prevents infrastructure issues from cascading and blocking development:
- **Single Source of Truth**: All core issues tracked in one place
- **Automated Detection**: Issues caught before they block scenarios
- **Shared Knowledge**: Workarounds available to all agents and developers
- **Self-Healing**: Foundation for automatic issue resolution

## Dependencies
### Required Resources
- **claude-code**: For intelligent issue analysis (only dependency)

### Zero Dependencies On
- No database resources (postgres, redis, etc.)
- No other scenarios
- No workflow engines (n8n, node-red, etc.)
- Works even when core systems are partially broken

## Integration Points
### For Other Scenarios
- Check core health: `core-debugger status`
- Get workarounds: `core-debugger get-workaround <error-message>`
- Report issues: `core-debugger report-issue --component cli --description "Error details"`

### API Endpoints
- `GET /api/v1/health` - Overall system health
- `GET /api/v1/issues` - List active issues
- `GET /api/v1/issues/{id}/workarounds` - Get specific workarounds
- `POST /api/v1/issues/{id}/analyze` - Trigger Claude analysis

## File Storage Structure
```
data/
├── issues/          # Active issues (git-tracked)
├── health/          # Component health states
├── workarounds/     # Workaround database
├── patterns/        # Issue patterns for detection
├── logs/           # Health check logs
└── archive/        # Old resolved issues
```

## UI Style
**Professional Status Dashboard**: Clean, technical interface inspired by GitHub Status and AWS Service Health Dashboard. Dark mode with clear status indicators (green/yellow/red), real-time updates, and actionable information.

## Quick Start
```bash
# Setup
cd scenarios/core-debugger
vrooli scenario setup core-debugger

# Start monitoring
vrooli scenario run core-debugger

# Check health
core-debugger status
core-debugger check-health --verbose

# Get help with an issue
core-debugger list-issues
core-debugger get-workaround <issue-id>
```

## Common Operations
```bash
# Monitor specific component
core-debugger check-health --component orchestrator

# List critical issues
core-debugger list-issues --severity critical

# Analyze an issue with Claude
core-debugger analyze-issue <issue-id>

# View status dashboard
open http://localhost:22100
```

## How It Works
1. **Health Monitoring**: Continuously checks core components
2. **Issue Detection**: Identifies problems via health checks and log analysis
3. **Deduplication**: Groups similar issues by error signature
4. **Workaround Matching**: Suggests known fixes from database
5. **Claude Analysis**: For unknown issues, uses AI to suggest solutions
6. **Knowledge Building**: Successful fixes added to workaround database

## Benefits
- **Prevents Cascading Failures**: Catches issues before they spread
- **Reduces Debugging Time**: Instant access to known solutions
- **Enables Self-Healing**: Foundation for automatic recovery
- **Improves Reliability**: Proactive issue detection
- **Shares Knowledge**: All agents benefit from collective debugging

## File-Based Advantages
- **Git Trackable**: Issue history in version control
- **No DB Required**: Works even if databases are down
- **Portable**: Easy to backup and share
- **Transparent**: Human-readable JSON files
- **Resilient**: No single point of failure

---

**Note**: This scenario is designed to be the most reliable component in Vrooli - it must work even when everything else is broken. Keep dependencies minimal and storage simple.
