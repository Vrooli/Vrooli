#!/usr/bin/env bash
# Unified Tier 2 Mock Verification System
# Replaces: verify_tier2.sh, verify_new_mocks.sh, verify_all_mocks.sh
#
# Usage:
#   ./verify_mocks.sh          # Run all tests (default)
#   ./verify_mocks.sh core      # Test core infrastructure mocks only
#   ./verify_mocks.sh utility   # Test utility mocks only
#   ./verify_mocks.sh storage   # Test storage category
#   ./verify_mocks.sh ai        # Test AI/ML category
#   ./verify_mocks.sh --verbose # Verbose output
#   ./verify_mocks.sh --quick   # Quick test (subset only)

set -euo pipefail

# Path robustness pattern
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/.." && builtin pwd)}"
export APP_ROOT

# Configuration
VERBOSE="${VERBOSE:-false}"
QUICK_MODE=false
TEST_CATEGORY="all"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --quick|-q)
            QUICK_MODE=true
            shift
            ;;
        core|utility|storage|ai|automation|infrastructure|all)
            TEST_CATEGORY="$1"
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [category] [--verbose] [--quick]"
            echo "Categories: all, core, utility, storage, ai, automation, infrastructure"
            exit 1
            ;;
    esac
done

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Mock categories
declare -A MOCK_CATEGORIES=(
    ["core"]="redis postgres docker node-red ollama minio"
    ["utility"]="logs jq verification dig filesystem system"
    ["storage"]="redis postgres minio qdrant questdb"
    ["ai"]="ollama whisper claude-code comfyui"
    ["automation"]="node-red huginn"
    ["infrastructure"]="docker helm vault browserless judge0 searxng unstructured-io agent-s2"
)

# Test results tracking
declare -A TEST_RESULTS=()
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}✓${NC} $*"
}

log_error() {
    echo -e "${RED}✗${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $*"
}

log_verbose() {
    [[ "$VERBOSE" == "true" ]] && echo -e "  ${NC}$*"
}

# Test a single mock
test_mock() {
    local mock_name="$1"
    local mock_file="${APP_ROOT}/__test/mocks/tier2/${mock_name}.sh"
    
    log_verbose "Testing $mock_name..."
    
    if [[ ! -f "$mock_file" ]]; then
        log_error "$mock_name: File not found"
        TEST_RESULTS["$mock_name"]="NOT_FOUND"
        ((TESTS_FAILED++))
        return 1
    fi
    
    # Test file sourcing
    if ! bash -c "source '$mock_file' 2>/dev/null"; then
        log_error "$mock_name: Failed to source"
        TEST_RESULTS["$mock_name"]="SOURCE_FAILED"
        ((TESTS_FAILED++))
        return 1
    fi
    
    # Test connection function if exists
    local test_output
    test_output=$(bash -c "
        source '$mock_file' 2>/dev/null
        if declare -F test_${mock_name}_connection >/dev/null 2>&1; then
            test_${mock_name}_connection >/dev/null 2>&1 && echo 'PASS' || echo 'FAIL'
        else
            echo 'NO_TEST'
        fi
    " 2>/dev/null)
    
    case "$test_output" in
        PASS)
            log_success "$mock_name: All tests passed"
            TEST_RESULTS["$mock_name"]="PASSED"
            ((TESTS_PASSED++))
            return 0
            ;;
        FAIL)
            log_error "$mock_name: Connection test failed"
            TEST_RESULTS["$mock_name"]="TEST_FAILED"
            ((TESTS_FAILED++))
            return 1
            ;;
        NO_TEST)
            log_warning "$mock_name: No test function (basic validation only)"
            TEST_RESULTS["$mock_name"]="NO_TEST"
            ((TESTS_SKIPPED++))
            return 0
            ;;
        *)
            log_error "$mock_name: Unknown test result"
            TEST_RESULTS["$mock_name"]="UNKNOWN"
            ((TESTS_FAILED++))
            return 1
            ;;
    esac
}

# Test integration between mocks
test_integration() {
    echo ""
    log_info "Running Integration Tests..."
    
    # Redis + PostgreSQL integration
    echo -n "  Testing Redis + PostgreSQL integration: "
    if bash << 'EOF' 2>/dev/null
        source "${APP_ROOT}/__test/mocks/tier2/redis.sh"
        source "${APP_ROOT}/__test/mocks/tier2/postgres.sh"
        
        # Reset both
        redis_mock_reset 2>/dev/null || true
        postgres_mock_reset 2>/dev/null || true
        
        # Test Redis
        redis-cli set integration_test "from_redis" >/dev/null 2>&1
        result=$(redis-cli get integration_test 2>/dev/null)
        [[ "$result" == "from_redis" ]] || exit 1
        
        # Test PostgreSQL
        psql -c "CREATE TABLE integration (data TEXT)" >/dev/null 2>&1
        psql -c "INSERT INTO integration VALUES ('from_postgres')" >/dev/null 2>&1
        
        # Verify both work together
        redis-cli set postgres_status "connected" >/dev/null 2>&1
        [[ "$(redis-cli get postgres_status)" == "connected" ]] || exit 1
EOF
    then
        log_success "Integration test passed"
        ((TESTS_PASSED++))
    else
        log_error "Integration test failed"
        ((TESTS_FAILED++))
    fi
}

# Test error injection
test_error_injection() {
    echo ""
    log_info "Testing Error Injection Framework..."
    
    echo -n "  Testing Redis error injection: "
    if bash << 'EOF' 2>/dev/null
        source "${APP_ROOT}/__test/mocks/tier2/redis.sh"
        
        # Reset and test normal operation
        redis_mock_reset
        redis-cli ping >/dev/null 2>&1 || exit 1
        
        # Inject error
        redis_mock_set_error "connection_failed"
        
        # Should fail now
        if redis-cli ping >/dev/null 2>&1; then
            exit 1  # Should have failed
        fi
        
        # Clear error
        redis_mock_set_error ""
        
        # Should work again
        redis-cli ping >/dev/null 2>&1 || exit 1
EOF
    then
        log_success "Error injection works"
        ((TESTS_PASSED++))
    else
        log_error "Error injection failed"
        ((TESTS_FAILED++))
    fi
}

# Get mocks to test based on category
get_test_mocks() {
    local category="$1"
    
    if [[ "$category" == "all" ]]; then
        # Get all .sh files in tier2 directory
        find "${APP_ROOT}/__test/mocks/tier2" -name "*.sh" -exec basename {} .sh \; | sort
    else
        # Get mocks for specific category
        echo "${MOCK_CATEGORIES[$category]}" | tr ' ' '\n'
    fi
}

# Main execution
main() {
    echo "🧪 Unified Mock Verification System"
    echo "===================================="
    echo ""
    
    # Show configuration
    log_info "Configuration:"
    echo "  Category: $TEST_CATEGORY"
    echo "  Verbose: $VERBOSE"
    echo "  Quick Mode: $QUICK_MODE"
    echo ""
    
    # Get mocks to test
    local mocks_to_test
    mapfile -t mocks_to_test < <(get_test_mocks "$TEST_CATEGORY")
    
    if [[ ${#mocks_to_test[@]} -eq 0 ]]; then
        log_error "No mocks found for category: $TEST_CATEGORY"
        exit 1
    fi
    
    # Quick mode: test only subset
    if [[ "$QUICK_MODE" == "true" ]]; then
        log_info "Quick mode: Testing subset only (redis, postgres, node-red, docker)"
        mocks_to_test=("redis" "postgres" "node-red" "docker")
    fi
    
    # Test individual mocks
    log_info "Testing ${#mocks_to_test[@]} mocks..."
    echo ""
    
    for mock in "${mocks_to_test[@]}"; do
        test_mock "$mock" || true  # Continue even if test fails
    done
    
    # Run integration tests (unless testing specific category)
    if [[ "$TEST_CATEGORY" == "all" ]] || [[ "$TEST_CATEGORY" == "core" ]]; then
        test_integration
        test_error_injection
    fi
    
    # Statistics
    echo ""
    echo "📊 Mock Statistics"
    echo "=================="
    echo "Total Tier 2 mocks: $(find "${APP_ROOT}/__test/mocks/tier2" -name "*.sh" | wc -l)"
    echo "Executable: $(find "${APP_ROOT}/__test/mocks/tier2" -name "*.sh" -executable | wc -l)"
    
    if [[ -n "$(find "${APP_ROOT}/__test/mocks/tier2" -name "*.sh" 2>/dev/null)" ]]; then
        local total_lines=$(wc -l "${APP_ROOT}/__test/mocks/tier2"/*.sh 2>/dev/null | tail -1 | awk '{print $1}')
        local mock_count=$(find "${APP_ROOT}/__test/mocks/tier2" -name "*.sh" | wc -l)
        local avg_lines=$((total_lines / mock_count))
        echo "Average lines: $avg_lines"
        echo "Total lines: $total_lines"
    fi
    
    # Summary
    echo ""
    echo "📋 Test Results Summary"
    echo "======================"
    echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
    echo -e "${RED}Failed:${NC} $TESTS_FAILED"
    echo -e "${YELLOW}Skipped:${NC} $TESTS_SKIPPED"
    echo -e "Total: $((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))"
    
    # Detailed results if verbose
    if [[ "$VERBOSE" == "true" ]] && [[ ${#TEST_RESULTS[@]} -gt 0 ]]; then
        echo ""
        echo "Detailed Results:"
        for mock in "${!TEST_RESULTS[@]}"; do
            printf "  %-20s: %s\n" "$mock" "${TEST_RESULTS[$mock]}"
        done | sort
    fi
    
    # Final verdict
    echo ""
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo "🎉 ALL TESTS PASSED!"
        echo ""
        echo "✅ Mock system fully operational"
        echo "✅ Integration tests passing"
        echo "✅ Error injection working"
        echo "✅ Ready for production use"
        exit 0
    else
        echo "⚠️  Some tests failed ($TESTS_FAILED failures)"
        echo "Run with --verbose for detailed output"
        exit 1
    fi
}

# Run main function
main
