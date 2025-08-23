# LiteLLM Resource

## Overview
LiteLLM is a self-hosted unified proxy server that provides OpenAI-compatible access to 100+ LLM APIs through a single endpoint. It acts as an intelligent gateway that can route, load balance, and manage requests across multiple AI providers including OpenAI, Anthropic, Google, OpenRouter, and many others.

## Why LiteLLM?

### Key Benefits
- **Unified Interface**: Single OpenAI-compatible API for 100+ models from different providers
- **Intelligent Routing**: Cost-based, latency-based, or load-based routing strategies
- **Fallback Support**: Automatic failover when primary models are unavailable
- **Cost Control**: Built-in budget management and rate limiting
- **Provider Independence**: Avoid vendor lock-in with multi-provider support
- **Self-Hosted**: Complete control over your data and API routing
- **Real-time Analytics**: Track usage, costs, and performance across all providers

### Use Cases in Vrooli
- **Claude Code Gateway**: Use LiteLLM as an API gateway for Claude Code with OpenRouter/other providers
- **Multi-Provider Scenarios**: Build scenarios that intelligently route between providers
- **Cost Optimization**: Automatically route simple tasks to cheaper models
- **High Availability**: Ensure AI features work even when primary providers are down
- **Model Experimentation**: Test same prompts across multiple models and providers
- **Privacy Control**: Keep sensitive prompts on-premise while accessing remote models

## Quick Start

```bash
# Install LiteLLM proxy server
vrooli resource litellm install --verbose

# Check status
vrooli resource litellm status

# Configure provider API keys (choose one method):
# Method 1: Environment file
vim ${var_ROOT_DIR}/data/litellm/config/.env

# Method 2: Vault (recommended)
vault kv put secret/vrooli/openai api_key="sk-your-key"
vault kv put secret/vrooli/anthropic api_key="sk-ant-your-key"
vault kv put secret/vrooli/openrouter api_key="sk-or-your-key"

# Test the installation
resource-litellm test
resource-litellm list-models
resource-litellm test-model gpt-3.5-turbo

# Use with Claude Code
export ANTHROPIC_BASE_URL="http://localhost:11435"
export ANTHROPIC_AUTH_TOKEN="$(grep LITELLM_MASTER_KEY ${var_ROOT_DIR}/data/litellm/config/.env | cut -d'=' -f2)"
```

## Architecture

### Self-Hosted Proxy Server
Unlike external API services, LiteLLM runs as a local Docker container that:
- Manages connections to multiple upstream AI providers
- Provides intelligent routing and load balancing
- Maintains local request/response logs and analytics
- Offers complete control over data flow and provider selection

### Multi-Provider Management
LiteLLM can simultaneously connect to:
- **OpenAI**: GPT-3.5, GPT-4, and newer models
- **Anthropic**: Claude 3 family (Haiku, Sonnet, Opus)
- **Google**: Gemini Pro and other models
- **OpenRouter**: Access to 300+ models through unified API
- **Local Models**: Ollama and other local inference servers
- **Custom Providers**: Easy integration of new API endpoints

### Intelligent Routing Strategies
1. **Simple Shuffle**: Weighted selection based on TPM/RPM limits
2. **Cost-Based**: Route to cheapest provider that meets requirements
3. **Latency-Based**: Route to fastest responding provider
4. **Least-Busy**: Route to provider with lowest current load
5. **Usage-Based**: Route based on current usage relative to limits

## Configuration

### API Key Setup
Configure your provider API keys using one of these methods:

#### Method 1: Environment File
```bash
# Edit the environment file
vim ${var_ROOT_DIR}/data/litellm/config/.env

# Add your API keys
OPENAI_API_KEY=sk-your-openai-key
ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
OPENROUTER_API_KEY=sk-or-your-openrouter-key
GOOGLE_API_KEY=your-google-key
```

#### Method 2: Vault (Recommended)
```bash
# Store keys in Vault for better security
vault kv put secret/vrooli/openai api_key="sk-your-key"
vault kv put secret/vrooli/anthropic api_key="sk-ant-your-key"
vault kv put secret/vrooli/openrouter api_key="sk-or-your-key"
vault kv put secret/vrooli/google api_key="your-google-key"
```

### Custom Configuration
```bash
# Add custom configuration
resource-litellm content add --type config --file my-config.yaml --name custom

# Apply custom configuration
resource-litellm content execute --type config --name custom

# List configurations
resource-litellm content list --type config
```

## Integration Examples

### With Claude Code
```bash
# Set up Claude Code to use LiteLLM proxy
export ANTHROPIC_BASE_URL="http://localhost:11435"
export ANTHROPIC_AUTH_TOKEN="your-litellm-master-key"

# Now Claude Code will route through LiteLLM to your configured providers
claude "Generate a Python script"
```

### With N8n Workflows
```json
{
  "nodes": [
    {
      "name": "LiteLLM Chat",
      "type": "n8n-nodes-base.httpRequest",
      "parameters": {
        "url": "http://vrooli-litellm:4000/chat/completions",
        "method": "POST",
        "headers": {
          "Authorization": "Bearer ${LITELLM_MASTER_KEY}",
          "Content-Type": "application/json"
        },
        "body": {
          "model": "gpt-3.5-turbo",
          "messages": [{"role": "user", "content": "Hello"}]
        }
      }
    }
  ]
}
```

