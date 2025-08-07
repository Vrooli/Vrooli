#!/usr/bin/env bash
# Claude Code CLI mocks for Bats tests.
# Provides comprehensive claude command mocking with a clear mock::claude::* namespace.

# Prevent duplicate loading
if [[ "${CLAUDE_MOCKS_LOADED:-}" == "true" ]]; then
  return 0
fi
export CLAUDE_MOCKS_LOADED="true"

# Load centralized logging utilities
if ! command -v mock::log_call &>/dev/null; then
  # Try to load from relative path
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  if [[ -f "$script_dir/logs.sh" ]]; then
    source "$script_dir/logs.sh"
  else
    echo "[CLAUDE_MOCK] WARNING: Could not load mock logging utilities" >&2
  fi
fi

# ----------------------------
# Global mock state & options
# ----------------------------
# Modes: normal, error, not_installed, auth_error, timeout, rate_limit, model_error
export CLAUDE_MOCK_MODE="${CLAUDE_MOCK_MODE:-normal}"

# Mock state for Claude operations
declare -A MOCK_CLAUDE_RESPONSES=()        # prompt -> response_content
declare -A MOCK_CLAUDE_SESSIONS=()         # session_id -> session_data
declare -A MOCK_CLAUDE_ERRORS=()           # command -> error_type
declare -A MOCK_CLAUDE_CONFIGS=()          # config_key -> config_value
declare -A MOCK_CLAUDE_CALL_HISTORY=()     # call_counter -> full_command
declare -A MOCK_CLAUDE_EXIT_CODES=()       # prompt_hash -> exit_code

# Call tracking
export CLAUDE_MOCK_CALL_COUNTER=0
export CLAUDE_MOCK_SESSION_COUNTER=0

# File-based state persistence for subshell access (BATS compatibility)
# Ensure MOCK_LOG_DIR exists before using it
if [[ -z "${MOCK_LOG_DIR:-}" ]]; then
  # Try common test output directories
  if [[ -d "/home/matthalloran8/Vrooli/data/test-outputs/mock-logs" ]]; then
    export MOCK_LOG_DIR="/home/matthalloran8/Vrooli/data/test-outputs/mock-logs"
  elif [[ -d "/tmp" ]]; then
    export MOCK_LOG_DIR="/tmp"
  else
    export MOCK_LOG_DIR="."
  fi
fi

# Ensure the directory exists
[[ ! -d "$MOCK_LOG_DIR" ]] && mkdir -p "$MOCK_LOG_DIR" 2>/dev/null || true

export CLAUDE_MOCK_STATE_FILE="${MOCK_LOG_DIR}/claude_mock_state.$$"

# Initialize state file
_claude_mock_init_state_file() {
  # Ensure directory exists first
  local state_dir
  state_dir=$(dirname "${CLAUDE_MOCK_STATE_FILE}")
  [[ ! -d "$state_dir" ]] && mkdir -p "$state_dir" 2>/dev/null || true
  
  if [[ -n "${CLAUDE_MOCK_STATE_FILE}" ]]; then
    {
      echo "declare -A MOCK_CLAUDE_RESPONSES=()"
      echo "declare -A MOCK_CLAUDE_SESSIONS=()"
      echo "declare -A MOCK_CLAUDE_ERRORS=()"
      echo "declare -A MOCK_CLAUDE_CONFIGS=()"
      echo "declare -A MOCK_CLAUDE_CALL_HISTORY=()"
      echo "declare -A MOCK_CLAUDE_EXIT_CODES=()"
      echo "export CLAUDE_MOCK_CALL_COUNTER=0"
      echo "export CLAUDE_MOCK_SESSION_COUNTER=0"
    } > "$CLAUDE_MOCK_STATE_FILE"
  fi
}

