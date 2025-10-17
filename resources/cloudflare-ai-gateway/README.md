# Cloudflare AI Gateway Resource

## Overview
The Cloudflare AI Gateway resource provides a resilient proxy layer for AI traffic with caching, rate limiting, analytics, retries, and model fallbacks. It acts as an intelligent intermediary between Vrooli's AI resources and external providers, optimizing costs and improving reliability.

## Key Features
- **Intelligent Caching**: Reduce API costs by caching repeated requests
- **Rate Limiting**: Prevent API quota exhaustion
- **Automatic Retries**: Handle transient failures gracefully
- **Provider Fallbacks**: Switch between providers when one fails
- **Analytics**: Track usage, costs, and performance
- **Request Logging**: Debug and audit AI interactions

## Benefits for Vrooli
- **Cost Reduction**: 30-50% reduction in AI API costs through caching
- **Improved Reliability**: Automatic failover between providers
- **Better Observability**: Detailed analytics and logging
- **Quota Management**: Prevent hitting provider rate limits
- **Unified Interface**: Single point for all AI traffic

## Quick Start

### Prerequisites
1. Cloudflare account (free tier available)
2. API token with AI Gateway permissions
3. Account ID from Cloudflare dashboard

### Installation
```bash
vrooli resource install cloudflare-ai-gateway
```

### Configuration
```bash
# Store credentials in Vault (recommended)
resource-vault set cloudflare_account_id YOUR_ACCOUNT_ID
resource-vault set cloudflare_api_token YOUR_API_TOKEN

# Or use environment variables
export CLOUDFLARE_ACCOUNT_ID=YOUR_ACCOUNT_ID
export CLOUDFLARE_API_TOKEN=YOUR_API_TOKEN
```

### Basic Usage
```bash
# Check status
resource-cloudflare-ai-gateway status

# Start the gateway
resource-cloudflare-ai-gateway start

# Configure providers
resource-cloudflare-ai-gateway configure --provider openrouter --cache-ttl 3600

# View logs and analytics
resource-cloudflare-ai-gateway logs
```

## Content Management
The gateway supports configuration management for rules, providers, and policies:

```bash
# Add a configuration
resource-cloudflare-ai-gateway content add --file gateway-rules.json

# List configurations
resource-cloudflare-ai-gateway content list

# Apply a configuration
resource-cloudflare-ai-gateway content execute --name production-rules
```

## Integration with Other Resources

### OpenRouter
Configure OpenRouter to route through the gateway:
```bash
# Set gateway URL in OpenRouter configuration
resource-openrouter configure --gateway-url https://gateway.ai.cloudflare.com/v1/YOUR_GATEWAY_ID
```

### Ollama
For external model access through Ollama:
```bash
# Configure Ollama to use gateway for remote models
resource-ollama configure --remote-gateway cloudflare
```

### Cline/Agents
Agents automatically benefit from the gateway's caching and fallback features.

## Example Configurations

### Basic Caching Rule
```json
{
  "rules": [
    {
      "pattern": "/chat/completions",
      "cache": {
        "enabled": true,
        "ttl": 3600,
        "key": ["model", "messages"]
      }
    }
  ]
}
```

### Provider Fallback Chain
```json
{
  "providers": [
    {
      "name": "openrouter",
      "priority": 1,
      "timeout": 30000
    },
    {
      "name": "openai",
      "priority": 2,
      "timeout": 30000
    }
  ],
  "fallback": {
    "enabled": true,
    "retry_count": 3
  }
}
```

## Troubleshooting

### Gateway not responding
1. Check credentials: `resource-cloudflare-ai-gateway status`
2. Verify network connectivity
3. Check Cloudflare dashboard for service status

### High cache miss rate
1. Review cache key configuration
2. Increase TTL for stable queries
3. Check request variations

### Rate limiting issues
1. Adjust rate limits in configuration
2. Enable request queuing
3. Consider upgrading Cloudflare plan

## Architecture
```
┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│   Vrooli    │────▶│  CF AI Gateway   │────▶│  Providers   │
│  Resources  │     │  (Proxy Layer)   │     │ (OpenRouter, │
└─────────────┘     └──────────────────┘     │   OpenAI)    │
                            │                 └──────────────┘
                            ▼
                    ┌──────────────────┐
                    │  Cache & Analytics│
                    └──────────────────┘
```

## Links
- [Cloudflare AI Gateway Docs](https://developers.cloudflare.com/ai-gateway/)
- [API Token Management](https://dash.cloudflare.com/profile/api-tokens)
- [Gateway Dashboard](https://dash.cloudflare.com/ai-gateway)