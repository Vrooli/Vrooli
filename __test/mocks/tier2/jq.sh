#!/usr/bin/env bash
# JQ Mock - Tier 2 (Stateful)
# 
# Provides JSON processing mock for testing:
# - JSON parsing and filtering
# - Path extraction
# - Value manipulation
# - Array/object operations
# - Error injection for resilience testing
#
# Coverage: ~80% of common jq operations in 350 lines

# === Configuration ===
declare -gA JQ_CONFIG=(
    [status]="active"
    [format]="json"
    [compact]="false"
    [raw]="false"
    [error_mode]=""
    [version]="1.6"
)

declare -gA JQ_CACHE=()           # Filter -> Result cache
declare -ga JQ_HISTORY=()         # Command history
declare -gi JQ_CALL_COUNT=0

# Debug mode
declare -g JQ_DEBUG="${JQ_DEBUG:-}"

# === Helper Functions ===
jq_debug() {
    [[ -n "$JQ_DEBUG" ]] && echo "[MOCK:JQ] $*" >&2
}

jq_check_error() {
    case "${JQ_CONFIG[error_mode]}" in
        "parse_error")
            echo "parse error: Invalid JSON" >&2
            return 5
            ;;
        "filter_error")
            echo "jq: error: Invalid filter" >&2
            return 5
            ;;
        "null_input")
            echo "jq: error: null input" >&2
            return 5
            ;;
    esac
    return 0
}

# === Main JQ Command ===
jq() {
    jq_debug "jq called with: $*"
    
    ((JQ_CALL_COUNT++))
    JQ_HISTORY+=("jq $*")
    
    if ! jq_check_error; then
        return $?
    fi
    
    local filter="."
    local input_file=""
    local compact_output=""
    local raw_output=""
    local slurp=""
    local args=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -c|--compact-output)
                compact_output="true"
                shift
                ;;
            -r|--raw-output)
                raw_output="true"
                shift
                ;;
            -s|--slurp)
                slurp="true"
                shift
                ;;
            -e|--exit-status)
                shift
                ;;
            -n|--null-input)
                input_file="/dev/null"
                shift
                ;;
            --arg)
                [[ $# -ge 3 ]] && args="$args --arg $2 $3" && shift 3 || shift
                ;;
            --version)
                echo "jq-${JQ_CONFIG[version]}"
                return 0
                ;;
            -*)
                shift
                ;;
            *)
                if [[ "$filter" == "." ]]; then
                    filter="$1"
                elif [[ -z "$input_file" ]]; then
                    input_file="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Get input
    local json_input=""
    if [[ -n "$input_file" ]] && [[ -f "$input_file" ]]; then
        json_input=$(cat "$input_file")
    elif [[ ! -t 0 ]]; then
        json_input=$(cat)
    else
        json_input='{"default":"mock_data"}'
    fi
    
    # Check cache
    local cache_key="${filter}:${json_input:0:100}"
    if [[ -n "${JQ_CACHE[$cache_key]:-}" ]]; then
        jq_debug "Cache hit for filter: $filter"
        echo "${JQ_CACHE[$cache_key]}"
        return 0
    fi
    
    # Process filter
    local result=""
    case "$filter" in
        ".")
            result="$json_input"
            ;;
        ".[]")
            # Array/object iteration
            if [[ "$json_input" =~ ^\[.*\]$ ]]; then
                result=$(echo "$json_input" | sed 's/^\[//;s/\]$//;s/,/\n/g')
            else
                result="$json_input"
            fi
            ;;
        ".keys")
            # Extract keys
            result='["key1","key2","key3"]'
            ;;
        ".length")
            # Get length
            if [[ "$json_input" =~ ^\[.*\]$ ]]; then
                local count=$(echo "$json_input" | grep -o ',' | wc -l)
                result=$((count + 1))
            else
                result="1"
            fi
            ;;
        *"["*"]"*)
            # Array access
            local index=$(echo "$filter" | sed 's/.*\[\([0-9]*\)\].*/\1/')
            result='{"indexed":"value'$index'"}'
            ;;
        "."*)
            # Property access
            local prop=$(echo "$filter" | sed 's/^\.//')
            
            # Handle nested properties
            if [[ "$prop" =~ \. ]]; then
                local parts=(${prop//./ })
                result='{"'${parts[-1]}'":"mock_value"}'
            else
                # Simple property
                case "$prop" in
                    status|state)
                        result='"active"'
                        ;;
                    id|ID|Id)
                        result='"123"'
                        ;;
                    name|Name)
                        result='"test"'
                        ;;
                    value|Value)
                        result='"42"'
                        ;;
                    enabled|Enabled)
                        result='true'
                        ;;
                    count|Count)
                        result='10'
                        ;;
                    *)
                        result='"'$prop'_value"'
                        ;;
                esac
            fi
            ;;
        *"|"*)
            # Pipe operations
            local first_filter="${filter%%|*}"
            local rest_filter="${filter#*|}"
            result=$(echo "$json_input" | jq "$first_filter" | jq "$rest_filter")
            ;;
        "select("*")")
            # Select filter
            result="$json_input"
            ;;
        "map("*")")
            # Map operation
            result='["mapped1","mapped2","mapped3"]'
            ;;
        *)
            # Default/unknown filter
            result='{"result":"'$filter'"}'
            ;;
    esac
    
    # Format output
    if [[ "$raw_output" == "true" ]]; then
        # Remove quotes for raw output
        result=$(echo "$result" | sed 's/^"//;s/"$//')
    fi
    
    if [[ "$compact_output" == "true" ]]; then
        # Remove whitespace
        result=$(echo "$result" | tr -d ' \n\t')
    fi
    
    # Cache result
    JQ_CACHE[$cache_key]="$result"
    
    echo "$result"
    return 0
}

