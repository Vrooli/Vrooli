#!/bin/bash
# LiteLLM Usage Examples - How to actually use the LiteLLM resource

set -e

echo "🤖 LiteLLM Usage Examples"
echo "========================"
echo

# Configuration
LITELLM_URL="http://localhost:11435"
MASTER_KEY_FILE="${VROOLI_ROOT:-${HOME}/Vrooli}/data/litellm/config/.env"

# Get master key
if [[ -f "$MASTER_KEY_FILE" ]]; then
    MASTER_KEY=$(grep "LITELLM_MASTER_KEY=" "$MASTER_KEY_FILE" | cut -d'=' -f2)
    echo "🔑 Using master key: ${MASTER_KEY:0:20}..."
else
    echo "❌ Master key file not found. Please set up LiteLLM first."
    exit 1
fi

echo
echo "1️⃣ METHOD 1: Direct API Calls"
echo "=============================="
echo

echo "📋 List available models:"
curl -s -H "Authorization: Bearer $MASTER_KEY" \
     "$LITELLM_URL/models" | jq '.data[].id' || echo "Service not running"

echo
echo "🗣️ Simple chat completion (auto routing):"
curl -s -X POST "$LITELLM_URL/chat/completions" \
    -H "Authorization: Bearer $MASTER_KEY" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "auto",
        "messages": [{"role": "user", "content": "Hello! What models are you using?"}],
        "max_tokens": 100
    }' | jq -r '.choices[0].message.content // .error.message // "No response"'

echo
echo "🎯 Specific model selection:"
echo "  - llama2-local (Ollama - FREE)"
echo "  - gpt-3.5-turbo (OpenAI - Paid)"
echo "  - claude-3-haiku (Anthropic - Paid)"

echo
echo "Test local Ollama model:"
curl -s -X POST "$LITELLM_URL/chat/completions" \
    -H "Authorization: Bearer $MASTER_KEY" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "llama2-local",
        "messages": [{"role": "user", "content": "Hi from Ollama!"}],
        "max_tokens": 50
    }' | jq -r '.choices[0].message.content // .error.message // "Model not available"'

echo
echo "2️⃣ METHOD 2: Claude Code Integration"
echo "===================================="
echo

cat << 'EOF'
# Set environment variables
export ANTHROPIC_BASE_URL="http://localhost:11435"
export ANTHROPIC_AUTH_TOKEN="your-master-key-here"

# Now Claude Code will route through LiteLLM!
claude "Generate a Python hello world script"

# The routing happens automatically:
# 1. Tries Ollama first (free)
# 2. Falls back to your API providers if needed
EOF

echo
echo "3️⃣ METHOD 3: Intelligent Routing Examples"
echo "=========================================="
echo

echo "🧠 How LiteLLM decides which model to use:"
echo

echo "Example 1: Simple Question (Routes to Ollama - FREE)"
curl -s -X POST "$LITELLM_URL/chat/completions" \
    -H "Authorization: Bearer $MASTER_KEY" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "auto",
        "messages": [{"role": "user", "content": "What is 2+2?"}],
        "max_tokens": 20
    }' | jq '.choices[0].message.content // .error.message // "No response"'

echo
echo "Example 2: Complex Question (May route to GPT-4 if Ollama fails)"
curl -s -X POST "$LITELLM_URL/chat/completions" \
    -H "Authorization: Bearer $MASTER_KEY" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "auto",
        "messages": [{"role": "user", "content": "Explain quantum computing in detail"}],
        "max_tokens": 200
    }' | jq '.choices[0].message.content // .error.message // "No response"'

echo
echo "4️⃣ METHOD 4: Model-Specific Requests"
echo "===================================="
echo

echo "Force use Ollama (even if it fails):"
echo 'curl -X POST "$LITELLM_URL/chat/completions" \'
echo '    -H "Authorization: Bearer $MASTER_KEY" \'
echo '    -d '"'"'{"model": "llama2-local", "messages": [...]}'"'"

