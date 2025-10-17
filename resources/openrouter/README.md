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

# List available models (text)
vrooli resource openrouter content models --limit 25

# Structured JSON (includes provider, pricing, context length)
vrooli resource openrouter content models --json | jq '.models[0]'

# Test a specific model
vrooli resource openrouter test-model "anthropic/claude-3-opus"

# View usage analytics
vrooli resource openrouter usage today      # Today's usage
vrooli resource openrouter usage week       # Last 7 days
vrooli resource openrouter usage month      # Last 30 days
vrooli resource openrouter usage all        # All-time usage

# Benchmark model performance (NEW)
vrooli resource openrouter benchmark test "openai/gpt-4"   # Test single model
vrooli resource openrouter benchmark compare               # Compare default models
vrooli resource openrouter benchmark list                  # List past benchmarks
```

## Configuration

### API Key Setup
1. Get a free API key from [OpenRouter](https://openrouter.ai)
2. Add credits ($5 minimum recommended)
3. Configure using one of these methods:
   - Environment variable: `export OPENROUTER_API_KEY="sk-or-..."`
   - Vault (recommended): `vault kv put secret/openrouter api_key="sk-or-..."`
   - Direct config: `vrooli resource openrouter configure --api-key "sk-or-..."`

### Model Selection (Enhanced 2025-09-11)
OpenRouter now supports intelligent automatic model selection and fallback chains:

#### Automatic Selection Strategies
- **Auto**: Intelligently selects based on task type (code, creative, analysis, etc.)
- **Cheapest**: Finds the most cost-effective model for your requirements
- **Fastest**: Routes to models with lowest latency
- **Quality**: Selects highest capability models within budget

#### Fallback Chains
The resource now automatically handles model failures with configurable fallback chains:
```bash
# Test with automatic fallback
vrooli resource openrouter test-model "primary-model" --fallback "backup1,backup2"

# List models by category
vrooli resource openrouter list-models --category code  # coding models
vrooli resource openrouter list-models --category cheap # budget models
vrooli resource openrouter list-models --category fast  # low-latency models
```

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

### Usage Tracking
The resource automatically tracks all API usage including tokens, costs, and models used:

```bash
# View usage for different periods
vrooli resource openrouter usage today      # Today's usage
vrooli resource openrouter usage week       # Last 7 days
vrooli resource openrouter usage month      # Last 30 days
vrooli resource openrouter usage all        # All-time usage
```

### Performance Benchmarking (NEW - v1.1.0)
Compare model performance to make informed decisions:

```bash
# Benchmark a single model
vrooli resource openrouter benchmark test "openai/gpt-4"

# Compare multiple models
vrooli resource openrouter benchmark compare "openai/gpt-3.5-turbo" "anthropic/claude-3-haiku" "mistralai/mistral-7b"

# Use default comparison set
vrooli resource openrouter benchmark compare

# View past benchmark results
vrooli resource openrouter benchmark list

# Clean old benchmarks (>30 days)
vrooli resource openrouter benchmark clean
```

Benchmarks measure:
- Response time (milliseconds)
- Token throughput (tokens/second)
- Success rate
- Relative performance

### Rate Limit Management
Built-in rate limit handling with automatic queuing:
- Requests are automatically queued when rate limited
- Exponential backoff for retries
- Priority queuing (high/normal/low)
- Automatic processing when limits reset

### Cost Optimization
The resource includes automatic model selection for cost optimization:
- Auto-selects cheapest model that meets requirements
- Fallback chains for failed requests
- Usage analytics to track spending

## Custom Routing Rules (NEW - v1.2.0)

Define custom rules to automatically select models based on prompt characteristics:

### Managing Routing Rules
```bash
# List all routing rules
vrooli resource openrouter routing list

# Show routing rule template
vrooli resource openrouter routing template

# Add a new routing rule from JSON file
vrooli resource openrouter routing add my-rule rule.json

# Enable/disable rules
vrooli resource openrouter routing enable code-specialist
vrooli resource openrouter routing disable fast-response

# Remove a rule
vrooli resource openrouter routing remove my-rule

# Test routing with a prompt
vrooli resource openrouter routing test "Write a Python function"

# Evaluate routing for a prompt (returns selected model)
vrooli resource openrouter routing evaluate "Analyze this data"

# View routing history
vrooli resource openrouter routing history 20
```

### Example Routing Rules

#### Cost Optimizer (Default Enabled)
```json
{
  "name": "cost-optimizer",
  "description": "Select cheapest model for simple tasks",
  "priority": 100,
  "conditions": {
    "max_cost_per_million": 5.0
  },
  "action": {
    "type": "select_cheapest",
    "fallback": "openai/gpt-3.5-turbo"
  },
  "enabled": true
}
```

#### Code Specialist
```json
{
  "name": "code-specialist",
  "description": "Use specialized models for coding tasks",
  "priority": 90,
  "conditions": {
    "prompt_contains": ["code", "function", "debug", "implement", "algorithm"]
  },
  "action": {
    "type": "select_model",
    "model": "anthropic/claude-3-opus",
    "fallback": "openai/gpt-4-turbo"
  },
  "enabled": true
}
```

#### Fast Response for Short Prompts
```json
{
  "name": "quick-answers",
  "description": "Use fast models for simple questions",
  "priority": 80,
  "conditions": {
    "prompt_length_less_than": 100,
    "response_time_target": 1000
  },
  "action": {
    "type": "select_fastest",
    "candidates": ["mistralai/mistral-7b", "openai/gpt-3.5-turbo"],
    "fallback": "openai/gpt-3.5-turbo"
  },
  "enabled": true
}
```

### How Routing Works
1. Rules are evaluated in priority order (highest first)
2. First matching rule determines model selection
3. If no rules match, default model is used
4. Routing decisions are logged for analytics

### Action Types
- **select_model**: Use a specific model
- **select_cheapest**: Choose most cost-effective model
- **select_fastest**: Pick fastest responding model from candidates

### Condition Types
- **prompt_contains**: Keywords to match (case-insensitive)
- **prompt_length_less_than**: Maximum prompt length
- **max_cost_per_million**: Cost limit per million tokens
- **response_time_target**: Target response time in ms

## Cloudflare AI Gateway Integration

OpenRouter can be configured to use Cloudflare AI Gateway as a proxy layer for additional features like caching, rate limiting, and analytics.

### Setup Cloudflare AI Gateway

```bash
# Check current status
vrooli resource openrouter cloudflare status

# Enable with explicit gateway URL
vrooli resource openrouter cloudflare configure true "https://gateway.ai.cloudflare.com/v1/ACCOUNT_ID/GATEWAY_ID/openrouter"

# Or enable with IDs (URL will be constructed)
vrooli resource openrouter cloudflare configure true "" "my-gateway" "abc123"

# Disable integration
vrooli resource openrouter cloudflare configure false

# Test connectivity
vrooli resource openrouter cloudflare test
```

### Benefits of Cloudflare AI Gateway
- **Caching**: Reduce API costs by caching repeated requests
- **Rate Limiting**: Prevent runaway costs with request limits
- **Analytics**: Track usage patterns and costs across your organization
- **Fallback**: Automatic failover if OpenRouter is unavailable
- **Security**: Additional layer of protection for API keys

When enabled, all OpenRouter API calls will automatically route through the configured Cloudflare AI Gateway.

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
