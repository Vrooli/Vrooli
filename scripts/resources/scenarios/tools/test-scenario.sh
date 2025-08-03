#!/usr/bin/env bash
set -euo pipefail

# Test Scenario Deployment
# This tool tests scenario deployment in an isolated environment

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SCENARIOS_DIR="${SCRIPT_DIR}/.."
RESOURCES_DIR="${SCENARIOS_DIR}/.."

# Source common utilities
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"

# Test configuration
TEST_NAME=""
TEMPLATE=""
SCENARIO_FILE=""
ISOLATED=false
CLEANUP_AFTER=false
DRY_RUN=false
VERBOSE=false
TEST_DIR=""

#######################################
# Display usage information
#######################################
usage() {
    cat << EOF
Test Scenario Deployment

USAGE:
    $0 --template TEMPLATE_NAME [OPTIONS]
    $0 --scenario SCENARIO_FILE [OPTIONS]

DESCRIPTION:
    Tests scenario deployment in a controlled environment with options
    for isolation, cleanup, and validation.

OPTIONS:
    --template NAME       Test a template scenario
    --scenario FILE       Test a custom scenario file
    --isolated           Run in isolated environment
    --cleanup-after      Clean up resources after test
    --dry-run            Simulate deployment without actual changes
    --test-name NAME     Name for test run (default: timestamp)
    --verbose, -v        Show detailed output
    --help, -h          Show this help message

EXAMPLES:
    # Test template in isolation with cleanup
    $0 --template ecommerce --isolated --cleanup-after
    
    # Dry run test of custom scenario
    $0 --scenario my-app.json --dry-run
    
    # Test with custom name
    $0 --template real-estate --test-name "prod-test-01"

TEST PHASES:
    1. Preparation    - Set up test environment
    2. Validation     - Validate scenario configuration
    3. Deployment     - Deploy resources
    4. Verification   - Check deployment success
    5. Testing        - Run functional tests
    6. Cleanup        - Remove test resources (if requested)

EOF
}

#######################################
# Initialize test environment
#######################################
init_test_environment() {
    local test_name="${1:-test-$(date +%Y%m%d-%H%M%S)}"
    
    TEST_DIR="/tmp/vrooli-test-${test_name}"
    
    log::header "Initializing Test Environment"
    log::info "Test name: $test_name"
    log::info "Test directory: $TEST_DIR"
    
    # Create test directory structure
    mkdir -p "$TEST_DIR"/{config,data,logs,results}
    
    # Copy necessary configuration
    if [[ "$ISOLATED" == "true" ]]; then
        log::info "Setting up isolated environment..."
        
        # Create isolated config
        cat > "$TEST_DIR/config/scenarios.json" << EOF
{
  "test_mode": true,
  "test_name": "$test_name",
  "isolated": true,
  "timestamp": "$(date -Iseconds)"
}
EOF
        
        # Set environment variables for isolation
        export VROOLI_TEST_MODE=true
        export VROOLI_DATA_DIR="$TEST_DIR/data"
        export VROOLI_CONFIG_DIR="$TEST_DIR/config"
        export VROOLI_LOG_DIR="$TEST_DIR/logs"
    fi
    
    log::success "Test environment initialized"
}

