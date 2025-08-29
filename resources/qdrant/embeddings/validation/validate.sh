#!/usr/bin/env bash
# Main Validation Script for Qdrant Embeddings
# Orchestrates all validation checks and provides comprehensive system validation

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
VALIDATION_DIR="${EMBEDDINGS_DIR}/validation"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source validation components
source "${VALIDATION_DIR}/schema-validator.sh"
source "${VALIDATION_DIR}/service-health.sh"
source "${VALIDATION_DIR}/performance-benchmark.sh"

#######################################
# Run pre-flight checks
# Returns: 0 if ready, 1 if issues found
#######################################
run_preflight_checks() {
    log::info "=== Pre-flight Validation Checks ==="
    local issues=0
    
    # Check if embedding CLI exists and is functional
    if [[ ! -f "${EMBEDDINGS_DIR}/cli.sh" ]]; then
        log::error "Embedding CLI not found: ${EMBEDDINGS_DIR}/cli.sh"
        ((issues++))
    else
        log::success "Embedding CLI found"
        
        # Test CLI help
        if bash "${EMBEDDINGS_DIR}/cli.sh" help >/dev/null 2>&1; then
            log::success "Embedding CLI is functional"
        else
            log::error "Embedding CLI help command failed"
            ((issues++))
        fi
    fi
    
    # Check extractor directories
    local extractors=("code" "docs" "scenarios" "resources" "filetrees" "initialization")
    local missing_extractors=()
    
    for extractor in "${extractors[@]}"; do
        local extractor_path="${EMBEDDINGS_DIR}/extractors/${extractor}/main.sh"
        if [[ -f "$extractor_path" ]]; then
            log::success "Extractor found: $extractor"
        else
            log::error "Extractor missing: $extractor ($extractor_path)"
            missing_extractors+=("$extractor")
            ((issues++))
        fi
    done
    
    # Check configuration
    if [[ -f "${EMBEDDINGS_DIR}/config/unified.sh" ]]; then
        log::success "Unified configuration found"
    else
        log::error "Unified configuration missing"
        ((issues++))
    fi
    
    # Check library files
    local libraries=("embedding-service.sh" "parallel.sh")
    for lib in "${libraries[@]}"; do
        if [[ -f "${EMBEDDINGS_DIR}/lib/${lib}" ]]; then
            log::success "Library found: $lib"
        else
            log::error "Library missing: $lib"
            ((issues++))
        fi
    done
    
    if [[ $issues -eq 0 ]]; then
        log::success "Pre-flight checks passed ✅"
        return 0
    else
        log::error "$issues pre-flight issues found ❌"
        return 1
    fi
}

#######################################
# Run system validation
# Returns: 0 if valid, 1 if issues found
#######################################
run_system_validation() {
    log::info "=== System Validation ==="
    local issues=0
    
    # Service health check
    if ! check_service_health "http://localhost:6333" "Qdrant"; then
        ((issues++))
    fi
    
    if ! check_service_health "http://localhost:11434/api/tags" "Ollama"; then
        ((issues++))
    fi
    
    # Test embedding generation
    if ! test_embedding_generation; then
        ((issues++))
    fi
    
    # Resource checks
    if ! check_disk_space; then
        ((issues++))
    fi
    
    if ! check_memory; then
        ((issues++))
    fi
    
    if [[ $issues -eq 0 ]]; then
        log::success "System validation passed ✅"
        return 0
    else
        log::error "$issues system validation issues found ❌"
        return 1
    fi
}

#######################################
# Run functional tests
# Returns: 0 if successful, 1 if issues found
#######################################
run_functional_tests() {
    log::info "=== Functional Tests ==="
    local issues=0
    
    # Test app initialization
    log::info "Testing app initialization..."
    if bash "${EMBEDDINGS_DIR}/cli.sh" status >/dev/null 2>&1; then
        log::success "Status command functional"
    else
        log::error "Status command failed"
        ((issues++))
    fi
    
    # Test validation command
    log::info "Testing validation command..."
    if bash "${EMBEDDINGS_DIR}/cli.sh" validate >/dev/null 2>&1; then
        log::success "Validate command functional"
    else
        log::error "Validate command failed"
        ((issues++))
    fi
    
    # Test basic extractor functionality (without full processing)
    local test_dir="/tmp/validation-test-$$"
    mkdir -p "$test_dir"
    trap "rm -rf $test_dir" EXIT
    
    # Create test code file
    cat > "$test_dir/test.js" << 'EOF'
function testFunction() {
    console.log("This is a test function");
    return true;
}
EOF
    
    # Test code extractor
    log::info "Testing code extractor..."
    if APP_ROOT="$APP_ROOT" bash -c "
        source '${EMBEDDINGS_DIR}/extractors/code/main.sh'
        qdrant::extract::code '$test_dir/test.js'
    " >/dev/null 2>&1; then
        log::success "Code extractor functional"
    else
        log::error "Code extractor failed"
        ((issues++))
    fi
    
    # Test docs extractor
    cat > "$test_dir/test.md" << 'EOF'
# Test Documentation

This is a test document for validation purposes.

## Features
- Feature A
- Feature B
EOF
    
    log::info "Testing docs extractor..."
    if APP_ROOT="$APP_ROOT" bash -c "
        source '${EMBEDDINGS_DIR}/extractors/docs/main.sh'
        qdrant::extract::docs_batch '$test_dir' '/tmp/test-docs.jsonl'
    " >/dev/null 2>&1; then
        log::success "Docs extractor functional"
    else
        log::error "Docs extractor failed"
        ((issues++))
    fi
    
    if [[ $issues -eq 0 ]]; then
        log::success "Functional tests passed ✅"
        return 0
    else
        log::error "$issues functional test issues found ❌"
        return 1
    fi
}

