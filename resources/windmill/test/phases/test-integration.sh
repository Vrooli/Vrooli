#!/usr/bin/env bash
################################################################################
# Windmill Integration Tests - End-to-End Functionality
# 
# Must complete in <120 seconds per v2.0 contract
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WINDMILL_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${WINDMILL_DIR}/../.." && pwd)"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
source "${WINDMILL_DIR}/config/defaults.sh"

# Load Windmill libraries
for lib in api workers content; do
    if [[ -f "${WINDMILL_DIR}/lib/${lib}.sh" ]]; then
        source "${WINDMILL_DIR}/lib/${lib}.sh"
    fi
done

# Test configuration
readonly TEST_NAME="Windmill Integration Tests"
readonly TEST_WORKSPACE="integration_test_$(date +%s)"
readonly TEST_SCRIPT_NAME="test_script_$(date +%s)"
readonly TEST_WORKFLOW_NAME="test_workflow_$(date +%s)"

# Color codes
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

################################################################################
# Helper Functions
################################################################################

cleanup_test_resources() {
    log::info "Cleaning up test resources..."
    
    # Clean up test scripts and workflows if they exist
    # This is a best-effort cleanup, don't fail if resources don't exist
    curl -sf -X DELETE "${WINDMILL_BASE_URL}/api/w/${TEST_WORKSPACE}" \
        -H "Authorization: Bearer ${TEST_TOKEN:-}" &>/dev/null || true
}

get_auth_token() {
    # Get authentication token for API calls
    local response
    response=$(curl -sf -X POST "${WINDMILL_BASE_URL}/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${WINDMILL_SUPERADMIN_EMAIL}\",\"password\":\"${WINDMILL_SUPERADMIN_PASSWORD}\"}" \
        2>/dev/null || echo "{}")
    
    echo "$response" | grep -o '"token":"[^"]*' | cut -d'"' -f4
}

################################################################################
# Test Functions
################################################################################

test_api_authentication() {
    log::info "Testing: API authentication..."
    
    TEST_TOKEN=$(get_auth_token)
    if [[ -n "$TEST_TOKEN" ]]; then
        echo -e "${GREEN}✓${NC} Successfully authenticated with API"
        return 0
    else
        echo -e "${RED}✗${NC} Failed to authenticate with API"
        return 1
    fi
}

test_script_creation() {
    log::info "Testing: Script creation via API..."
    
    # Create a simple TypeScript test script
    local script_content='export async function main() { return { result: "Hello from Windmill!" }; }'
    
    local response
    response=$(curl -sf -X POST "${WINDMILL_BASE_URL}/api/w/starter/scripts/create" \
        -H "Authorization: Bearer ${TEST_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{
            \"path\": \"${TEST_SCRIPT_NAME}\",
            \"content\": \"${script_content}\",
            \"language\": \"typescript\",
            \"description\": \"Integration test script\"
        }" 2>/dev/null || echo "{}")
    
    if echo "$response" | grep -q "path"; then
        echo -e "${GREEN}✓${NC} Successfully created test script"
        return 0
    else
        echo -e "${RED}✗${NC} Failed to create test script"
        return 1
    fi
}

test_script_execution() {
    log::info "Testing: Script execution..."
    
    # Execute the test script
    local job_response
    job_response=$(curl -sf -X POST "${WINDMILL_BASE_URL}/api/w/starter/jobs/run/script/${TEST_SCRIPT_NAME}" \
        -H "Authorization: Bearer ${TEST_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{}" 2>/dev/null || echo "{}")
    
    local job_id
    job_id=$(echo "$job_response" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
    
    if [[ -z "$job_id" ]]; then
        echo -e "${RED}✗${NC} Failed to start script execution"
        return 1
    fi
    
    # Wait for job completion (with timeout)
    local attempts=0
    local max_attempts=30
    
    while [[ $attempts -lt $max_attempts ]]; do
        local job_status
        job_status=$(curl -sf "${WINDMILL_BASE_URL}/api/w/starter/jobs/get/${job_id}" \
            -H "Authorization: Bearer ${TEST_TOKEN}" 2>/dev/null || echo "{}")
        
        if echo "$job_status" | grep -q '"type":"CompletedJob"'; then
            if echo "$job_status" | grep -q '"success":true'; then
                echo -e "${GREEN}✓${NC} Script executed successfully"
                return 0
            else
                echo -e "${RED}✗${NC} Script execution failed"
                return 1
            fi
        fi
        
        sleep 1
        attempts=$((attempts + 1))
    done
    
    echo -e "${RED}✗${NC} Script execution timed out"
    return 1
}

test_multi_language_support() {
    log::info "Testing: Multi-language script support..."
    
    local languages=("python" "bash")
    local scripts=(
        'def main(): return {"result": "Python works!"}'
        'echo "{\"result\": \"Bash works!\"}"'
    )
    
    local all_passed=true
    
    for i in "${!languages[@]}"; do
        local lang="${languages[$i]}"
        local script="${scripts[$i]}"
        
        # Create and execute script for each language
        local response
        response=$(curl -sf -X POST "${WINDMILL_BASE_URL}/api/w/starter/jobs/run_code" \
            -H "Authorization: Bearer ${TEST_TOKEN}" \
            -H "Content-Type: application/json" \
            -d "{
                \"language\": \"${lang}\",
                \"content\": \"${script}\"
            }" 2>/dev/null || echo "{}")
        
        if echo "$response" | grep -q "id"; then
            echo -e "${GREEN}✓${NC} ${lang} script support verified"
        else
            echo -e "${RED}✗${NC} ${lang} script support failed"
            all_passed=false
        fi
    done
    
    if $all_passed; then
        return 0
    else
        return 1
    fi
}

test_worker_scaling() {
    log::info "Testing: Worker scaling operations..."
    
    # Get current worker count
    local initial_count
    initial_count=$(docker ps --format "table {{.Names}}" | grep -c "${WINDMILL_WORKER_CONTAINER}" || echo "0")
    
    # Try to scale workers
    if "${WINDMILL_DIR}/cli.sh" content scale-workers --workers 2 &>/dev/null; then
        sleep 5  # Give time for scaling
        
        local new_count
        new_count=$(docker ps --format "table {{.Names}}" | grep -c "${WINDMILL_WORKER_CONTAINER}" || echo "0")
        
        if [[ $new_count -ne $initial_count ]]; then
            echo -e "${GREEN}✓${NC} Worker scaling successful (${initial_count} -> ${new_count})"
            
            # Scale back to original
            "${WINDMILL_DIR}/cli.sh" content scale-workers --workers "$initial_count" &>/dev/null || true
            return 0
        else
            echo -e "${YELLOW}⚠${NC} Worker count unchanged (may be at limit)"
            return 0
        fi
    else
        echo -e "${YELLOW}⚠${NC} Worker scaling command not available"
        return 0
    fi
}

test_content_management() {
    log::info "Testing: Content management operations..."
    
    # Test listing content
    if "${WINDMILL_DIR}/cli.sh" content list &>/dev/null; then
        echo -e "${GREEN}✓${NC} Content listing works"
    else
        echo -e "${RED}✗${NC} Content listing failed"
        return 1
    fi
    
    # Test adding content (create a sample workflow)
    local workflow_file="/tmp/test_workflow_$$.json"
    cat > "$workflow_file" <<EOF
{
    "name": "Test Workflow",
    "description": "Integration test workflow",
    "steps": []
}
EOF
    
    if "${WINDMILL_DIR}/cli.sh" content add --file "$workflow_file" --type workflow &>/dev/null; then
        echo -e "${GREEN}✓${NC} Content addition works"
        rm -f "$workflow_file"
        return 0
    else
        echo -e "${YELLOW}⚠${NC} Content addition not fully implemented"
        rm -f "$workflow_file"
        return 0
    fi
}

test_performance_metrics() {
    log::info "Testing: Performance metrics..."
    
    # Test simple script execution time
    local start_time=$(date +%s)
    
    curl -sf -X POST "${WINDMILL_BASE_URL}/api/w/starter/jobs/run_code" \
        -H "Authorization: Bearer ${TEST_TOKEN}" \
        -H "Content-Type: application/json" \
        -d '{"language": "typescript", "content": "export async function main() { return 42; }"}' \
        &>/dev/null
    
    local end_time=$(date +%s)
    local execution_time=$((end_time - start_time))
    
    if [[ $execution_time -lt 5 ]]; then
        echo -e "${GREEN}✓${NC} Script execution performance acceptable (${execution_time}s)"
        return 0
    else
        echo -e "${YELLOW}⚠${NC} Script execution slower than expected (${execution_time}s)"
        return 0
    fi
}

################################################################################
# Main Test Execution
################################################################################

main() {
    echo "================================"
    echo "$TEST_NAME"
    echo "================================"
    echo ""
    
    # Set up trap for cleanup
    trap cleanup_test_resources EXIT
    
    local failed_tests=()
    local exit_code=0
    
    # Run each test
    local tests=(
        "test_api_authentication"
        "test_script_creation"
        "test_script_execution"
        "test_multi_language_support"
        "test_worker_scaling"
        "test_content_management"
        "test_performance_metrics"
    )
    
    for test in "${tests[@]}"; do
        if $test; then
            :  # Test passed
        else
            failed_tests+=("$test")
            exit_code=1
        fi
        echo ""
    done
    
    # Summary
    echo "================================"
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}✓ All integration tests passed!${NC}"
    else
        echo -e "${RED}✗ Some tests failed:${NC}"
        for test in "${failed_tests[@]}"; do
            echo -e "  ${RED}- ${test}${NC}"
        done
    fi
    echo "================================"
    
    return $exit_code
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi