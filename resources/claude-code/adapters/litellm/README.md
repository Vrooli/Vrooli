# LiteLLM Adapter for Claude Code

## Overview

The LiteLLM adapter provides a seamless fallback mechanism for Claude Code when rate limits are encountered. It automatically routes requests through LiteLLM's unified API interface, allowing continued operation even when Claude's native limits are reached.

## Features

- **Automatic Rate Limit Detection**: Detects when Claude hits usage limits and can automatically switch to LiteLLM
- **Manual Control**: Connect/disconnect from LiteLLM backend on demand
- **Smart Auto-Disconnect**: Automatically reverts to native Claude after a configurable time period
- **Model Mapping**: Translates Claude model names to LiteLLM-compatible versions
- **Cost Tracking**: Monitors usage costs for both native Claude and LiteLLM
- **State Persistence**: Maintains connection state and history across sessions
- **Configuration Management**: Flexible configuration with sensible defaults

## Installation

The adapter is automatically available with the Claude Code resource. To use LiteLLM as a fallback, ensure the LiteLLM resource is running:

```bash
# Check if LiteLLM is available in your Vrooli installation
vrooli resource list

# If available, start the LiteLLM service
resource-litellm start

# Verify it's running
resource-litellm status
```

Note: LiteLLM must be included in your Vrooli resource configuration. If it's not available, you may need to add it to your `.vrooli/service.json` configuration.

## Usage

### Basic Commands

```bash
# Check adapter status
resource-claude-code for litellm status

# Connect to LiteLLM backend
resource-claude-code for litellm connect

# Disconnect and return to native Claude
resource-claude-code for litellm disconnect

# View configuration
resource-claude-code for litellm config show

# Test connection
resource-claude-code for litellm test
```

### Configuration Options

```bash
# Enable/disable auto-fallback on rate limits
resource-claude-code for litellm config set auto-fallback on

# Set preferred model
resource-claude-code for litellm config set model claude-3-5-sonnet-latest

# Set auto-disconnect time (hours)
resource-claude-code for litellm config set auto-disconnect-hours 5

# Enable cost tracking
resource-claude-code for litellm config set cost-tracking on

# Add custom model mapping
resource-claude-code for litellm config set add-model-mapping claude-3-opus:claude-3-opus-20240229
```

### Advanced Connection Options

```bash
# Connect with specific model
resource-claude-code for litellm connect --model claude-3-5-sonnet-latest

# Connect with custom auto-disconnect time
resource-claude-code for litellm connect --auto-disconnect-in 3

# Force reconnection
resource-claude-code for litellm connect --force
```

## Architecture

### Directory Structure

```
adapters/litellm/
├── README.md          # This file
├── state.sh           # State and configuration management
├── status.sh          # Connection status and health checks
├── connect.sh         # Connection establishment
├── disconnect.sh      # Disconnection and cleanup
├── config.sh          # Configuration interface
└── test.sh            # Test suite
```

### State Management

The adapter maintains two JSON files in `~/.claude/`:

1. **litellm_state.json**: Runtime state including connection status, request counts, and history
2. **litellm_config.json**: User configuration including model mappings and preferences

### Integration Flow

1. **Rate Limit Detection**: Claude Code's execute.sh detects rate limits
2. **Automatic Fallback**: If configured, adapter connects to LiteLLM
3. **Request Routing**: Subsequent requests go through LiteLLM
4. **Auto-Disconnect**: After configured time or when limits reset
5. **Return to Native**: Seamlessly switches back to native Claude

## Model Mappings

Default model mappings:

| Claude Model | LiteLLM Model (Ollama) |
|-------------|------------------------|
| claude-3-5-sonnet | qwen2.5-coder |
| claude-3-5-sonnet-latest | qwen2.5-coder |
| claude-3-haiku | qwen2.5 |
| claude-3-opus | llama3.1-8b |

## Configuration Reference

### Auto-Fallback Settings

- `auto_fallback_enabled`: Enable automatic switching on rate limits (default: true)
- `auto_fallback_on_rate_limit`: Trigger on rate limit errors (default: true)
- `auto_fallback_on_error`: Trigger on other errors (default: false)
- `auto_disconnect_after_hours`: Hours until auto-disconnect (default: 5)

### Model Configuration

- `preferred_model`: Default model to use with LiteLLM
- `model_mappings`: Custom model name mappings

### Cost Tracking

- `cost_tracking.enabled`: Enable cost monitoring (default: true)
- `cost_tracking.native_claude_cost`: Accumulated native Claude costs
- `cost_tracking.litellm_cost`: Accumulated LiteLLM costs

## Testing

Run the test suite to verify adapter functionality:

```bash
resources/claude-code/adapters/litellm/test.sh
```

Tests cover:
- State initialization
- Configuration management
- Connection state transitions
- Request counting
- Error recording
- Model mapping
- Auto-disconnect detection

## Troubleshooting

### LiteLLM Not Available

If you see "LiteLLM Resource: Not available":
1. Check available resources: `vrooli resource list`
2. If LiteLLM is listed, start it: `resource-litellm start`
3. Verify it's running: `resource-litellm status`
4. If not listed, add it to `.vrooli/service.json` with `"enabled": true`

### Connection Test Fails

If connection tests fail:
1. Check LiteLLM is running: `resource-litellm status`
2. Verify API keys are configured
3. Test LiteLLM directly: `resource-litellm run --model claude-3-5-sonnet --prompt "test"`

### Auto-Fallback Not Working

If automatic fallback doesn't trigger:
1. Verify it's enabled: `resource-claude-code for litellm config get auto-fallback`
2. Check rate limit detection: `resource-claude-code test-rate-limit`
3. Review logs for errors

## Future Enhancements

Planned improvements for subsequent phases:

- **Phase 2**: Automatic rate limit triggered connection
- **Phase 3**: Advanced prompt/response translation layer
- **Phase 4**: Tool call format conversion
- **Phase 5**: Performance metrics and quality tracking
- **Phase 6**: Multi-provider support beyond LiteLLM

## Contributing

When modifying the adapter:
1. Follow the established pattern from other adapters (e.g., browserless)
2. Update tests in test.sh
3. Document new configuration options
4. Maintain backwards compatibility

## Support

For issues or questions:
- Check adapter status: `resource-claude-code for litellm status`
- Review configuration: `resource-claude-code for litellm config show`
- Run tests: `./test.sh`
- Check Claude Code logs: `resource-claude-code logs`