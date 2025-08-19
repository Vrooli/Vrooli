#!/usr/bin/env bash
# K6 Test Execution Functions

# Run a K6 test
k6::test::run() {
    local script="$1"
    shift  # Remove script from arguments
    
    if [[ -z "$script" ]]; then
        log::error "Test script required"
        echo "Usage: resource-k6 run-test <script.js> [k6-options]"
        return 1
    fi
    
    # Check if script exists
    local script_path=""
    if [[ -f "$script" ]]; then
        script_path="$script"
    elif [[ -f "$K6_SCRIPTS_DIR/$script" ]]; then
        script_path="$K6_SCRIPTS_DIR/$script"
    else
        log::error "Test script not found: $script"
        return 1
    fi
    
    log::info "Running K6 test: $script_path"
    
    # Generate results filename
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local script_name=$(basename "$script" .js)
    local results_filename="${script_name}_${timestamp}.json"
    local results_file="$K6_RESULTS_DIR/${results_filename}"
    
    # Convert local path to container path
    local container_script="/scripts/$(basename "$script_path")"
    
    # Build k6 command with proper escaping
    local k6_cmd="k6 run ${container_script} --out json=/results/${results_filename}"
    
    # Add any additional k6 arguments
    if [[ $# -gt 0 ]]; then
        k6_cmd="$k6_cmd $*"
    fi
    
    # Run test with JSON output (using docker exec with sh -c)
    if docker exec vrooli-k6 sh -c "$k6_cmd"; then
        local success=true
    else
        local success=false
    fi
    
    if [[ "$success" == "true" ]]; then
        log::success "Test completed successfully"
        log::info "Results saved to: $results_file"
        
        # Show summary if jq is available
        if command -v jq >/dev/null 2>&1; then
            echo ""
            echo "Test Summary:"
            echo "============="
            jq -r 'select(.type=="Point" and .metric=="http_req_duration") | 
                   "Response Time (p95): \(.data.tags.p95 // .data.value)ms"' "$results_file" 2>/dev/null | head -1
            jq -r 'select(.type=="Point" and .metric=="http_reqs") | 
                   "Total Requests: \(.data.value)"' "$results_file" 2>/dev/null | tail -1
            jq -r 'select(.type=="Point" and .metric=="http_req_failed") | 
                   "Failed Requests: \(.data.value)"' "$results_file" 2>/dev/null | tail -1
        fi
    else
        log::error "Test failed"
        return 1
    fi
}

# List available test scripts
k6::test::list() {
    log::info "Available K6 test scripts:"
    echo ""
    
    if [[ ! -d "$K6_SCRIPTS_DIR" ]] || [[ -z "$(ls -A "$K6_SCRIPTS_DIR" 2>/dev/null)" ]]; then
        log::warn "No test scripts found in $K6_SCRIPTS_DIR"
        return 0
    fi
    
    # List scripts with descriptions
    for script in "$K6_SCRIPTS_DIR"/*.js; do
        if [[ -f "$script" ]]; then
            local name=$(basename "$script")
            local desc=""
            
            # Try to extract description from first comment
            desc=$(head -n 10 "$script" | grep -m1 "^//" | sed 's|^//\s*||' || echo "No description")
            
            printf "  %-30s %s\n" "$name" "$desc"
        fi
    done
    
    echo ""
    echo "Run a test with: resource-k6 run-test <script-name>"
}

# Show test results
k6::test::results() {
    local limit="${1:-10}"
    
    log::info "Recent K6 test results (last $limit):"
    echo ""
    
    if [[ ! -d "$K6_RESULTS_DIR" ]] || [[ -z "$(ls -A "$K6_RESULTS_DIR" 2>/dev/null)" ]]; then
        log::warn "No test results found in $K6_RESULTS_DIR"
        return 0
    fi
    
    # List recent results
    ls -lt "$K6_RESULTS_DIR"/*.json 2>/dev/null | head -n "$limit" | while read -r line; do
        local file=$(echo "$line" | awk '{print $NF}')
        local size=$(echo "$line" | awk '{print $5}')
        local date=$(echo "$line" | awk '{print $6, $7, $8}')
        local name=$(basename "$file" .json)
        
        printf "  %-40s %10s  %s\n" "$name" "$size" "$date"
    done
    
    echo ""
    echo "View results with: cat $K6_RESULTS_DIR/<result-file>.json | jq"
}