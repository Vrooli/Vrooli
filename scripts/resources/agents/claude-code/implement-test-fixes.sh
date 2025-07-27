#!/usr/bin/env bash
# Implement proper test fixes for Claude Code BATS tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Implementing Permanent Test Fixes ==="
echo

# Fix 1: Create test helper for better function mocking
cat > test_helper.bash << 'EOF'
#!/usr/bin/env bash

# Helper function to test without using 'run' for better variable control
test_without_run() {
    local func="$1"
    shift
    local output
    local status
    
    output=$("$func" "$@" 2>&1) || status=$?
    status=${status:-0}
    
    echo "$output"
    return $status
}

# Helper to override readonly variables in tests
override_readonly() {
    local var_name="$1"
    local var_value="$2"
    
    # Use gdb if available, otherwise try declare +r
    if command -v gdb &>/dev/null; then
        gdb -batch -ex "call unbind_variable(\"$var_name\")" -ex quit $$ &>/dev/null || true
    else
        declare +r "$var_name" 2>/dev/null || true
    fi
    
    eval "$var_name=\"$var_value\""
}
EOF

# Fix 2: Update status.bats to use better testing patterns
cat > lib/status.bats.fixed << 'EOF'
#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

load ../test_helper

setup() {
    export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME:-$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)}"
    export CLAUDE_CODE_DIR="$BATS_TEST_DIRNAME/.."
    export RESOURCES_DIR="$CLAUDE_CODE_DIR/../.."
    export HELPERS_DIR="$RESOURCES_DIR/../helpers"
    export SCRIPT_PATH="$BATS_TEST_DIRNAME/status.sh"
    
    source "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/flow.sh" 2>/dev/null || true
    source "$RESOURCES_DIR/common.sh" 2>/dev/null || true
    source "$CLAUDE_CODE_DIR/config/defaults.sh"
    source "$CLAUDE_CODE_DIR/config/messages.sh" 2>/dev/null || true
    source "$CLAUDE_CODE_DIR/lib/common.sh"
    source "$SCRIPT_PATH"
    
    # Default mocks
    confirm() { return 0; }
    system::is_command() { 
        case "$1" in
            jq) return 0 ;;
            *) return 0 ;;
        esac
    }
}

@test "status.sh defines required functions" {
    declare -f claude_code::status
    declare -f claude_code::info
    declare -f claude_code::logs
}

@test "claude_code::status shows not installed when missing" {
    output=$(
        claude_code::is_installed() { return 1; }
        claude_code::status 2>&1
    )
    [[ "$output" =~ "Not installed" ]]
}

@test "claude_code::status shows installed when available" {
    output=$(
        claude_code::is_installed() { return 0; }
        claude_code::status 2>&1
    )
    [[ "$output" =~ "Installed" ]]
}

@test "claude_code::info shows message when not installed" {
    output=$(
        claude_code::is_installed() { return 1; }
        claude_code::info 2>&1
    )
    [[ "$output" =~ "not installed" ]]
}

@test "claude_code::info displays version info" {
    output=$(
        claude_code::is_installed() { return 0; }
        claude() { echo "Claude Code 1.0.0"; }
        claude_code::info 2>&1
    )
    [[ "$output" =~ "1.0.0" ]]
}

@test "claude_code::logs handles missing log file" {
    output=$(
        CLAUDE_LOGS_FILE="/nonexistent/claude.log"
        claude_code::logs 2>&1
    )
    [[ "$output" =~ "No log file found" ]]
}

@test "claude_code::logs shows recent logs with tail" {
    TMP_FILE=$(mktemp)
    for i in {1..30}; do
        echo "Log line $i" >> "$TMP_FILE"
    done
    
    output=$(
        CLAUDE_LOGS_FILE="$TMP_FILE"
        tail() { /usr/bin/tail "$@"; }
        claude_code::logs 2>&1
    )
    
    rm -f "$TMP_FILE"
    [[ "$output" =~ "Log line 30" ]]
    [[ ! "$output" =~ "Log line 1" ]]
}

@test "claude_code::logs follows logs with -f flag" {
    TMP_FILE=$(mktemp)
    echo "Initial log" > "$TMP_FILE"
    
    output=$(
        CLAUDE_LOGS_FILE="$TMP_FILE"
        LOG_FOLLOW=1
        tail() { echo "Following logs..."; }
        claude_code::logs 2>&1
    )
    
    rm -f "$TMP_FILE"
    [[ "$output" =~ "Following logs" ]]
}

@test "claude_code::logs handles custom line count" {
    TMP_FILE=$(mktemp)
    for i in {1..100}; do
        echo "Log line $i" >> "$TMP_FILE"
    done
    
    output=$(
        CLAUDE_LOGS_FILE="$TMP_FILE"
        LOG_LINES=5
        tail() { /usr/bin/tail "$@"; }
        claude_code::logs 2>&1
    )
    
    rm -f "$TMP_FILE"
    [[ "$output" =~ "Log line 100" ]]
    [[ "$output" =~ "Log line 96" ]]
    [[ ! "$output" =~ "Log line 95" ]]
}
EOF

# Fix 3: Update execute.bats for better command testing
cat > lib/execute.bats.fixed << 'EOF'
#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

load ../test_helper

