#!/bin/bash
# ====================================================================
# Vrooli Resource Integration Test Runner
# ====================================================================
# 
# Comprehensive integration testing for Vrooli's resource ecosystem.
# Tests both individual resources and multi-resource workflows to
# ensure reliable operation without mocking dependencies.
#
# Usage:
#   ./run.sh [options]
#
# Options:
#   --help                 Show help message
#   --verbose              Enable verbose logging
#   --debug                Enable debug mode (verbose + HTTP logging + extended output)
#   --single-only          Run only single-resource tests
#   --scenarios-only       Run only business scenario tests
#   --scenarios <filter>   Run scenarios matching criteria (e.g., category=ai,complexity=intermediate)
#   --list-scenarios       List all available scenarios and exit
#   --resource <name>      Test specific resource only
#   --timeout <seconds>    Override default timeout (300s)
#   --output-format <fmt>  Output format: text|json (default: text)
#   --fail-fast            Stop on first test failure
#   --cleanup              Clean up test artifacts after run
#   --business-report      Generate business readiness assessment
#
# Examples:
#   ./run.sh                                    # Run all tests
#   ./run.sh --verbose --single-only           # Single resource tests with verbose output
#   ./run.sh --resource ollama                 # Test only Ollama
#   ./run.sh --output-format json --timeout 600 # JSON output with 10min timeout
#
# ====================================================================

set -euo pipefail

# Script directory for relative imports
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source framework components
source "$SCRIPT_DIR/framework/discovery.sh"
source "$SCRIPT_DIR/framework/runner.sh"
source "$SCRIPT_DIR/framework/reporter.sh"

# Source enhanced debug functionality
if [[ -f "$SCRIPT_DIR/framework/helpers/debug-enhanced.sh" ]]; then
    source "$SCRIPT_DIR/framework/helpers/debug-enhanced.sh"
fi

# Default configuration
VERBOSE=false
DEBUG=false
SINGLE_ONLY=false
SCENARIOS_ONLY=false
INTERFACE_ONLY=false
SKIP_INTERFACE=false
SCENARIO_FILTER=""
LIST_SCENARIOS=false
SPECIFIC_RESOURCE=""
TEST_TIMEOUT=300
OUTPUT_FORMAT="text"
FAIL_FAST=false
CLEANUP=true
BUSINESS_REPORT=false
HTTP_LOG_ENABLED=false

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Test tracking arrays
declare -a ALL_RESOURCES=()
declare -a ENABLED_RESOURCES=()
declare -a HEALTHY_RESOURCES=()
declare -a DISABLED_RESOURCES=()
declare -a UNHEALTHY_RESOURCES=()
declare -a ALL_SCENARIOS=()
declare -a RUNNABLE_SCENARIOS=()
declare -a BLOCKED_SCENARIOS=()
declare -a FILTERED_SCENARIOS=()
declare -a FAILED_TEST_NAMES=()
declare -a SKIPPED_TEST_NAMES=()
declare -A SCENARIO_METADATA=()
declare -a MISSING_TEST_RESOURCES=()

# Colors for output
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    PURPLE='\033[0;35m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m' # No Color
else
    RED='' GREEN='' YELLOW='' BLUE='' PURPLE='' CYAN='' BOLD='' NC=''
fi

# Logging functions
log_header() {
    echo -e "${BOLD}${BLUE}[HEADER]${NC} ${BOLD}$1${NC}"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC}    $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} ✅ $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC}    ⚠️  $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC}   ❌ $1"
}

log_debug() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${PURPLE}[DEBUG]${NC}   🔍 $1"
    fi
}

