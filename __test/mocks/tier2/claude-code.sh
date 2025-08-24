#!/usr/bin/env bash
# Claude-Code Mock - Tier 2 (Stateful)
# 
# Provides stateful Claude CLI mocking for testing:
# - Interactive chat sessions
# - Code generation and analysis
# - Project context management
# - Tool execution simulation
# - Error injection for resilience testing
#
# Coverage: ~80% of common Claude CLI operations in 400 lines

# === Configuration ===
declare -gA CLAUDE_SESSIONS=()           # Session_id -> "status|turns|context"
declare -gA CLAUDE_RESPONSES=()          # Prompt_hash -> "response"
declare -gA CLAUDE_PROJECTS=()           # Project_name -> "path|files|status"
declare -gA CLAUDE_TOOLS=()              # Tool_name -> "enabled|calls"
declare -gA CLAUDE_CONFIG=(              # Service configuration
    [api_key]="mock-api-key"
    [model]="claude-3-opus"
    [max_tokens]="4096"
    [temperature]="0.7"
    [error_mode]=""
    [version]="1.0.0"
)

# Debug mode
declare -g CLAUDE_DEBUG="${CLAUDE_DEBUG:-}"

# === Helper Functions ===
claude_debug() {
    [[ -n "$CLAUDE_DEBUG" ]] && echo "[MOCK:CLAUDE] $*" >&2
}

claude_check_error() {
    case "${CLAUDE_CONFIG[error_mode]}" in
        "auth_error")
            echo "Error: Authentication failed - invalid API key" >&2
            return 1
            ;;
        "rate_limit")
            echo "Error: Rate limit exceeded" >&2
            return 1
            ;;
        "model_error")
            echo "Error: Model ${CLAUDE_CONFIG[model]} not available" >&2
            return 1
            ;;
        "timeout")
            echo "Error: Request timed out" >&2
            return 1
            ;;
    esac
    return 0
}

claude_generate_id() {
    printf "session_%08x" $RANDOM
}

claude_hash_prompt() {
    echo "$1" | md5sum | cut -d' ' -f1
}

# === Main Claude Command ===
claude() {
    claude_debug "claude called with: $*"
    
    if ! claude_check_error; then
        return $?
    fi
    
    local command="${1:-help}"
    
    # Handle single prompt without subcommand
    if [[ "$command" != "chat" && "$command" != "analyze" && "$command" != "generate" && 
          "$command" != "project" && "$command" != "config" && "$command" != "help" && 
          "$command" != "version" && "$command" != "tools" ]]; then
        # Treat entire input as prompt
        claude_cmd_prompt "$*"
        return $?
    fi
    
    shift || true
    
    case "$command" in
        chat)
            claude_cmd_chat "$@"
            ;;
        analyze)
            claude_cmd_analyze "$@"
            ;;
        generate)
            claude_cmd_generate "$@"
            ;;
        project)
            claude_cmd_project "$@"
            ;;
        config)
            claude_cmd_config "$@"
            ;;
        tools)
            claude_cmd_tools "$@"
            ;;
        version)
            echo "Claude CLI v${CLAUDE_CONFIG[version]}"
            echo "Model: ${CLAUDE_CONFIG[model]}"
            ;;
        help|*)
            echo "Claude CLI - AI Assistant"
            echo "Commands:"
            echo "  chat     - Start interactive chat"
            echo "  analyze  - Analyze code or files"
            echo "  generate - Generate code"
            echo "  project  - Manage project context"
            echo "  config   - Configure settings"
            echo "  tools    - Manage tool usage"
            echo "  version  - Show version info"
            echo ""
            echo "Or simply: claude 'your prompt here'"
            ;;
    esac
}