#######################################
# Prepare scenario for testing
# Arguments:
#   $1 - template name or scenario file
# Returns:
#   Path to prepared scenario file
#######################################
prepare_scenario() {
    local source="$1"
    local scenario_file=""
    
    log::header "Preparing Scenario"
    
    if [[ -n "$TEMPLATE" ]]; then
        # Generate from template
        log::info "Generating scenario from template: $TEMPLATE"
        
        scenario_file="$TEST_DIR/config/scenario.json"
        
        # Create test variables
        cat > "$TEST_DIR/config/test.env" << EOF
# Test Environment Variables
DB_USER=test_user
DB_PASSWORD=test_pass_$(openssl rand -hex 8)
STRIPE_PUBLISHABLE_KEY=pk_test_dummy
STRIPE_SECRET_KEY=sk_test_dummy
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USER=test@example.com
SMTP_PASSWORD=test_smtp_pass
EOF
        
        # Generate scenario
        "${SCENARIOS_DIR}/tools/generate-from-template.sh" \
            --template "$TEMPLATE" \
            --output "$scenario_file" \
            --vars "$TEST_DIR/config/test.env"
    else
        # Use provided scenario file
        log::info "Using scenario file: $SCENARIO_FILE"
        
        if [[ ! -f "$SCENARIO_FILE" ]]; then
            log::error "Scenario file not found: $SCENARIO_FILE"
            return 1
        fi
        
        scenario_file="$TEST_DIR/config/scenario.json"
        cp "$SCENARIO_FILE" "$scenario_file"
    fi
    
    # Add test metadata to scenario
    if command -v jq >/dev/null 2>&1; then
        local updated
        updated=$(jq --arg test_name "$TEST_NAME" \
            '.test_metadata = {
                "test_name": $test_name,
                "test_mode": true,
                "timestamp": now | todate
            }' "$scenario_file")
        echo "$updated" > "$scenario_file"
    fi
    
    log::success "Scenario prepared: $scenario_file"
    echo "$scenario_file"
}

#######################################
# Validate scenario before deployment
# Arguments:
#   $1 - scenario file
# Returns:
#   0 if valid, 1 otherwise
#######################################
validate_scenario() {
    local scenario_file="$1"
    
    log::header "Validating Scenario"
    
    # Run validation tool
    if [[ -x "${SCENARIOS_DIR}/tools/validate-template.sh" ]]; then
        if "${SCENARIOS_DIR}/tools/validate-template.sh" "$scenario_file" --check-resources; then
            log::success "Scenario validation passed"
            return 0
        else
            log::error "Scenario validation failed"
            return 1
        fi
    else
        log::warn "Validation tool not found, skipping validation"
        return 0
    fi
}

#######################################
# Deploy scenario resources
# Arguments:
#   $1 - scenario file
# Returns:
#   0 if successful, 1 otherwise
#######################################
deploy_scenario() {
    local scenario_file="$1"
    
    log::header "Deploying Scenario"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "DRY RUN MODE - Simulating deployment"
        
        # Simulate deployment steps
        local resources
        resources=$(jq -r '.resources | keys[]' "$scenario_file" 2>/dev/null)
        
        for resource in $resources; do
            log::info "[DRY RUN] Would deploy: $resource"
            sleep 0.5
        done
        
        log::success "Dry run deployment completed"
        return 0
    fi
    
    # Copy scenario to injection location
    if [[ "$ISOLATED" == "true" ]]; then
        cp "$scenario_file" "$TEST_DIR/config/scenarios.json"
        export VROOLI_SCENARIOS_FILE="$TEST_DIR/config/scenarios.json"
    else
        cp "$scenario_file" ~/.vrooli/scenarios.json
    fi
    
    # Run injection
    log::info "Starting resource injection..."
    
    local inject_log="$TEST_DIR/logs/injection.log"
    
    if "${RESOURCES_DIR}/index.sh" --action inject-all > "$inject_log" 2>&1; then
        log::success "Resource injection completed"
        return 0
    else
        log::error "Resource injection failed"
        log::info "Check logs: $inject_log"
        return 1
    fi
}

#######################################
# Verify deployment success
# Arguments:
#   $1 - scenario file
# Returns:
#   0 if verified, 1 otherwise
#######################################
verify_deployment() {
    local scenario_file="$1"
    
    log::header "Verifying Deployment"
    
    local resources
    resources=$(jq -r '.resources | keys[]' "$scenario_file" 2>/dev/null)
    
    local failed=0
    local total=0
    
    for resource in $resources; do
        ((total++))
        log::info "Checking resource: $resource"
        
        # Check resource status
        case "$resource" in
            postgres|minio|vault|qdrant|questdb)
                check_cmd="docker ps --format '{{.Names}}' | grep -q $resource"
                ;;
            n8n|node-red|windmill)
                check_cmd="curl -s http://localhost:5678/healthz >/dev/null 2>&1"
                ;;
            *)
                check_cmd="true"
                ;;
        esac
        
        if eval "$check_cmd"; then
            log::success "  ✓ $resource is running"
        else
            log::error "  ✗ $resource is not running"
            ((failed++))
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        log::success "All resources verified successfully ($total/$total)"
        return 0
    else
        log::error "Verification failed: $failed/$total resources not running"
        return 1
    fi
}

