# Codex Resource

AI-powered code completion and generation via OpenAI Codex API.

## Overview

Codex provides intelligent code suggestions and completions for multiple programming languages. It enables automated code generation, completion, and transformation capabilities for scenarios and workflows.

## Features

- Code completion and generation
- Multi-language support (Python, JavaScript, Go, etc.)
- Script injection and management
- API-based processing with configurable parameters
- Integration with Vault for secure credential storage

## Requirements

- OpenAI API key with Codex access
- Network connectivity to OpenAI API endpoints
- jq for JSON processing

## Configuration

### API Key Setup

The resource checks for API keys in this order:
1. Environment variable: `OPENAI_API_KEY`
2. Vault secret: `secret/openai` (field: `api_key`)
3. Credentials file: `~/.openai/credentials`

### Environment Variables

- `CODEX_API_ENDPOINT`: API endpoint (default: https://api.openai.com/v1)
- `CODEX_DEFAULT_MODEL`: Default model (default: code-davinci-002)
- `CODEX_DEFAULT_TEMPERATURE`: Generation temperature (default: 0.0)
- `CODEX_DEFAULT_MAX_TOKENS`: Maximum tokens (default: 2048)
- `CODEX_TIMEOUT`: API timeout in seconds (default: 30)

## Usage

### Basic Commands

```bash
# Check status
resource-codex status

# Start service (mark as running)
resource-codex start

# Stop service (mark as stopped)
resource-codex stop

# List injected scripts
resource-codex list

# Inject a Python script
resource-codex inject my_script.py

# Run a script with Codex
resource-codex run my_script.py
```

### Status Check

```bash
# Text format
resource-codex status

# JSON format
resource-codex status --format json
```

### Script Injection

Scripts are stored in `~/.codex/scripts/` for processing:

```bash
# Inject a new script
resource-codex inject path/to/script.py

# List all injected scripts
resource-codex list

# Run a specific script
resource-codex run script.py
```

## Directory Structure

- `config/` - Configuration defaults
- `lib/` - Core functionality libraries
- `injected/` - Backup of injected scripts
- `~/.codex/scripts/` - Active script storage
- `~/.codex/outputs/` - Generated code outputs

## Integration

### With Scenarios

Codex can be used in scenarios for:
- Generating boilerplate code
- Converting between languages
- Creating test cases
- Implementing algorithms
- Code refactoring

### With Other Resources

- **Vault**: Secure API key storage
- **Judge0**: Execute generated code
- **n8n/Node-RED**: Workflow automation with code generation
- **Ollama**: Fallback for local code generation

## Troubleshooting

### API Key Issues

```bash
# Check if API key is configured
resource-codex status | grep "API Configured"

# Test API connectivity
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
  https://api.openai.com/v1/models
```

### Model Access

Ensure your API key has access to Codex models. Some models require special access or may be deprecated.

## Security

- API keys are never logged or displayed
- Use Vault for production deployments
- Generated code is stored locally
- No automatic code execution without explicit user action