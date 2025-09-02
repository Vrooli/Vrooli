# OpenRouter Resource

## Overview
OpenRouter is a unified AI model gateway that provides access to 100+ models from OpenAI, Anthropic, Google, Meta, and more through a single API. It acts as a smart router that can automatically select the best model for your needs based on cost, speed, and capability requirements.

## Why OpenRouter?

### Key Benefits
- **Model Diversity**: Access GPT-4, Claude, Gemini, Llama, Mistral, and dozens more through one API
- **Cost Optimization**: Automatically route to cheaper models when appropriate
- **Fallback Handling**: Automatic failover when primary models are unavailable  
- **Usage Analytics**: Track spending and usage across all models
- **Rate Limit Management**: Built-in queuing and retry logic
- **Provider Redundancy**: No single point of failure across AI providers

### Use Cases in Vrooli
- **Scenario Generation**: Use specialized models for different scenario types
- **Cost-Sensitive Tasks**: Route simple tasks to cheaper models automatically
- **High-Availability**: Ensure AI features work even when primary providers are down
- **Model Comparison**: Test same prompt across multiple models for quality assessment
- **Specialized Tasks**: Access domain-specific models (code, math, creative writing)

## Quick Start

```bash
# Check status
vrooli resource openrouter status

# Configure with API key
vrooli resource openrouter configure --api-key "your-key-here"

# Or store in Vault (recommended)
vault kv put secret/openrouter api_key="your-key-here"

# Test the configuration
vrooli resource openrouter test

# List available models
vrooli resource openrouter list-models

# Test a specific model
vrooli resource openrouter test-model "anthropic/claude-3-opus"
```

## Configuration

### API Key Setup
1. Get a free API key from [OpenRouter](https://openrouter.ai)
2. Add credits ($5 minimum recommended)
3. Configure using one of these methods:
   - Environment variable: `export OPENROUTER_API_KEY="sk-or-..."`
   - Vault (recommended): `vault kv put secret/openrouter api_key="sk-or-..."`
   - Direct config: `vrooli resource openrouter configure --api-key "sk-or-..."`

### Model Selection
OpenRouter supports automatic model selection based on your requirements:
- **Speed**: Route to fastest available model
- **Cost**: Route to cheapest model that meets quality threshold
- **Quality**: Route to best model within budget
- **Capability**: Route to models with specific features (vision, coding, etc.)

## Integration Examples

### With N8n Workflows
```json
{
  "model": "auto",
  "route": "fallback",
  "models": [
    "anthropic/claude-3-opus",
    "openai/gpt-4-turbo",
    "google/gemini-pro"
  ]
}
```

### With AI Scenarios
```bash
# Use specialized model for AI-powered scenarios
vrooli scenario run ai-code-generator

# Use efficient models for documentation tasks
vrooli scenario run document-processor
```

## Available Models (Sample)

### Premium Models
- `anthropic/claude-3-opus` - Best for complex reasoning
- `openai/gpt-4-turbo` - Versatile, multimodal
- `google/gemini-pro` - Good for long context

### Cost-Effective Models  
- `mistralai/mistral-7b` - Fast, cheap, capable
- `meta-llama/llama-3-70b` - Open source powerhouse
- `anthropic/claude-instant` - Quick responses

### Specialized Models
- `anthropic/claude-3-opus:beta` - Coding specialist
- `perplexity/sonar-medium` - Web-aware responses
- `cohere/command-r` - RAG-optimized

## Monitoring & Analytics

Check usage and costs:
```bash
# Show current usage
vrooli resource openrouter usage

# Show cost breakdown by model
vrooli resource openrouter costs --breakdown

# Show rate limit status
vrooli resource openrouter limits
```

## Troubleshooting

### Common Issues

1. **"Placeholder key" error**
   - Solution: Configure a real API key (see Configuration section)

2. **Rate limits**
   - OpenRouter handles this automatically with queuing
   - Consider upgrading tier for higher limits

3. **Model not available**
   - Some models require specific account tiers
   - Check model availability: `vrooli resource openrouter list-models`

## Documentation
- [API Documentation](docs/api.md)
- [Model Comparison](docs/models.md)
- [Integration Guide](docs/integration.md)
- [Cost Optimization](docs/cost-optimization.md)

## Support
- OpenRouter Dashboard: https://openrouter.ai/dashboard
- API Docs: https://openrouter.ai/docs
- Model Playground: https://openrouter.ai/playground