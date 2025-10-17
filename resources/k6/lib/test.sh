#!/usr/bin/env bash
# K6 Resource Testing Functions (Resource Validation, not Performance Testing)
# Performance testing moved to lib/content.sh

# ==================== RESOURCE VALIDATION FUNCTIONS ====================
# These test K6 as a resource (health, connectivity, functionality)

# Quick smoke test - validate K6 resource health
k6::test::smoke() {
    bash "${K6_CLI_DIR}/test/phases/test-smoke.sh" "$@"
}

# Full integration test - validate K6 end-to-end functionality  
k6::test::integration() {
    bash "${K6_CLI_DIR}/test/phases/test-integration.sh" "$@"
}

# Unit tests - validate K6 library functions
k6::test::unit() {
    bash "${K6_CLI_DIR}/test/phases/test-unit.sh" "$@"
}

# Run all resource tests
k6::test::all() {
    bash "${K6_CLI_DIR}/test/run-tests.sh" all "$@"
}

# ==================== LEGACY PERFORMANCE TESTING FUNCTIONS ====================
# DEPRECATED: These functions moved to lib/content.sh
# Kept for backward compatibility with deprecation warnings

# DEPRECATED: Use content execute instead
k6::test::run() {
    log::warning "DEPRECATED: k6::test::run is deprecated"
    log::warning "Use 'resource-k6 content execute' instead"
    log::warning "This function will be removed in v3.0 (December 2025)"
    
    # Delegate to new content system
    k6::content::run_performance_test "$@"
}

# DEPRECATED: Use content list instead
k6::test::list() {
    log::warning "DEPRECATED: k6::test::list is deprecated"
    log::warning "Use 'resource-k6 content list' instead"
    log::warning "This function will be removed in v3.0 (December 2025)"
    
    # Delegate to new content system
    k6::content::list_scripts "$@"
}

# DEPRECATED: Use content results instead
k6::test::results() {
    log::warning "DEPRECATED: k6::test::results is deprecated"
    log::warning "Use 'resource-k6 content results' instead"
    log::warning "This function will be removed in v3.0 (December 2025)"
    
    # Delegate to new content system
    k6::content::show_results "$@"
}

# Enhanced stress test function (adds value over basic content execute)
k6::test::stress() {
    log::info "Running K6 stress test with escalating load..."
    
    # Use enhanced stress testing parameters
    k6::content::run_performance_test "$1" --stages "1m:10,5m:50,1m:100,2m:100,1m:0" --thresholds "http_req_duration[p(95)]<500" "${@:2}"
}