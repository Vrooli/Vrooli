# Investigation Workflow Guide

## Overview
The App Issue Tracker uses direct script execution for AI-powered issue investigation, eliminating the need for external workflow engines.

## How It Works

### Investigation Flow
1. **User triggers investigation** via API, CLI, or UI
2. **API executes investigation script** in background
3. **Script interfaces with Claude Code** for analysis
4. **Results are saved to issue file** in YAML format
5. **Issue moves through status folders** automatically

### Direct Script Execution
Instead of complex workflow orchestration, the system now uses simple bash scripts:

```bash
# Trigger investigation via API
curl -X POST http://localhost:8090/api/investigate \
  -H "Content-Type: application/json" \
  -d '{"issue_id": "issue-123", "agent_id": "deep-investigator"}'

# Or directly via script
bash scripts/claude-investigator.sh "issue-123" "deep-investigator"
```

### File-Based Status Management
Issues automatically move between folders based on their status:
- `open/` - New issues awaiting triage
- `investigating/` - Issues being analyzed
- `in-progress/` - Issues being fixed
- `fixed/` - Resolved issues
- `closed/` - Archived issues
- `failed/` - Issues that couldn't be resolved

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
- `deep-investigator` - Thorough code analysis
- `auto-fixer` - Automated fix generation
- `quick-analyzer` - Rapid triage

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

## Integration with Claude Code

The `claude-investigator.sh` script creates structured prompts for Claude Code:

1. Loads issue details from YAML file
2. Generates investigation prompt with context
3. Executes Claude Code with project context
4. Parses response and updates issue file
5. Moves issue to appropriate status folder

This direct integration is more reliable and easier to maintain than workflow-based systems.