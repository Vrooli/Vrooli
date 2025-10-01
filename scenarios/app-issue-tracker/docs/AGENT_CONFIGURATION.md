# Agent Backend Configuration

The App Issue Tracker now supports multiple AI agent backends for issue investigation and fix generation. This provides flexibility to use different AI providers based on availability, cost, or performance requirements.

## Supported Backends

### 1. Resource Codex (Default)
- **CLI Command**: `resource-codex`
- **Description**: Primary AI agent for code analysis and investigation
- **Capabilities**: Investigation, fix generation, code analysis, pattern detection

### 2. Resource Claude Code
- **CLI Command**: `resource-claude-code`
- **Description**: Alternative AI agent with interactive capabilities
- **Capabilities**: Investigation, fix generation, code analysis, pattern detection, interactive fixes

## Configuration

### Settings File Location
```
scenarios/app-issue-tracker/initialization/configuration/agent-settings.json
```

### Default Configuration
```json
{
  "agent_backend": {
    "provider": "codex",
    "fallback_order": ["codex", "claude-code"],
    "auto_fallback": true
  }
}
```

### Configuration Options

#### `agent_backend.provider`
The preferred AI agent to use for investigations.
- **Type**: `string`
- **Options**: `"codex"`, `"claude-code"`
- **Default**: `"codex"`

#### `agent_backend.fallback_order`
Order of backends to try if the primary is unavailable.
- **Type**: `array<string>`
- **Default**: `["codex", "claude-code"]`

#### `agent_backend.auto_fallback`
Whether to automatically try the next backend if the current one fails.
- **Type**: `boolean`
- **Default**: `true`

## How It Works

### Backend Detection
The system automatically detects which agent backends are available:

1. Checks the configured `provider` setting
2. Verifies the CLI command is available in PATH
3. Checks if the resource is running via `status` command
4. If unavailable and `auto_fallback` is enabled, tries next in `fallback_order`

### Unified Interface
Both backends are called using the same interface:
```bash
# Investigation
${AGENT_CLI_COMMAND} content execute --context text --operation analyze "${prompt}"

# Fix Generation
${AGENT_CLI_COMMAND} content execute --context text --operation analyze "${fix_prompt}"
```

## API Endpoints

### Get Agent Settings
```http
GET /api/v1/agent/settings
```

**Response**:
```json
{
  "agent_backend": {
    "provider": "codex",
    "fallback_order": ["codex", "claude-code"],
    "auto_fallback": true
  },
  "providers": {
    "codex": { ... },
    "claude-code": { ... }
  }
}
```

### Update Agent Settings
```http
PATCH /api/v1/agent/settings
Content-Type: application/json

{
  "provider": "claude-code",
  "auto_fallback": false
}
```

**Response**:
```json
{
  "message": "Agent settings updated successfully",
  "settings": { ... }
}
```

## Usage Examples

### Switching to Claude Code
```bash
curl -X PATCH http://localhost:${API_PORT}/api/v1/agent/settings \
  -H "Content-Type: application/json" \
  -d '{"provider": "claude-code"}'
```

### Disabling Auto-Fallback
```json
{
  "provider": "codex",
  "auto_fallback": false
}
```

### Manual Script Execution
The investigation script automatically detects and uses the configured backend:
```bash
./scripts/claude-investigator.sh investigate <issue_id> <agent_id> [project_path]
```

Backend information is shown in the script output:
```
[2025-10-01 12:00:00] Using agent backend: codex (resource-codex)
```

## Provider-Specific Configuration

### Codex Configuration
Located at: `initialization/configuration/codex-config.json`

Includes:
- Model selection (primary, secondary, fast)
- Rate limits
- Cost tracking
- Investigation settings

### Claude Code Configuration
Claude Code uses default settings or can be configured via its own resource configuration.

## Resource Dependencies

Both agent backends are defined in `.vrooli/service.json`:

```json
{
  "resources": {
    "codex": {
      "type": "codex",
      "enabled": true,
      "required": false,
      "description": "AI agent for issue investigation (primary)"
    },
    "claude-code": {
      "type": "claude-code",
      "enabled": true,
      "required": false,
      "description": "AI agent for issue investigation (alternative)"
    }
  }
}
```

## Troubleshooting

### Backend Not Found
```
[ERROR] No available agent backend found. Tried: codex, claude-code
```

**Solutions**:
1. Verify resource is installed: `resource-codex --version` or `resource-claude-code --version`
2. Check resource status: `resource-codex status`
3. Start the resource: `vrooli resource start codex`

### Backend Not Running
```
[WARN] Agent backend 'codex' CLI found but resource not running
```

**Solution**: Start the resource
```bash
resource-codex start
# or
vrooli resource start codex
```

### Fallback in Action
When auto-fallback is enabled, you'll see:
```
[WARN] Agent backend 'codex' CLI not found: resource-codex
[2025-10-01 12:00:00] Using agent backend: claude-code (resource-claude-code)
```

## Best Practices

1. **Enable Auto-Fallback**: Keep `auto_fallback: true` for production to ensure investigations continue even if one backend fails

2. **Monitor Costs**: If using Codex, monitor the cost tracking in `codex-config.json`

3. **Test Both Backends**: Verify both backends work in your environment before relying on auto-fallback

4. **Resource Management**: Ensure at least one agent resource is running before triggering investigations

5. **Configuration Changes**: Changes to `agent-settings.json` take effect immediately on the next investigation

## Integration with UI

The UI can display the current backend and allow users to switch providers through the settings panel by calling the agent settings API endpoints.

Example UI integration:
```javascript
// Get current settings
const response = await fetch('/api/v1/agent/settings');
const settings = await response.json();
console.log('Current provider:', settings.agent_backend.provider);

// Update provider
await fetch('/api/v1/agent/settings', {
  method: 'PATCH',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ provider: 'claude-code' })
});
```
