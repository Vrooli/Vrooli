#!/bin/bash
# LiteLLM Complete Setup Guide
# This script demonstrates how to manually set up LiteLLM with API keys and Ollama

set -e

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"

echo "🚀 LiteLLM Complete Setup Guide"
echo "=============================="
echo

# Configuration paths
LITELLM_CONFIG_DIR="${APP_ROOT}/data/litellm/config"
LITELLM_CONFIG_FILE="${LITELLM_CONFIG_DIR}/config.yaml"
LITELLM_ENV_FILE="${LITELLM_CONFIG_DIR}/.env"

echo "1️⃣ Creating LiteLLM directories..."
mkdir -p "${APP_ROOT}/data/litellm"/{config,logs,data}
echo "✅ Directories created"
echo

echo "2️⃣ Setting up API keys..."
echo "Choose your method:"
echo "A) Environment file (simple)"
echo "B) Vault (secure)"
echo "C) Manual environment variables"
echo

read -p "Choose method (A/B/C): " method

case "$method" in
    "A"|"a")
        echo "📝 Creating environment file..."
        cat > "$LITELLM_ENV_FILE" <<EOF
# LiteLLM Environment Variables
# Add your API keys here

# Core LiteLLM settings
LITELLM_MASTER_KEY=sk-vrooli-litellm-$(openssl rand -hex 8)

# Provider API Keys
# Uncomment and add your keys:

# OpenAI
# OPENAI_API_KEY=sk-your-openai-key-here

# Anthropic
# ANTHROPIC_API_KEY=sk-ant-your-anthropic-key-here

# OpenRouter (gives access to many models)
# OPENROUTER_API_KEY=sk-or-your-openrouter-key-here

# Google
# GOOGLE_API_KEY=your-google-key-here

# Azure OpenAI (if using)
# AZURE_API_KEY=your-azure-key
# AZURE_API_BASE=https://your-resource.openai.azure.com/
# AZURE_API_VERSION=2023-12-01-preview

# Database
DATABASE_URL=sqlite:///app/data/litellm.db
LOG_LEVEL=INFO
EOF
        echo "✅ Environment file created at: $LITELLM_ENV_FILE"
        echo "📝 Edit this file to add your API keys"
        ;;
    "B"|"b")
        echo "🔐 Setting up Vault keys..."
        if command -v vault >/dev/null 2>&1; then
            echo "Enter your API keys (press Enter to skip):"
            
            read -p "OpenAI API Key: " openai_key
            if [[ -n "$openai_key" ]]; then
                vault kv put secret/vrooli/openai api_key="$openai_key"
                echo "✅ OpenAI key stored in Vault"
            fi
            
            read -p "Anthropic API Key: " anthropic_key
            if [[ -n "$anthropic_key" ]]; then
                vault kv put secret/vrooli/anthropic api_key="$anthropic_key"
                echo "✅ Anthropic key stored in Vault"
            fi
            
            read -p "OpenRouter API Key: " openrouter_key
            if [[ -n "$openrouter_key" ]]; then
                vault kv put secret/vrooli/openrouter api_key="$openrouter_key"
                echo "✅ OpenRouter key stored in Vault"
            fi
        else
            echo "❌ Vault not found. Please install Vault first or choose option A."
            exit 1
        fi
        ;;
    "C"|"c")
        echo "🔧 Manual environment setup..."
        echo "Add these to your shell profile (.bashrc, .zshrc, etc.):"
        echo
        echo "export OPENAI_API_KEY='sk-your-openai-key'"
        echo "export ANTHROPIC_API_KEY='sk-ant-your-anthropic-key'"
        echo "export OPENROUTER_API_KEY='sk-or-your-openrouter-key'"
        echo "export GOOGLE_API_KEY='your-google-key'"
        echo "export LITELLM_MASTER_KEY='sk-vrooli-litellm-$(openssl rand -hex 8)'"
        echo
        ;;
esac

echo
echo "3️⃣ Setting up LiteLLM configuration..."

# Create configuration with Ollama integration
cat > "$LITELLM_CONFIG_FILE" <<'EOF'
# LiteLLM Configuration with Multi-Provider Support
model_list:
  # Local Ollama Models (if available)
  - model_name: llama2-local
    litellm_params:
      model: ollama/llama2
      api_base: http://localhost:11434
      
  - model_name: codellama-local
    litellm_params:
      model: ollama/codellama
      api_base: http://localhost:11434
      
  # OpenAI Models
  - model_name: gpt-3.5-turbo
    litellm_params:
      model: openai/gpt-3.5-turbo
      api_key: ${OPENAI_API_KEY}
      
  - model_name: gpt-4
    litellm_params:
      model: openai/gpt-4
      api_key: ${OPENAI_API_KEY}
      
  # Anthropic Models
  - model_name: claude-3-haiku
    litellm_params:
      model: anthropic/claude-3-haiku-20240307
      api_key: ${ANTHROPIC_API_KEY}
      
  - model_name: claude-3-sonnet
    litellm_params:
      model: anthropic/claude-3-5-sonnet-20241022
      api_key: ${ANTHROPIC_API_KEY}
      
  # OpenRouter Models (fallback)
  - model_name: openrouter-gpt-3.5
    litellm_params:
      model: openai/gpt-3.5-turbo
      api_base: https://openrouter.ai/api/v1
      api_key: ${OPENROUTER_API_KEY}

# Router Settings - Prefer local models
router_settings:
  routing_strategy: "cost-based-routing"  # Free local models first
  enable_fallbacks: true
  fallbacks:
    - llama2-local: ["gpt-3.5-turbo", "openrouter-gpt-3.5"]
    - codellama-local: ["gpt-4", "claude-3-haiku"]
    - gpt-3.5-turbo: ["claude-3-haiku", "openrouter-gpt-3.5"]
  num_retries: 3

# General Settings
general_settings:
  master_key: "${LITELLM_MASTER_KEY}"
  database_url: "sqlite:///app/data/litellm.db"
  
# Logging
litellm_settings:
  set_verbose: false
  json_logs: true
  log_file: "/app/logs/litellm.log"

# Budget Controls
budget_settings:
  enable_budget_control: true
  max_budget: 100.0
EOF

echo "✅ Configuration created at: $LITELLM_CONFIG_FILE"
echo

echo "4️⃣ Docker Setup..."
echo "Starting LiteLLM container..."

# Check if container exists and remove it
if docker ps -a --format '{{.Names}}' | grep -q '^vrooli-litellm$'; then
    echo "🗑️ Removing existing container..."
    docker rm -f vrooli-litellm
fi

# Create network if it doesn't exist
if ! docker network ls --format '{{.Name}}' | grep -q '^vrooli-network$'; then
    echo "🌐 Creating vrooli-network..."
    docker network create vrooli-network
fi

echo "📦 Starting LiteLLM container..."
docker run -d \
    --name vrooli-litellm \
    --hostname vrooli-litellm \
    --network vrooli-network \
    -p 11435:4000 \
    -v "${LITELLM_CONFIG_FILE}:/app/config.yaml:ro" \
    -v "${LITELLM_ENV_FILE}:/app/.env:ro" \
    -v "${APP_ROOT}/data/litellm/data:/app/data" \
    -v "${APP_ROOT}/data/litellm/logs:/app/logs" \
    --restart unless-stopped \
    --env-file "${LITELLM_ENV_FILE}" \
    ghcr.io/berriai/litellm:main-latest \
    --config /app/config.yaml \
    --port 4000 \
    --detailed_debug

echo "⏳ Waiting for LiteLLM to start..."
sleep 10

# Test the setup
echo
echo "5️⃣ Testing setup..."

# Get master key for testing
MASTER_KEY=$(grep "LITELLM_MASTER_KEY=" "$LITELLM_ENV_FILE" | cut -d'=' -f2)

echo "🧪 Testing API endpoint..."
response=$(curl -s -w "%{http_code}" -o /tmp/litellm_test.json -X POST \
    -H "Authorization: Bearer $MASTER_KEY" \
    -H "Content-Type: application/json" \
    -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello!"}], "max_tokens": 5}' \
    http://localhost:11435/chat/completions)

if [[ "$response" == "200" ]]; then
    echo "✅ API is working!"
    echo "Response: $(cat /tmp/litellm_test.json | jq -r '.choices[0].message.content' 2>/dev/null || echo 'API responded successfully')"
else
    echo "⚠️  API test failed (HTTP $response)"
    echo "This might be normal if you haven't configured API keys yet"
fi

echo
echo "6️⃣ Ollama Integration Setup..."
echo "To connect to Ollama:"

if docker ps --format '{{.Names}}' | grep -q 'ollama'; then
    echo "✅ Ollama container is running"
    echo "📋 Available Ollama models:"
    docker exec ollama ollama list 2>/dev/null || echo "  (Run 'docker exec ollama ollama pull llama2' to install models)"
else
    echo "⚠️  Ollama not running. To set up Ollama:"
    echo "  1. Install Ollama: vrooli resource ollama install"
    echo "  2. Pull models: docker exec ollama ollama pull llama2"
    echo "  3. Restart LiteLLM: docker restart vrooli-litellm"
fi

echo
echo "🎉 Setup Complete!"
echo "================="
echo
echo "📋 Summary:"
echo "• LiteLLM URL: http://localhost:11435"
echo "• Master Key: $MASTER_KEY"
echo "• Config: $LITELLM_CONFIG_FILE"
echo "• Environment: $LITELLM_ENV_FILE"
echo
echo "🔑 Next Steps:"
echo "1. Edit $LITELLM_ENV_FILE to add your API keys"
echo "2. Restart: docker restart vrooli-litellm"
echo "3. Test models: curl -X POST http://localhost:11435/models -H 'Authorization: Bearer $MASTER_KEY'"
echo
echo "🦙 Ollama Integration:"
echo "• Local models are automatically preferred (cost = \$0)"
echo "• Falls back to API models if local models fail"
echo "• No API keys needed for local Ollama models"
echo
echo "🤖 Claude Code Integration:"
echo "export ANTHROPIC_BASE_URL='http://localhost:11435'"
echo "export ANTHROPIC_AUTH_TOKEN='$MASTER_KEY'"

rm -f /tmp/litellm_test.json