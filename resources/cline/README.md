# Cline Resource

VS Code AI coding assistant powered by Claude and other LLMs.

## Overview

Cline (formerly Claude Dev) is a powerful VS Code extension that brings AI-powered coding assistance directly into your IDE. It supports multiple AI providers including OpenRouter and local models via Ollama.

## Installation

```bash
# Install Cline (v2.0 compliant)
vrooli resource cline manage install

# Or using the direct CLI
./resources/cline/cli.sh manage install
```

## Configuration

Cline can be configured to use different AI providers:

```bash
# Configure with OpenRouter (default)
vrooli resource cline content execute provider openrouter

# Configure with Ollama for local models
export CLINE_USE_OLLAMA=true
export CLINE_OLLAMA_BASE_URL=http://localhost:11434
vrooli resource cline configure
```

## Usage

### Check Status

```bash
# Check if Cline is installed and configured
vrooli resource cline status

# Run smoke tests for quick health check
vrooli resource cline test smoke
```

### Resource Integrations

```bash
# List available integrations
vrooli resource cline integrate list

# Enable integrations
vrooli resource cline integrate enable judge0    # Auto-generate tests
vrooli resource cline integrate enable ollama    # Local LLM models
vrooli resource cline integrate enable redis     # Response caching

# Execute integration commands
vrooli resource cline integrate execute judge0 test script.py
```

### Performance Optimization

```bash
# View cache statistics
vrooli resource cline cache stats

# Enable/disable caching
vrooli resource cline cache enable
vrooli resource cline cache disable

# Clear cache
vrooli resource cline cache clear
```

### Terminal Integration

```bash
# List available AI models
vrooli resource cline content execute models

# Switch provider
vrooli resource cline content execute provider ollama

# Send prompt from terminal (future integration)
vrooli resource cline content execute prompt "Explain this code"
```

### Workspace Context

```bash
# Load workspace context
vrooli resource cline content execute context load /path/to/project

# Show active context
vrooli resource cline content execute context show

# List all contexts
vrooli resource cline content execute context list

# Clear context
vrooli resource cline content execute context clear
```

### Usage Analytics

```bash
# View usage statistics
vrooli resource cline content execute analytics show

# Track a session (automatically done by Cline)
vrooli resource cline content execute analytics track ollama llama3.2 1500 0.001

# Reset analytics
vrooli resource cline content execute analytics reset
```

### Batch Operations

```bash
# Analyze multiple files
vrooli resource cline content execute batch analyze "*.js" /path/to/dir

# Create batch processing job
vrooli resource cline content execute batch process "*.py" refactor

# Check batch job status
vrooli resource cline content execute batch status
```

### Custom Instructions

```bash
# Add custom instruction
vrooli resource cline content execute instructions add "code-style" "Use 2 spaces for indentation"

# List instructions
vrooli resource cline content execute instructions list

# Show specific instruction
vrooli resource cline content execute instructions show code-style

# Activate instruction
vrooli resource cline content execute instructions activate code-style

# Remove instruction
vrooli resource cline content execute instructions remove code-style
```

### Lifecycle Management

```bash
# Start Cline (ensure configuration is active)
vrooli resource cline manage start

# Stop Cline (clean shutdown)
vrooli resource cline manage stop

# Restart Cline
vrooli resource cline manage restart

# Uninstall Cline
vrooli resource cline manage uninstall
```

## Integration with Vrooli

Cline integrates with other Vrooli resources:

- **Vault**: Automatically fetches API keys from Vault if available
- **Ollama**: Can use local models running in Ollama
- **OpenRouter**: Default provider for accessing multiple AI models

## Environment Variables

- `CLINE_DEFAULT_PROVIDER`: Default API provider (openrouter, ollama)
- `CLINE_DEFAULT_MODEL`: Default model to use
- `CLINE_USE_OLLAMA`: Enable Ollama integration (true/false)
- `CLINE_OLLAMA_BASE_URL`: Ollama server URL
- `CLINE_AUTO_APPROVE`: Auto-approve AI suggestions (use with caution)

## Troubleshooting

### VS Code Not Found

If VS Code is not detected, ensure it's installed and the `code` command is available in your PATH.

### API Key Issues

Cline will attempt to fetch API keys from:
1. Vault (if available)
2. Environment variables (OPENROUTER_API_KEY for OpenRouter)

### Extension Not Loading

Try reinstalling with clean option:
```bash
vrooli resource cline manage uninstall
vrooli resource cline manage install
```