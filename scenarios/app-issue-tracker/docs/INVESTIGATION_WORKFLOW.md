# Investigation Workflow Guide

## Overview
The App Issue Tracker uses direct script execution for AI-powered issue investigation, eliminating the need for external workflow engines.

## How It Works

### Investigation Flow
1. **User triggers unified agent run** via API, CLI, or UI
2. **API executes the resolution script** in background
3. **Script interfaces with Codex** for analysis and remediation
4. **Results are saved to the issue bundle** in YAML/Markdown format
5. **Issue transitions through simplified status folders** automatically

### Direct Script Execution
Instead of complex workflow orchestration, the system now uses simple bash scripts:

```bash
# Trigger unified run via API
curl -X POST http://localhost:8090/api/v1/investigate \
  -H "Content-Type: application/json" \
  -d '{"issue_id": "issue-123", "agent_id": "unified-resolver", "auto_resolve": true}'

# Or directly via script
bash scripts/claude-investigator.sh resolve "issue-123" "unified-resolver" \
  "/path/to/project" "Investigate the failing login flow" true
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
# API configuration
export API_PORT=8090
export ISSUES_DIR=/path/to/issues

# Optional services
export QDRANT_URL=http://localhost:6333  # For semantic search
```

### Agent Configuration
Agents are defined in the API with capabilities:
- `unified-resolver` - Single-pass agent that triages, investigates, and drafts fixes
- Prompt instructions are sourced from `prompts/unified-resolver.md` so updates do not require database migrations.

## Benefits of Direct Execution

1. **Simplicity** - No external workflow engines to configure
2. **Portability** - Works anywhere bash and Go are available
3. **Git-Friendly** - All issues are YAML files, easily tracked
4. **Debugging** - Simple scripts are easier to debug than workflows
5. **Performance** - Direct execution without middleware overhead

## Testing

Run the test suite to verify everything works:

```bash
# Test API functionality
cd test
./test-investigation-workflow.sh

# Test CLI operations
cd cli
./app-issue-tracker.bats
```

## Integration with Codex

The `claude-investigator.sh` script creates structured prompts for Codex:

1. Loads issue details from YAML file
2. Generates investigation prompt with context
3. Executes Codex with project context
4. Parses response and updates issue file
5. Moves issue to appropriate status folder

This direct integration is more reliable and easier to maintain than workflow-based systems.
