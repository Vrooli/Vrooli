#!/usr/bin/env bash
# Mock Verification System
# Enhanced tracking and validation of mock calls and interactions

# Prevent duplicate loading
if [[ "${MOCK_VERIFICATION_LOADED:-}" == "true" ]]; then
    return 0
fi
export MOCK_VERIFICATION_LOADED="true"

# Mock verification state
declare -gA MOCK_CALL_COUNTS
declare -gA MOCK_CALL_HISTORY
declare -gA MOCK_EXPECTED_CALLS
declare -gA MOCK_CALL_PATTERNS
declare -gA MOCK_CALL_SEQUENCES

# Global verification configuration
export MOCK_VERIFICATION_ENABLED="${MOCK_VERIFICATION_ENABLED:-true}"
export MOCK_VERIFICATION_STRICT="${MOCK_VERIFICATION_STRICT:-false}"

#######################################
# Initialize mock verification system
# Sets up tracking directories and logging
#######################################
mock::verify::init() {
    if [[ "$MOCK_VERIFICATION_ENABLED" != "true" ]]; then
        return 0
    fi
    
    # Ensure mock responses directory exists
    mkdir -p "${MOCK_RESPONSES_DIR}"
    
    # Initialize tracking files
    echo "" > "${MOCK_RESPONSES_DIR}/command_calls.log"
    echo "" > "${MOCK_RESPONSES_DIR}/http_calls.log"
    echo "" > "${MOCK_RESPONSES_DIR}/docker_calls.log"
    echo "" > "${MOCK_RESPONSES_DIR}/verification_errors.log"
    
    # Reset verification state
    MOCK_CALL_COUNTS=()
    MOCK_CALL_HISTORY=()
    MOCK_EXPECTED_CALLS=()
    MOCK_CALL_PATTERNS=()
    MOCK_CALL_SEQUENCES=()
    
    echo "[MOCK_VERIFICATION] Verification system initialized"
}

#######################################
# Record a mock call
# Arguments: $1 - call type, $2 - call details, $3 - timestamp (optional)
#######################################
mock::verify::record_call() {
    if [[ "$MOCK_VERIFICATION_ENABLED" != "true" ]]; then
        return 0
    fi
    
    local call_type="$1"
    local call_details="$2"
    local timestamp="${3:-$(date +%s)}"
    local call_id="${call_type}:${call_details}"
    
    # Increment call count
    local current_count="${MOCK_CALL_COUNTS[$call_id]:-0}"
    MOCK_CALL_COUNTS["$call_id"]=$((current_count + 1))
    
    # Add to call history
    local history_key="${call_id}_history"
    local current_history="${MOCK_CALL_HISTORY[$history_key]:-}"
    MOCK_CALL_HISTORY["$history_key"]="${current_history}${timestamp};"
    
    # Log to appropriate file
    case "$call_type" in
        "http")
            echo "${timestamp}: ${call_details}" >> "${MOCK_RESPONSES_DIR}/http_calls.log"
            ;;
        "docker")
            echo "${timestamp}: ${call_details}" >> "${MOCK_RESPONSES_DIR}/docker_calls.log"
            ;;
        "command")
            echo "${timestamp}: ${call_details}" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
            ;;
        *)
            echo "${timestamp}: ${call_type}: ${call_details}" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
            ;;
    esac
}

#######################################
# Set expected call count for verification
# Arguments: $1 - call type, $2 - call pattern, $3 - expected count
#######################################
mock::verify::expect_calls() {
    local call_type="$1"
    local call_pattern="$2"
    local expected_count="$3"
    local expectation_id="${call_type}:${call_pattern}"
    
    MOCK_EXPECTED_CALLS["$expectation_id"]="$expected_count"
    MOCK_CALL_PATTERNS["$expectation_id"]="$call_pattern"
    
    echo "[MOCK_VERIFICATION] Expecting $expected_count calls matching $call_type:$call_pattern"
}

#######################################
# Set expected call sequence for verification
# Arguments: $1 - sequence name, $@ - expected calls in order
#######################################
mock::verify::expect_sequence() {
    local sequence_name="$1"
    shift
    local expected_sequence="$*"
    
    MOCK_CALL_SEQUENCES["$sequence_name"]="$expected_sequence"
    
    echo "[MOCK_VERIFICATION] Expecting sequence '$sequence_name': $expected_sequence"
}

