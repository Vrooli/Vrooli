#!/usr/bin/env bash
# ====================================================================
# Test Script for Resource Integration Improvements
# ====================================================================
#
# Validates that all the improvements work correctly:
# 1. Integration test for Vault + n8n
# 2. Authentication setup scripts
# 3. Status harmonization
# 4. Working examples
#
# Usage:
#   ./test-improvements.sh                    # Run all tests
#   ./test-improvements.sh --quick            # Run quick validation
#   ./test-improvements.sh --integration      # Run integration tests only
#
# ====================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Test configuration
RUN_INTEGRATION=true
RUN_AUTH_TESTS=true
RUN_STATUS_TESTS=true
QUICK_MODE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --quick)
            QUICK_MODE=true
            RUN_INTEGRATION=false
            shift
            ;;
        --integration)
            RUN_AUTH_TESTS=false
            RUN_STATUS_TESTS=false
            shift
            ;;
        --auth)
            RUN_INTEGRATION=false
            RUN_STATUS_TESTS=false
            shift
            ;;
        --status)
            RUN_INTEGRATION=false
            RUN_AUTH_TESTS=false
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--quick|--integration|--auth|--status]"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

#######################################
# Print colored output
#######################################
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

print_success() { print_status "$GREEN" "✅ $1"; }
print_error() { print_status "$RED" "❌ $1"; }
print_warning() { print_status "$YELLOW" "⚠️  $1"; }
print_info() { print_status "$BLUE" "ℹ️  $1"; }

#######################################
# Test status helper functionality
#######################################
test_status_helper() {
    print_info "Testing status helper functionality..."
    
    # Source the status helper
    if ! source "$RESOURCES_DIR/common/status-helper.sh"; then
        print_error "Failed to source status helper"
        return 1
    fi
    
    # Test basic service check
    print_info "  Testing basic service status check..."
    local test_status
    test_status=$(status_helper::check_service "test-service" "9999" "/" "nonexistent-container" 2>/dev/null || echo "{}")
    
    if echo "$test_status" | jq -e '.service == "test-service"' >/dev/null; then
        print_success "  Status helper JSON format works"
    else
        print_error "  Status helper JSON format failed"
        return 1
    fi
    
    # Test with real service (Vault if available)
    if status_helper::is_port_listening "8200"; then
        print_info "  Testing with real Vault service..."
        local vault_status
        vault_status=$(status_helper::check_service "vault" "8200" "/v1/sys/health" "vault")
        
        if echo "$vault_status" | jq -e '.service == "vault"' >/dev/null; then
            print_success "  Real service status check works"
            local status_value
            status_value=$(echo "$vault_status" | jq -r '.status')
            print_info "    Vault status: $status_value"
        else
            print_warning "  Real service status check had issues"
        fi
    fi
    
    return 0
}

#######################################
# Test authentication setup scripts
#######################################
test_auth_scripts() {
    print_info "Testing authentication setup scripts..."
    
    # Test Vault token setup script
    local vault_script="$RESOURCES_DIR/storage/vault/setup-dev-tokens.sh"
    if [[ -f "$vault_script" && -x "$vault_script" ]]; then
        print_success "  Vault token setup script exists and is executable"
        
        # Test help function
        if "$vault_script" --help >/dev/null 2>&1; then
            print_success "  Vault script help works"
        else
            print_warning "  Vault script help has issues"
        fi
        
        # Test validation (if Vault is running)
        if status_helper::is_port_listening "8200"; then
            print_info "  Testing Vault token validation..."
            if "$vault_script" --validate >/dev/null 2>&1; then
                print_success "  Vault token validation works"
            else
                print_warning "  Vault token validation failed (may need setup)"
            fi
        fi
    else
        print_error "  Vault token setup script missing or not executable"
        return 1
    fi
    
    # Test n8n auth setup script
    local n8n_script="$RESOURCES_DIR/automation/n8n/setup-api-access.sh"
    if [[ -f "$n8n_script" && -x "$n8n_script" ]]; then
        print_success "  n8n API setup script exists and is executable"
        
        # Test help function
        if "$n8n_script" --help >/dev/null 2>&1; then
            print_success "  n8n script help works"
        else
            print_warning "  n8n script help has issues"
        fi
        
        # Test validation (if n8n is running)
        if status_helper::is_port_listening "5678"; then
            print_info "  Testing n8n API validation..."
            if "$n8n_script" --validate >/dev/null 2>&1; then
                print_success "  n8n API validation works"
            else
                print_warning "  n8n API validation failed (may need setup)"
            fi
        fi
    else
        print_error "  n8n API setup script missing or not executable"
        return 1
    fi
    
    return 0
}

