#!/usr/bin/env bash
# Test Performance Profiler - Phase 3 Optimization
# Analyzes test performance patterns and provides optimization recommendations

set -euo pipefail

TESTS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SCRIPTS_DIR=$(dirname "${TESTS_DIR}")
PROFILE_DIR="/tmp/vrooli-test-profile-$$"
PROFILE_DATA="$PROFILE_DIR/profile.data"

mkdir -p "$PROFILE_DIR"

# Source utilities
source "${TESTS_DIR}/../helpers/utils/log.sh"

#######################################
# Performance Analysis Functions
#######################################

# Profile individual test file performance
profile_test_file() {
    local test_file="$1"
    local profile_output="$PROFILE_DIR/$(basename "$test_file").profile"
    
    log::debug "🔍 Profiling $(basename "$test_file")..."
    
    # Measure test execution time and resource usage
    local start_time=$(date +%s.%N)
    local start_memory=$(ps -o rss= -p $$ 2>/dev/null || echo 0)
    
    # Run test with timeout and capture detailed output
    local output=""
    local exit_code=0
    local timed_out=false
    
    if timeout 30s bats --tap "$test_file" > "$profile_output.output" 2>&1; then
        exit_code=0
    else
        exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            timed_out=true
            echo "TIMEOUT after 30s" >> "$profile_output.output"
        fi
    fi
    
    local end_time=$(date +%s.%N)
    local end_memory=$(ps -o rss= -p $$ 2>/dev/null || echo 0)
    
    # Calculate metrics
    local duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "0")
    local memory_delta=$((end_memory - start_memory))
    
    # Count test results
    local test_count=0
    local failure_count=0
    local skip_count=0
    
    if [[ -f "$profile_output.output" ]]; then
        output=$(cat "$profile_output.output")
        test_count=$(echo "$output" | grep -c "^ok\|^not ok" 2>/dev/null || echo 0)
        failure_count=$(echo "$output" | grep -c "^not ok" 2>/dev/null || echo 0)
        skip_count=$(echo "$output" | grep -c "# skip" 2>/dev/null || echo 0)
        
        # Clean up any potential whitespace/newlines
        test_count=$(echo "$test_count" | tr -d '\n\r ' 2>/dev/null || echo 0)
        failure_count=$(echo "$failure_count" | tr -d '\n\r ' 2>/dev/null || echo 0)
        skip_count=$(echo "$skip_count" | tr -d '\n\r ' 2>/dev/null || echo 0)
        
        # Ensure they're valid numbers
        [[ "$test_count" =~ ^[0-9]+$ ]] || test_count=0
        [[ "$failure_count" =~ ^[0-9]+$ ]] || failure_count=0
        [[ "$skip_count" =~ ^[0-9]+$ ]] || skip_count=0
    fi
    
    # Determine performance category
    local perf_category="FAST"
    if $timed_out || (( $(echo "$duration > 10" | bc -l 2>/dev/null || echo 0) )); then
        perf_category="SLOW"
    elif (( $(echo "$duration > 3" | bc -l 2>/dev/null || echo 0) )); then
        perf_category="MEDIUM"
    fi
    
    # Analyze test characteristics
    local has_docker=$(grep -c "docker\|container" "$test_file" 2>/dev/null || echo 0)
    local has_curl=$(grep -c "curl\|http" "$test_file" 2>/dev/null || echo 0)
    local has_sleep=$(grep -c "sleep\|timeout" "$test_file" 2>/dev/null || echo 0)
    local has_external=$(grep -c "source.*manage\.sh\|localhost:[0-9]" "$test_file" 2>/dev/null || echo 0)
    
    # Write profile record
    cat >> "$PROFILE_DATA" << EOF
FILE:$(basename "$test_file")
PATH:$test_file
DURATION:$duration
MEMORY_DELTA:$memory_delta
TESTS:$test_count
FAILURES:$failure_count
SKIPS:$skip_count
EXIT_CODE:$exit_code
TIMED_OUT:$timed_out
PERF_CATEGORY:$perf_category
HAS_DOCKER:$has_docker
HAS_CURL:$has_curl
HAS_SLEEP:$has_sleep
HAS_EXTERNAL:$has_external
TIMESTAMP:$(date +%s)
---
EOF
    
    # Show progress
    if $timed_out; then
        log::error "⏰ $(basename "$test_file"): TIMEOUT (>30s)"
    elif [[ "$perf_category" == "SLOW" ]]; then
        log::warning "🐌 $(basename "$test_file"): ${duration}s (SLOW)"
    elif [[ "$failure_count" -gt 0 ]]; then
        log::error "❌ $(basename "$test_file"): ${duration}s, $failure_count failures"
    else
        log::success "✅ $(basename "$test_file"): ${duration}s"
    fi
}

