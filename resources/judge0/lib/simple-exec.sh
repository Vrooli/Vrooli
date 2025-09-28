#!/bin/bash

# Simplified executor for testing - runs code directly without Docker
# This is a proof-of-concept to show code execution can work

set -euo pipefail

execute_python() {
    local code="$1"
    local stdin="${2:-}"
    
    # Create temp file
    local temp_file="/tmp/judge0-test-$$.py"
    echo "$code" > "$temp_file"
    
    # Execute with timeout
    local output=""
    local error=""
    local exit_code=0
    
    if [[ -n "$stdin" ]]; then
        output=$(echo "$stdin" | timeout 5 python3 "$temp_file" 2>&1) || exit_code=$?
    else
        output=$(timeout 5 python3 "$temp_file" 2>&1) || exit_code=$?
    fi
    
    # Clean up
    rm -f "$temp_file"
    
    # Return JSON result
    cat <<EOF
{
    "status": "$( [[ $exit_code -eq 0 ]] && echo "accepted" || echo "error" )",
    "stdout": $(echo -n "$output" | jq -Rs .),
    "exit_code": $exit_code
}
EOF
}

# Test
if [[ "${1:-}" == "test" ]]; then
    echo "Testing simple Python execution..."
    execute_python 'print("Hello from simple executor!")'
else
    execute_python "$@"
fi