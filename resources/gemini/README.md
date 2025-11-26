# Gemini Resource

Google's Gemini AI API integration for Vrooli.

## Overview

The Gemini resource provides access to Google's multimodal AI models through their API, supporting text generation, vision capabilities, and more.

## Features

- Text generation with Gemini Pro
- Vision capabilities with Gemini Pro Vision
- Credential management via Vault
- Response caching with Redis integration
- Token usage tracking and cost estimation
- Integration with automation platforms (n8n, Huginn, etc.)
- JSON and text output formats

## Installation

```bash
# Install the resource
resource-gemini install

# Or using vrooli CLI
vrooli resource install gemini
```

## Configuration

### API Key Setup

1. Get an API key from [Google AI Studio](https://makersuite.google.com/app/apikey)

2. Store the key in Vault (recommended):
```bash
docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv put secret/vrooli/gemini api_key='your-api-key'"
```

3. Or set as environment variable:
```bash
export GEMINI_API_KEY="your-api-key"
```

## Usage

### Check Status
```bash
resource-gemini status
resource-gemini status --verbose
resource-gemini status --format json
```

### List Available Models
```bash
resource-gemini list-models
```

### Generate Content
```bash
resource-gemini generate "Explain quantum computing"
resource-gemini generate "What is in this image?" gemini-pro-vision
```

### Cache Management
```bash
# View cache statistics
resource-gemini cache-stats

# Clear cache
resource-gemini cache-clear
```

### Token Usage Management
```bash
# View token usage statistics
resource-gemini token-stats

# Estimate costs based on usage
resource-gemini token-cost

# Clear token usage history
resource-gemini token-clear
```

### Test Connection
```bash
resource-gemini test
```

### Inject into Other Resources
```bash
resource-gemini inject n8n
resource-gemini inject huginn
```

## Models

- `gemini-pro` - Text generation
- `gemini-pro-vision` - Multimodal (text + vision)

## Environment Variables

### Core Configuration
- `GEMINI_API_KEY` - Google AI API key
- `GEMINI_API_BASE` - API base URL (default: https://generativelanguage.googleapis.com/v1beta)
- `GEMINI_DEFAULT_MODEL` - Default model (default: gemini-pro)
- `GEMINI_TIMEOUT` - Request timeout in seconds (default: 30)

### Cache Configuration
- `GEMINI_CACHE_ENABLED` - Enable Redis caching (default: true)
- `GEMINI_CACHE_TTL` - Cache time-to-live in seconds (default: 3600)
- `GEMINI_CACHE_PREFIX` - Cache key prefix (default: gemini:cache)
- `REDIS_HOST` - Redis host (default: localhost)
- `REDIS_PORT` - Redis port (default: 6379)

### Token Tracking
- `GEMINI_TOKEN_TRACKING_ENABLED` - Enable token usage tracking (default: true)
- `GEMINI_TOKEN_LOG_FILE` - Token usage log file path

## Integration

The Gemini resource can be integrated with:
- n8n workflows
- Huginn agents
- Automation scripts
- Node-RED flows

## Troubleshooting

### API Key Issues
- Ensure your API key is valid and has proper permissions
- Check if the key is properly stored in Vault or environment

### Connection Issues
- Verify network connectivity to Google's API servers
- Check firewall/proxy settings

### Rate Limiting
- Google AI has rate limits on API usage
- Consider implementing retry logic in workflows