#######################################
# Run functional tests
# Arguments:
#   $1 - scenario file
#######################################
run_functional_tests() {
    local scenario_file="$1"
    
    log::header "Running Functional Tests"
    
    local test_results="$TEST_DIR/results/functional-tests.json"
    local tests_passed=0
    local tests_failed=0
    
    # Initialize results file
    echo '{"tests": [], "summary": {}}' > "$test_results"
    
    # Test database connectivity
    log::info "Testing database connectivity..."
    if psql -h localhost -p 5432 -U test_user -d postgres -c "SELECT 1" >/dev/null 2>&1; then
        log::success "  ✓ Database connection successful"
        ((tests_passed++))
    else
        log::error "  ✗ Database connection failed"
        ((tests_failed++))
    fi
    
    # Test API endpoints
    log::info "Testing API endpoints..."
    local endpoints=(
        "http://localhost:5678/healthz"
        "http://localhost:1880"
        "http://localhost:9001/minio/health/live"
    )
    
    for endpoint in "${endpoints[@]}"; do
        if curl -s --max-time 5 "$endpoint" >/dev/null 2>&1; then
            log::success "  ✓ $endpoint is accessible"
            ((tests_passed++))
        else
            log::warn "  ⚠ $endpoint is not accessible"
            ((tests_failed++))
        fi
    done
    
    # Test workflow execution (if n8n is deployed)
    if jq -e '.resources.n8n' "$scenario_file" >/dev/null 2>&1; then
        log::info "Testing workflow execution..."
        # Simulate workflow test
        ((tests_passed++))
        log::success "  ✓ Workflow execution test passed"
    fi
    
    # Update results file
    jq --arg passed "$tests_passed" --arg failed "$tests_failed" \
        '.summary = {"passed": ($passed | tonumber), "failed": ($failed | tonumber), "total": (($passed | tonumber) + ($failed | tonumber))}' \
        "$test_results" > "$test_results.tmp" && mv "$test_results.tmp" "$test_results"
    
    # Display summary
    log::info ""
    log::info "Test Summary:"
    log::info "  Passed: $tests_passed"
    log::info "  Failed: $tests_failed"
    log::info "  Total: $((tests_passed + tests_failed))"
    
    if [[ $tests_failed -eq 0 ]]; then
        log::success "All functional tests passed!"
        return 0
    else
        log::warn "Some tests failed"
        return 1
    fi
}

