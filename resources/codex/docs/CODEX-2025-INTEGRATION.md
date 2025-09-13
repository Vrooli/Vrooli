# Integrating 2025 Codex Agent into Resource-Codex

## Current Situation

**What we have:** A text-generation wrapper using GPT-5/GPT-4 models
**What's available:** OpenAI's 2025 Codex CLI agent with full tool execution

## Three Integration Approaches

### Option 1: Direct Codex CLI Integration (RECOMMENDED)

Install and use the official OpenAI Codex CLI as the backend:

```bash
# Install globally
npm install -g @openai/codex

# Configure API key
export OPENAI_API_KEY="your-key-here"

# Or use ChatGPT auth (if you have Plus/Pro)
codex  # Then select "Sign in with ChatGPT"
```

#### Implementation Plan

1. **Detect Codex CLI Installation**
```bash
codex::cli::is_installed() {
    command -v codex &>/dev/null
}
```

2. **Wrap Codex CLI Commands**
```bash
codex::cli::execute() {
    local prompt="$1"
    
    # Use Codex CLI in non-interactive mode
    echo "$prompt" | codex --mode auto --no-color
}
```

3. **Update resource-codex to use Codex CLI when available**
```bash
codex::generate_code() {
    if codex::cli::is_installed; then
        # Use real Codex agent
        codex::cli::execute "$1"
    else
        # Fall back to current GPT text generation
        codex::api::generate "$1"
    fi
}
```

### Option 2: Use codex-mini-latest Model via API

Switch to the new `codex-mini-latest` model (optimized for coding):

```bash
# Update defaults.sh
export CODEX_DEFAULT_MODEL="${CODEX_DEFAULT_MODEL:-codex-mini-latest}"
# Pricing: $1.50/1M input, $6/1M output (cheaper than GPT-5!)
```

#### Benefits
- Better coding performance than general GPT models
- Specifically tuned for code generation
- 75% prompt caching discount
- No CLI dependency

#### Implementation
```bash
# In core.sh
codex::generate_code() {
    local model="codex-mini-latest"  # Use Codex-specific model
    
    # Rest of implementation stays the same
    # But model is optimized for coding tasks
}
```

### Option 3: Build Our Own Agent Loop

Implement tool execution similar to Codex CLI:

```bash
codex::agent::run() {
    local task="$1"
    local workspace="/tmp/codex-workspace"
    
    # 1. Create isolated workspace
    mkdir -p "$workspace"
    cd "$workspace"
    
    # 2. Initialize conversation with tools
    local conversation=$(codex::agent::init "$task")
    
    # 3. Execute tool loop
    while true; do
        # Get next action from model
        local action=$(codex::agent::get_action "$conversation")
        
        case "$action" in
            write_file:*)
                codex::agent::write_file "$action"
                ;;
            run_command:*)
                codex::agent::run_command "$action"
                ;;
            complete)
                break
                ;;
        esac
        
        # Add result to conversation
        conversation=$(codex::agent::add_result "$conversation" "$result")
    done
}
```

## Recommended Implementation Strategy

### Phase 1: Install Codex CLI (Immediate)

```bash
# Add to resource-codex install function
codex::install::cli() {
    if ! command -v codex &>/dev/null; then
        log::info "Installing OpenAI Codex CLI..."
        npm install -g @openai/codex
        
        # Configure with existing API key
        if [[ -n "$OPENAI_API_KEY" ]]; then
            mkdir -p ~/.codex
            cat > ~/.codex/config.toml <<EOF
model = "codex-mini-latest"
model_provider = "openai"
preferred_auth_method = "apikey"
EOF
        fi
    fi
}
```

### Phase 2: Dual-Mode Operation

```bash
# Smart detection and routing
codex::execute() {
    local input="$1"
    
    # Check available backends
    if command -v codex &>/dev/null; then
        # Use full Codex agent
        log::info "Using Codex CLI agent..."
        codex::cli::execute "$input"
    elif [[ "$CODEX_DEFAULT_MODEL" == "codex-mini-latest" ]]; then
        # Use Codex model via API
        log::info "Using codex-mini model..."
        codex::api::generate_with_model "codex-mini-latest" "$input"
    else
        # Fall back to GPT models
        log::info "Using GPT model..."
        codex::api::generate "$input"
    fi
}
```

### Phase 3: Enhanced CLI Commands