# === Direct Prompt Handler ===
claude_cmd_prompt() {
    local prompt="$*"
    local prompt_hash=$(claude_hash_prompt "$prompt")
    
    # Check for cached response
    if [[ -n "${CLAUDE_RESPONSES[$prompt_hash]}" ]]; then
        echo "${CLAUDE_RESPONSES[$prompt_hash]}"
        return 0
    fi
    
    # Generate contextual response
    local response=""
    if [[ "$prompt" =~ (test|spec|unit) ]]; then
        response="I'll help you create comprehensive tests.

\`\`\`javascript
describe('Component', () => {
  it('should work correctly', () => {
    expect(component.render()).toBeTruthy();
  });
});
\`\`\`

The tests cover all edge cases and maintain 100% coverage."
    elif [[ "$prompt" =~ (fix|debug|error) ]]; then
        response="I've identified the issue in your code:

The problem is on line 42 where the variable is undefined.
Here's the fix:

\`\`\`diff
- const result = undefinedVar.property;
+ const result = definedVar?.property || defaultValue;
\`\`\`

This handles the undefined case gracefully."
    elif [[ "$prompt" =~ (explain|what|how) ]]; then
        response="Let me explain this concept:

This code implements a design pattern that separates concerns and improves maintainability. 
The key components are:
1. Data layer - handles persistence
2. Business logic - processes rules
3. Presentation - manages UI

This architecture ensures scalability and testability."
    else
        response="I understand your request. Here's my response:

Based on the context provided, I recommend implementing a solution that balances performance and maintainability.
The approach should be modular and follow best practices.

Would you like me to elaborate on any specific aspect?"
    fi
    
    # Cache and return
    CLAUDE_RESPONSES[$prompt_hash]="$response"
    echo "$response"
}

# === Chat Command ===
claude_cmd_chat() {
    local session_id=$(claude_generate_id)
    CLAUDE_SESSIONS[$session_id]="active|0|"
    
    echo "Starting Claude chat session: $session_id"
    echo "Type 'exit' to end the session"
    echo ""
    echo "Claude: Hello! How can I help you today?"
    
    # Simulate interactive session (in real usage would read stdin)
    local turns=1
    CLAUDE_SESSIONS[$session_id]="active|$turns|context"
}

# === Analyze Command ===
claude_cmd_analyze() {
    local file="${1:-}"
    [[ -z "$file" ]] && { echo "Error: file path required" >&2; return 1; }
    
    echo "Analyzing: $file"
    echo ""
    echo "Code Analysis Results:"
    echo "====================="
    echo "• Complexity: Medium (Cyclomatic: 8)"
    echo "• Lines of Code: 156"
    echo "• Test Coverage: 75%"
    echo ""
    echo "Potential Issues:"
    echo "• Line 23: Possible null reference"
    echo "• Line 45: Unused variable 'temp'"
    echo "• Line 89: Consider extracting method"
    echo ""
    echo "Suggestions:"
    echo "• Add error handling for edge cases"
    echo "• Improve variable naming consistency"
    echo "• Consider adding unit tests for uncovered branches"
}

# === Generate Command ===
claude_cmd_generate() {
    local type="${1:-code}"
    local description="${*:2}"
    
    case "$type" in
        code)
            echo "Generating code based on requirements..."
            echo ""
            echo "\`\`\`python"
            echo "def process_data(input_data):"
            echo "    \"\"\"Process input data according to specifications.\"\"\""
            echo "    result = []"
            echo "    for item in input_data:"
            echo "        if validate(item):"
            echo "            result.append(transform(item))"
            echo "    return result"
            echo "\`\`\`"
            ;;
        test)
            echo "Generating test suite..."
            echo ""
            echo "\`\`\`python"
            echo "import pytest"
            echo ""
            echo "def test_process_data():"
            echo "    assert process_data([1, 2, 3]) == [2, 4, 6]"
            echo "    assert process_data([]) == []"
            echo "\`\`\`"
            ;;
        docs)
            echo "Generating documentation..."
            echo ""
            echo "# API Documentation"
            echo "## Overview"
            echo "This module provides data processing capabilities."
            echo ""
            echo "## Functions"
            echo "### process_data(input_data)"
            echo "Processes input data and returns transformed results."
            ;;
        *)
            echo "Generating $type..."
            echo "Generated content for: $description"
            ;;
    esac
}

# === Project Command ===
claude_cmd_project() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Projects:"
            if [[ ${#CLAUDE_PROJECTS[@]} -eq 0 ]]; then
                echo "  (none)"
            else
                for name in "${!CLAUDE_PROJECTS[@]}"; do
                    local data="${CLAUDE_PROJECTS[$name]}"
                    IFS='|' read -r path files status <<< "$data"
                    echo "  $name - Path: $path, Files: $files, Status: $status"
                done
            fi
            ;;
        add)
            local name="${1:-}"
            local path="${2:-.}"
            [[ -z "$name" ]] && { echo "Error: project name required" >&2; return 1; }
            
            CLAUDE_PROJECTS[$name]="$path|0|active"
            echo "Project '$name' added at $path"
            ;;
        remove)
            local name="${1:-}"
            [[ -z "$name" ]] && { echo "Error: project name required" >&2; return 1; }
            
            if [[ -n "${CLAUDE_PROJECTS[$name]}" ]]; then
                unset CLAUDE_PROJECTS[$name]
                echo "Project '$name' removed"
            else
                echo "Error: project not found: $name" >&2
                return 1
            fi
            ;;
        *)
            echo "Usage: claude project {list|add|remove} [args]"
            return 1
            ;;
    esac
}

# === Config Command ===
claude_cmd_config() {
    local key="${1:-}"
    local value="${2:-}"
    
    if [[ -z "$key" ]]; then
        echo "Configuration:"
        for k in "${!CLAUDE_CONFIG[@]}"; do
            echo "  $k: ${CLAUDE_CONFIG[$k]}"
        done
    elif [[ -z "$value" ]]; then
        echo "${CLAUDE_CONFIG[$key]:-}"
    else
        CLAUDE_CONFIG[$key]="$value"
        echo "Set $key = $value"
    fi
}

# === Tools Command ===
claude_cmd_tools() {
    local action="${1:-list}"
    shift || true
    
    case "$action" in
        list)
            echo "Available tools:"
            echo "  read_file - Read file contents"
            echo "  write_file - Write to files"
            echo "  run_command - Execute commands"
            echo "  search - Search in files"
            for tool in "${!CLAUDE_TOOLS[@]}"; do
                local data="${CLAUDE_TOOLS[$tool]}"
                IFS='|' read -r enabled calls <<< "$data"
                echo "  $tool - Enabled: $enabled, Calls: $calls"
            done
            ;;
        enable)
            local tool="${1:-}"
            [[ -z "$tool" ]] && { echo "Error: tool name required" >&2; return 1; }
            CLAUDE_TOOLS[$tool]="true|0"
            echo "Tool '$tool' enabled"
            ;;
        disable)
            local tool="${1:-}"
            [[ -z "$tool" ]] && { echo "Error: tool name required" >&2; return 1; }
            CLAUDE_TOOLS[$tool]="false|0"
            echo "Tool '$tool' disabled"
            ;;
        *)
            echo "Usage: claude tools {list|enable|disable} [tool]"
            return 1
            ;;
    esac
}

# === Mock Control Functions ===
claude_mock_reset() {
    claude_debug "Resetting mock state"
    
    CLAUDE_SESSIONS=()
    CLAUDE_RESPONSES=()
    CLAUDE_PROJECTS=()
    CLAUDE_TOOLS=()
    CLAUDE_CONFIG[error_mode]=""
    
    # Initialize defaults
    claude_mock_init_defaults
}

claude_mock_init_defaults() {
    # Default tools
    CLAUDE_TOOLS["read_file"]="true|0"
    CLAUDE_TOOLS["write_file"]="true|0"
    CLAUDE_TOOLS["run_command"]="false|0"
}

claude_mock_set_error() {
    CLAUDE_CONFIG[error_mode]="$1"
    claude_debug "Set error mode: $1"
}

claude_mock_set_response() {
    local prompt="$1"
    local response="$2"
    local prompt_hash=$(claude_hash_prompt "$prompt")
    CLAUDE_RESPONSES[$prompt_hash]="$response"
    claude_debug "Set custom response for prompt"
}

claude_mock_dump_state() {
    echo "=== Claude Mock State ==="
    echo "Model: ${CLAUDE_CONFIG[model]}"
    echo "Max Tokens: ${CLAUDE_CONFIG[max_tokens]}"
    echo "Sessions: ${#CLAUDE_SESSIONS[@]}"
    echo "Cached Responses: ${#CLAUDE_RESPONSES[@]}"
    echo "Projects: ${#CLAUDE_PROJECTS[@]}"
    echo "Tools: ${#CLAUDE_TOOLS[@]}"
    echo "Error Mode: ${CLAUDE_CONFIG[error_mode]:-none}"
    echo "==================="
}

# === Convention-based Test Functions ===
test_claude_connection() {
    claude_debug "Testing connection..."
    
    local result
    result=$(claude version 2>&1)
    
    if [[ "$result" =~ "Claude CLI" ]]; then
        claude_debug "Connection test passed"
        return 0
    else
        claude_debug "Connection test failed"
        return 1
    fi
}

test_claude_health() {
    claude_debug "Testing health..."
    
    test_claude_connection || return 1
    
    claude "test prompt" >/dev/null 2>&1 || return 1
    claude config model >/dev/null 2>&1 || return 1
    
    claude_debug "Health test passed"
    return 0
}

test_claude_basic() {
    claude_debug "Testing basic operations..."
    
    claude project add test-project /tmp >/dev/null 2>&1 || return 1
    claude tools enable search >/dev/null 2>&1 || return 1
    claude generate code "hello world function" >/dev/null 2>&1 || return 1
    
    claude_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f claude
export -f test_claude_connection test_claude_health test_claude_basic
export -f claude_mock_reset claude_mock_set_error claude_mock_set_response
export -f claude_mock_dump_state
export -f claude_debug claude_check_error

# Initialize with defaults
claude_mock_reset
claude_debug "Claude CLI Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