#######################################
# Run performance tests
# Returns: 0 if successful, 1 if issues found
#######################################
run_performance_tests() {
    log::info "=== Performance Tests ==="
    local issues=0
    
    # Quick performance check
    log::info "Running quick performance test..."
    local single_time=$(benchmark_single_embedding "Test content for validation")
    
    if [[ "$single_time" -gt 0 && "$single_time" -lt 10000 ]]; then  # Less than 10 seconds
        log::success "Single embedding performance: ${single_time}ms ✅"
    elif [[ "$single_time" -gt 0 ]]; then
        log::warn "Single embedding performance slow: ${single_time}ms ⚠️"
    else
        log::error "Single embedding failed ❌"
        ((issues++))
    fi
    
    # Quick batch test
    log::info "Running quick batch test..."
    local test_file=$(create_test_data 5 "code")
    local batch_results=$(benchmark_batch_processing "$test_file" 5)
    
    local success_rate=$(echo "$batch_results" | jq -r '.success_rate')
    local throughput=$(echo "$batch_results" | jq -r '.throughput_per_second')
    
    if [[ "${success_rate%.*}" -ge 80 ]]; then  # At least 80% success rate
        log::success "Batch processing success rate: ${success_rate}% ✅"
    else
        log::error "Batch processing success rate too low: ${success_rate}% ❌"
        ((issues++))
    fi
    
    log::info "Batch throughput: ${throughput} items/second"
    
    if [[ $issues -eq 0 ]]; then
        log::success "Performance tests passed ✅"
        return 0
    else
        log::error "$issues performance test issues found ❌"
        return 1
    fi
}

#######################################
# Generate validation report
#######################################
generate_validation_report() {
    local report_file="${VALIDATION_DIR}/validation-report-$(date +%Y%m%d-%H%M%S).txt"
    
    {
        echo "=== QDRANT EMBEDDINGS VALIDATION REPORT ==="
        echo "Timestamp: $(date)"
        echo "System: $(uname -a)"
        echo "App Root: $APP_ROOT"
        echo
        
        echo "=== PRE-FLIGHT CHECKS ==="
        if run_preflight_checks; then
            echo "Status: PASSED ✅"
        else
            echo "Status: FAILED ❌"
        fi
        echo
        
        echo "=== SYSTEM VALIDATION ==="
        if run_system_validation; then
            echo "Status: PASSED ✅"
        else
            echo "Status: FAILED ❌"
        fi
        echo
        
        echo "=== FUNCTIONAL TESTS ==="
        if run_functional_tests; then
            echo "Status: PASSED ✅"
        else
            echo "Status: FAILED ❌"
        fi
        echo
        
        echo "=== PERFORMANCE TESTS ==="
        if run_performance_tests; then
            echo "Status: PASSED ✅"
        else
            echo "Status: FAILED ❌"
        fi
        echo
        
        echo "=== SYSTEM INFO ==="
        echo "Memory: $(free -h | grep Mem | awk '{print $3 "/" $2}')"
        echo "Disk /tmp: $(df -h /tmp | tail -1 | awk '{print $4 " available"}')"
        echo "Qdrant Status: $(curl -s http://localhost:6333/health 2>/dev/null | jq -r '.status // "unavailable"')"
        echo "Ollama Models: $(curl -s http://localhost:11434/api/tags 2>/dev/null | jq -r '.models | length // 0') available"
        
    } > "$report_file"
    
    log::info "Validation report generated: $report_file"
    return 0
}

#######################################
# Run comprehensive validation
#######################################
run_comprehensive_validation() {
    log::info "=== Comprehensive Qdrant Embeddings Validation ==="
    local total_issues=0
    
    # Run all validation checks
    if ! run_preflight_checks; then
        ((total_issues++))
    fi
    
    if ! run_system_validation; then
        ((total_issues++))
    fi
    
    if ! run_functional_tests; then
        ((total_issues++))
    fi
    
    if ! run_performance_tests; then
        ((total_issues++))
    fi
    
    echo
    if [[ $total_issues -eq 0 ]]; then
        log::success "🎉 ALL VALIDATION CHECKS PASSED! 🎉"
        log::info "The Qdrant embeddings system is ready for production use."
    else
        log::error "❌ $total_issues VALIDATION SUITE(S) FAILED"
        log::warn "Please address the issues before production use."
    fi
    
    generate_validation_report
    return $total_issues
}

#######################################
# Main function
#######################################
main() {
    local validation_type="${1:-comprehensive}"
    
    case "$validation_type" in
        preflight)
            run_preflight_checks
            ;;
        system)
            run_system_validation
            ;;
        functional)
            run_functional_tests
            ;;
        performance)
            run_performance_tests
            ;;
        report)
            generate_validation_report
            ;;
        comprehensive|all)
            run_comprehensive_validation
            ;;
        *)
            log::error "Unknown validation type: $validation_type"
            log::info "Usage: $0 <preflight|system|functional|performance|report|comprehensive>"
            return 1
            ;;
    esac
}

# Run main if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi