#!/bin/bash
# ====================================================================
# File System Assertion Functions
# ====================================================================
#
# File and directory assertion functions for testing file operations
# and system state.
#
# Functions:
#   - assert_file_exists()     - Assert file exists
#   - assert_dir_exists()      - Assert directory exists
#   - assert_command_success() - Assert command executes successfully
#   - assert_command_fails()   - Assert command fails
#   - wait_for_condition()     - Wait for condition with timeout
#
# ====================================================================

# Assert file exists
assert_file_exists() {
    local file_path="$1"
    local message="${2:-File existence assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ -f "$file_path" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  File not found: '$file_path'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert directory exists
assert_dir_exists() {
    local dir_path="$1"
    local message="${2:-Directory existence assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if [[ -d "$dir_path" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Directory not found: '$dir_path'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert command executes successfully
assert_command_success() {
    local command="$1"
    local message="${2:-Command success assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if eval "$command" >/dev/null 2>&1; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Command failed: '$command'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert command fails (expects non-zero exit code)
assert_command_fails() {
    local command="$1"
    local message="${2:-Command failure assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    if ! eval "$command" >/dev/null 2>&1; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Command should have failed: '$command'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Wait for condition with timeout
wait_for_condition() {
    local condition="$1"
    local timeout="${2:-30}"
    local interval="${3:-1}"
    local message="${4:-Waiting for condition}"
    
    local elapsed=0
    echo -n "⏳ $message"
    
    while [[ $elapsed -lt $timeout ]]; do
        if eval "$condition" >/dev/null 2>&1; then
            echo -e " ${ASSERT_GREEN}✓${ASSERT_NC}"
            return 0
        fi
        
        echo -n "."
        sleep "$interval"
        elapsed=$((elapsed + interval))
    done
    
    echo -e " ${ASSERT_RED}X${ASSERT_NC} (timeout after ${timeout}s)"
    return 1
}