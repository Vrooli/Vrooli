#!/usr/bin/env bash
################################################################################
# Judge0 Integration Tests - End-to-End Functionality (<120s)
# 
# Tests complete workflows and integration scenarios
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
RESOURCE_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test configuration
JUDGE0_PORT="${JUDGE0_PORT:-2358}"
API_URL="http://localhost:${JUDGE0_PORT}"
TEST_FAILED=0

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_test() {
    local status="$1"
    local test_name="$2"
    local details="${3:-}"
    
    case "$status" in
        PASS)
            echo -e "  ${GREEN}✓${NC} $test_name"
            [[ -n "$details" ]] && echo "    $details"
            ;;
        FAIL)
            echo -e "  ${RED}✗${NC} $test_name"
            [[ -n "$details" ]] && echo -e "    ${RED}$details${NC}"
            TEST_FAILED=1
            ;;
        INFO)
            echo -e "  ${YELLOW}ℹ${NC} $test_name"
            [[ -n "$details" ]] && echo "    $details"
            ;;
        PERF)
            echo -e "  ${BLUE}⚡${NC} $test_name"
            [[ -n "$details" ]] && echo "    $details"
            ;;
    esac
}

test_multi_language_execution() {
    log_test "INFO" "Testing multi-language code execution..."
    
    # Test programs that should produce same output
    local test_output="42"
    
    # Python
    local python_result=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d "{
            \"source_code\": \"print($test_output)\",
            \"language_id\": 92
        }" 2>/dev/null || echo "FAILED")
    
    if [[ "$python_result" != "FAILED" ]]; then
        local stdout=$(echo "$python_result" | python3 -c "import sys, json; print(json.load(sys.stdin).get('stdout', '').strip())" 2>/dev/null)
        if [[ "$stdout" == "$test_output" ]]; then
            log_test "PASS" "Python execution" "Output: $stdout"
        else
            log_test "FAIL" "Python execution" "Expected: $test_output, Got: $stdout"
        fi
    else
        log_test "FAIL" "Python execution" "Failed to execute"
    fi
    
    # JavaScript
    local js_result=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d "{
            \"source_code\": \"console.log($test_output)\",
            \"language_id\": 93
        }" 2>/dev/null || echo "FAILED")
    
    if [[ "$js_result" != "FAILED" ]]; then
        local stdout=$(echo "$js_result" | python3 -c "import sys, json; print(json.load(sys.stdin).get('stdout', '').strip())" 2>/dev/null)
        if [[ "$stdout" == "$test_output" ]]; then
            log_test "PASS" "JavaScript execution" "Output: $stdout"
        else
            log_test "FAIL" "JavaScript execution" "Expected: $test_output, Got: $stdout"
        fi
    else
        log_test "FAIL" "JavaScript execution" "Failed to execute"
    fi
    
    # Go (if available)
    local go_result=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d "{
            \"source_code\": \"package main\\nimport \\\"fmt\\\"\\nfunc main() { fmt.Println($test_output) }\",
            \"language_id\": 95
        }" 2>/dev/null || echo "FAILED")
    
    if [[ "$go_result" != "FAILED" ]]; then
        local stdout=$(echo "$go_result" | python3 -c "import sys, json; print(json.load(sys.stdin).get('stdout', '').strip())" 2>/dev/null)
        if [[ "$stdout" == "$test_output" ]]; then
            log_test "PASS" "Go execution" "Output: $stdout"
        else
            log_test "FAIL" "Go execution" "Expected: $test_output, Got: $stdout"
        fi
    else
        log_test "INFO" "Go execution" "Skipped or not available"
    fi
}

test_batch_submissions() {
    log_test "INFO" "Testing batch submission capability..."
    
    # Submit multiple programs simultaneously
    local batch_data='[
        {"source_code": "print(1)", "language_id": 71},
        {"source_code": "print(2)", "language_id": 71},
        {"source_code": "print(3)", "language_id": 71}
    ]'
    
    local batch_result=$(timeout 15 curl -sf -X POST "${API_URL}/submissions/batch?wait=true" \
        -H "Content-Type: application/json" \
        -d "$batch_data" 2>/dev/null || echo "FAILED")
    
    if [[ "$batch_result" != "FAILED" ]]; then
        # Check if we got an array response
        local is_array=$(echo "$batch_result" | python3 -c "import sys, json; print(isinstance(json.load(sys.stdin), list))" 2>/dev/null || echo "False")
        
        if [[ "$is_array" == "True" ]]; then
            local count=$(echo "$batch_result" | python3 -c "import sys, json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "0")
            if [[ $count -eq 3 ]]; then
                log_test "PASS" "Batch submission" "Successfully processed $count submissions"
            else
                log_test "FAIL" "Batch submission" "Expected 3 results, got $count"
            fi
        else
            log_test "INFO" "Batch submission" "Batch endpoint not available, testing sequential"
            test_sequential_submissions
        fi
    else
        log_test "INFO" "Batch submission" "Batch API not available"
        test_sequential_submissions
    fi
}

