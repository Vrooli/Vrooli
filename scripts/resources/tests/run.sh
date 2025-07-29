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
#   --single-only          Run only single-resource tests
#   --multi-only           Run only multi-resource tests
#   --resource <name>      Test specific resource only
#   --timeout <seconds>    Override default timeout (300s)
#   --output-format <fmt>  Output format: text|json (default: text)
#   --fail-fast            Stop on first test failure
#   --cleanup              Clean up test artifacts after run
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

# Default configuration
VERBOSE=false
SINGLE_ONLY=false
MULTI_ONLY=false
SPECIFIC_RESOURCE=""
TEST_TIMEOUT=300
OUTPUT_FORMAT="text"
FAIL_FAST=false
CLEANUP=true

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
declare -a FAILED_TEST_NAMES=()
declare -a SKIPPED_TEST_NAMES=()

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
    --single-only          Run only single-resource tests
    --multi-only           Run only multi-resource tests
    --resource <name>      Test specific resource only
    --timeout <seconds>    Override default timeout (default: 300s)
    --output-format <fmt>  Output format: text|json (default: text)
    --fail-fast            Stop on first test failure
    --cleanup              Clean up test artifacts after run (default: true)

EXAMPLES:
    $0                                      # Run all available tests
    $0 --verbose --single-only             # Single resource tests with verbose output
    $0 --resource ollama                   # Test only Ollama resource
    $0 --output-format json --timeout 600  # JSON output with 10min timeout
    $0 --multi-only --fail-fast            # Multi-resource tests, stop on first failure

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
            --single-only)
                SINGLE_ONLY=true
                shift
                ;;
            --multi-only)
                MULTI_ONLY=true
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
    echo
    
    # Validate environment
    validate_prerequisites
    
    # Phase 1: Resource Discovery
    log_header "🔍 Phase 1: Resource Discovery"
    discover_all_resources
    filter_enabled_resources
    validate_resource_health
    
    if [[ ${#HEALTHY_RESOURCES[@]} -eq 0 ]]; then
        log_error "No healthy resources found for testing"
        exit 1
    fi
    
    # Phase 2: Single Resource Tests
    if [[ "$MULTI_ONLY" != "true" ]]; then
        log_header "📦 Phase 2: Single Resource Tests"
        run_single_resource_tests
    fi
    
    # Phase 3: Multi-Resource Tests  
    if [[ "$SINGLE_ONLY" != "true" ]]; then
        log_header "🔗 Phase 3: Multi-Resource Integration Tests"
        run_multi_resource_tests
    fi
    
    # Phase 4: Generate Report
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    log_header "📊 Phase 4: Test Results"
    generate_final_report "$duration"
    
    # Cleanup if requested
    if [[ "$CLEANUP" == "true" ]]; then
        log_header "🧹 Phase 5: Cleanup"
        cleanup_test_artifacts
    fi
    
    # Exit with appropriate code
    if [[ $FAILED_TESTS -gt 0 ]]; then
        exit 1
    else
        exit 0
    fi
}

# Run main function with all arguments
main "$@"