```bash
# New commands leveraging Codex agent
cli::register_command "agent" "Run Codex agent on task" "codex::agent::task"
cli::register_command "fix" "Fix code issues" "codex::agent::fix"
cli::register_command "test" "Generate and run tests" "codex::agent::test"
cli::register_command "refactor" "Refactor code" "codex::agent::refactor"
```

## Configuration Updates

### config/defaults.sh
```bash
# Codex CLI settings
export CODEX_CLI_ENABLED="${CODEX_CLI_ENABLED:-auto}"  # auto|true|false
export CODEX_CLI_MODE="${CODEX_CLI_MODE:-auto}"        # auto|approve|always
export CODEX_WORKSPACE="${CODEX_WORKSPACE:-/tmp/codex-workspace}"

# Model selection (in order of preference)
export CODEX_MODEL_PRIORITY="${CODEX_MODEL_PRIORITY:-codex-mini-latest,gpt-5-nano,gpt-4o}"

# Pricing awareness
export CODEX_MINI_PRICE_INPUT="1.50"   # per 1M tokens
export CODEX_MINI_PRICE_OUTPUT="6.00"  # per 1M tokens
```

### config/secrets.yaml
```yaml
secrets:
  api_keys:
    - name: "openai_api_key"
      path: "secret/resources/codex/api/openai"
      description: "OpenAI API key for Codex CLI and models"
      required: true
      supports:
        - "codex-cli"
        - "codex-mini-latest" 
        - "gpt-5-*"
        - "gpt-4o-*"
```

## Testing the Integration

### 1. Check Installation
```bash
resource-codex status --verbose

# Should show:
# Codex CLI: Installed (v0.34.0)
# Model: codex-mini-latest
# Auth: API Key configured
```

### 2. Test Basic Generation
```bash
# Should use Codex CLI if installed
resource-codex content execute "Write a binary search in Python"
```

### 3. Test Agent Capabilities
```bash
# This should create actual files and run tests
resource-codex agent "Create a REST API with FastAPI including tests"
```

## Migration Path

1. **No Breaking Changes**: Existing commands continue to work
2. **Progressive Enhancement**: Codex CLI used when available
3. **Fallback Chain**: Codex CLI → codex-mini API → GPT-5 → GPT-4
4. **User Choice**: Can force specific backend via env vars

## Cost Comparison

| Method | Input Cost | Output Cost | Capabilities |
|--------|------------|-------------|--------------|
| Codex CLI (codex-mini) | $1.50/1M | $6/1M | Full agent, tool execution |
| GPT-5-nano | $0.05/1M | $0.40/1M | Text only |
| GPT-5 | $1.25/1M | $10/1M | Text only |
| GPT-4o | $2.50/1M | $10/1M | Text only |

**Recommendation**: Use Codex CLI with codex-mini for best coding performance at reasonable cost.

## Security Considerations

### Codex CLI Sandboxing
- Runs in specified directory only
- Requires approval for sensitive operations
- No internet access during execution
- Can configure approval modes

### Resource-Codex Integration
- Inherit Codex CLI's security model
- Add additional Vrooli-specific controls
- Log all agent actions
- Workspace isolation in /tmp/codex-workspace

## Next Steps

1. **Install Codex CLI**: `npm install -g @openai/codex`
2. **Configure API key**: Already stored in Vault
3. **Update resource-codex**: Implement Phase 1 changes
4. **Test integration**: Verify both modes work
5. **Document usage**: Update README with examples

## Example Implementation

```bash
#!/usr/bin/env bash
# Enhanced resource-codex with Codex CLI integration

codex::content::execute() {
    local input="$1"
    
    # Try Codex CLI first
    if command -v codex &>/dev/null; then
        log::info "Executing with Codex agent..."
        
        # Create temporary file for complex prompts
        local prompt_file="/tmp/codex-prompt-$$.txt"
        echo "$input" > "$prompt_file"
        
        # Run Codex CLI with appropriate flags
        codex \
            --mode "${CODEX_CLI_MODE:-auto}" \
            --model "${CODEX_DEFAULT_MODEL:-codex-mini-latest}" \
            --file "$prompt_file"
        
        rm -f "$prompt_file"
        return $?
    fi
    
    # Fall back to API-based generation
    log::info "Codex CLI not found, using API..."
    codex::api::generate "$input"
}
```

This approach gives us the best of both worlds:
- Full agent capabilities when Codex CLI is available
- Fallback to API for simpler deployments
- No breaking changes to existing workflows