# Save current state to file
_claude_mock_save_state() {
  if [[ -n "${CLAUDE_MOCK_STATE_FILE}" ]]; then
    # Ensure directory exists first
    local state_dir
    state_dir=$(dirname "${CLAUDE_MOCK_STATE_FILE}")
    [[ ! -d "$state_dir" ]] && mkdir -p "$state_dir" 2>/dev/null || true
    
    # Ensure state file exists
    [[ ! -f "${CLAUDE_MOCK_STATE_FILE}" ]] && _claude_mock_init_state_file
    
    {
      declare -p MOCK_CLAUDE_RESPONSES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_CLAUDE_RESPONSES=()"
      declare -p MOCK_CLAUDE_SESSIONS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_CLAUDE_SESSIONS=()"
      declare -p MOCK_CLAUDE_ERRORS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_CLAUDE_ERRORS=()"
      declare -p MOCK_CLAUDE_CONFIGS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_CLAUDE_CONFIGS=()"
      declare -p MOCK_CLAUDE_CALL_HISTORY 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_CLAUDE_CALL_HISTORY=()"
      declare -p MOCK_CLAUDE_EXIT_CODES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_CLAUDE_EXIT_CODES=()"
      echo "export CLAUDE_MOCK_CALL_COUNTER=${CLAUDE_MOCK_CALL_COUNTER:-0}"
      echo "export CLAUDE_MOCK_SESSION_COUNTER=${CLAUDE_MOCK_SESSION_COUNTER:-0}"
    } > "$CLAUDE_MOCK_STATE_FILE" 2>/dev/null || true
  fi
}

# Load state from file
_claude_mock_load_state() {
  if [[ -n "${CLAUDE_MOCK_STATE_FILE}" && -f "$CLAUDE_MOCK_STATE_FILE" ]]; then
    # Use eval to execute in global scope, not function scope
    eval "$(cat "$CLAUDE_MOCK_STATE_FILE" 2>/dev/null)" 2>/dev/null || true
  fi
}

# Initialize state file
_claude_mock_init_state_file

# ----------------------------
# Utilities
# ----------------------------

