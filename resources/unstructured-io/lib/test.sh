#!/usr/bin/env bash
# Unstructured.io Resource Testing Functions (Resource Validation, not Document Processing)
# Document processing functionality moved to lib/process.sh

# ==================== RESOURCE VALIDATION FUNCTIONS ====================
# These test Unstructured.io as a resource (health, connectivity, functionality)

# Quick smoke test - validate Unstructured.io resource health
unstructured_io::test::smoke() {
    # Delegate to v2.0 test runner as per universal.yaml
    bash "${UNSTRUCTURED_IO_CLI_DIR}/test/run-tests.sh" smoke "$@"
}

# Full integration test - validate Unstructured.io end-to-end functionality  
unstructured_io::test::integration() {
    # Delegate to v2.0 test runner as per universal.yaml
    bash "${UNSTRUCTURED_IO_CLI_DIR}/test/run-tests.sh" integration "$@"
}

# Unit tests - validate Unstructured.io library functions
unstructured_io::test::unit() {
    # Delegate to v2.0 test runner as per universal.yaml
    bash "${UNSTRUCTURED_IO_CLI_DIR}/test/run-tests.sh" unit "$@"
}

# Run all resource tests
unstructured_io::test::all() {
    # Delegate to v2.0 test runner as per universal.yaml
    bash "${UNSTRUCTURED_IO_CLI_DIR}/test/run-tests.sh" all "$@"
}

# ==================== LEGACY DOCUMENT PROCESSING FUNCTIONS ====================
# DEPRECATED: These functions moved to lib/process.sh
# Kept for backward compatibility with deprecation warnings

# DEPRECATED: Use content execute instead
unstructured_io::test::process() {
    log::warning "DEPRECATED: unstructured_io::test::process is deprecated"
    log::warning "Use 'resource-unstructured-io content execute' instead"
    log::warning "This function will be removed in v3.0 (December 2025)"
    
    # Delegate to new content system
    unstructured_io::process_document "$@"
}

# DEPRECATED: Use content list instead
unstructured_io::test::list() {
    log::warning "DEPRECATED: unstructured_io::test::list is deprecated"
    log::warning "Use 'resource-unstructured-io content list' instead"
    log::warning "This function will be removed in v3.0 (December 2025)"
    
    # Delegate to new content system
    unstructured_io::content::list "$@"
}