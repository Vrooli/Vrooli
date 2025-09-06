# Claude Code Integration Update

## Summary
Updated the Scenario Generator V1 to properly use the `resource-claude-code` CLI instead of mock implementations and removed all n8n references.

## Changes Made

### 1. Claude Client Implementation (`api/pipeline/claude_client.go`)
- ✅ Updated to use `resource-claude-code execute -` command
- ✅ Modified to pass prompts via stdin to avoid shell escaping issues  
- ✅ Added proper error handling for common Claude Code errors
- ✅ Implemented file parsing from Claude's markdown responses

### 2. N8N References Removed From:
- ✅ `README.md` - Replaced with pipeline orchestration references
- ✅ `IMPLEMENTATION_PLAN.md` - Updated architecture descriptions
- ✅ `.vrooli/service.json` - Removed n8n resource configuration
- ✅ `initialization/postgres/schema.sql` - Updated template resources
- ✅ All backlog YAML files - Replaced n8n with claude-code
- ✅ `backlog/templates/scenario-template.yaml` - Updated template
- ❌ Deleted `MIGRATION_FROM_N8N.md` (obsolete file)

### 3. Documentation Updates
- Updated all references from "vrooli resource claude" to "claude-code"
- Clarified that the system uses Go-based pipeline orchestration
- Updated debugging commands and configuration examples

## Usage

### Prerequisites
1. Install claude-code resource:
   ```bash
   vrooli resource install claude-code
   ```

2. Authenticate with Claude:
   ```bash
   claude auth login
   ```

### Running the Scenario Generator

1. Build and start the API:
   ```bash
   cd api
   go build -o scenario-generator-v1-api .
   ./scenario-generator-v1-api
   ```

2. Start the UI:
   ```bash
   cd ui
   npm install
   node server.js
   ```

3. Access the dashboard:
   ```
   http://localhost:3000
   ```

## Testing

Run the integration test to verify Claude Code is working:
```bash
./test-claude-integration.sh
```

## Architecture Overview

The system now uses:
- **Go API** for pipeline orchestration (no n8n dependency)
- **resource-claude-code** for AI-powered scenario generation
- **PostgreSQL** for data storage
- **React UI** for user interface
- **File-based backlog** for queue management

## Known Limitations

1. Claude Code authentication is required for generation features
2. Rate limits apply based on your Claude API plan
3. Generation quality depends on prompt clarity and detail

## Next Steps

To complete the implementation:
1. Implement actual file generation from Claude responses
2. Add validation logic for generated scenarios  
3. Implement pattern learning from successful generations
4. Add retry mechanisms for failed generations

## Support

For issues with:
- Claude Code: Run `resource-claude-code help`
- API: Check logs in the terminal running the API
- UI: Check browser console for errors