# Analyze all test files and generate performance report
analyze_test_performance() {
    local -a all_test_files
    local total_files=0
    local profiled_files=0
    
    log::header "🔍 Test Performance Analysis"
    
    # Find all test files
    while IFS= read -r test_file; do
        all_test_files+=("$test_file")
    done < <(find "$SCRIPTS_DIR" -path "$SCRIPTS_DIR/__tests/helpers" -prune -o -type f -name '*.bats' -print)
    
    total_files=${#all_test_files[@]}
    log::info "📊 Found $total_files test files to profile"
    
    # Profile each test file
    for test_file in "${all_test_files[@]}"; do
        profile_test_file "$test_file"
        ((profiled_files++))
        
        # Show progress every 10 files
        if ((profiled_files % 10 == 0)); then
            log::info "📈 Progress: $profiled_files/$total_files files profiled"
        fi
    done
    
    log::info "✅ Profiling complete: $profiled_files files analyzed"
}

# Generate performance optimization recommendations
generate_recommendations() {
    log::header "🎯 Performance Optimization Recommendations"
    
    if [[ ! -f "$PROFILE_DATA" ]]; then
        log::error "No profile data found. Run analysis first."
        return 1
    fi
    
    # Parse profile data
    local total_duration=0
    local slow_tests=0
    local timeout_tests=0
    local failing_tests=0
    local docker_tests=0
    local external_tests=0
    
    local -a slowest_tests=()
    local -a timeout_tests_list=()
    local -a failing_tests_list=()
    
    while IFS= read -r line; do
        if [[ "$line" =~ ^FILE:(.*) ]]; then
            current_file="${BASH_REMATCH[1]}"
        elif [[ "$line" =~ ^DURATION:(.*) ]]; then
            local duration="${BASH_REMATCH[1]}"
            total_duration=$(echo "$total_duration + $duration" | bc -l 2>/dev/null || echo "$total_duration")
            
            # Track slow tests (>3s)
            if (( $(echo "$duration > 3" | bc -l 2>/dev/null || echo 0) )); then
                ((slow_tests++))
                slowest_tests+=("$current_file:$duration")
            fi
        elif [[ "$line" =~ ^TIMED_OUT:true ]]; then
            ((timeout_tests++))
            timeout_tests_list+=("$current_file")
        elif [[ "$line" =~ ^FAILURES:([1-9][0-9]*) ]]; then
            ((failing_tests++))
            failing_tests_list+=("$current_file")
        elif [[ "$line" =~ ^HAS_DOCKER:([1-9][0-9]*) ]]; then
            ((docker_tests++))
        elif [[ "$line" =~ ^HAS_EXTERNAL:([1-9][0-9]*) ]]; then
            ((external_tests++))
        fi
    done < "$PROFILE_DATA"
    
    # Calculate statistics
    local avg_duration=$(echo "scale=2; $total_duration / $(grep -c "^FILE:" "$PROFILE_DATA")" | bc -l 2>/dev/null || echo "0")
    
    # Generate report
    echo ""
    log::info "📊 Performance Statistics:"
    log::info "  Total execution time: ${total_duration}s"
    log::info "  Average test time: ${avg_duration}s"
    log::info "  Slow tests (>3s): $slow_tests"
    log::info "  Timeout tests (>30s): $timeout_tests"
    log::info "  Failing tests: $failing_tests"
    log::info "  Tests with Docker deps: $docker_tests"
    log::info "  Tests with external deps: $external_tests"
    
    echo ""
    log::info "🎯 Optimization Recommendations:"
    
    # Recommendation 1: Skip timeout tests
    if [[ $timeout_tests -gt 0 ]]; then
        echo "  1. 🚫 SKIP TIMEOUT TESTS ($timeout_tests tests)"
        echo "     These tests consistently timeout and should be skipped:"
        for test in "${timeout_tests_list[@]}"; do
            echo "     - $test"
        done
        echo ""
    fi
    
    # Recommendation 2: Mock external dependencies  
    if [[ $external_tests -gt 10 ]]; then
        echo "  2. 🎭 MOCK EXTERNAL DEPENDENCIES ($external_tests tests)"
        echo "     Replace real service calls with fast mocks"
        echo "     Estimated time savings: $(echo "$external_tests * 2" | bc)s"
        echo ""
    fi
    
    # Recommendation 3: Parallelize slow tests
    if [[ $slow_tests -gt 5 ]]; then
        echo "  3. ⚡ PARALLELIZE SLOW TESTS ($slow_tests tests)"
        echo "     Run slow tests in parallel pools"
        echo "     Estimated time savings: $(echo "scale=0; $total_duration * 0.4" | bc -l)s"
        echo ""
    fi
    
    # Recommendation 4: Fix failing tests
    if [[ $failing_tests -gt 0 ]]; then
        echo "  4. 🔧 FIX FAILING TESTS ($failing_tests tests)"
        echo "     These tests have consistent failures:"
        for test in "${failing_tests_list[@]}"; do
            echo "     - $test"
        done
        echo ""
    fi
    
    # Recommendation 5: Docker optimization
    if [[ $docker_tests -gt 20 ]]; then
        echo "  5. 🐳 OPTIMIZE DOCKER TESTS ($docker_tests tests)"
        echo "     Use shared Docker environments and faster mocks"
        echo "     Estimated time savings: $(echo "$docker_tests * 1.5" | bc)s"
        echo ""
    fi
    
    # Show top 10 slowest tests
    if [[ ${#slowest_tests[@]} -gt 0 ]]; then
        echo ""
        log::info "🐌 Top 10 Slowest Tests:"
        printf '%s\n' "${slowest_tests[@]}" | sort -t: -k2 -nr | head -10 | while IFS=: read -r file duration; do
            echo "  - $file: ${duration}s"
        done
    fi
    
    # Calculate potential improvements
    local potential_savings=$((timeout_tests * 30 + external_tests * 2 + docker_tests * 1))
    local optimized_time=$(echo "$total_duration - $potential_savings" | bc -l)
    
    echo ""
    log::success "💡 Potential Improvements:"
    log::success "  Current total time: ${total_duration}s"
    log::success "  Potential time savings: ${potential_savings}s"
    log::success "  Optimized time: ${optimized_time}s"
    log::success "  Improvement: $(echo "scale=1; $potential_savings * 100 / $total_duration" | bc -l)%"
}

# Create smart test configuration based on analysis
create_optimized_config() {
    log::header "⚙️ Creating Optimized Test Configuration"
    
    local config_file="$TESTS_DIR/optimized_test_config.json"
    
    # Extract data for optimization
    local -a skip_patterns=()
    local -a parallel_candidates=()
    local -a mock_candidates=()
    
    # Process profile data to build configuration
    while IFS= read -r line; do
        if [[ "$line" =~ ^FILE:(.*) ]]; then
            current_file="${BASH_REMATCH[1]}"
        elif [[ "$line" =~ ^TIMED_OUT:true ]]; then
            skip_patterns+=("$(basename "$current_file" .bats)")
        elif [[ "$line" =~ ^DURATION:([3-9]|[1-9][0-9]+) ]]; then
            parallel_candidates+=("$current_file")
        elif [[ "$line" =~ ^HAS_EXTERNAL:([1-9][0-9]*) ]]; then
            mock_candidates+=("$current_file")
        fi
    done < "$PROFILE_DATA"
    
    # Generate JSON configuration
    cat > "$config_file" << EOF
{
  "optimized_test_config": {
    "version": "1.0",
    "generated": "$(date -Iseconds)",
    "skip_patterns": [
$(printf '      "%s"' "${skip_patterns[@]}" | paste -sd, -)
    ],
    "parallel_candidates": [
$(printf '      "%s"' "${parallel_candidates[@]}" | paste -sd, -)
    ],
    "mock_candidates": [
$(printf '      "%s"' "${mock_candidates[@]}" | paste -sd, -)
    ],
    "recommendations": {
      "skip_timeout_tests": $(( ${#skip_patterns[@]} > 0 )),
      "enable_parallel_execution": $(( ${#parallel_candidates[@]} > 5 )),
      "implement_mocking": $(( ${#mock_candidates[@]} > 10 )),
      "estimated_time_savings": "$((${#skip_patterns[@]} * 30 + ${#mock_candidates[@]} * 2))s"
    }
  }
}
EOF
    
    log::success "✅ Optimized configuration saved to: $config_file"
    log::info "📋 Configuration includes:"
    log::info "  - Skip patterns: ${#skip_patterns[@]} tests"
    log::info "  - Parallel candidates: ${#parallel_candidates[@]} tests"
    log::info "  - Mock candidates: ${#mock_candidates[@]} tests"
}

#######################################
# Main Profiler Function
#######################################

main() {
    local action="analyze"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            analyze|profile)
                action="analyze"
                shift
                ;;
            recommend|recommendations)
                action="recommend"
                shift
                ;;
            config|configure)
                action="config"
                shift
                ;;
            all)
                action="all"
                shift
                ;;
            --help|-h)
                echo "Test Performance Profiler - Phase 3 Optimization"
                echo ""
                echo "Usage: $0 [ACTION]"
                echo ""
                echo "Actions:"
                echo "  analyze              Profile all test files (default)"
                echo "  recommend            Generate optimization recommendations"
                echo "  config               Create optimized test configuration"
                echo "  all                  Run analysis + recommendations + config"
                echo ""
                echo "Examples:"
                echo "  $0 analyze           # Profile test performance"
                echo "  $0 recommend         # Get optimization suggestions"
                echo "  $0 all               # Complete optimization analysis"
                exit 0
                ;;
            *)
                log::error "Unknown action: $1"
                exit 1
                ;;
        esac
    done
    
    # Execute requested action
    case $action in
        analyze)
            analyze_test_performance
            ;;
        recommend)
            generate_recommendations
            ;;
        config)
            create_optimized_config
            ;;
        all)
            analyze_test_performance
            echo ""
            generate_recommendations
            echo ""
            create_optimized_config
            ;;
    esac
    
    # Cleanup
    # rm -rf "$PROFILE_DIR"  # Keep for debugging
    log::info "🗂️  Profile data saved in: $PROFILE_DIR"
}

main "$@"