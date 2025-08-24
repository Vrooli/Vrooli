#!/usr/bin/env bash
# Verification Mock - Tier 2 (Stateful)
# 
# Provides test verification and tracking for mocks:
# - Mock call tracking and validation
# - State verification
# - Expected vs actual comparisons
# - Test assertions
# - Error injection for testing
#
# Coverage: ~80% of common verification operations in 300 lines

# === Configuration ===
declare -gA VERIFY_CONFIG=(
    [status]="active"
    [strict]="false"
    [auto_verify]="false"
    [error_mode]=""
    [version]="2.0.0"
)

declare -gA VERIFY_CALLS=()       # Call tracking
declare -gA VERIFY_EXPECTED=()    # Expected calls
declare -gA VERIFY_STATE=()       # State tracking
declare -ga VERIFY_HISTORY=()     # Call history
declare -gi VERIFY_CALL_COUNT=0

# Debug mode
declare -g VERIFY_DEBUG="${VERIFY_DEBUG:-}"

# === Helper Functions ===
verify_debug() {
    [[ -n "$VERIFY_DEBUG" ]] && echo "[MOCK:VERIFY] $*" >&2
}

verify_check_error() {
    case "${VERIFY_CONFIG[error_mode]}" in
        "assertion_failed")
            echo "Error: Assertion failed" >&2
            return 1
            ;;
        "unexpected_call")
            echo "Error: Unexpected mock call" >&2
            return 1
            ;;
        "missing_call")
            echo "Error: Expected call not made" >&2
            return 1
            ;;
    esac
    return 0
}

# === Mock Verification Functions ===
mock::verify_call() {
    local command="$1"
    shift
    local args="$*"
    local call_key="${command}:${args}"
    
    verify_debug "Verifying call: $call_key"
    
    if ! verify_check_error; then
        return $?
    fi
    
    ((VERIFY_CALL_COUNT++))
    VERIFY_HISTORY+=("$call_key")
    
    # Track call count
    local count=${VERIFY_CALLS[$call_key]:-0}
    VERIFY_CALLS[$call_key]=$((count + 1))
    
    # Check if call was expected in strict mode
    if [[ "${VERIFY_CONFIG[strict]}" == "true" ]]; then
        if [[ -z "${VERIFY_EXPECTED[$call_key]:-}" ]]; then
            echo "Unexpected call: $call_key" >&2
            return 1
        fi
    fi
    
    return 0
}

mock::expect_call() {
    local command="$1"
    shift
    local args="$*"
    local count="${1:-1}"
    local call_key="${command}:${args}"
    
    verify_debug "Expecting call: $call_key (count: $count)"
    
    VERIFY_EXPECTED[$call_key]="$count"
}

mock::verify_expectations() {
    verify_debug "Verifying all expectations"
    
    local failed=0
    
    # Check expected calls were made
    for expected_call in "${!VERIFY_EXPECTED[@]}"; do
        local expected_count="${VERIFY_EXPECTED[$expected_call]}"
        local actual_count="${VERIFY_CALLS[$expected_call]:-0}"
        
        if [[ "$actual_count" -ne "$expected_count" ]]; then
            echo "Expected $expected_call to be called $expected_count times, got $actual_count" >&2
            ((failed++))
        fi
    done
    
    return $failed
}

# === State Verification ===
mock::verify_state() {
    local component="$1"
    local expected_state="$2"
    
    verify_debug "Verifying state: $component = $expected_state"
    
    local actual_state="${VERIFY_STATE[$component]:-unknown}"
    
    if [[ "$actual_state" == "$expected_state" ]]; then
        return 0
    else
        echo "State verification failed: $component expected '$expected_state', got '$actual_state'" >&2
        return 1
    fi
}

mock::set_state() {
    local component="$1"
    local state="$2"
    
    verify_debug "Setting state: $component = $state"
    
    VERIFY_STATE[$component]="$state"
}

# === Assertion Functions ===
mock::assert_called() {
    local command="$1"
    shift
    local args="$*"
    local call_key="${command}:${args}"
    
    if [[ "${VERIFY_CALLS[$call_key]:-0}" -gt 0 ]]; then
        return 0
    else
        echo "Assertion failed: $call_key was not called" >&2
        return 1
    fi
}

mock::assert_called_times() {
    local expected_count="$1"
    local command="$2"
    shift 2
    local args="$*"
    local call_key="${command}:${args}"
    
    local actual_count="${VERIFY_CALLS[$call_key]:-0}"
    
    if [[ "$actual_count" -eq "$expected_count" ]]; then
        return 0
    else
        echo "Assertion failed: $call_key called $actual_count times, expected $expected_count" >&2
        return 1
    fi
}