#######################################
# Verify all expected calls were made
# Returns: 0 if all expectations met, 1 otherwise
#######################################
mock::verify::validate_all() {
    if [[ "$MOCK_VERIFICATION_ENABLED" != "true" ]]; then
        return 0
    fi
    
    local verification_failed=false
    local error_file="${MOCK_RESPONSES_DIR}/verification_errors.log"
    
    echo "[MOCK_VERIFICATION] Validating all expectations..."
    
    # Validate call count expectations
    for expectation_id in "${!MOCK_EXPECTED_CALLS[@]}"; do
        local expected_count="${MOCK_EXPECTED_CALLS[$expectation_id]}"
        local pattern="${MOCK_CALL_PATTERNS[$expectation_id]}"
        local actual_count=$(mock::verify::count_calls_matching "$expectation_id" "$pattern")
        
        if [[ "$actual_count" != "$expected_count" ]]; then
            echo "EXPECTATION FAILED: $expectation_id" >> "$error_file"
            echo "  Expected: $expected_count calls" >> "$error_file"
            echo "  Actual: $actual_count calls" >> "$error_file"
            echo "  Pattern: $pattern" >> "$error_file"
            echo "" >> "$error_file"
            verification_failed=true
        fi
    done
    
    # Validate call sequences
    for sequence_name in "${!MOCK_CALL_SEQUENCES[@]}"; do
        local expected_sequence="${MOCK_CALL_SEQUENCES[$sequence_name]}"
        if ! mock::verify::validate_sequence "$sequence_name" "$expected_sequence"; then
            echo "SEQUENCE FAILED: $sequence_name" >> "$error_file"
            echo "  Expected: $expected_sequence" >> "$error_file"
            echo "" >> "$error_file"
            verification_failed=true
        fi
    done
    
    if [[ "$verification_failed" == "true" ]]; then
        echo "[MOCK_VERIFICATION] Verification FAILED - see $error_file for details"
        cat "$error_file" >&2
        return 1
    else
        echo "[MOCK_VERIFICATION] All verifications PASSED"
        return 0
    fi
}

#######################################
# Count calls matching a pattern
# Arguments: $1 - expectation ID, $2 - pattern
# Returns: count of matching calls
#######################################
mock::verify::count_calls_matching() {
    local expectation_id="$1"
    local pattern="$2"
    local count=0
    
    # Extract call type from expectation ID
    local call_type="${expectation_id%%:*}"
    
    # Count exact matches first
    local exact_match_id="$expectation_id"
    count="${MOCK_CALL_COUNTS[$exact_match_id]:-0}"
    
    # If no exact matches, try pattern matching
    if [[ "$count" == "0" ]]; then
        for call_id in "${!MOCK_CALL_COUNTS[@]}"; do
            if [[ "$call_id" =~ $call_type:.*$pattern.* ]]; then
                count=$((count + MOCK_CALL_COUNTS[$call_id]))
            fi
        done
    fi
    
    echo "$count"
}

#######################################
# Validate call sequence
# Arguments: $1 - sequence name, $2 - expected sequence
# Returns: 0 if sequence matches, 1 otherwise
#######################################
mock::verify::validate_sequence() {
    local sequence_name="$1"
    local expected_sequence="$2"
    
    # Get actual call sequence from logs
    local actual_sequence=""
    if [[ -f "${MOCK_RESPONSES_DIR}/command_calls.log" ]]; then
        # Extract command sequence (simplified for demonstration)
        actual_sequence=$(grep -o '^[^:]*: [^[:space:]]*' "${MOCK_RESPONSES_DIR}/command_calls.log" | cut -d' ' -f2 | tr '\n' ' ')
    fi
    
    # Simple sequence validation (can be enhanced with more sophisticated matching)
    [[ "$actual_sequence" =~ $expected_sequence ]]
}

#######################################
# Get call count for specific call
# Arguments: $1 - call type, $2 - call details
# Returns: call count
#######################################
mock::verify::get_call_count() {
    local call_type="$1"
    local call_details="$2"
    local call_id="${call_type}:${call_details}"
    
    echo "${MOCK_CALL_COUNTS[$call_id]:-0}"
}

#######################################
# Check if specific call was made
# Arguments: $1 - call type, $2 - call details
# Returns: 0 if call was made, 1 otherwise
#######################################
mock::verify::was_called() {
    local call_type="$1"
    local call_details="$2"
    local count=$(mock::verify::get_call_count "$call_type" "$call_details")
    
    [[ "$count" -gt 0 ]]
}