# Show usage information
show_help() {
    cat << EOF
Vrooli Resource Integration Test Runner

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --help                 Show this help message
    --verbose              Enable verbose logging
    --debug                Enable debug mode (verbose + HTTP logging + extended output)
    --single-only          Run only single-resource tests
    --scenarios-only       Run only business scenario tests
    --interface-only       Run only interface compliance tests
    --skip-interface       Skip mandatory interface compliance validation
    --scenarios <filter>   Run scenarios matching criteria (e.g., category=ai,complexity=intermediate)
    --list-scenarios       List all available scenarios and exit
    --resource <name>      Test specific resource only
    --timeout <seconds>    Override default timeout (default: 300s)
    --output-format <fmt>  Output format: text|json (default: text)
    --fail-fast            Stop on first test failure
    --cleanup              Clean up test artifacts after run (default: true)
    --business-report      Generate business readiness assessment
    --validate-tests       Run test file compliance validation before tests

EXAMPLES:
    $0                                              # Run all available tests
    $0 --verbose --single-only                     # Single resource tests with verbose output
    $0 --interface-only                            # Run only interface compliance tests
    $0 --debug --resource qdrant                   # Debug mode for specific resource
    $0 --resource ollama                           # Test only Ollama resource
    $0 --scenarios-only                            # Run only business scenarios
    $0 --scenarios "category=customer-service"     # Run customer service scenarios
    $0 --scenarios "complexity=intermediate"       # Run intermediate complexity scenarios
    $0 --list-scenarios                            # List all available scenarios
    $0 --business-report                           # Generate business readiness report
    $0 --validate-tests --verbose                  # Validate test files with detailed output
    $0 --output-format json --timeout 600          # JSON output with 10min timeout

RESOURCE SELECTION:
    Tests automatically discover and adapt to available resources.
    Resources must be both enabled in configuration AND running/healthy to be tested.

OUTPUT:
    - ✅ Passed tests
    - ❌ Failed tests  
    - ⚠️  Skipped tests (missing dependencies)
    - 🔍 Debug information (--verbose only)

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                show_help
                exit 0
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --debug|-d)
                DEBUG=true
                VERBOSE=true
                HTTP_LOG_ENABLED=true
                shift
                ;;
            --single-only)
                SINGLE_ONLY=true
                shift
                ;;
            --scenarios-only)
                SCENARIOS_ONLY=true
                shift
                ;;
            --interface-only)
                INTERFACE_ONLY=true
                shift
                ;;
            --skip-interface)
                SKIP_INTERFACE=true
                shift
                ;;
            --scenarios)
                SCENARIO_FILTER="$2"
                shift 2
                ;;
            --list-scenarios)
                LIST_SCENARIOS=true
                shift
                ;;
            --resource)
                SPECIFIC_RESOURCE="$2"
                shift 2
                ;;
            --timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            --output-format)
                OUTPUT_FORMAT="$2"
                if [[ "$OUTPUT_FORMAT" != "text" && "$OUTPUT_FORMAT" != "json" ]]; then
                    log_error "Invalid output format: $OUTPUT_FORMAT. Must be 'text' or 'json'"
                    exit 1
                fi
                shift 2
                ;;
            --fail-fast)
                FAIL_FAST=true
                shift
                ;;
            --cleanup)
                CLEANUP=true
                shift
                ;;
            --no-cleanup)
                CLEANUP=false
                shift
                ;;
            --business-report)
                BUSINESS_REPORT=true
                shift
                ;;
            --validate-tests)
                VALIDATE_TESTS=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Validate prerequisites
validate_prerequisites() {
    log_debug "Validating test prerequisites..."
    
    # Check if we're in the right directory
    if [[ ! -f "$RESOURCES_DIR/index.sh" ]]; then
        log_error "Cannot find resources index.sh. Are you running from the correct directory?"
        exit 1
    fi
    
    # Check required tools
    local missing_tools=()
    for tool in curl jq docker; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi
    
    log_debug "Prerequisites validation complete"
}