# Generate realistic Claude responses based on prompt
_claude_mock_generate_response() {
  local prompt="$1"
  local output_format="${2:-text}"
  local max_turns="${3:-1}"
  
  # Check for custom response first (avoid empty key issues)
  if [[ -n "$prompt" && -n "${MOCK_CLAUDE_RESPONSES[$prompt]:-}" ]]; then
    echo "${MOCK_CLAUDE_RESPONSES[$prompt]}"
    return 0
  fi
  
  # Generate context-aware responses based on prompt content
  local response=""
  if [[ "$prompt" =~ (write.*test|unit.*test|integration.*test|create.*test) ]]; then
    response="I've created comprehensive tests for your code:

\`\`\`javascript
describe('Feature Tests', () => {
  it('should work correctly', () => {
    expect(newFeature()).toBe(true);
  });
});
\`\`\`

All tests are passing successfully."
  elif [[ "$prompt" =~ (status|health|health.*check|system.*test) ]]; then
    response="Claude Code is running and healthy. All systems operational."
  elif [[ "$prompt" =~ (debug|error|fix) ]]; then
    response="I've analyzed the issue. The problem appears to be a missing dependency. Here's how to fix it: npm install missing-package"
  elif [[ "$prompt" =~ (create|build|implement) ]]; then
    response="I'll help you create that feature. Here's the implementation:

\`\`\`javascript
function newFeature() {
  console.log('Feature implemented successfully');
  return true;
}
\`\`\`

The feature has been created and is ready to use."
  elif [[ "$prompt" =~ (explain|what|how) ]]; then
    response="Let me explain this concept. The code works by using event-driven architecture to handle asynchronous operations efficiently."
  else
    response="I understand your request: \"$prompt\". I'm ready to help you with this task. Let me know if you need any clarification or have additional requirements."
  fi
  
  # Format response based on output format
  case "$output_format" in
    "json"|"stream-json")
      cat <<EOF
{
  "id": "msg_$(date +%s)_$$",
  "type": "message", 
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "$response"
    }
  ],
  "model": "claude-3-5-sonnet",
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": $((${#prompt} / 4)),
    "output_tokens": $((${#response} / 4))
  }
}
EOF
      ;;
    *)
      echo "$response"
      ;;
  esac
}

# Generate session data
_claude_mock_generate_session() {
  local session_id="$1"
  local working_dir="${2:-$(pwd)}"
  
  cat <<EOF
{
  "session_id": "$session_id",
  "status": "active",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "working_directory": "$working_dir",
  "model": "claude-3-5-sonnet",
  "turns": 1,
  "files_loaded": ["package.json", "README.md"],
  "context_size": 4096
}
EOF
}

# ----------------------------
# Public functions used by tests
# ----------------------------
mock::claude::reset() {
  # Clear existing arrays by unsetting and recreating
  unset MOCK_CLAUDE_RESPONSES MOCK_CLAUDE_SESSIONS MOCK_CLAUDE_ERRORS
  unset MOCK_CLAUDE_CONFIGS MOCK_CLAUDE_CALL_HISTORY MOCK_CLAUDE_EXIT_CODES
  
  # Recreate as associative arrays
  declare -gA MOCK_CLAUDE_RESPONSES=()
  declare -gA MOCK_CLAUDE_SESSIONS=()
  declare -gA MOCK_CLAUDE_ERRORS=()
  declare -gA MOCK_CLAUDE_CONFIGS=()
  declare -gA MOCK_CLAUDE_CALL_HISTORY=()
  declare -gA MOCK_CLAUDE_EXIT_CODES=()
  
  export CLAUDE_MOCK_CALL_COUNTER=0
  export CLAUDE_MOCK_SESSION_COUNTER=0
  
  # Clean up state file first
  if [[ -n "${CLAUDE_MOCK_STATE_FILE}" && -f "$CLAUDE_MOCK_STATE_FILE" ]]; then
    rm -f "$CLAUDE_MOCK_STATE_FILE"
  fi
  
  # Initialize state file for subshell access
  _claude_mock_init_state_file
  
  echo "[MOCK] claude state reset"
}

# Set mock response for a specific prompt
mock::claude::set_response() {
  local prompt="$1"
  local response="$2"
  
  MOCK_CLAUDE_RESPONSES["$prompt"]="$response"
  
  # Save state to file for subshell access
  _claude_mock_save_state
  
  return 0
}

# Set mock session data
mock::claude::set_session() {
  local session_id="$1"
  local session_data="$2"
  
  MOCK_CLAUDE_SESSIONS["$session_id"]="$session_data"
  
  # Save state to file for subshell access
  _claude_mock_save_state
  
  return 0
}

# Set mock configuration
mock::claude::set_config() {
  local key="$1"
  local value="$2"
  
  MOCK_CLAUDE_CONFIGS["$key"]="$value"
  
  # Save state to file for subshell access
  _claude_mock_save_state
  
  return 0
}

# Inject errors for testing failure scenarios
mock::claude::inject_error() {
  local command="$1"
  local error_type="${2:-general_error}"
  
  MOCK_CLAUDE_ERRORS["$command"]="$error_type"
  
  # Save state to file for subshell access
  _claude_mock_save_state
  
  echo "[MOCK] Injected error for claude command '$command': $error_type"
}

# Set exit code for specific prompt
mock::claude::set_exit_code() {
  local prompt="$1"
  local exit_code="$2"
  
  local prompt_hash=$(echo "$prompt" | sha256sum | cut -d' ' -f1)
  MOCK_CLAUDE_EXIT_CODES["$prompt_hash"]="$exit_code"
  
  # Save state to file for subshell access
  _claude_mock_save_state
  
  return 0
}

# ----------------------------
# claude() main interceptor
# ----------------------------
claude() {
  # Load state from file for subshell access
  if [[ -n "${CLAUDE_MOCK_STATE_FILE}" && -f "$CLAUDE_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$CLAUDE_MOCK_STATE_FILE")" 2>/dev/null || true
  fi

  # Increment call counter (ensure it's initialized)
  CLAUDE_MOCK_CALL_COUNTER=$((${CLAUDE_MOCK_CALL_COUNTER:-0} + 1))
  local call_id="$CLAUDE_MOCK_CALL_COUNTER"
  
  # Store call history
  MOCK_CLAUDE_CALL_HISTORY["$call_id"]="$*"

  # Use centralized logging and verification if available
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "claude" "$*"
  elif command -v mock::log_call &>/dev/null; then
    mock::log_call "claude" "$*"
  fi

  # Check global mock mode
  case "$CLAUDE_MOCK_MODE" in
    error)       echo "Claude: An unexpected error occurred" >&2; return 1 ;;
    not_installed) echo "claude: command not found" >&2; return 127 ;;
    auth_error)  echo "Claude: Authentication failed. Please run 'claude auth login'" >&2; return 1 ;;
    timeout)     echo "Claude: Request timed out" >&2; return 124 ;;
    rate_limit)  echo "Claude: Rate limit exceeded. Please try again later" >&2; return 1 ;;
    model_error) echo "Claude: Model unavailable. Please try a different model" >&2; return 1 ;;
  esac

  # Parse arguments
  local prompt=""
  local prompt_set=false
  local print_mode=false
  local continue_mode=false
  local resume_session=""
  local model="claude-3-5-sonnet"
  local output_format="text"
  local max_turns="1"
  local verbose=false
  local interactive=true
  
  # Handle no arguments (interactive mode)
  if [[ $# -eq 0 ]]; then
    echo "Starting Claude interactive session..."
    echo "Type your message and press Enter. Type 'exit' to quit."
    return 0
  fi

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --version)
        echo "claude-code v3.5.0"
        return 0
        ;;
      -h|--help)
        cat <<'EOF'
Claude Code - AI assistant for software development

Usage: claude [options] [prompt]

Options:
  -p, --print           Print response without interactive mode
  -c, --continue        Continue most recent conversation
      --resume ID       Resume specific session
      --model MODEL     Set model for session (sonnet, haiku, opus)
      --output-format   Specify output format (text, json, stream-json)
      --max-turns N     Limit number of agentic turns
      --verbose         Enable detailed logging
  -h, --help            Show this help message
      --version         Show version information

Authentication:
  claude auth login     Login to Claude
  claude auth logout    Logout from Claude
  claude auth status    Show authentication status

MCP (Model Context Protocol):
  claude mcp            Configure MCP servers
  claude mcp list       List configured MCP servers
  claude mcp add SERVER Add MCP server
  claude mcp remove ID  Remove MCP server

Examples:
  claude                           Start interactive session
  claude "Explain this code"       Quick query
  claude -p "Fix this bug"         Print response and exit
  claude --continue                Continue previous session
  claude --model haiku "Quick task" Use faster model
  
For more information, visit: https://docs.anthropic.com/claude-code
EOF
        return 0
        ;;
      -p|--print)
        print_mode=true
        interactive=false
        shift
        ;;
      -c|--continue)
        continue_mode=true
        shift
        ;;
      --resume)
        resume_session="$2"
        shift 2
        ;;
      --model)
        model="$2"
        shift 2
        ;;
      --output-format)
        output_format="$2"
        shift 2
        ;;
      --max-turns)
        max_turns="$2"
        shift 2
        ;;
      --verbose)
        verbose=true
        shift
        ;;
      auth)
        # Handle auth subcommands
        local auth_cmd="${2:-help}"
        case "$auth_cmd" in
          login)
            echo "Successfully logged in to Claude"
            return 0
            ;;
          logout)
            echo "Successfully logged out from Claude"
            return 0
            ;;
          status)
            echo "Logged in as: user@example.com"
            echo "API Key: cl-***-***"
            echo "Model Access: claude-3-5-sonnet, claude-3-haiku"
            return 0
            ;;
          *)
            echo "Usage: claude auth [login|logout|status]"
            return 1
            ;;
        esac
        ;;
      mcp)
        # Handle MCP subcommands
        local mcp_cmd="${2:-help}"
        case "$mcp_cmd" in
          list)
            echo "Configured MCP servers:"
            echo "  filesystem: @modelcontextprotocol/server-filesystem"
            echo "  browser: @modelcontextprotocol/server-brave-search"
            return 0
            ;;
          add)
            echo "Added MCP server: $3"
            return 0
            ;;
          remove)
            echo "Removed MCP server: $3" 
            return 0
            ;;
          *)
            echo "MCP server configuration..."
            return 0
            ;;
        esac
        ;;
      update)
        echo "Updating Claude Code..."
        echo "Claude Code updated to latest version"
        return 0
        ;;
      -*)
        # Unknown flag, consume it
        shift
        ;;
      *)
        # This is the prompt
        prompt="$1"
        prompt_set=true
        shift
        ;;
    esac
  done

  # Check for injected errors specific to this command
  local command_key
  if [[ "$prompt_set" == "true" ]]; then
    command_key="prompt:$prompt"
  elif [[ "$continue_mode" == "true" ]]; then
    command_key="continue"
  elif [[ -n "$resume_session" ]]; then
    command_key="resume:$resume_session"
  else
    command_key="interactive"
  fi
  
  # Check for specific error first, then check for wildcard errors
  local error_type=""
  if [[ -n "${MOCK_CLAUDE_ERRORS[$command_key]:-}" ]]; then
    error_type="${MOCK_CLAUDE_ERRORS[$command_key]}"
  elif [[ "$prompt_set" == "true" && -n "${MOCK_CLAUDE_ERRORS[prompt:any]:-}" ]]; then
    error_type="${MOCK_CLAUDE_ERRORS[prompt:any]}"
  fi
  
  if [[ -n "$error_type" ]]; then
    case "$error_type" in
      auth_error)
        echo "Claude: Authentication required. Run 'claude auth login'" >&2
        return 1
        ;;
      model_error)
        echo "Claude: Model '$model' is not available" >&2
        return 1
        ;;
      session_error)
        echo "Claude: Session '$resume_session' not found" >&2
        return 1
        ;;
      rate_limit)
        echo "Claude: Rate limit exceeded" >&2
        return 1
        ;;
      timeout)
        echo "Claude: Request timed out after 30s" >&2
        return 124
        ;;
      parse_error)
        echo "Claude: Failed to parse response" >&2
        return 1
        ;;
      *)
        echo "Claude: Error - $error_type" >&2
        return 1
        ;;
    esac
  fi

  # Handle different modes
  if [[ "$continue_mode" == "true" ]]; then
    echo "Continuing previous conversation..."
    local response="I'm continuing from where we left off. How can I help you further?"
    if [[ "$output_format" =~ json ]]; then
      _claude_mock_generate_response "continue conversation" "$output_format" "$max_turns"
    else
      echo "$response"
    fi
  elif [[ -n "$resume_session" ]]; then
    if [[ -n "${MOCK_CLAUDE_SESSIONS[$resume_session]:-}" ]]; then
      echo "Resuming session: $resume_session"
      echo "${MOCK_CLAUDE_SESSIONS[$resume_session]}"
    else
      # Generate default session
      ((CLAUDE_MOCK_SESSION_COUNTER++))
      local session_data
      session_data="$(_claude_mock_generate_session "$resume_session")"
      mock::claude::set_session "$resume_session" "$session_data"
      echo "Resuming session: $resume_session"
      echo "$session_data"
    fi
  elif [[ "$prompt_set" == "true" ]]; then
    # Handle prompt execution (including empty prompts)
    if [[ "$verbose" == "true" ]]; then
      echo "[DEBUG] Processing prompt with model: $model" >&2
      echo "[DEBUG] Output format: $output_format" >&2
      echo "[DEBUG] Max turns: $max_turns" >&2
    fi
    
    # Check for custom exit code
    local prompt_hash=$(echo "$prompt" | sha256sum | cut -d' ' -f1)
    local custom_exit_code="${MOCK_CLAUDE_EXIT_CODES[$prompt_hash]:-0}"
    
    # Generate and output response
    local response
    response="$(_claude_mock_generate_response "$prompt" "$output_format" "$max_turns")"
    echo "$response"
    
    # Return custom exit code if set
    if [[ "$custom_exit_code" != "0" ]]; then
      return "$custom_exit_code"
    fi
  else
    # Interactive mode (no prompt provided)
    echo "Starting Claude interactive session..."
    echo "Model: $model"
    echo "Type your message and press Enter. Type 'exit' to quit."
  fi

  # Save state after command
  _claude_mock_save_state

  return 0
}

# ----------------------------
# Test Helper Functions
# ----------------------------

# Set up common scenarios
mock::claude::scenario::setup_healthy() {
  # Ensure state file is initialized before trying to use it
  if [[ ! -f "${CLAUDE_MOCK_STATE_FILE:-}" ]]; then
    _claude_mock_init_state_file
  fi
  
  mock::claude::set_config "auth_status" "logged_in"
  mock::claude::set_config "model_access" "claude-3-5-sonnet,claude-3-haiku"
  mock::claude::set_response "health check" "Claude Code is running and healthy. All systems operational."
  echo "[MOCK] Set up healthy Claude scenario"
}

mock::claude::scenario::setup_auth_required() {
  mock::claude::inject_error "prompt:any" "auth_error" >/dev/null
  mock::claude::inject_error "continue" "auth_error" >/dev/null
  mock::claude::inject_error "interactive" "auth_error" >/dev/null
  echo "[MOCK] Set up auth required scenario"
}

mock::claude::scenario::setup_development() {
  mock::claude::set_response "create function" "I'll help you create that function. Here's the implementation:

\`\`\`javascript
function myFunction() {
  return 'Hello, World!';
}
\`\`\`"
  mock::claude::set_response "debug error" "I found the issue. The variable is undefined on line 15. Here's the fix:

\`\`\`diff
- const result = undefinedVar.process();
+ const result = myVar.process();
\`\`\`"
  echo "[MOCK] Set up development scenario"
}

# Assertion helpers
mock::claude::assert::response_contains() {
  local expected="$1"
  local prompt="$2"
  
  local actual
  if ! actual="$(claude -p "$prompt")"; then
    echo "ASSERTION FAILED: claude command failed for prompt '$prompt'" >&2
    return 1
  fi
  
  if [[ "$actual" != *"$expected"* ]]; then
    echo "ASSERTION FAILED: Response doesn't contain '$expected'" >&2
    echo "Actual response: $actual" >&2
    return 1
  fi
  
  return 0
}

mock::claude::assert::command_called() {
  local command_pattern="$1"
  
  for call in "${MOCK_CLAUDE_CALL_HISTORY[@]}"; do
    if [[ "$call" =~ $command_pattern ]]; then
      return 0
    fi
  done
  
  echo "ASSERTION FAILED: Command matching '$command_pattern' was not called" >&2
  return 1
}

mock::claude::assert::call_count() {
  local expected_count="$1"
  
  # Load state from file for subshell access
  if [[ -n "${CLAUDE_MOCK_STATE_FILE}" && -f "$CLAUDE_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$CLAUDE_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  if [[ "$CLAUDE_MOCK_CALL_COUNTER" != "$expected_count" ]]; then
    echo "ASSERTION FAILED: Expected $expected_count calls, got $CLAUDE_MOCK_CALL_COUNTER" >&2
    return 1
  fi
  
  return 0
}

mock::claude::assert::exit_code() {
  local expected_code="$1"
  local prompt="$2"
  
  local actual_code
  claude -p "$prompt" >/dev/null 2>&1
  actual_code=$?
  
  if [[ "$actual_code" != "$expected_code" ]]; then
    echo "ASSERTION FAILED: Expected exit code $expected_code, got $actual_code" >&2
    return 1
  fi
  
  return 0
}

# Get call history for analysis
mock::claude::get::call_history() {
  for i in "${!MOCK_CLAUDE_CALL_HISTORY[@]}"; do
    echo "$i: ${MOCK_CLAUDE_CALL_HISTORY[$i]}"
  done
}

mock::claude::get::last_call() {
  if [[ "$CLAUDE_MOCK_CALL_COUNTER" -gt 0 ]]; then
    echo "${MOCK_CLAUDE_CALL_HISTORY[$CLAUDE_MOCK_CALL_COUNTER]}"
  else
    echo ""
  fi
}

# Debug helper
mock::claude::debug::dump_state() {
  echo "=== Claude Mock State Dump ==="
  echo "Call Counter: $CLAUDE_MOCK_CALL_COUNTER"
  echo "Session Counter: $CLAUDE_MOCK_SESSION_COUNTER"
  echo "Mode: $CLAUDE_MOCK_MODE"
  echo "Custom Responses:"
  for prompt in "${!MOCK_CLAUDE_RESPONSES[@]}"; do
    echo "  '$prompt': ${MOCK_CLAUDE_RESPONSES[$prompt]:0:50}..."
  done
  echo "Sessions:"
  for session_id in "${!MOCK_CLAUDE_SESSIONS[@]}"; do
    echo "  $session_id: ${MOCK_CLAUDE_SESSIONS[$session_id]:0:50}..."
  done
  echo "Configurations:"
  for key in "${!MOCK_CLAUDE_CONFIGS[@]}"; do
    echo "  $key: ${MOCK_CLAUDE_CONFIGS[$key]}"
  done
  echo "Injected Errors:"
  for command in "${!MOCK_CLAUDE_ERRORS[@]}"; do
    echo "  $command: ${MOCK_CLAUDE_ERRORS[$command]}"
  done
  echo "Call History:"
  mock::claude::get::call_history
  echo "=========================="
}

# ----------------------------
# Export functions into subshells
# ----------------------------
export -f claude
export -f _claude_mock_generate_response
export -f _claude_mock_generate_session
export -f _claude_mock_init_state_file
export -f _claude_mock_save_state
export -f _claude_mock_load_state

export -f mock::claude::reset
export -f mock::claude::set_response
export -f mock::claude::set_session
export -f mock::claude::set_config
export -f mock::claude::inject_error
export -f mock::claude::set_exit_code

# Export test helper functions
export -f mock::claude::scenario::setup_healthy
export -f mock::claude::scenario::setup_auth_required
export -f mock::claude::scenario::setup_development
export -f mock::claude::assert::response_contains
export -f mock::claude::assert::command_called
export -f mock::claude::assert::call_count
export -f mock::claude::assert::exit_code
export -f mock::claude::get::call_history
export -f mock::claude::get::last_call
export -f mock::claude::debug::dump_state

echo "[MOCK] claude mocks loaded successfully"