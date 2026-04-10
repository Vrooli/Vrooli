# Investigation Workflow Guide

## Overview
The App Issue Tracker runs investigations through **agent-manager**. The API creates a task/run using a stored profile, and run events are surfaced back to the issue bundle.

## How It Works

### Investigation Flow
1. **User triggers an investigation** via API, CLI, or UI
2. **API builds the investigation prompt** and creates an agent-manager run
3. **Agent-manager executes the configured runner** (claude-code, codex, or opencode)
4. **Run events + summary** are stored on the issue metadata
5. **Issue transitions** from open → active → completed/failed automatically

### Triggering a Run
```bash
curl -X POST http://localhost:8090/api/v1/investigate \
  -H "Content-Type: application/json" \
  -d '{"issue_id": "issue-123", "agent_id": "unified-resolver", "auto_resolve": true}'
```

### File-Based Status Management
Issues automatically move between folders based on their status:
- `open/` - New issues awaiting triage
- `active/` - Agent is currently working the issue
- `completed/` - Agent produced a recommended fix
- `failed/` - Agent run failed, was cancelled, or needs manual attention
- `archived/` - Historical issues kept for reference

Cancelled or idle-timed-out agent runs are automatically moved into `failed/` so follow-up workflows can immediately see they need human attention.

## Configuration

### Environment Variables
```bash
export API_PORT=8090
export ISSUES_DIR=/path/to/issues
export QDRANT_URL=http://localhost:6333  # Optional semantic search
```

### Agent Profile Settings
Agent-manager settings live in `initialization/configuration/agent-settings.json` and can be updated via `PATCH /api/v1/agent/settings`.
Key options include:
- `runner_type` (claude-code, codex, opencode)
- `max_turns`
- `allowed_tools`
- `timeout_seconds`
- `skip_permissions`

The profile key is `app-issue-tracker-investigations` and is kept in sync with the configured settings.

## Benefits of Agent-Manager Execution
1. **Centralized lifecycle** with consistent run metadata and events
2. **Runner flexibility** across supported agent types
3. **Operational visibility** via run status/events instead of parsing local files
4. **Cleaner shutdowns** and cancellation handling

## Testing
Run the scenario test suite:

```bash
make test
```
