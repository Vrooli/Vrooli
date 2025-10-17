# Resource-Codex Quick Start Guide

## TL;DR - Two Modes Available

### 🚀 FAST: Text Generation Only (Default)
```bash
resource-codex content execute "Write a function to sort an array"
# → Returns code as text (copy/paste to use)
```

### 🤖 POWERFUL: Full Agent with Tool Execution
```bash
# One-time setup
resource-codex manage install-cli

# Now you get full agent capabilities
resource-codex content execute "Create a REST API with tests"
# → Actually creates files and runs tests
```

---

## What's the Difference?

| Without Codex CLI | With Codex CLI |
|-------------------|----------------|
| 📝 Generates code as text | 📁 Creates actual files |
| ❌ You copy/paste manually | ✅ Runs commands automatically |
| ❌ Can't test code | ✅ Tests and fixes errors |
| ⚡ Fast and simple | 🤖 Full software engineering |

---

## Commands Available

### Everyone Gets (No Installation Required)
```bash
resource-codex status                    # Check status
resource-codex content execute "prompt" # Generate code
resource-codex content list             # List scripts
resource-codex help                     # Show all commands
```

### With Codex CLI Installed
```bash
# Setup commands (one-time)
resource-codex manage install-cli       # Install Codex CLI
resource-codex manage configure-cli     # Configure with API key

# Agent commands (full tool execution)
resource-codex agent "complex task"     # Run agent on any task
resource-codex fix app.py "issue"       # Fix specific code issues
resource-codex generate-tests file.py   # Create and run tests
resource-codex refactor old_code.py     # Improve code quality
```

---

## Installation Guide

### Step 1: Basic Setup (Already Done)
✅ Resource-codex is already configured
✅ API key is stored in Vault
✅ Ready for text generation

### Step 2: Enable Agent Mode (Optional)
```bash
# Install Codex CLI
resource-codex manage install-cli

# Check it worked
resource-codex status | grep "CLI Installed"
# Should show: CLI Installed: true
```

### Step 3: Test It
```bash
# Simple test
resource-codex content execute "Write hello world in Python"
# Should show: "Using Codex CLI agent..." and create actual file

# Complex test
resource-codex agent "Create a calculator app with GUI and tests"
# Should build complete working application
```

---

## Troubleshooting

### "CLI Installed: false"
```bash
# Install it
resource-codex manage install-cli

# Or manually
npm install -g @openai/codex
```

### "No API key configured"
```bash
# Configure it
resource-codex manage configure-cli

# Check Vault has the key
resource-vault content get --path "resources/codex/api/openai" --key "api_key"
```

### Commands fail with "not available"
```bash
# Check status
resource-codex status

# Try basic execution
resource-codex content execute "print('hello')"
```

---

## Examples

### Generate a Simple Function
```bash
resource-codex content execute "Write a Python function to calculate fibonacci"
```

### Build a Complete Project (Requires CLI)
```bash
resource-codex agent "Create a Flask web app with user registration, login, and a dashboard. Include tests and error handling."
```

### Fix Existing Code (Requires CLI)
```bash
resource-codex fix buggy_script.py "Fix the memory leak and add error handling"
```

### Generate Tests (Requires CLI)
```bash
resource-codex generate-tests src/calculator.py
```

---

## Pro Tips

1. **Start Simple**: Use text generation first, install CLI when you need it
2. **Complex Tasks**: Use `resource-codex agent` for multi-step projects
3. **Specific Issues**: Use `resource-codex fix` for targeted improvements
4. **Cost Conscious**: Text generation is much cheaper than agent mode
5. **Check Status**: Run `resource-codex status` to see what's available

---

## Cost Information

- **Text Generation**: $0.05-1.25 per 1M tokens (very cheap)
- **Agent Mode**: $1.50 per 1M tokens + execution time (more expensive but incredibly powerful)

The system automatically uses the cheapest effective method for your task.

---

## Need Help?

```bash
resource-codex help                    # Show all commands
./resources/codex/examples/codex-cli-demo.sh  # Run interactive demo
```

Or check the full documentation in `resources/codex/README.md`