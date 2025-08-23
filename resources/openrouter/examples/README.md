# OpenRouter Examples

This directory contains examples demonstrating how to use the OpenRouter resource.

## Prerequisites

Before running these examples, ensure you have:

1. **API Key**: Obtain an API key from [OpenRouter](https://openrouter.ai/keys)
2. **Configure Key**: Store your API key using one of these methods:
   - Vault (recommended): `docker exec vault sh -c 'vault kv put secret/vrooli/openrouter api_key=YOUR_KEY'`
   - Environment variable: `export OPENROUTER_API_KEY=YOUR_KEY`
   - Credentials file: Stored automatically when key is configured

## Available Examples

### basic-chat.sh
Demonstrates basic usage of OpenRouter API including:
- Simple chat completion
- Multi-turn conversations with system prompts
- Listing available models
- Checking API usage/credits

Run with:
```bash
./basic-chat.sh
```

## Integration with Other Resources

OpenRouter can be injected into other Vrooli resources like n8n, Node-RED, and Windmill to provide AI capabilities in workflows. Use the inject command:

```bash
vrooli resource openrouter inject n8n
```

This will configure the target resource with OpenRouter credentials for use in AI-powered workflows.

## Common Use Cases

1. **Content Generation**: Generate text, code, or creative content
2. **Data Analysis**: Analyze and summarize data
3. **Translation**: Translate between languages
4. **Code Assistance**: Generate, review, or explain code
5. **Question Answering**: Build Q&A systems

## Model Selection

OpenRouter provides access to models from multiple providers. Common models include:
- `openai/gpt-3.5-turbo` - Fast, cost-effective
- `openai/gpt-4` - Most capable
- `anthropic/claude-2` - Strong reasoning
- `google/palm-2` - Google's LLM

List all available models:
```bash
vrooli resource openrouter list-models
```

## Rate Limits and Usage

Check your current usage and limits:
```bash
vrooli resource openrouter usage
```

## Troubleshooting

If examples fail:
1. Verify API key is configured: `vrooli resource openrouter status`
2. Check connectivity: `vrooli resource openrouter test-connection`
3. Ensure you have credits: `vrooli resource openrouter usage`