echo
echo "Force use OpenAI:"
echo 'curl -X POST "$LITELLM_URL/chat/completions" \'
echo '    -H "Authorization: Bearer $MASTER_KEY" \'
echo '    -d '"'"'{"model": "gpt-3.5-turbo", "messages": [...]}'"'"

echo
echo "5️⃣ METHOD 5: Programming Language Examples"
echo "==========================================="
echo

echo "Python example:"
cat << 'EOF'
import requests

url = "http://localhost:11435/chat/completions"
headers = {
    "Authorization": "Bearer your-master-key",
    "Content-Type": "application/json"
}
data = {
    "model": "auto",  # Let LiteLLM choose
    "messages": [{"role": "user", "content": "Hello!"}],
    "max_tokens": 100
}

response = requests.post(url, headers=headers, json=data)
print(response.json()["choices"][0]["message"]["content"])
EOF

echo
echo "JavaScript/Node.js example:"
cat << 'EOF'
const response = await fetch('http://localhost:11435/chat/completions', {
    method: 'POST',
    headers: {
        'Authorization': 'Bearer your-master-key',
        'Content-Type': 'application/json'
    },
    body: JSON.stringify({
        model: 'auto',
        messages: [{role: 'user', content: 'Hello!'}],
        max_tokens: 100
    })
});

const data = await response.json();
console.log(data.choices[0].message.content);
EOF

echo
echo "6️⃣ Understanding the Routing Logic"
echo "=================================="
echo

cat << 'EOF'
🧠 How LiteLLM Chooses Models:

1. COST-BASED ROUTING (Default):
   - Ollama models: $0.00 (FREE) → Always tried first
   - OpenAI models: $0.002/1K tokens → Second choice
   - Anthropic models: $0.003/1K tokens → Third choice
   - OpenRouter: Varies → Fallback option

2. FALLBACK CHAIN:
   llama2-local → gpt-3.5-turbo → claude-3-haiku → openrouter-gpt-3.5

3. AUTOMATIC DECISIONS:
   - Simple tasks → Ollama (free)
   - Ollama unavailable → OpenAI (cheap)
   - OpenAI rate limited → Anthropic
   - All APIs down → OpenRouter (backup)

4. YOU CAN OVERRIDE:
   - model: "auto" → LiteLLM decides
   - model: "llama2-local" → Force Ollama
   - model: "gpt-4" → Force OpenAI
   - model: "claude-3-sonnet" → Force Anthropic
EOF

echo
echo "7️⃣ Monitoring and Debugging"
echo "==========================="
echo

echo "Check which model was actually used:"
echo 'curl -s "$LITELLM_URL/chat/completions" [...] | jq .model'

echo
echo "View LiteLLM logs:"
echo 'docker logs vrooli-litellm --tail 50'

echo
echo "Get usage statistics:"
curl -s -H "Authorization: Bearer $MASTER_KEY" \
     "$LITELLM_URL/health" | jq '.' || echo "Health endpoint not available"

echo
echo "🎉 Summary"
echo "=========="
echo
echo "✅ LiteLLM automatically routes requests based on:"
echo "   • Cost (free Ollama first)"
echo "   • Availability (fallback if models are down)"
echo "   • Your API key configuration"
echo
echo "✅ You can:"
echo "   • Let it choose automatically (model: 'auto')"
echo "   • Force specific models (model: 'gpt-4')"
echo "   • Set up fallback chains"
echo "   • Monitor usage and costs"
echo
echo "✅ Best practices:"
echo "   • Use 'auto' for cost optimization"
echo "   • Keep Ollama running for free inference"
echo "   • Configure API keys for important fallbacks"
echo "   • Monitor logs to see routing decisions"
EOF

echo
echo "🚀 Ready to use LiteLLM!"
echo "Start with: curl -X POST $LITELLM_URL/chat/completions [...]"