mock::assert_not_called() {
    local command="$1"
    shift
    local args="$*"
    local call_key="${command}:${args}"
    
    if [[ "${VERIFY_CALLS[$call_key]:-0}" -eq 0 ]]; then
        return 0
    else
        echo "Assertion failed: $call_key was called ${VERIFY_CALLS[$call_key]} times" >&2
        return 1
    fi
}

# === Query Functions ===
mock::get_call_count() {
    local command="$1"
    shift
    local args="$*"
    local call_key="${command}:${args}"
    
    echo "${VERIFY_CALLS[$call_key]:-0}"
}

mock::get_total_calls() {
    echo "$VERIFY_CALL_COUNT"
}

mock::list_calls() {
    local command="${1:-}"
    
    if [[ -n "$command" ]]; then
        printf '%s\n' "${VERIFY_HISTORY[@]}" | grep "^${command}:"
    else
        printf '%s\n' "${VERIFY_HISTORY[@]}"
    fi
}

# === Mock Control Functions ===
verification_mock_reset() {
    verify_debug "Resetting mock state"
    
    VERIFY_CALLS=()
    VERIFY_EXPECTED=()
    VERIFY_STATE=()
    VERIFY_HISTORY=()
    VERIFY_CALL_COUNT=0
    VERIFY_CONFIG[error_mode]=""
    VERIFY_CONFIG[strict]="false"
}

verification_mock_set_error() {
    VERIFY_CONFIG[error_mode]="$1"
    verify_debug "Set error mode: $1"
}

verification_mock_dump_state() {
    echo "=== Verification Mock State ==="
    echo "Status: ${VERIFY_CONFIG[status]}"
    echo "Strict Mode: ${VERIFY_CONFIG[strict]}"
    echo "Total Calls: $VERIFY_CALL_COUNT"
    echo "Unique Calls: ${#VERIFY_CALLS[@]}"
    echo "Expected Calls: ${#VERIFY_EXPECTED[@]}"
    echo "Tracked States: ${#VERIFY_STATE[@]}"
    echo "Error Mode: ${VERIFY_CONFIG[error_mode]:-none}"
    echo "======================="
}

# === Configuration Functions ===
mock::set_strict_mode() {
    local enabled="${1:-true}"
    VERIFY_CONFIG[strict]="$enabled"
    verify_debug "Strict mode: $enabled"
}

mock::enable_auto_verify() {
    VERIFY_CONFIG[auto_verify]="true"
    verify_debug "Auto-verify enabled"
}

mock::disable_auto_verify() {
    VERIFY_CONFIG[auto_verify]="false"
    verify_debug "Auto-verify disabled"
}

# === Convention-based Test Functions ===
test_verification_connection() {
    verify_debug "Testing connection..."
    
    # Test basic verification functionality
    mock::verify_call "test" "connection"
    if mock::assert_called "test" "connection"; then
        verify_debug "Connection test passed"
        return 0
    else
        verify_debug "Connection test failed"
        return 1
    fi
}

test_verification_health() {
    verify_debug "Testing health..."
    
    test_verification_connection || return 1
    
    # Test state tracking
    mock::set_state "health" "good"
    mock::verify_state "health" "good" || return 1
    
    verify_debug "Health test passed"
    return 0
}

test_verification_basic() {
    verify_debug "Testing basic operations..."
    
    # Test call tracking
    mock::verify_call "basic" "test" "operation"
    [[ $(mock::get_call_count "basic" "test" "operation") -eq 1 ]] || return 1
    
    # Test expectations
    mock::expect_call "expected" "call"
    mock::verify_call "expected" "call"
    mock::verify_expectations || return 1
    
    verify_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f mock::verify_call
export -f mock::expect_call
export -f mock::verify_expectations
export -f mock::verify_state
export -f mock::set_state
export -f mock::assert_called
export -f mock::assert_called_times
export -f mock::assert_not_called
export -f mock::get_call_count
export -f mock::get_total_calls
export -f mock::list_calls
export -f mock::set_strict_mode
export -f mock::enable_auto_verify
export -f mock::disable_auto_verify
export -f test_verification_connection
export -f test_verification_health
export -f test_verification_basic
export -f verification_mock_reset
export -f verification_mock_set_error
export -f verification_mock_dump_state
export -f verify_debug

# Initialize
verification_mock_reset
verify_debug "Verification Tier 2 mock initialized"
# Ensure we return success when sourced
return 0 2>/dev/null || true