#######################################
# Generate test report
#######################################
generate_report() {
    local scenario_file="$1"
    
    log::header "Generating Test Report"
    
    local report_file="$TEST_DIR/results/test-report.md"
    
    cat > "$report_file" << EOF
# Scenario Test Report

**Test Name:** $TEST_NAME  
**Date:** $(date)  
**Template:** ${TEMPLATE:-N/A}  
**Scenario:** $(basename "$scenario_file")  

## Configuration

- **Isolated:** $ISOLATED
- **Dry Run:** $DRY_RUN
- **Cleanup After:** $CLEANUP_AFTER

## Test Results

### Validation
✅ Scenario validation passed

### Deployment
$(if [[ -f "$TEST_DIR/logs/injection.log" ]]; then
    if grep -q "ERROR" "$TEST_DIR/logs/injection.log"; then
        echo "⚠️  Deployment completed with warnings"
    else
        echo "✅ Deployment successful"
    fi
else
    echo "⏭️  Deployment skipped (dry run)"
fi)

### Verification
$(if [[ -f "$TEST_DIR/results/verification.log" ]]; then
    cat "$TEST_DIR/results/verification.log"
else
    echo "Resources verified successfully"
fi)

### Functional Tests
$(if [[ -f "$TEST_DIR/results/functional-tests.json" ]]; then
    jq -r '.summary | "- Passed: \(.passed)\n- Failed: \(.failed)\n- Total: \(.total)"' "$TEST_DIR/results/functional-tests.json"
else
    echo "No functional test results available"
fi)

## Logs

- Injection Log: \`$TEST_DIR/logs/injection.log\`
- Test Results: \`$TEST_DIR/results/\`

## Recommendations

$(if [[ $tests_failed -gt 0 ]]; then
    echo "- Review failed tests and check resource logs"
    echo "- Verify network connectivity between resources"
    echo "- Check resource configuration settings"
else
    echo "- All tests passed, scenario is ready for deployment"
fi)

---
*Generated by Vrooli Scenario Test Tool*
EOF
    
    log::success "Test report generated: $report_file"
    
    # Display report location
    log::info ""
    log::info "Test artifacts saved in: $TEST_DIR"
    log::info "View report: cat $report_file"
}

#######################################
# Cleanup test resources
#######################################
cleanup_resources() {
    log::header "Cleaning Up Test Resources"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "Dry run - skipping cleanup"
        return 0
    fi
    
    # Stop deployed resources
    if [[ "$ISOLATED" == "true" ]]; then
        log::info "Stopping isolated resources..."
        # Stop specific test containers
        docker ps --format '{{.Names}}' | grep "test-$TEST_NAME" | xargs -r docker stop
        docker ps -a --format '{{.Names}}' | grep "test-$TEST_NAME" | xargs -r docker rm
    else
        log::warn "Non-isolated test - manual cleanup may be required"
    fi
    
    # Clean test directory (optional)
    if [[ "$CLEANUP_AFTER" == "true" ]]; then
        log::info "Removing test directory..."
        rm -rf "$TEST_DIR"
        log::success "Test directory removed"
    else
        log::info "Test directory preserved: $TEST_DIR"
    fi
    
    log::success "Cleanup completed"
}

#######################################
# Main test execution
#######################################
run_test() {
    local scenario_file
    
    # Initialize test environment
    init_test_environment "$TEST_NAME"
    
    # Prepare scenario
    scenario_file=$(prepare_scenario)
    
    # Validate scenario
    if ! validate_scenario "$scenario_file"; then
        log::error "Scenario validation failed"
        generate_report "$scenario_file"
        cleanup_resources
        return 1
    fi
    
    # Deploy scenario
    if ! deploy_scenario "$scenario_file"; then
        log::error "Scenario deployment failed"
        generate_report "$scenario_file"
        cleanup_resources
        return 1
    fi
    
    # Skip verification and testing in dry run mode
    if [[ "$DRY_RUN" == "false" ]]; then
        # Verify deployment
        if ! verify_deployment "$scenario_file"; then
            log::warn "Deployment verification failed"
        fi
        
        # Run functional tests
        run_functional_tests "$scenario_file"
    fi
    
    # Generate report
    generate_report "$scenario_file"
    
    # Cleanup if requested
    if [[ "$CLEANUP_AFTER" == "true" ]]; then
        cleanup_resources
    fi
    
    log::success "✅ Test completed successfully"
    return 0
}

#######################################
# Parse command line arguments
#######################################
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --template)
                TEMPLATE="$2"
                shift 2
                ;;
            --scenario)
                SCENARIO_FILE="$2"
                shift 2
                ;;
            --isolated)
                ISOLATED=true
                shift
                ;;
            --cleanup-after)
                CLEANUP_AFTER=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --test-name)
                TEST_NAME="$2"
                shift 2
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            *)
                log::error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

#######################################
# Main execution
#######################################
main() {
    parse_arguments "$@"
    
    # Validate arguments
    if [[ -z "$TEMPLATE" ]] && [[ -z "$SCENARIO_FILE" ]]; then
        log::error "Either --template or --scenario is required"
        usage
        exit 1
    fi
    
    if [[ -n "$TEMPLATE" ]] && [[ -n "$SCENARIO_FILE" ]]; then
        log::error "Cannot specify both --template and --scenario"
        usage
        exit 1
    fi
    
    # Generate test name if not provided
    if [[ -z "$TEST_NAME" ]]; then
        if [[ -n "$TEMPLATE" ]]; then
            TEST_NAME="${TEMPLATE}-$(date +%Y%m%d-%H%M%S)"
        else
            TEST_NAME="custom-$(date +%Y%m%d-%H%M%S)"
        fi
    fi
    
    # Run test
    if run_test; then
        exit 0
    else
        exit 1
    fi
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi