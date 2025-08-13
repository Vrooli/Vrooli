#!/usr/bin/env bash
################################################################################
# Orchestrator Test Script
# 
# Simple test to verify the Vrooli Orchestrator is working correctly.
# This script tests basic functionality like starting the daemon, registering
# processes, starting/stopping them, and monitoring.
################################################################################

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Test configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ORCHESTRATOR_CTL="$SCRIPT_DIR/orchestrator-ctl.sh"
CLIENT_SCRIPT="$SCRIPT_DIR/orchestrator-client.sh"
TEST_HOME="$HOME/.vrooli/orchestrator-test"

# Test state
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

# Test framework
test_start() {
    local test_name="$1"
    TESTS_RUN=$((TESTS_RUN + 1))
    echo ""
    echo -e "${CYAN}🧪 Test $TESTS_RUN: $test_name${NC}"
}

test_pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    log_success "✅ PASSED"
}

test_fail() {
    local reason="$1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    log_error "❌ FAILED: $reason"
}

# Cleanup function
cleanup() {
    echo ""
    log_info "🧹 Cleaning up test environment..."
    
    # Stop orchestrator if running
    "$ORCHESTRATOR_CTL" stop >/dev/null 2>&1 || true
    
    # Clean up test home
    if [[ -d "$TEST_HOME" ]]; then
        rm -rf "$TEST_HOME"
    fi
    
    # Reset environment
    unset VROOLI_ORCHESTRATOR_HOME
    
    log_success "Cleanup complete"
}

# Setup test environment
setup_test_env() {
    log_info "🛠️ Setting up test environment..."
    
    # Use a test-specific orchestrator home
    export VROOLI_ORCHESTRATOR_HOME="$TEST_HOME"
    
    # Create test directory
    mkdir -p "$TEST_HOME"
    
    log_success "Test environment ready: $TEST_HOME"
}

# Test 1: Orchestrator dependencies
test_dependencies() {
    test_start "Check orchestrator dependencies"
    
    local missing=()
    local required=("jq" "curl" "ps" "mkfifo")
    
    for cmd in "${required[@]}"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing+=("$cmd")
        fi
    done
    
    if [[ ${#missing[@]} -eq 0 ]]; then
        test_pass
    else
        test_fail "Missing dependencies: ${missing[*]}"
    fi
}

# Test 2: Orchestrator scripts exist
test_orchestrator_scripts() {
    test_start "Verify orchestrator scripts exist and are executable"
    
    local scripts=(
        "vrooli-orchestrator.sh"
        "orchestrator-client.sh"
        "orchestrator-ctl.sh"
        "multi-app-runner.sh"
    )
    
    local missing=()
    
    for script in "${scripts[@]}"; do
        local script_path="$SCRIPT_DIR/$script"
        if [[ ! -x "$script_path" ]]; then
            missing+=("$script")
        fi
    done
    
    if [[ ${#missing[@]} -eq 0 ]]; then
        test_pass
    else
        test_fail "Missing or non-executable scripts: ${missing[*]}"
    fi
}

# Test 3: Start orchestrator daemon
test_start_daemon() {
    test_start "Start orchestrator daemon"
    
    if "$ORCHESTRATOR_CTL" start >/dev/null 2>&1; then
        # Wait a moment for daemon to fully start
        sleep 3
        
        # Check if daemon is actually running
        if "$ORCHESTRATOR_CTL" status >/dev/null 2>&1; then
            test_pass
        else
            test_fail "Daemon started but status check failed"
        fi
    else
        test_fail "Failed to start daemon"
    fi
}

# Test 4: Register a test process
test_register_process() {
    test_start "Register a test process"
    
    # Change to test directory for proper context
    cd "$TEST_HOME"
    
    # Source client library
    # shellcheck disable=SC1090
    source "$CLIENT_SCRIPT" >/dev/null
    
    # Register a simple test process
    if orchestrator::register "test-echo" "echo 'Hello from test process'" >/dev/null 2>&1; then
        test_pass
    else
        test_fail "Failed to register test process"
    fi
}

# Test 5: Start and stop process
test_process_lifecycle() {
    test_start "Test process lifecycle (start/stop)"
    
    # Change to test directory
    cd "$TEST_HOME"
    
    # Source client library
    # shellcheck disable=SC1090
    source "$CLIENT_SCRIPT" >/dev/null
    
    # Start the test process
    if orchestrator::start "test-echo" >/dev/null 2>&1; then
        sleep 2
        
        # Stop the test process
        if orchestrator::stop "test-echo" >/dev/null 2>&1; then
            test_pass
        else
            test_fail "Failed to stop test process"
        fi
    else
        test_fail "Failed to start test process"
    fi
}

# Test 6: Process status and monitoring
test_process_status() {
    test_start "Test process status and monitoring"
    
    # Check overall status
    if "$ORCHESTRATOR_CTL" status >/dev/null 2>&1; then
        # Check process tree
        if "$ORCHESTRATOR_CTL" tree >/dev/null 2>&1; then
            test_pass
        else
            test_fail "Process tree command failed"
        fi
    else
        test_fail "Status command failed"
    fi
}

# Test 7: Long-running process
test_long_running_process() {
    test_start "Test long-running process management"
    
    # Change to test directory
    cd "$TEST_HOME"
    
    # Source client library
    # shellcheck disable=SC1090
    source "$CLIENT_SCRIPT" >/dev/null
    
    # Register a long-running process
    if orchestrator::register "test-sleep" "sleep 30" >/dev/null 2>&1; then
        # Start it
        if orchestrator::start "test-sleep" >/dev/null 2>&1; then
            sleep 2
            
            # Check if it's actually running
            if pgrep -f "sleep 30" >/dev/null; then
                # Stop it
                if orchestrator::stop "test-sleep" >/dev/null 2>&1; then
                    # Verify it stopped
                    sleep 2
                    if ! pgrep -f "sleep 30" >/dev/null; then
                        test_pass
                    else
                        test_fail "Process didn't stop properly"
                    fi
                else
                    test_fail "Failed to stop long-running process"
                    # Kill any remaining sleep processes
                    pkill -f "sleep 30" >/dev/null 2>&1 || true
                fi
            else
                test_fail "Long-running process didn't start"
            fi
        else
            test_fail "Failed to start long-running process"
        fi
    else
        test_fail "Failed to register long-running process"
    fi
}

# Test 8: Configuration and limits
test_configuration() {
    test_start "Test orchestrator configuration"
    
    if "$ORCHESTRATOR_CTL" config >/dev/null 2>&1; then
        test_pass
    else
        test_fail "Configuration command failed"
    fi
}

# Test 9: Stop daemon
test_stop_daemon() {
    test_start "Stop orchestrator daemon"
    
    if "$ORCHESTRATOR_CTL" stop >/dev/null 2>&1; then
        # Wait a moment for daemon to fully stop
        sleep 2
        
        # Verify daemon is stopped
        if ! "$ORCHESTRATOR_CTL" status >/dev/null 2>&1; then
            test_pass
        else
            test_fail "Daemon still running after stop command"
        fi
    else
        test_fail "Failed to stop daemon"
    fi
}

# Show test summary
show_summary() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "${CYAN}🧪 Orchestrator Test Summary${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Total Tests: $TESTS_RUN"
    echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
    echo ""
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}🎉 All tests passed! Orchestrator is working correctly.${NC}"
        echo ""
        echo "Next steps:"
        echo "  1. Run 'vrooli setup' to initialize the system"
        echo "  2. Run 'vrooli develop' to start all your apps"
        echo "  3. Use 'orchestrator-ctl' commands to manage processes"
        return 0
    else
        echo -e "${RED}❌ Some tests failed. Please check the orchestrator setup.${NC}"
        echo ""
        echo "Troubleshooting:"
        echo "  1. Ensure all dependencies are installed"
        echo "  2. Check file permissions on orchestrator scripts"
        echo "  3. Verify no other processes are using orchestrator ports"
        echo "  4. Check system logs for error details"
        return 1
    fi
}

# Main test runner
main() {
    # Handle command line options
    local verbose=false
    local cleanup_only=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                verbose=true
                shift
                ;;
            --cleanup)
                cleanup_only=true
                shift
                ;;
            --help|-h)
                cat << 'EOF'
Orchestrator Test Script

Usage: test-orchestrator.sh [options]

Options:
  --verbose, -v    Show detailed output
  --cleanup        Only perform cleanup, no tests
  --help, -h       Show this help message

This script tests the Vrooli Orchestrator functionality including:
- Dependency checks
- Script verification
- Daemon lifecycle
- Process registration and management
- Status and monitoring commands

EOF
                exit 0
                ;;
            *)
                echo "Unknown option: $1" >&2
                echo "Use --help for usage information" >&2
                exit 1
                ;;
        esac
    done
    
    # Set up cleanup trap
    trap cleanup EXIT
    
    # Handle cleanup-only mode
    if [[ "$cleanup_only" == "true" ]]; then
        cleanup
        exit 0
    fi
    
    # Show header
    echo -e "${CYAN}🎼 Vrooli Orchestrator Test Suite${NC}"
    echo "Testing orchestrator functionality..."
    
    # Setup test environment
    setup_test_env
    
    # Run tests in order
    test_dependencies
    test_orchestrator_scripts
    test_start_daemon
    test_register_process
    test_process_lifecycle
    test_process_status
    test_long_running_process
    test_configuration
    test_stop_daemon
    
    # Show summary and exit with appropriate code
    show_summary
}

# Execute main function
main "$@"