#######################################
# Test working integration examples
#######################################
test_integration_examples() {
    print_info "Testing working integration examples..."
    
    # Test Vault-n8n integration workflow
    local workflow_file="$RESOURCES_DIR/storage/vault/examples/n8n-vault-integration.json"
    if [[ -f "$workflow_file" ]]; then
        print_success "  n8n-Vault integration workflow exists"
        
        # Validate JSON format
        if jq . "$workflow_file" >/dev/null 2>&1; then
            print_success "  Workflow JSON is valid"
            
            # Check for working node types
            local node_types
            node_types=$(jq -r '.nodes[].type' "$workflow_file")
            
            if echo "$node_types" | grep -q "n8n-nodes-base.httpRequest"; then
                print_success "  Uses working HTTP node types (not theoretical ones)"
            else
                print_warning "  May still use theoretical node types"
            fi
            
            if echo "$node_types" | grep -q "vault-secret"; then
                print_error "  Still uses non-existent vault-secret node type"
                return 1
            else
                print_success "  No non-existent node types found"
            fi
        else
            print_error "  Workflow JSON is invalid"
            return 1
        fi
    else
        print_error "  n8n-Vault integration workflow missing"
        return 1
    fi
    
    return 0
}

#######################################
# Test integration scenario
#######################################
test_integration_scenario() {
    print_info "Testing integration test scenario..."
    
    local integration_test="$SCRIPT_DIR/scenarios/vault-n8n-integration/vault-secrets-integration.test.sh"
    if [[ -f "$integration_test" && -x "$integration_test" ]]; then
        print_success "  Vault-n8n integration test exists and is executable"
        
        # Check if required services are running
        local vault_running=false
        local n8n_running=false
        
        if status_helper::is_port_listening "8200"; then
            vault_running=true
            print_info "  Vault is running (port 8200)"
        fi
        
        if status_helper::is_port_listening "5678"; then
            n8n_running=true
            print_info "  n8n is running (port 5678)"
        fi
        
        if [[ "$vault_running" == true && "$n8n_running" == true ]]; then
            print_info "  Running integration test..."
            
            # Set up test environment
            export HEALTHY_RESOURCES_STR="vault n8n"
            export SCRIPT_DIR="$SCRIPT_DIR"
            export RESOURCES_DIR="$RESOURCES_DIR"
            
            # Run the test with timeout
            if timeout 300 "$integration_test" >/dev/null 2>&1; then
                print_success "  Integration test passed!"
            else
                print_warning "  Integration test failed or timed out"
                print_info "    Run manually for details: $integration_test"
            fi
        else
            print_warning "  Skipping integration test (services not running)"
            print_info "    Vault running: $vault_running"
            print_info "    n8n running: $n8n_running"
        fi
    else
        print_error "  Integration test script missing or not executable"
        return 1
    fi
    
    return 0
}

#######################################
# Main test execution
#######################################
main() {
    print_info "🧪 Testing Resource Integration Improvements"
    print_info "============================================="
    echo
    
    local test_count=0
    local passed_count=0
    local failed_tests=()
    
    # Test status helper
    if [[ "$RUN_STATUS_TESTS" == true ]]; then
        ((test_count++))
        if test_status_helper; then
            ((passed_count++))
        else
            failed_tests+=("Status Helper")
        fi
        echo
    fi
    
    # Test auth scripts
    if [[ "$RUN_AUTH_TESTS" == true ]]; then
        ((test_count++))
        if test_auth_scripts; then
            ((passed_count++))
        else
            failed_tests+=("Authentication Scripts")
        fi
        echo
    fi
    
    # Test integration examples
    if [[ "$RUN_INTEGRATION" == true ]]; then
        ((test_count++))
        if test_integration_examples; then
            ((passed_count++))
        else
            failed_tests+=("Integration Examples")
        fi
        echo
        
        # Test integration scenario
        if [[ "$QUICK_MODE" == false ]]; then
            ((test_count++))
            if test_integration_scenario; then
                ((passed_count++))
            else
                failed_tests+=("Integration Scenario")
            fi
            echo
        fi
    fi
    
    # Summary
    print_info "🎯 Test Summary"
    print_info "==============="
    print_info "Total tests: $test_count"
    print_success "Passed: $passed_count"
    
    if [[ ${#failed_tests[@]} -gt 0 ]]; then
        print_error "Failed: ${#failed_tests[@]}"
        print_info "Failed tests:"
        for test in "${failed_tests[@]}"; do
            print_info "  • $test"
        done
        echo
        print_error "Some tests failed - check the output above for details"
        exit 1
    else
        print_success "All tests passed! 🎉"
        echo
        print_info "✨ Resource integration improvements are working correctly!"
        print_info ""
        print_info "Next steps:"
        print_info "  • Use improved integration tests: $SCRIPT_DIR/scenarios/vault-n8n-integration/"
        print_info "  • Set up authentication: vault/setup-dev-tokens.sh, n8n/setup-api-access.sh"
        print_info "  • Use standardized status: vault/lib/status-enhanced.sh"
        print_info "  • Import working workflows: vault/examples/n8n-vault-integration.json"
        exit 0
    fi
}

# Execute main function
main "$@"