#######################################
# Assert specific call was made
# Arguments: $1 - call type, $2 - call details, $3 - expected count (optional)
# Returns: 0 if assertion passes, 1 otherwise
#######################################
mock::verify::assert_called() {
    local call_type="$1"
    local call_details="$2"
    local expected_count="${3:-1}"
    local actual_count=$(mock::verify::get_call_count "$call_type" "$call_details")
    
    if [[ "$actual_count" != "$expected_count" ]]; then
        echo "Mock call assertion failed:" >&2
        echo "  Call: $call_type:$call_details" >&2
        echo "  Expected: $expected_count" >&2
        echo "  Actual: $actual_count" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Assert call was never made
# Arguments: $1 - call type, $2 - call details
# Returns: 0 if assertion passes, 1 otherwise
#######################################
mock::verify::assert_not_called() {
    local call_type="$1"
    local call_details="$2"
    
    mock::verify::assert_called "$call_type" "$call_details" "0"
}

#######################################
# Assert minimum number of calls
# Arguments: $1 - call type, $2 - call pattern, $3 - minimum count
# Returns: 0 if assertion passes, 1 otherwise
#######################################
mock::verify::assert_min_calls() {
    local call_type="$1"
    local call_pattern="$2"
    local min_count="$3"
    local actual_count=$(mock::verify::count_calls_matching "${call_type}:${call_pattern}" "$call_pattern")
    
    if [[ "$actual_count" -lt "$min_count" ]]; then
        echo "Mock call minimum assertion failed:" >&2
        echo "  Pattern: $call_type:$call_pattern" >&2
        echo "  Minimum: $min_count" >&2
        echo "  Actual: $actual_count" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Assert maximum number of calls
# Arguments: $1 - call type, $2 - call pattern, $3 - maximum count
# Returns: 0 if assertion passes, 1 otherwise
#######################################
mock::verify::assert_max_calls() {
    local call_type="$1"
    local call_pattern="$2"
    local max_count="$3"
    local actual_count=$(mock::verify::count_calls_matching "${call_type}:${call_pattern}" "$call_pattern")
    
    if [[ "$actual_count" -gt "$max_count" ]]; then
        echo "Mock call maximum assertion failed:" >&2
        echo "  Pattern: $call_type:$call_pattern" >&2
        echo "  Maximum: $max_count" >&2
        echo "  Actual: $actual_count" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Print verification summary
#######################################
mock::verify::print_summary() {
    echo "[MOCK_VERIFICATION] Call Summary:"
    echo "================================"
    
    for call_id in "${!MOCK_CALL_COUNTS[@]}"; do
        local count="${MOCK_CALL_COUNTS[$call_id]}"
        echo "  $call_id: $count calls"
    done
    
    echo ""
    echo "Log files:"
    echo "  Commands: ${MOCK_RESPONSES_DIR}/command_calls.log"
    echo "  HTTP: ${MOCK_RESPONSES_DIR}/http_calls.log"
    echo "  Docker: ${MOCK_RESPONSES_DIR}/docker_calls.log"
    echo "  Errors: ${MOCK_RESPONSES_DIR}/verification_errors.log"
}

#######################################
# Reset verification state
#######################################
mock::verify::reset() {
    MOCK_CALL_COUNTS=()
    MOCK_CALL_HISTORY=()
    MOCK_EXPECTED_CALLS=()
    MOCK_CALL_PATTERNS=()
    MOCK_CALL_SEQUENCES=()
    
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        rm -f "${MOCK_RESPONSES_DIR}/command_calls.log"
        rm -f "${MOCK_RESPONSES_DIR}/http_calls.log"
        rm -f "${MOCK_RESPONSES_DIR}/docker_calls.log"
        rm -f "${MOCK_RESPONSES_DIR}/verification_errors.log"
    fi
    
    echo "[MOCK_VERIFICATION] Verification state reset"
}

#######################################
# Export verification functions
#######################################
export -f mock::verify::init
export -f mock::verify::record_call
export -f mock::verify::expect_calls
export -f mock::verify::expect_sequence
export -f mock::verify::validate_all
export -f mock::verify::count_calls_matching
export -f mock::verify::validate_sequence
export -f mock::verify::get_call_count
export -f mock::verify::was_called
export -f mock::verify::assert_called
export -f mock::verify::assert_not_called
export -f mock::verify::assert_min_calls
export -f mock::verify::assert_max_calls
export -f mock::verify::print_summary
export -f mock::verify::reset

echo "[MOCK_VERIFICATION] Mock verification system loaded"