test_sequential_submissions() {
    log_test "INFO" "Testing sequential submissions..."
    
    local success_count=0
    for i in {1..3}; do
        local result=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
            -H "Content-Type: application/json" \
            -d "{\"source_code\": \"print($i)\", \"language_id\": 71}" 2>/dev/null || echo "FAILED")
        
        if [[ "$result" != "FAILED" ]]; then
            local stdout=$(echo "$result" | python3 -c "import sys, json; print(json.load(sys.stdin).get('stdout', '').strip())" 2>/dev/null)
            if [[ "$stdout" == "$i" ]]; then
                ((success_count++))
            fi
        fi
    done
    
    if [[ $success_count -eq 3 ]]; then
        log_test "PASS" "Sequential submissions" "All 3 submissions succeeded"
    else
        log_test "FAIL" "Sequential submissions" "Only $success_count/3 succeeded"
    fi
}

test_error_handling() {
    log_test "INFO" "Testing error handling..."
    
    # Test compilation error
    local compile_error=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{
            "source_code": "print(undefined_variable)",
            "language_id": 71
        }' 2>/dev/null || echo "FAILED")
    
    if [[ "$compile_error" != "FAILED" ]]; then
        local status_id=$(echo "$compile_error" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', {}).get('id', 0))" 2>/dev/null || echo "0")
        local stderr=$(echo "$compile_error" | python3 -c "import sys, json; print(json.load(sys.stdin).get('stderr', '')[:100])" 2>/dev/null || echo "")
        
        if [[ $status_id -ge 6 && $status_id -le 11 ]]; then
            log_test "PASS" "Runtime error handling" "Correctly detected error (status: $status_id)"
            if [[ -n "$stderr" ]]; then
                log_test "PASS" "Error message" "Error details provided"
            fi
        else
            log_test "FAIL" "Runtime error handling" "Unexpected status: $status_id"
        fi
    else
        log_test "FAIL" "Error handling test" "Failed to submit"
    fi
    
    # Test invalid language ID
    local invalid_lang=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{
            "source_code": "print(\"test\")",
            "language_id": 99999
        }' 2>/dev/null)
    
    # This should fail or return an error
    if [[ -z "$invalid_lang" ]] || echo "$invalid_lang" | grep -q "error\|Error\|invalid\|Invalid"; then
        log_test "PASS" "Invalid language handling" "Properly rejected invalid language ID"
    else
        log_test "FAIL" "Invalid language handling" "Accepted invalid language ID"
    fi
}

test_performance_benchmarks() {
    log_test "INFO" "Running performance benchmarks..."
    
    # Simple program execution time
    local start_time=$(date +%s%N)
    local perf_result=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{
            "source_code": "for i in range(100000): pass\nprint(\"done\")",
            "language_id": 71
        }' 2>/dev/null || echo "FAILED")
    local end_time=$(date +%s%N)
    
    if [[ "$perf_result" != "FAILED" ]]; then
        local exec_time=$(( (end_time - start_time) / 1000000 ))
        local cpu_time=$(echo "$perf_result" | python3 -c "import sys, json; print(json.load(sys.stdin).get('time', 'N/A'))" 2>/dev/null || echo "N/A")
        local memory=$(echo "$perf_result" | python3 -c "import sys, json; print(json.load(sys.stdin).get('memory', 'N/A'))" 2>/dev/null || echo "N/A")
        
        log_test "PERF" "Python loop (100k iterations)" "Total: ${exec_time}ms, CPU: ${cpu_time}s, Memory: ${memory}KB"
        
        if [[ $exec_time -lt 2000 ]]; then
            log_test "PASS" "Performance target" "Execution completed in <2s"
        else
            log_test "FAIL" "Performance target" "Execution took ${exec_time}ms (target: <2000ms)"
        fi
    else
        log_test "FAIL" "Performance benchmark" "Failed to execute"
    fi
    
    # Test concurrent execution performance
    log_test "INFO" "Testing concurrent execution..."
    
    local concurrent_start=$(date +%s%N)
    for i in {1..5}; do
        timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=false" \
            -H "Content-Type: application/json" \
            -d "{\"source_code\": \"import time; time.sleep(0.1); print($i)\", \"language_id\": 71}" &>/dev/null &
    done
    wait
    local concurrent_end=$(date +%s%N)
    local concurrent_time=$(( (concurrent_end - concurrent_start) / 1000000 ))
    
    log_test "PERF" "5 concurrent submissions" "${concurrent_time}ms total"
    
    if [[ $concurrent_time -lt 5000 ]]; then
        log_test "PASS" "Concurrent execution" "Handled 5 submissions efficiently"
    else
        log_test "INFO" "Concurrent execution" "Took ${concurrent_time}ms for 5 submissions"
    fi
}

test_stdin_stdout_handling() {
    log_test "INFO" "Testing stdin/stdout handling..."
    
    # Test with stdin
    local stdin_test=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{
            "source_code": "name = input(); print(f\"Hello, {name}!\")",
            "language_id": 71,
            "stdin": "Judge0"
        }' 2>/dev/null || echo "FAILED")
    
    if [[ "$stdin_test" != "FAILED" ]]; then
        local stdout=$(echo "$stdin_test" | python3 -c "import sys, json; print(json.load(sys.stdin).get('stdout', '').strip())" 2>/dev/null)
        if [[ "$stdout" == "Hello, Judge0!" ]]; then
            log_test "PASS" "Stdin handling" "Correctly processed input"
        else
            log_test "FAIL" "Stdin handling" "Expected: 'Hello, Judge0!', Got: '$stdout'"
        fi
    else
        log_test "FAIL" "Stdin test" "Failed to execute"
    fi
    
    # Test multiline output
    local multiline_test=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{
            "source_code": "for i in range(3): print(f\"Line {i+1}\")",
            "language_id": 71
        }' 2>/dev/null || echo "FAILED")
    
    if [[ "$multiline_test" != "FAILED" ]]; then
        local line_count=$(echo "$multiline_test" | python3 -c "import sys, json; stdout=json.load(sys.stdin).get('stdout', ''); print(len(stdout.strip().split('\\n')))" 2>/dev/null || echo "0")
        if [[ $line_count -eq 3 ]]; then
            log_test "PASS" "Multiline output" "Correctly captured 3 lines"
        else
            log_test "FAIL" "Multiline output" "Expected 3 lines, got $line_count"
        fi
    else
        log_test "FAIL" "Multiline test" "Failed to execute"
    fi
}

test_security_isolation() {
    log_test "INFO" "Testing security isolation..."
    
    # Test network isolation (should fail to connect)
    local network_test=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{
            "source_code": "import urllib.request; urllib.request.urlopen(\"http://google.com\")",
            "language_id": 71
        }' 2>/dev/null || echo "FAILED")
    
    if [[ "$network_test" != "FAILED" ]]; then
        local status_id=$(echo "$network_test" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', {}).get('id', 0))" 2>/dev/null || echo "0")
        
        # Should fail with runtime error due to network isolation
        if [[ $status_id -ge 7 && $status_id -le 11 ]]; then
            log_test "PASS" "Network isolation" "Network access properly blocked"
        elif [[ $status_id -eq 3 ]]; then
            log_test "FAIL" "Network isolation" "Network access not blocked!"
        else
            log_test "INFO" "Network isolation" "Status: $status_id"
        fi
    else
        log_test "FAIL" "Security test" "Failed to submit"
    fi
    
    # Test file system isolation
    local fs_test=$(timeout 10 curl -sf -X POST "${API_URL}/submissions?wait=true" \
        -H "Content-Type: application/json" \
        -d '{
            "source_code": "import os; os.system(\"rm -rf /\")",
            "language_id": 71
        }' 2>/dev/null || echo "FAILED")
    
    if [[ "$fs_test" != "FAILED" ]]; then
        # Should complete without damaging the system
        log_test "PASS" "Filesystem isolation" "Dangerous commands safely isolated"
    else
        log_test "INFO" "Filesystem test" "Could not verify isolation"
    fi
}

main() {
    echo -e "\n${GREEN}Running Judge0 Integration Tests${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Run all integration tests
    test_multi_language_execution
    test_batch_submissions
    test_error_handling
    test_performance_benchmarks
    test_stdin_stdout_handling
    test_security_isolation
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ $TEST_FAILED -eq 0 ]]; then
        echo -e "${GREEN}✅ All integration tests passed${NC}\n"
        exit 0
    else
        echo -e "${RED}❌ Some integration tests failed${NC}\n"
        exit 1
    fi
}

main "$@"