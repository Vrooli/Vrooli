#!/usr/bin/env bash
################################################################################
# Codex CLI Integration Demo - Shows 2025 Agent Capabilities
#
# This demonstrates how resource-codex now integrates with OpenAI's 
# 2025 Codex CLI agent for full tool execution capabilities
################################################################################

echo "==================================================================="
echo "            Resource-Codex with 2025 Codex CLI Agent              "
echo "==================================================================="
echo ""

# Check current status
echo "📋 Current Status:"
echo "=================="
resource-codex status | head -15
echo ""

# Check if Codex CLI is installed
echo "🔍 Checking Codex CLI Installation:"
echo "===================================="
if command -v codex &>/dev/null; then
    echo "✅ Codex CLI is installed!"
    echo "   Version: $(codex --version 2>/dev/null || echo 'unknown')"
    echo "   Location: $(which codex)"
else
    echo "❌ Codex CLI not installed"
    echo ""
    echo "📦 To install Codex CLI:"
    echo "   npm install -g @openai/codex"
    echo "   OR"
    echo "   resource-codex manage install-cli"
fi
echo ""

# Show available commands
echo "🚀 New Commands Available:"
echo "=========================="
echo ""
echo "1. Install/Manage Codex CLI:"
echo "   resource-codex manage install-cli    # Install OpenAI Codex CLI"
echo "   resource-codex manage update-cli     # Update to latest version"
echo "   resource-codex manage configure-cli  # Configure with API key"
echo ""
echo "2. Agent Commands (full tool execution):"
echo "   resource-codex agent <task>          # Run full agent on task"
echo "   resource-codex fix <file> [issue]    # Fix code issues"
echo "   resource-codex generate-tests <file> # Generate and run tests"
echo "   resource-codex refactor <file>       # Refactor code"
echo ""
echo "3. Smart Execution (auto-selects best backend):"
echo "   resource-codex content execute <prompt>"
echo "   → Uses Codex CLI if installed (full agent)"
echo "   → Falls back to API if not (text only)"
echo ""

# Show model hierarchy
echo "📊 Model Selection Hierarchy:"
echo "============================="
echo "1. Codex CLI with codex-mini-latest ($1.50/1M input)"
echo "   → Full agent with tool execution"
echo "   → Can create files, run commands, fix errors"
echo ""
echo "2. API with codex-mini-latest model"
echo "   → Optimized for coding tasks"
echo "   → Text generation only"
echo ""
echo "3. API with GPT-5-nano ($0.05/1M input)"
echo "   → General purpose, very cheap"
echo "   → Text generation only"
echo ""

# Show example usage
echo "💡 Example Usage:"
echo "================="
echo ""
echo "# Install Codex CLI (one-time setup):"
echo "resource-codex manage install-cli"
echo ""
echo "# Run agent on a complex task:"
echo 'resource-codex agent "Create a REST API with FastAPI including CRUD operations for a todo app with SQLite database and tests"'
echo ""
echo "# Fix a specific issue:"
echo 'resource-codex fix app.py "The database connection is leaking"'
echo ""
echo "# Generate tests for existing code:"
echo "resource-codex generate-tests src/calculator.py"
echo ""
echo "# Smart execution (uses best available):"
echo 'resource-codex content execute "Write a binary search tree implementation with all operations"'
echo ""

# Show workspace info
echo "📁 Workspace Information:"
echo "========================"
echo "Codex CLI workspace: ${CODEX_WORKSPACE:-/tmp/codex-workspace}"
echo "Scripts directory: ${HOME}/.codex/scripts"
echo "Outputs directory: ${HOME}/.codex/outputs"
echo "Config file: ${HOME}/.codex/config.toml"
echo ""

# Show authentication options
echo "🔐 Authentication Options:"
echo "========================="
echo "1. Use existing OpenAI API key from Vault"
echo "2. Sign in with ChatGPT (Plus/Pro/Team required)"
echo "3. Manual API key configuration"
echo ""
echo "Current: $(if [[ -n "${OPENAI_API_KEY}" ]]; then echo "API key configured"; else echo "Not configured"; fi)"
echo ""

echo "==================================================================="
echo "                       What's Changed?                             "
echo "==================================================================="
echo ""
echo "OLD (Text Generation Only):"
echo "  Request → OpenAI API → Text Response"
echo "  ❌ Cannot create files"
echo "  ❌ Cannot run commands"
echo "  ❌ Cannot fix errors"
echo ""
echo "NEW (With Codex CLI Agent):"
echo "  Request → Codex Agent → [Create Files, Run Commands, Test] → Result"
echo "  ✅ Creates actual files"
echo "  ✅ Runs and tests code"
echo "  ✅ Fixes errors automatically"
echo "  ✅ Full software engineering agent"
echo ""
echo "==================================================================="