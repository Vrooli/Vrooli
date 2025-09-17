# Codex Resource

AI-powered code generation with two modes:
1. **Text Generation**: Via OpenAI API (GPT-5/GPT-4 models)
2. **Full Agent**: Via OpenAI Codex CLI (2025) with tool execution

## Overview

Codex provides intelligent code generation through either simple text generation or full agent capabilities with the 2025 Codex CLI. When the Codex CLI is installed, it can create files, run commands, test code, and fix errors automatically - acting as a complete software engineering agent.

## Features

### Text Generation Mode (Default)
- Code completion and generation as text
- Multi-language support (Python, JavaScript, Go, etc.)
- Script injection and management
- API-based processing with configurable parameters
- Integration with Vault for secure credential storage

### Agent Mode (With Codex CLI)
- **Full tool execution**: Creates files, runs commands, tests code
- **Error correction**: Automatically fixes issues and retries
- **Multi-step tasks**: Handles complex workflows autonomously
- **Local execution**: Runs in your terminal with full control
- **Approval workflow**: Choose automatic or manual approval for actions
- **Non-interactive pipelines**: Uses `codex exec` under the hood (workspace sandbox + network enabled)

## Requirements

- OpenAI API key with Codex access
- Network connectivity to OpenAI API endpoints
- jq for JSON processing

## Configuration

### API Key Setup

The resource checks for API keys in this order:
1. Environment variable: `OPENAI_API_KEY`
2. Vault secret: `secret/openai` (field: `api_key`)
3. Credentials file: `~/.openai/credentials`

### Environment Variables

