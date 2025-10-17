#!/bin/bash

################################################################################
# Batch Submission Support for Judge0
# 
# Provides functionality for processing multiple code submissions efficiently
################################################################################

judge0::batch::submit() {
    local submissions_json="$1"
    local wait="${2:-false}"
    local base_url="${3:-http://localhost:2358}"
    
    if [[ -z "$submissions_json" ]]; then
        echo "[ERROR] No submissions provided" >&2
        return 1
    fi
    
    # Parse submissions array
    local num_submissions
    num_submissions=$(echo "$submissions_json" | jq 'length' 2>/dev/null)
    
    if [[ -z "$num_submissions" ]] || [[ "$num_submissions" -eq 0 ]]; then
        echo "[ERROR] Invalid or empty submissions array" >&2
        return 1
    fi
    
    echo "[INFO] Processing batch of $num_submissions submissions"
    
    # Array to store tokens
    local tokens=()
    
    # Submit each code snippet
    local i
    for ((i=0; i<num_submissions; i++)); do
        local submission
        submission=$(echo "$submissions_json" | jq ".[$i]")
        
        local response
        response=$(curl -s -X POST "$base_url/submissions" \
            -H "Content-Type: application/json" \
            -H "X-Judge0-Token: ${JUDGE0_API_KEY:-YourAuthTokenHere}" \
            -d "$submission")
        
        local token
        token=$(echo "$response" | jq -r '.token')
        
        if [[ -n "$token" ]] && [[ "$token" != "null" ]]; then
            tokens+=("$token")
            echo "[SUCCESS] Submission $((i+1))/$num_submissions - Token: $token"
        else
            echo "[ERROR] Failed to submit code $((i+1)): $response"
        fi
    done
    
    # Wait for results if requested
    if [[ "$wait" == "true" ]]; then
        echo "[INFO] Waiting for batch results..."
        sleep 2
        judge0::batch::get_results "${tokens[@]}"
    else
        # Return tokens as JSON array
        printf '%s\n' "${tokens[@]}" | jq -R . | jq -s .
    fi
}

judge0::batch::get_results() {
    local tokens=("$@")
    local base_url="${JUDGE0_BASE_URL:-http://localhost:2358}"
    local results=()
    
    for token in "${tokens[@]}"; do
        local result
        result=$(curl -s -X GET "$base_url/submissions/$token" \
            -H "X-Judge0-Token: ${JUDGE0_API_KEY:-YourAuthTokenHere}")
        
        results+=("$result")
    done
    
    # Return results as JSON array
    printf '%s\n' "${results[@]}" | jq -s .
}

judge0::batch::submit_file() {
    local file_path="$1"
    local language_id="${2:-71}"  # Default to Python
    local wait="${3:-false}"
    
    if [[ ! -f "$file_path" ]]; then
        echo "[ERROR] File not found: $file_path" >&2
        return 1
    fi
    
    # Read file and create submission JSON
    local source_code
    source_code=$(base64 -w 0 < "$file_path")
    
    local submission_json
    submission_json=$(jq -n \
        --arg src "$source_code" \
        --arg lang "$language_id" \
        '[{source_code: $src, language_id: ($lang | tonumber)}]')
    
    judge0::batch::submit "$submission_json" "$wait"
}

judge0::batch::benchmark() {
    local num_submissions="${1:-10}"
    local language_id="${2:-71}"
    
    echo "[INFO] Running batch benchmark with $num_submissions submissions"
    
    # Create test submissions
    local submissions=()
    local i
    for ((i=1; i<=num_submissions; i++)); do
        local code="print('Test $i: ' + str($i * $i))"
        local encoded
        encoded=$(echo "$code" | base64 -w 0)
        
        submissions+=("{\"source_code\": \"$encoded\", \"language_id\": $language_id}")
    done
    
    # Create JSON array
    local json_array
    json_array=$(printf '%s\n' "${submissions[@]}" | jq -s .)
    
    # Time the batch submission
    local start_time
    start_time=$(date +%s%3N)
    
    local result
    result=$(judge0::batch::submit "$json_array" "false")
    
    local end_time
    end_time=$(date +%s%3N)
    
    local duration=$((end_time - start_time))
    
    echo "[INFO] Batch submission completed in ${duration}ms"
    echo "[INFO] Average time per submission: $((duration / num_submissions))ms"
    
    # Get tokens count
    local token_count
    token_count=$(echo "$result" | jq 'length' 2>/dev/null || echo "0")
    echo "[INFO] Successfully submitted: $token_count/$num_submissions"
}

# Export functions
export -f judge0::batch::submit
export -f judge0::batch::get_results
export -f judge0::batch::submit_file
export -f judge0::batch::benchmark