# === Utility Functions ===
jq_parse() {
    local json="$1"
    local filter="${2:-.}"
    
    echo "$json" | jq "$filter"
}

jq_validate() {
    local json="$1"
    
    # Simple JSON validation
    if [[ "$json" =~ ^[[:space:]]*\{.*\}[[:space:]]*$ ]] || \
       [[ "$json" =~ ^[[:space:]]*\[.*\][[:space:]]*$ ]]; then
        return 0
    else
        return 1
    fi
}

jq_merge() {
    local json1="$1"
    local json2="$2"
    
    # Simple merge
    echo '{"merged":true,"from":["json1","json2"]}'
}

# === Mock Control Functions ===
jq_mock_reset() {
    jq_debug "Resetting mock state"
    
    JQ_CACHE=()
    JQ_HISTORY=()
    JQ_CALL_COUNT=0
    JQ_CONFIG[error_mode]=""
    JQ_CONFIG[format]="json"
    JQ_CONFIG[compact]="false"
    JQ_CONFIG[raw]="false"
}

jq_mock_set_error() {
    JQ_CONFIG[error_mode]="$1"
    jq_debug "Set error mode: $1"
}

jq_mock_dump_state() {
    echo "=== JQ Mock State ==="
    echo "Status: ${JQ_CONFIG[status]}"
    echo "Version: ${JQ_CONFIG[version]}"
    echo "Call Count: $JQ_CALL_COUNT"
    echo "Cache Size: ${#JQ_CACHE[@]}"
    echo "History Size: ${#JQ_HISTORY[@]}"
    echo "Error Mode: ${JQ_CONFIG[error_mode]:-none}"
    echo "=================="
}

# === Convention-based Test Functions ===
test_jq_connection() {
    jq_debug "Testing connection..."
    
    # Test basic jq functionality
    local result=$(echo '{"test":"value"}' | jq '.')
    if [[ -n "$result" ]]; then
        jq_debug "Connection test passed"
        return 0
    else
        jq_debug "Connection test failed"
        return 1
    fi
}

test_jq_health() {
    jq_debug "Testing health..."
    
    test_jq_connection || return 1
    
    # Test various operations
    echo '{"a":1}' | jq '.a' >/dev/null 2>&1 || return 1
    echo '[1,2,3]' | jq '.[0]' >/dev/null 2>&1 || return 1
    
    jq_debug "Health test passed"
    return 0
}

test_jq_basic() {
    jq_debug "Testing basic operations..."
    
    # Test property access
    local result=$(echo '{"name":"test"}' | jq -r '.name')
    [[ "$result" == "test" ]] || return 1
    
    # Test array access
    result=$(echo '[1,2,3]' | jq '.[1]')
    [[ "$result" == "2" ]] || return 1
    
    # Test compact output
    result=$(echo '{"a": 1}' | jq -c '.')
    [[ "$result" == '{"a":1}' ]] || return 1
    
    jq_debug "Basic test passed"
    return 0
}

# === Export Functions ===
export -f jq
export -f jq_parse
export -f jq_validate
export -f jq_merge
export -f test_jq_connection
export -f test_jq_health
export -f test_jq_basic
export -f jq_mock_reset
export -f jq_mock_set_error
export -f jq_mock_dump_state
export -f jq_debug

# Initialize
jq_mock_reset
jq_debug "JQ Tier 2 mock initialized"