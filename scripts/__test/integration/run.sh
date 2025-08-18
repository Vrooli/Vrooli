#!/usr/bin/env bash
# Integration Test Runner - Main entry point for all integration tests
# This runner coordinates real service testing (not mocked)

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Script directory
INTEGRATION_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Default configuration
readonly DEFAULT_TIMEOUT=300  # 5 minutes for integration tests
readonly DEFAULT_PARALLEL=false
readonly DEFAULT_VERBOSE=false

# Parse command line arguments
SERVICES=()
SCENARIOS=()
RUN_ALL=true
VERBOSE=false
PARALLEL=false
TIMEOUT=$DEFAULT_TIMEOUT
HEALTH_CHECK_ONLY=false

print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Run integration tests for Vrooli resources and services.

OPTIONS:
    -s, --service NAME      Run tests for specific service
    -S, --scenario NAME     Run specific integration scenario
    -a, --all              Run all integration tests (default)
    -h, --health           Only run health checks
    -p, --parallel         Run tests in parallel
    -t, --timeout SECONDS  Set timeout for tests (default: 300)
    -v, --verbose          Enable verbose output
    --help                 Show this help message

EXAMPLES:
    $0                     # Run all integration tests
    $0 -s ollama          # Test only Ollama service
    $0 -S multi-ai        # Run multi-AI scenario
    $0 -h                 # Run health checks only
    $0 -p -t 600          # Run parallel with 10min timeout

EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--service)
            SERVICES+=("$2")
            RUN_ALL=false
            shift 2
            ;;
        -S|--scenario)
            SCENARIOS+=("$2")
            RUN_ALL=false
            shift 2
            ;;
        -a|--all)
            RUN_ALL=true
            shift
            ;;
        -h|--health)
            HEALTH_CHECK_ONLY=true
            shift
            ;;
        -p|--parallel)
            PARALLEL=true
            shift
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            print_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Health check function
run_health_checks() {
    log_info "Running health checks..."
    
    if [[ -x "$INTEGRATION_DIR/health-check.sh" ]]; then
        if timeout "$TIMEOUT" "$INTEGRATION_DIR/health-check.sh" ${VERBOSE:+-v}; then
            log_success "Health checks passed"
            return 0
        else
            log_error "Health checks failed"
            return 1
        fi
    else
        log_warning "Health check script not found or not executable"
        return 1
    fi
}

# Run service tests
run_service_tests() {
    local service="$1"
    local test_file="$INTEGRATION_DIR/services/${service}.sh"
    
    if [[ ! -f "$test_file" ]]; then
        log_warning "No test found for service: $service"
        return 1
    fi
    
    log_info "Testing service: $service"
    
    if timeout "$TIMEOUT" bash "$test_file" ${VERBOSE:+-v}; then
        log_success "Service test passed: $service"
        return 0
    else
        log_error "Service test failed: $service"
        return 1
    fi
}

# Run scenario tests
run_scenario_tests() {
    local scenario="$1"
    local test_file="$INTEGRATION_DIR/scenarios/${scenario}.sh"
    
    if [[ ! -f "$test_file" ]]; then
        log_warning "No test found for scenario: $scenario"
        return 1
    fi
    
    log_info "Running scenario: $scenario"
    
    if timeout "$TIMEOUT" bash "$test_file" ${VERBOSE:+-v}; then
        log_success "Scenario test passed: $scenario"
        return 0
    else
        log_error "Scenario test failed: $scenario"
        return 1
    fi
}

# Discover all available tests
discover_tests() {
    local test_type="$1"
    local test_dir="$INTEGRATION_DIR/$test_type"
    
    if [[ -d "$test_dir" ]]; then
        find "$test_dir" -name "*.sh" -type f -printf "%f\n" | sed 's/\.sh$//' | sort
    fi
}

# Main execution
main() {
    local exit_code=0
    local failed_tests=()
    local passed_tests=()
    
    log_info "Starting integration tests..."
    log_info "Configuration: Timeout=${TIMEOUT}s, Parallel=${PARALLEL}, Verbose=${VERBOSE}"
    
    # Run health checks first (if not skipped)
    if [[ "$HEALTH_CHECK_ONLY" == "true" ]]; then
        run_health_checks
        exit $?
    fi
    
    # Always run health checks first
    if ! run_health_checks; then
        log_error "Health checks failed - skipping integration tests"
        exit 1
    fi
    
    # Determine which tests to run
    if [[ "$RUN_ALL" == "true" ]]; then
        SERVICES=($(discover_tests "services"))
        SCENARIOS=($(discover_tests "scenarios"))
    fi
    
    # Run service tests
    if [[ ${#SERVICES[@]} -gt 0 ]]; then
        log_info "Running ${#SERVICES[@]} service test(s)..."
        
        for service in "${SERVICES[@]}"; do
            if run_service_tests "$service"; then
                passed_tests+=("service:$service")
            else
                failed_tests+=("service:$service")
                exit_code=1
            fi
        done
    fi
    
    # Run scenario tests
    if [[ ${#SCENARIOS[@]} -gt 0 ]]; then
        log_info "Running ${#SCENARIOS[@]} scenario test(s)..."
        
        for scenario in "${SCENARIOS[@]}"; do
            if run_scenario_tests "$scenario"; then
                passed_tests+=("scenario:$scenario")
            else
                failed_tests+=("scenario:$scenario")
                exit_code=1
            fi
        done
    fi
    
    # Print summary
    echo
    echo "======================================="
    echo "         Integration Test Summary       "
    echo "======================================="
    
    if [[ ${#passed_tests[@]} -gt 0 ]]; then
        log_success "Passed: ${#passed_tests[@]} test(s)"
        if [[ "$VERBOSE" == "true" ]]; then
            for test in "${passed_tests[@]}"; do
                echo "  ✓ $test"
            done
        fi
    fi
    
    if [[ ${#failed_tests[@]} -gt 0 ]]; then
        log_error "Failed: ${#failed_tests[@]} test(s)"
        for test in "${failed_tests[@]}"; do
            echo "  ✗ $test"
        done
    fi
    
    echo "======================================="
    
    exit $exit_code
}

# Run main function
main