- `CODEX_API_ENDPOINT`: API endpoint (default: https://api.openai.com/v1)
- `CODEX_DEFAULT_MODEL`: Default model (default: gpt-5-nano)
- `CODEX_DEFAULT_TEMPERATURE`: Generation temperature (default: 0.2)
- `CODEX_DEFAULT_MAX_TOKENS`: Maximum tokens (default: 8192)
- `CODEX_TIMEOUT`: End-to-end timeout in seconds (default: 30, apply profiles for longer runs)
- `CODEX_MAX_TURNS`: Maximum conversation turns before the agent stops (default: 10)
- `CODEX_ALLOWED_TOOLS`: Comma-separated allow list (e.g. `read_file,write_file,execute_command(git *)`)
- `CODEX_SKIP_PERMISSIONS`: Set to `true` to disable permission checks (equivalent to `--skip-permissions`)
- `CODEX_CLI_MODE`: Approval policy for Codex CLI (`auto`, `approve`, `always`, `yolo`)
- `CODEX_CLI_SANDBOX`: Override sandbox policy passed to `codex exec` (default: `workspace-write`)
- `CODEX_CLI_EXTRA_ARGS`: Additional raw arguments appended to `codex exec`

### Available Models (GPT-5 Released August 2025)

#### GPT-5 Series (Latest - 400K context, 128K output)
- **gpt-5-nano** (DEFAULT) - Best value: $0.05/1M input, $0.40/1M output
- **gpt-5-mini** - Mid-tier: $0.25/1M input, $2/1M output  
- **gpt-5** - Flagship: $1.25/1M input, $10/1M output

#### GPT-4o Series (Still available - 128K context)
- **gpt-4o-mini** - $0.15/1M input, $0.60/1M output
- **gpt-4o** - $2.50/1M input, $10/1M output

#### O1 Reasoning Models
- **o1-mini** - Cost-efficient reasoning
- **o1-preview** - Advanced reasoning for hardest problems

## Installing Codex CLI (Optional but Recommended)

To enable full agent capabilities with tool execution:

```bash
# Install via resource-codex
resource-codex manage install-cli

# Or install directly with npm
npm install -g @openai/codex

# Configure with your API key
resource-codex manage configure-cli

# Check installation
resource-codex status | grep "CLI"
```

The Codex CLI is a 2025 release from OpenAI that provides:
- Local code execution in sandboxed environments
- File creation and modification
- Command execution with safety controls
- Automatic error detection and correction
- Multi-step task orchestration

## Usage

### Basic Commands

```bash
# Check status
resource-codex status

# Start service (mark as running)
resource-codex start

# Stop service (mark as stopped)
resource-codex stop

# List injected scripts
resource-codex list

# Inject a Python script
resource-codex inject my_script.py

# Run a script with Codex
resource-codex run my_script.py
```

### Status Check

```bash
# Text format
resource-codex status

# JSON format
resource-codex status --format json
```

### Agent Commands (Requires Codex CLI)

When Codex CLI is installed, you get full agent capabilities:

```bash
# Run agent on any task
resource-codex agent "Create a FastAPI app with user authentication"

# Fix code issues
resource-codex fix app.py "Fix the memory leak"

# Generate and run tests
resource-codex generate-tests src/calculator.py

# Refactor code
resource-codex refactor legacy_code.py "Improve readability and add type hints"
```

### Script Injection

Scripts are stored in `$CODEX_HOME/scripts/` (default `~/.codex/scripts/`, fallback to `/tmp/codex-workspace/.codex-home/scripts`) for processing:

```bash
# Inject a new script
resource-codex inject path/to/script.py

# List all injected scripts
resource-codex list

# Run a specific script
resource-codex run script.py
```

### Permission Profiles & Limits

- Use `--allowed-tools` to narrow execution (e.g. `read_file,write_file,execute_command(git *)`).
- Apply predefined profiles (`safe`, `development`, `admin`) or set `--max-turns` / `--timeout` to cap sessions.
- `--skip-permissions` (or `CODEX_SKIP_PERMISSIONS=true`) now forces `CODEX_CLI_MODE=yolo` and disables sandboxing, giving Codex full control – combine with care.
- All agent runs use `codex exec` with workspace sandboxing; network is explicitly enabled for sandbox runs.

## Directory Structure

- `config/` - Configuration defaults
- `lib/` - Core functionality libraries
- `injected/` - Backup of injected scripts
- `$CODEX_HOME/scripts/` - Active script storage
- `$CODEX_HOME/outputs/` - Generated code outputs

## Integration

### With Scenarios

Codex can be used in scenarios for:
- Generating boilerplate code
- Converting between languages
- Creating test cases
- Implementing algorithms
- Code refactoring

### With Other Resources

- **Vault**: Secure API key storage
- **Judge0**: Execute generated code
- **n8n/Node-RED**: Workflow automation with code generation
- **Ollama**: Fallback for local code generation

## Troubleshooting

### API Key Issues

```bash
# Check if API key is configured
resource-codex status | grep "API Configured"

# Test API connectivity
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
  https://api.openai.com/v1/models
```

### CLI Fails with "failed to initialize rollout recorder"

- Ensure Codex has a writable home directory. The resource automatically falls back to `/tmp/codex-workspace/.codex-home` when `~/.codex` is read-only.
- You can manually override with `export CODEX_HOME=/path/to/writable/dir` before running agent commands.

### Model Selection Guide

**Use gpt-5-nano (default) for:**
- Quick code generation at lowest cost ($0.05/1M)
- Simple scripts and functions
- High-volume operations
- Everyday coding tasks
- 400K context window for large files!

**Use gpt-5-mini for:**
- Balanced performance/cost
- Medium complexity tasks
- Better accuracy than nano

**Use gpt-5 for:**
- Most complex algorithms
- Best coding performance (94.6% on AIME 2025)
- Architecture design
- Production-critical code

**Use o1-mini/o1-preview for:**
- Deep reasoning tasks
- Mathematical proofs
- Logic-heavy implementations

## Text Generation vs Agent Mode

| Feature | Text Generation | Agent Mode (CLI) |
|---------|----------------|------------------|
| **Code Generation** | ✅ Returns code as text | ✅ Creates actual files |
| **File Operations** | ❌ | ✅ Read/write/modify files |
| **Code Execution** | ❌ | ✅ Run commands and tests |
| **Error Fixing** | ❌ | ✅ Auto-detect and fix issues |
| **Multi-step Tasks** | ❌ | ✅ Handle complex workflows |
| **Installation** | Built-in | Requires `npm install -g @openai/codex` |
| **Cost** | API usage only | CLI + API usage |
| **Use Case** | Quick code snippets | Full software development |

**Smart Routing**: The system automatically uses the best available backend:
1. Codex CLI (if installed) → Full agent capabilities
2. codex-mini-latest API → Optimized text generation
3. GPT-5-nano API → Fallback text generation

## Security

### Text Generation Mode
- No code execution, completely safe
- Only generates text responses
- API keys secured in Vault

### Agent Mode (Codex CLI)
- Executes in controlled sandboxes
- Approval workflows for sensitive operations
- Local execution with file system access
- Audit logging of all operations
- No internet access during execution
- User-controllable workspace isolation