setup() {
    export BATS_TEST_DIRNAME="${BATS_TEST_DIRNAME:-$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)}"
    export CLAUDE_CODE_DIR="$BATS_TEST_DIRNAME/.."
    export RESOURCES_DIR="$CLAUDE_CODE_DIR/../.."
    export HELPERS_DIR="$RESOURCES_DIR/../helpers"
    export SCRIPT_PATH="$BATS_TEST_DIRNAME/execute.sh"
    
    source "$HELPERS_DIR/utils/log.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/system.sh" 2>/dev/null || true
    source "$HELPERS_DIR/utils/flow.sh" 2>/dev/null || true
    source "$RESOURCES_DIR/common.sh" 2>/dev/null || true
    source "$CLAUDE_CODE_DIR/config/defaults.sh"
    source "$CLAUDE_CODE_DIR/config/messages.sh" 2>/dev/null || true
    source "$CLAUDE_CODE_DIR/lib/common.sh"
    source "$SCRIPT_PATH"
    
    confirm() { return 0; }
    system::is_command() { return 0; }
}

@test "execute.sh defines required functions" {
    declare -f claude_code::run
    declare -f claude_code::batch
}

@test "claude_code::run fails when not installed" {
    claude_code::is_installed() { return 1; }
    run claude_code::run 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Claude Code is not installed" ]]
}

@test "claude_code::run fails without prompt" {
    claude_code::is_installed() { return 0; }
    PROMPT=''
    run claude_code::run 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No prompt provided" ]]
}

@test "claude_code::run builds basic command correctly" {
    output=$(
        claude_code::is_installed() { return 0; }
        PROMPT='Test prompt'
        MAX_TURNS=5
        OUTPUT_FORMAT='text'
        ALLOWED_TOOLS=''
        eval() { echo "Command: $*"; }
        claude_code::run 2>&1
    )
    [[ "$output" =~ "Command: claude --prompt \"Test prompt\" --max-turns 5" ]]
}

@test "claude_code::run handles stream-json format" {
    output=$(
        claude_code::is_installed() { return 0; }
        PROMPT='Test'
        OUTPUT_FORMAT='stream-json'
        ALLOWED_TOOLS=''
        eval() { echo "Command: $*"; }
        claude_code::run 2>&1
    )
    [[ "$output" =~ "--output-format stream-json" ]]
}

@test "claude_code::run adds allowed tools" {
    output=$(
        claude_code::is_installed() { return 0; }
        PROMPT='Test'
        ALLOWED_TOOLS='tool1,tool2'
        eval() { echo "Command: $*"; }
        claude_code::run 2>&1
    )
    [[ "$output" =~ '--allowedTools "tool1"' ]]
    [[ "$output" =~ '--allowedTools "tool2"' ]]
}

@test "claude_code::run sets timeout environment variables" {
    output=$(
        claude_code::is_installed() { return 0; }
        PROMPT='Test'
        TIMEOUT=300
        ALLOWED_TOOLS=''
        eval() { 
            echo "BASH_DEFAULT_TIMEOUT_MS=$BASH_DEFAULT_TIMEOUT_MS"
            echo "BASH_MAX_TIMEOUT_MS=$BASH_MAX_TIMEOUT_MS"
            echo "MCP_TOOL_TIMEOUT=$MCP_TOOL_TIMEOUT"
        }
        claude_code::run 2>&1
    )
    [[ "$output" =~ "BASH_DEFAULT_TIMEOUT_MS=300000" ]]
    [[ "$output" =~ "BASH_MAX_TIMEOUT_MS=300000" ]]
    [[ "$output" =~ "MCP_TOOL_TIMEOUT=300000" ]]
}

@test "claude_code::batch fails when not installed" {
    claude_code::is_installed() { return 1; }
    run claude_code::batch 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Claude Code is not installed" ]]
}

@test "claude_code::batch fails without prompt" {
    claude_code::is_installed() { return 0; }
    PROMPT=''
    run claude_code::batch 2>&1
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No prompt provided" ]]
}

@test "claude_code::batch uses stream-json and no-interactive" {
    output=$(
        claude_code::is_installed() { return 0; }
        PROMPT='Test batch'
        MAX_TURNS=10
        ALLOWED_TOOLS=''
        eval() { echo "Command: $*"; }
        claude_code::batch 2>&1
    )
    [[ "$output" =~ "--output-format stream-json" ]]
    [[ "$output" =~ "--no-interactive" ]]
}

@test "claude_code::batch creates output file" {
    output=$(
        claude_code::is_installed() { return 0; }
        PROMPT='Test batch'
        ALLOWED_TOOLS=''
        eval() { echo 'test output' > /tmp/claude-batch-test.json; return 0; }
        RANDOM=test
        claude_code::batch 2>&1
    )
    rm -f /tmp/claude-batch-test.json
    [[ "$output" =~ "Batch completed" ]]
    [[ "$output" =~ "Output saved to:" ]]
}

@test "claude_code::batch handles eval failure" {
    output=$(
        claude_code::is_installed() { return 0; }
        PROMPT='Test batch'
        ALLOWED_TOOLS=''
        eval() { return 1; }
        claude_code::batch 2>&1
    ) || status=$?
    [ "${status:-0}" -eq 1 ]
    [[ "$output" =~ "Batch execution failed" ]]
}
EOF

echo "Created test fixes. To apply them:"
echo "1. Review the fixed test files (*.fixed)"
echo "2. Run: mv lib/status.bats.fixed lib/status.bats"
echo "3. Run: mv lib/execute.bats.fixed lib/execute.bats"
echo "4. Test with: bats lib/status.bats lib/execute.bats"