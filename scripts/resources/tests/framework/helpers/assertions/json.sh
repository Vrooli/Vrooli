#!/bin/bash
# ====================================================================
# JSON Assertion Functions
# ====================================================================
#
# JSON validation and field assertion functions for testing APIs and
# structured data.
#
# Functions:
#   - assert_json_valid()      - Assert string is valid JSON
#   - assert_json_field()      - Assert JSON contains specific field
#   - assert_json_true()       - Assert JSON boolean is true
#   - debug_json_response()    - Pretty print JSON for debugging
#
# ====================================================================

# Assert string is valid JSON
assert_json_valid() {
    local json_string="$1"
    local message="${2:-JSON validity assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    # First check if the string is empty
    if [[ -z "$json_string" ]]; then
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  JSON string is empty"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
    
    # Capture jq error output for detailed error reporting
    local jq_error
    jq_error=$(echo "$json_string" | jq . 2>&1 >/dev/null)
    local jq_exit_code=$?
    
    if [[ $jq_exit_code -eq 0 ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  JSON validation error: $jq_error"
        echo "  JSON string: '${json_string:0:500}$([ ${#json_string} -gt 500 ] && echo "..." || echo "")'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert JSON contains specific field
assert_json_field() {
    local json_string="$1"
    local field_path="$2"
    local message="${3:-JSON field assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    # First validate the JSON
    if ! echo "$json_string" | jq . >/dev/null 2>&1; then
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Invalid JSON provided for field check"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
    
    local field_value
    local jq_error
    
    # Capture both value and any error
    {
        field_value=$(echo "$json_string" | jq -r "$field_path" 2>&1)
        jq_error=""
    } || {
        jq_error="jq command failed"
    }
    
    # Check if field exists and is not null
    if echo "$json_string" | jq -e "$field_path" >/dev/null 2>&1; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message (value: '${field_value:0:100}$([ ${#field_value} -gt 100 ] && echo "..." || echo "")')"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Field path '$field_path' not found or is null"
        echo "  Available fields: $(echo "$json_string" | jq -r 'keys[]?' 2>/dev/null | head -5 | tr '\n' ' ')..."
        echo "  JSON preview: '${json_string:0:200}$([ ${#json_string} -gt 200 ] && echo "..." || echo "")'"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Assert JSON boolean is true
assert_json_true() {
    local json="$1"
    local path="$2"
    local message="${3:-JSON boolean assertion}"
    
    TEST_ASSERTIONS=$((TEST_ASSERTIONS + 1))
    
    local value
    value=$(echo "$json" | jq -r "$path" 2>/dev/null)
    
    if [[ "$value" == "true" ]]; then
        echo -e "${ASSERT_GREEN}✓${ASSERT_NC} $message"
        PASSED_ASSERTIONS=$((PASSED_ASSERTIONS + 1))
        return 0
    else
        echo -e "${ASSERT_RED}✗${ASSERT_NC} $message"
        echo "  Path: $path"
        echo "  Expected: true, got: $value"
        FAILED_ASSERTIONS=$((FAILED_ASSERTIONS + 1))
        return 1
    fi
}

# Pretty print JSON response for debugging
debug_json_response() {
    local json_string="$1"
    local label="${2:-JSON Response}"
    
    if [[ "${TEST_VERBOSE:-false}" == "true" ]]; then
        echo "🔍 $label:"
        
        # Handle extremely large JSON responses to prevent SIGPIPE
        local json_size=${#json_string}
        
        if [[ $json_size -gt 100000 ]]; then
            # For very large JSON (>100KB), just show basic info
            echo "  Large JSON response (${json_size} characters)"
            if echo "$json_string" | head -c 1000 | jq . >/dev/null 2>&1; then
                echo "  First 1KB (formatted):"
                echo "$json_string" | head -c 1000 | jq . 2>/dev/null | head -10
                echo "  ... (truncated due to size: ${json_size} characters)"
            else
                echo "  Raw preview (first 200 chars): ${json_string:0:200}..."
            fi
        elif echo "$json_string" | jq . >/dev/null 2>&1; then
            # For normal-sized JSON, show formatted with line limit
            local formatted_output
            formatted_output=$(echo "$json_string" | jq . 2>/dev/null)
            local line_count=$(echo "$formatted_output" | wc -l)
            
            if [[ $line_count -gt 20 ]]; then
                echo "$formatted_output" | head -20
                echo "  ... (truncated, total lines: $line_count)"
            else
                echo "$formatted_output"
            fi
        else
            echo "  Invalid JSON: ${json_string:0:200}..."
        fi
        echo
    fi
}