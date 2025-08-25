# Cline Resource

VS Code AI coding assistant powered by Claude and other LLMs.

## Overview

Cline (formerly Claude Dev) is a powerful VS Code extension that brings AI-powered coding assistance directly into your IDE. It supports multiple AI providers including OpenRouter, Anthropic, and local models via Ollama.

## Installation

```bash
# Install Cline
vrooli resource cline install

# Or using the direct CLI
./resources/cline/cli.sh install
```

## Configuration

Cline can be configured to use different AI providers:

```bash
# Configure with OpenRouter (default)
vrooli resource cline inject api openrouter

# Configure with Anthropic
vrooli resource cline inject api anthropic YOUR_API_KEY

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

# Detailed health check
vrooli resource cline health
```

### Inject Prompts

```bash
# Inject custom prompts
vrooli resource cline inject prompts /path/to/prompts/

# Inject model configuration
vrooli resource cline inject model gpt-4 0.7 4096
```

### Update

```bash
# Update to latest version
vrooli resource cline update
```

## Integration with Vrooli

Cline integrates with other Vrooli resources:

- **Vault**: Automatically fetches API keys from Vault if available
- **Ollama**: Can use local models running in Ollama
- **OpenRouter**: Default provider for accessing multiple AI models

## Environment Variables

- `CLINE_DEFAULT_PROVIDER`: Default API provider (openrouter, anthropic, openai, ollama)
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
2. Environment variables (OPENROUTER_API_KEY, ANTHROPIC_API_KEY, etc.)

### Extension Not Loading

Try reinstalling with force:
```bash
vrooli resource cline install --force
```