### With Scenario Generation
```bash
# Use LiteLLM for scenario generation with cost optimization
vrooli scenario generate \
  --ai-provider "litellm" \
  --model "auto" \
  --routing "cost-based" \
  --type "documentation"
```

## Available Models

LiteLLM provides access to models from multiple providers. Use `resource-litellm list-models` to see currently available models.

### Example Model Categories

**OpenAI Models:**
- `gpt-3.5-turbo` - Fast, cost-effective for most tasks
- `gpt-4` - Best reasoning and complex tasks
- `gpt-4-turbo` - Faster GPT-4 with larger context

**Anthropic Models:**
- `claude-3-haiku` - Fastest, most cost-effective Claude
- `claude-3-sonnet` - Balanced performance and cost
- `claude-3-opus` - Most capable Claude model

**Google Models:**
- `gemini-pro` - Google's flagship model
- `gemini-flash` - Fast, efficient responses

**OpenRouter Models:**
- Access to 300+ models through OpenRouter integration
- Automatic fallback when primary providers are unavailable

## Management Commands

### Service Management
```bash
# Start/stop the proxy server
resource-litellm start --verbose
resource-litellm stop
resource-litellm restart

# Check service status
resource-litellm status
resource-litellm status-detailed

# Monitor logs
resource-litellm logs --follow
resource-litellm stats
```

### Testing and Validation
```bash
# Test proxy connectivity
resource-litellm test

# Test specific models
resource-litellm test-model gpt-3.5-turbo
resource-litellm test-model claude-3-haiku

# Get proxy metrics
resource-litellm proxy-status

# Run integration tests
resource-litellm run-tests
```

### Content Management
```bash
# Add custom configurations
resource-litellm content add --type config --file config.yaml --name production
resource-litellm content add --type provider --file provider.json --name custom-ai

# List and manage content
resource-litellm content list
resource-litellm content get --type config --name production
resource-litellm content execute --type config --name production

# Remove content
resource-litellm content remove --type config --name old-config
```

### Backup and Maintenance
```bash
# Create backup
resource-litellm backup --verbose

# Update to latest version
resource-litellm update --verbose

# Reset to defaults (keeps data)
resource-litellm reset --verbose

# Full reset (removes all data)
resource-litellm reset --purge-data --verbose
```

## Monitoring & Analytics

### Built-in Metrics
- Request counts and success rates per provider
- Response times and latencies
- Cost tracking across all providers
- Model usage statistics
- Error rates and failure patterns

### Accessing Metrics
```bash
# Get current proxy status
resource-litellm proxy-status

# View detailed status
resource-litellm status-detailed

# Monitor resource usage
resource-litellm stats

# Check logs for debugging
resource-litellm logs 100
```

## Troubleshooting

### Common Issues

1. **"Container not found" error**
   ```bash
   # Reinstall LiteLLM
   resource-litellm install --force --verbose
   ```

2. **"API key not configured" errors**
   ```bash
   # Check API key configuration
   resource-litellm status-detailed
   
   # Reconfigure keys
   vim ${var_ROOT_DIR}/data/litellm/config/.env
   resource-litellm restart
   ```

3. **Model not available**
   ```bash
   # Check available models
   resource-litellm list-models
   
   # Test specific provider
   resource-litellm test-model gpt-3.5-turbo
   ```

4. **Connection timeouts**
   ```bash
   # Check proxy health
   resource-litellm test --timeout 30
   
   # Check container logs
   resource-litellm logs 50
   ```

5. **Rate limiting issues**
   - Configure appropriate TPM/RPM limits in configuration
   - Use multiple providers for higher throughput
   - Enable intelligent routing to distribute load

### Debug Mode
```bash
# Enable verbose logging
resource-litellm logs --follow

# Get container details
resource-litellm inspect

# Execute commands inside container
resource-litellm exec bash
```

## Configuration Files

### Core Configuration
- **Config**: `${var_ROOT_DIR}/data/litellm/config/config.yaml`
- **Environment**: `${var_ROOT_DIR}/data/litellm/config/.env`
- **Data**: `${var_ROOT_DIR}/data/litellm/data/`
- **Logs**: `${var_ROOT_DIR}/data/litellm/logs/`

### Network Configuration
- **Service URL**: `http://localhost:11435`
- **Internal Port**: `4000`
- **Container Name**: `vrooli-litellm`
- **Network**: `vrooli-network`

## Documentation
- [API Documentation](docs/api.md)
- [Provider Configuration](docs/providers.md)
- [Routing Strategies](docs/routing.md)
- [Integration Guide](docs/integration.md)
- [Advanced Configuration](docs/advanced.md)

## Support
- Container Image: https://github.com/BerriAI/litellm
- Official Docs: https://docs.litellm.ai/
- API Reference: https://docs.litellm.ai/docs/proxy/
- Provider Support: https://docs.litellm.ai/docs/providers