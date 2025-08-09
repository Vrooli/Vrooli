#!/usr/bin/env bash
# Agent S2 Test Runner
# Comprehensive test suite runner for dual-mode functionality

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get proper directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"

# Source common utilities using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"

AGENT_S2_DIR="${SCRIPT_DIR}/.."
PYTHON_VENV=""
TEST_RESULTS_DIR="${var_ROOT_DIR}/data/test-outputs/agent-s2-tests"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Use log functions from common.sh instead of custom print functions
print_status() { log::info "$@"; }
print_success() { log::success "$@"; }
print_warning() { log::warning "$@"; }
print_error() { log::error "$@"; }
print_header() { log::header "$@"; }

#######################################
# Show usage information
#######################################
show_usage() {
    cat << EOF
Agent S2 Test Runner

Usage: $0 [OPTIONS] [TEST_TYPE]

TEST_TYPES:
    unit         Run unit tests only
    integration  Run integration tests only  
    modes        Run mode-specific tests only
    security     Run security tests only
    all          Run all tests (default)
    quick        Run quick tests (exclude slow tests)

OPTIONS:
    -h, --help          Show this help message
    -v, --verbose       Verbose output
    -q, --quiet         Quiet output (errors only)
    --setup-venv        Set up Python virtual environment
    --clean             Clean previous test results
    --parallel          Run tests in parallel (when supported)
    --coverage          Generate code coverage report
    --html-report       Generate HTML test report
    --junit-xml         Generate JUnit XML report
    --bail              Stop on first test failure

EXAMPLES:
    $0                  # Run all tests
    $0 unit             # Run only unit tests
    $0 integration -v   # Run integration tests with verbose output
    $0 --coverage all   # Run all tests with coverage report
    $0 quick            # Run quick tests only

EOF
}

#######################################
# Parse command line arguments
#######################################
parse_arguments() {
    VERBOSE=false
    QUIET=false
    SETUP_VENV=false
    CLEAN=false
    PARALLEL=false
    COVERAGE=false
    HTML_REPORT=false
    JUNIT_XML=false
    BAIL=false
    TEST_TYPE="all"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -q|--quiet)
                QUIET=true
                shift
                ;;
            --setup-venv)
                SETUP_VENV=true
                shift
                ;;
            --clean)
                CLEAN=true
                shift
                ;;
            --parallel)
                PARALLEL=true
                shift
                ;;
            --coverage)
                COVERAGE=true
                shift
                ;;
            --html-report)
                HTML_REPORT=true
                shift
                ;;
            --junit-xml)
                JUNIT_XML=true
                shift
                ;;
            --bail)
                BAIL=true
                shift
                ;;
            unit|integration|modes|security|all|quick)
                TEST_TYPE="$1"
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
}

#######################################
# Setup Python virtual environment
#######################################
setup_python_venv() {
    local venv_dir="${AGENT_S2_DIR}/.venv-test"
    
    print_status "Setting up Python virtual environment..."
    
    # Create virtual environment if it doesn't exist
    if [[ ! -d "$venv_dir" ]]; then
        python3 -m venv "$venv_dir"
        print_success "Created virtual environment: $venv_dir"
    fi
    
    # Activate virtual environment
    source "$venv_dir/bin/activate"
    PYTHON_VENV="$venv_dir"
    
    # Upgrade pip
    pip install --upgrade pip > /dev/null 2>&1
    
    # Install test dependencies
    print_status "Installing test dependencies..."
    pip install pytest pytest-asyncio pytest-cov pytest-html requests > /dev/null 2>&1
    
    print_success "Python virtual environment ready"
}

#######################################
# Check prerequisites
#######################################
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check Python
    if ! command -v python3 >/dev/null 2>&1; then
        print_error "Python 3 is required but not installed"
        exit 1
    fi
    
    # Check Docker (for integration tests)
    if ! command -v docker >/dev/null 2>&1; then
        print_warning "Docker not found - integration tests may fail"
    fi
    
    # Check if Agent S2 directory structure exists
    if [[ ! -d "${AGENT_S2_DIR}/agent_s2" ]]; then
        print_error "Agent S2 Python package not found"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

#######################################
# Setup test environment
#######################################
setup_test_environment() {
    print_status "Setting up test environment..."
    
    # Create results directory
    mkdir -p "$TEST_RESULTS_DIR"
    
    # Clean previous results if requested
    if [[ "$CLEAN" == "true" ]]; then
        print_status "Cleaning previous test results..."
        rm -rf "${TEST_RESULTS_DIR:?}"/*
    fi
    
    # Set environment variables for testing
    export PYTHONPATH="${AGENT_S2_DIR}:${PYTHONPATH:-}"
    export AGENT_S2_MODE="${AGENT_S2_MODE:-sandbox}"
    export AGENT_S2_HOST_MODE_ENABLED="${AGENT_S2_HOST_MODE_ENABLED:-false}"
    export AGENT_S2_HOST_AUDIT_LOGGING="${AGENT_S2_HOST_AUDIT_LOGGING:-true}"
    
    print_success "Test environment ready"
}

#######################################
# Build pytest command
#######################################
build_pytest_command() {
    local test_paths=()
    local pytest_args=()
    
    # Determine test paths based on test type
    case "$TEST_TYPE" in
        "unit")
            test_paths=("${SCRIPT_DIR}/unit")
            ;;
        "integration") 
            test_paths=("${SCRIPT_DIR}/integration")
            ;;
        "modes")
            test_paths=("${SCRIPT_DIR}/modes")
            ;;
        "security")
            test_paths=("${SCRIPT_DIR}/unit/test_security.py")
            ;;
        "quick")
            test_paths=("${SCRIPT_DIR}")
            pytest_args+=("-m" "not slow")
            ;;
        "all")
            test_paths=("${SCRIPT_DIR}")
            ;;
        *)
            print_error "Unknown test type: $TEST_TYPE"
            exit 1
            ;;
    esac
    
    # Add verbosity settings
    if [[ "$VERBOSE" == "true" ]]; then
        pytest_args+=("-v" "-s")
    elif [[ "$QUIET" == "true" ]]; then
        pytest_args+=("-q")
    else
        pytest_args+=("-v")
    fi
    
    # Add other options
    if [[ "$BAIL" == "true" ]]; then
        pytest_args+=("-x")
    fi
    
    if [[ "$PARALLEL" == "true" && "$TEST_TYPE" != "integration" ]]; then
        # Only use parallel for unit tests (integration tests may conflict)
        if command -v pytest-xdist >/dev/null 2>&1; then
            pytest_args+=("-n" "auto")
        else
            print_warning "pytest-xdist not available, running tests serially"
        fi
    fi
    
    # Add coverage reporting
    if [[ "$COVERAGE" == "true" ]]; then
        pytest_args+=(
            "--cov=${AGENT_S2_DIR}/agent_s2"
            "--cov-report=term"
            "--cov-report=html:${TEST_RESULTS_DIR}/coverage_html"
            "--cov-report=xml:${TEST_RESULTS_DIR}/coverage.xml"
        )
    fi
    
    # Add HTML report
    if [[ "$HTML_REPORT" == "true" ]]; then
        pytest_args+=(
            "--html=${TEST_RESULTS_DIR}/report_${TIMESTAMP}.html"
            "--self-contained-html"
        )
    fi
    
    # Add JUnit XML report
    if [[ "$JUNIT_XML" == "true" ]]; then
        pytest_args+=(
            "--junit-xml=${TEST_RESULTS_DIR}/junit_${TIMESTAMP}.xml"
        )
    fi
    
    # Add timing information
    pytest_args+=("--durations=10")
    
    # Build final command
    echo "pytest ${pytest_args[*]} ${test_paths[*]}"
}

#######################################
# Run tests
#######################################
run_tests() {
    local pytest_cmd
    pytest_cmd=$(build_pytest_command)
    
    print_header "Running Agent S2 Tests"
    print_status "Test Type: $TEST_TYPE"
    print_status "Command: $pytest_cmd"
    print_status "Results Directory: $TEST_RESULTS_DIR"
    
    # Change to test directory
    cd "$SCRIPT_DIR"
    
    # Run tests
    local start_time
    start_time=$(date +%s)
    
    if eval "$pytest_cmd"; then
        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        print_success "All tests passed! (Duration: ${duration}s)"
        
        # Show results summary
        show_results_summary
        
        return 0
    else
        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        print_error "Some tests failed! (Duration: ${duration}s)"
        
        # Show results summary
        show_results_summary
        
        return 1
    fi
}

#######################################
# Show results summary
#######################################
show_results_summary() {
    print_header "Test Results Summary"
    
    # Show coverage report if generated
    if [[ "$COVERAGE" == "true" && -f "${TEST_RESULTS_DIR}/coverage.xml" ]]; then
        print_status "Coverage report: ${TEST_RESULTS_DIR}/coverage_html/index.html"
    fi
    
    # Show HTML report if generated
    if [[ "$HTML_REPORT" == "true" ]]; then
        local html_report="${TEST_RESULTS_DIR}/report_${TIMESTAMP}.html"
        if [[ -f "$html_report" ]]; then
            print_status "HTML report: $html_report"
        fi
    fi
    
    # Show JUnit XML if generated
    if [[ "$JUNIT_XML" == "true" ]]; then
        local junit_report="${TEST_RESULTS_DIR}/junit_${TIMESTAMP}.xml"
        if [[ -f "$junit_report" ]]; then
            print_status "JUnit XML: $junit_report"
        fi
    fi
    
    # List all files in results directory
    if [[ -d "$TEST_RESULTS_DIR" ]] && [[ -n "$(ls -A "$TEST_RESULTS_DIR" 2>/dev/null)" ]]; then
        print_status "All results saved to: $TEST_RESULTS_DIR"
        ls -la "$TEST_RESULTS_DIR"
    fi
}

#######################################
# Agent S2 health check
#######################################
check_agent_health() {
    local base_url="http://localhost:4113"
    
    if curl -sf "${base_url}/health" >/dev/null 2>&1; then
        print_success "Agent S2 is running and healthy"
        
        # Get current mode
        local mode_info
        mode_info=$(curl -s "${base_url}/modes/current" 2>/dev/null || echo '{"current_mode":"unknown"}')
        local current_mode
        current_mode=$(echo "$mode_info" | jq -r '.current_mode // "unknown"' 2>/dev/null || echo "unknown")
        
        print_status "Current mode: $current_mode"
        return 0
    else
        print_warning "Agent S2 is not running - integration tests may fail"
        print_status "Start Agent S2 with: ${AGENT_S2_DIR}/manage.sh --action start"
        return 1
    fi
}

#######################################
# Main function
#######################################
main() {
    print_header "Agent S2 Test Suite"
    
    # Parse arguments
    parse_arguments "$@"
    
    # Check prerequisites
    check_prerequisites
    
    # Setup virtual environment if requested
    if [[ "$SETUP_VENV" == "true" ]] || [[ ! -d "${AGENT_S2_DIR}/.venv-test" ]]; then
        setup_python_venv
    elif [[ -d "${AGENT_S2_DIR}/.venv-test" ]]; then
        # Activate existing virtual environment
        source "${AGENT_S2_DIR}/.venv-test/bin/activate"
        PYTHON_VENV="${AGENT_S2_DIR}/.venv-test"
    fi
    
    # Setup test environment
    setup_test_environment
    
    # Check Agent S2 health for integration tests
    if [[ "$TEST_TYPE" == "integration" || "$TEST_TYPE" == "all" ]]; then
        check_agent_health
    fi
    
    # Run tests
    if run_tests; then
        print_success "Test suite completed successfully!"
        exit 0
    else
        print_error "Test suite failed!"
        exit 1
    fi
}

# Run main function
main "$@"