# Main test execution function
main() {
    local start_time=$(date +%s)
    
    # Parse arguments first
    parse_args "$@"
    
    log_header "🧪 Vrooli Resource Integration Tests"
    echo "Test run started: $(date)"
    echo "Configuration: timeout=${TEST_TIMEOUT}s, format=${OUTPUT_FORMAT}, cleanup=${CLEANUP}"
    if [[ "$DEBUG" == "true" ]]; then
        echo "Debug mode: ENABLED (verbose + HTTP logging)"
    fi
    echo
    
    # Export debug settings for tests
    export TEST_VERBOSE="$VERBOSE"
    export HTTP_LOG_ENABLED="$HTTP_LOG_ENABLED"
    export TEST_DEBUG="$DEBUG"
    
    # Initialize enhanced debug mode
    if command -v init_enhanced_debug >/dev/null 2>&1; then
        init_enhanced_debug
    fi
    
    # Validate environment
    validate_prerequisites
    
    # Phase 1: Resource Discovery
    log_header "🔍 Phase 1: Resource Discovery"
    discover_all_resources
    filter_enabled_resources
    validate_resource_health
    
    # Phase 1b: Scenario Discovery
    log_header "📋 Phase 1b: Scenario Discovery"
    discover_scenarios
    filter_runnable_scenarios
    
    if [[ -n "$SCENARIO_FILTER" ]]; then
        filter_scenarios_by_criteria "$SCENARIO_FILTER"
    fi
    
    # Handle list scenarios option
    if [[ "$LIST_SCENARIOS" == "true" ]]; then
        log_header "📋 Available Business Scenarios"
        list_scenarios "$OUTPUT_FORMAT"
        exit 0
    fi
    
    if [[ ${#HEALTHY_RESOURCES[@]} -eq 0 ]]; then
        log_error "No healthy resources found for testing"
        exit 1
    fi
    
    # Phase 2: Test File Validation (optional)
    if [[ "${VALIDATE_TESTS:-false}" == "true" ]]; then
        log_header "🔍 Phase 2: Test File Validation"
        run_test_validation
    fi
    
    # Phase 2b: Interface Compliance Validation (mandatory unless --skip-interface)
    if [[ "${SKIP_INTERFACE:-false}" != "true" && "$INTERFACE_ONLY" != "true" ]]; then
        log_header "🔧 Phase 2b: Interface Compliance Validation"
        run_interface_compliance_validation
    fi
    
    # Phase 3: Interface Compliance Tests (if requested)
    if [[ "$INTERFACE_ONLY" == "true" ]]; then
        log_header "🔧 Phase 3: Interface Compliance Tests"
        run_interface_compliance_tests
    # Phase 3: Single Resource Tests
    elif [[ "$SCENARIOS_ONLY" != "true" ]]; then
        log_header "📦 Phase 3: Single Resource Tests"
        run_single_resource_tests
    fi
    
    # Phase 4: Business Scenario Tests  
    if [[ "$SINGLE_ONLY" != "true" && "$INTERFACE_ONLY" != "true" ]]; then
        log_header "🎯 Phase 4: Business Scenario Tests"
        run_scenario_tests
    fi
    
    # Phase 5: Generate Report
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    log_header "📊 Phase 5: Test Results"
    generate_final_report "$duration"
    
    # Phase 6: Business Readiness Assessment (if requested)
    if [[ "$BUSINESS_REPORT" == "true" && ${#ALL_SCENARIOS[@]} -gt 0 ]]; then
        log_header "💼 Phase 6: Business Readiness Assessment"
        generate_business_readiness_report "$SCRIPT_DIR/scenarios" "${HEALTHY_RESOURCES[*]}" "${PASSED_TEST_NAMES[*]:-}" "${FAILED_TEST_NAMES[*]:-}"
    fi
    
    # Cleanup if requested
    if [[ "$CLEANUP" == "true" ]]; then
        log_header "🧹 Phase 7: Cleanup"
        cleanup_test_artifacts
    fi
    
    # Finalize enhanced debug mode
    if command -v finalize_enhanced_debug >/dev/null 2>&1; then
        finalize_enhanced_debug
    fi
    
    # Exit with appropriate code
    if [[ $FAILED_TESTS -gt 0 ]]; then
        exit 1
    else
        exit 0
    fi
}

# Run test file validation
run_test_validation() {
    log_info "Validating test file compliance..."
    
    # Check if validator exists
    local validator_script="$SCRIPT_DIR/framework/helpers/test-validator.sh"
    if [[ ! -f "$validator_script" ]]; then
        log_warning "Test validator not found at $validator_script"
        log_info "Skipping validation phase"
        return 0
    fi
    
    # Run validation
    local validation_output
    local validation_exit_code
    
    if [[ "${VERBOSE:-false}" == "true" ]]; then
        validation_output=$("$validator_script" --verbose 2>&1)
    else
        validation_output=$("$validator_script" 2>&1)
    fi
    validation_exit_code=$?
    
    # Report results
    case $validation_exit_code in
        0)
            log_success "✅ All test files are compliant"
            if [[ "${VERBOSE:-false}" == "true" ]]; then
                echo "$validation_output" | sed 's/^/  /'
            fi
            ;;
        1)
            log_warning "⚠️  Test file compliance issues found"
            echo "$validation_output" | sed 's/^/  /'
            echo
            log_info "💡 Consider running: $validator_script --fix"
            echo
            ;;
        2)
            log_error "❌ Critical validation errors"
            echo "$validation_output" | sed 's/^/  /'
            echo
            log_error "Fix critical issues before running tests"
            exit 2
            ;;
        *)
            log_warning "⚠️  Test validation completed with warnings (exit code: $validation_exit_code)"
            ;;
    esac
    
    echo
}

# Run interface compliance tests only
run_interface_compliance_tests() {
    log_info "Running interface compliance tests for discovered resources..."
    
    # Source the interface compliance framework
    if [[ ! -f "$SCRIPT_DIR/framework/interface-compliance.sh" ]]; then
        log_error "Interface compliance framework not found: $SCRIPT_DIR/framework/interface-compliance.sh"
        exit 1
    fi
    
    source "$SCRIPT_DIR/framework/interface-compliance.sh"
    
    local tests_run=0
    local tests_passed=0
    local tests_failed=0
    local failed_resources=()
    
    # Test each discovered resource
    for resource in "${HEALTHY_RESOURCES[@]}"; do
        if [[ -n "$SPECIFIC_RESOURCE" && "$resource" != "$SPECIFIC_RESOURCE" ]]; then
            continue
        fi
        
        # Find the manage.sh script for this resource
        local manage_script=""
        local categories=("ai" "automation" "agents" "search" "storage" "browser" "visual")
        
        # First, try to find it using the resource directory structure
        for category in "${categories[@]}"; do
            local potential_path="$RESOURCES_DIR/$category/$resource/manage.sh"
            if [[ -f "$potential_path" ]]; then
                manage_script="$potential_path"
                break
            fi
        done
        
        # If not found in categorized structure, try direct search
        if [[ -z "$manage_script" ]]; then
            manage_script=$(find "$RESOURCES_DIR" -name "manage.sh" -path "*/$resource/*" -type f 2>/dev/null | head -1)
        fi
        
        if [[ -z "$manage_script" || ! -f "$manage_script" ]]; then
            log_warning "⚠️  No manage.sh found for $resource (skipping interface compliance test)"
            SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
            SKIPPED_TEST_NAMES+=("$resource: no manage.sh script available")
            continue
        fi
        
        log_info "Testing interface compliance for $resource..."
        log_debug "Using manage script: $manage_script"
        
        tests_run=$((tests_run + 1))
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        
        # Run interface compliance test
        if test_resource_interface_compliance "$resource" "$manage_script"; then
            tests_passed=$((tests_passed + 1))
            PASSED_TESTS=$((PASSED_TESTS + 1))
            log_success "✅ $resource passes interface compliance"
        else
            tests_failed=$((tests_failed + 1))
            FAILED_TESTS=$((FAILED_TESTS + 1))
            FAILED_TEST_NAMES+=("$resource")
            failed_resources+=("$resource")
            log_error "❌ $resource fails interface compliance"
            
            if [[ "$FAIL_FAST" == "true" ]]; then
                log_error "Stopping due to --fail-fast option"
                return 1
            fi
        fi
        
        echo "========================================"
    done
    
    # Print interface compliance summary
    log_info "Interface compliance tests complete:"
    echo "  Resources tested: $tests_run"
    echo "  Passed: $tests_passed" 
    echo "  Failed: $tests_failed"
    
    if [[ $tests_failed -gt 0 ]]; then
        log_error "Resources failing interface compliance:"
        for resource in "${failed_resources[@]}"; do
            echo "    • $resource"
        done
        return 1
    else
        if [[ $tests_run -gt 0 ]]; then
            log_success "All tested resources pass interface compliance!"
        else
            log_warning "No resources were tested for interface compliance"
        fi
        return 0
    fi
}

# Run interface compliance validation (mandatory phase)
run_interface_compliance_validation() {
    log_info "Running mandatory interface compliance validation..."
    log_info "💡 Use --skip-interface to skip this phase if needed"
    
    # Source the interface compliance framework
    if [[ ! -f "$SCRIPT_DIR/framework/interface-compliance.sh" ]]; then
        log_warning "Interface compliance framework not found - skipping validation"
        return 0
    fi
    
    source "$SCRIPT_DIR/framework/interface-compliance.sh"
    
    local resources_tested=0
    local resources_passed=0
    local resources_failed=0
    local failed_resources=()
    
    # Test interface compliance for all healthy resources
    for resource in "${HEALTHY_RESOURCES[@]}"; do
        if [[ -n "$SPECIFIC_RESOURCE" && "$resource" != "$SPECIFIC_RESOURCE" ]]; then
            continue
        fi
        
        # Find the manage.sh script for this resource
        local manage_script=""
        local categories=("ai" "automation" "agents" "search" "storage" "browser" "visual")
        
        # Try to find it using the resource directory structure
        for category in "${categories[@]}"; do
            local potential_path="$RESOURCES_DIR/$category/$resource/manage.sh"
            if [[ -f "$potential_path" ]]; then
                manage_script="$potential_path"
                break
            fi
        done
        
        # If not found in categorized structure, try direct search
        if [[ -z "$manage_script" ]]; then
            manage_script=$(find "$RESOURCES_DIR" -name "manage.sh" -path "*/$resource/*" -type f 2>/dev/null | head -1)
        fi
        
        if [[ -z "$manage_script" || ! -f "$manage_script" ]]; then
            log_warning "⚠️  No manage.sh found for $resource (skipping interface validation)"
            continue
        fi
        
        resources_tested=$((resources_tested + 1))
        log_debug "Validating interface compliance for $resource..."
        
        # Run interface compliance test (capture output to avoid cluttering)
        local compliance_output
        if compliance_output=$(test_resource_interface_compliance "$resource" "$manage_script" 2>&1); then
            resources_passed=$((resources_passed + 1))
            log_success "✅ $resource passes interface compliance"
        else
            resources_failed=$((resources_failed + 1))
            failed_resources+=("$resource")
            log_warning "⚠️  $resource fails interface compliance"
            
            # Show brief summary of failures
            if [[ "$VERBOSE" == "true" ]]; then
                echo "$compliance_output" | grep -E "\[INTERFACE\].*❌" | head -3 | sed 's/^/    /'
            fi
        fi
    done
    
    # Report validation summary
    echo
    log_info "Interface compliance validation summary:"
    echo "  Resources tested: $resources_tested"
    echo "  Passed: $resources_passed"
    echo "  Failed: $resources_failed"
    
    if [[ $resources_failed -gt 0 ]]; then
        log_warning "Resources with interface compliance issues:"
        for resource in "${failed_resources[@]}"; do
            echo "    • $resource"
        done
        echo
        log_info "💡 These resources have interface compliance issues but tests will continue."
        log_info "   Run --interface-only to see detailed compliance issues."
        log_info "   See: scripts/resources/tests/INTERFACE_COMPLIANCE_INTEGRATION.md"
    else
        if [[ $resources_tested -gt 0 ]]; then
            log_success "All tested resources pass interface compliance! 🎉"
        else
            log_warning "No resources were tested for interface compliance"
        fi
    fi
    
    echo
}

# Run main function with all arguments
main "$@"