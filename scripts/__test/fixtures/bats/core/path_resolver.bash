#!/usr/bin/env bash
# Path Resolver for BATS Testing Infrastructure
# Provides centralized, robust path resolution for all test fixtures
# 
# This file MUST be sourced before any other fixture files to ensure
# consistent path resolution across the entire testing infrastructure.

# Prevent duplicate loading
if [[ "${PATH_RESOLVER_LOADED:-}" == "true" ]]; then
    return 0
fi
export PATH_RESOLVER_LOADED="true"

#######################################
# Resolve the absolute path to the bats fixtures directory
# This function is self-contained and doesn't rely on external variables
# Returns: Sets VROOLI_TEST_FIXTURES_DIR environment variable
#######################################
vrooli_resolve_fixtures_path() {
    local source_file="${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}"
    local resolved_path
    
    # Handle different sourcing contexts
    if [[ -n "${VROOLI_TEST_FIXTURES_DIR:-}" ]] && [[ -d "${VROOLI_TEST_FIXTURES_DIR}" ]]; then
        # Already set and valid
        return 0
    fi
    
    # Resolve from the location of this file (path_resolver.bash)
    # This file is in core/, so fixtures is the parent directory
    local this_dir
    this_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    resolved_path="$(cd "${this_dir}/.." && pwd)"
    
    # Validate the resolved path
    if [[ ! -d "${resolved_path}/core" ]] || [[ ! -d "${resolved_path}/mocks" ]]; then
        echo "[PATH_RESOLVER] ERROR: Invalid fixtures directory structure at: ${resolved_path}" >&2
        echo "[PATH_RESOLVER] Expected directories 'core' and 'mocks' not found" >&2
        return 1
    fi
    
    export VROOLI_TEST_FIXTURES_DIR="${resolved_path}"
    export VROOLI_TEST_ROOT="$(cd "${resolved_path}/../.." && pwd)"
    
    # Set derived paths
    export VROOLI_TEST_HELPERS_DIR="${VROOLI_TEST_ROOT}/helpers"
    export VROOLI_TEST_RESOURCES_DIR="${VROOLI_TEST_ROOT}/resources"
    export VROOLI_TEST_SINGLE_DIR="${VROOLI_TEST_ROOT}/single"
    export VROOLI_TEST_SHELL_DIR="${VROOLI_TEST_ROOT}/shell"
    
    # Validate critical paths exist
    local critical_paths=(
        "${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash"
        "${VROOLI_TEST_FIXTURES_DIR}/core/assertions.bash"
        "${VROOLI_TEST_FIXTURES_DIR}/mocks/mock_registry.bash"
    )
    
    for path in "${critical_paths[@]}"; do
        if [[ ! -f "${path}" ]]; then
            echo "[PATH_RESOLVER] ERROR: Critical file not found: ${path}" >&2
            return 1
        fi
    done
    
    return 0
}

#######################################
# Get the absolute path to a fixture file
# Arguments: 
#   $1 - relative path from fixtures directory
# Returns: Absolute path to the file
#######################################
vrooli_fixture_path() {
    local relative_path="$1"
    
    if [[ -z "${VROOLI_TEST_FIXTURES_DIR:-}" ]]; then
        vrooli_resolve_fixtures_path || return 1
    fi
    
    echo "${VROOLI_TEST_FIXTURES_DIR}/${relative_path}"
}

#######################################
# Source a fixture file with error handling
# Arguments:
#   $1 - relative path from fixtures directory
# Returns: 0 on success, 1 on failure
#######################################
vrooli_source_fixture() {
    local relative_path="$1"
    local full_path
    
    full_path="$(vrooli_fixture_path "${relative_path}")"
    
    if [[ ! -f "${full_path}" ]]; then
        echo "[PATH_RESOLVER] ERROR: Cannot source fixture - file not found: ${full_path}" >&2
        return 1
    fi
    
    # shellcheck disable=SC1090
    source "${full_path}"
}

#######################################
# Validate that the test environment is properly configured
# Returns: 0 if valid, 1 if not
#######################################
vrooli_validate_test_environment() {
    local errors=0
    
    # Check required environment variables
    if [[ -z "${VROOLI_TEST_FIXTURES_DIR:-}" ]]; then
        echo "[PATH_RESOLVER] ERROR: VROOLI_TEST_FIXTURES_DIR not set" >&2
        ((errors++))
    fi
    
    # Check directory structure
    local required_dirs=(
        "${VROOLI_TEST_FIXTURES_DIR}/core"
        "${VROOLI_TEST_FIXTURES_DIR}/mocks"
        "${VROOLI_TEST_FIXTURES_DIR}/docs"
        "${VROOLI_TEST_FIXTURES_DIR}/templates"
    )
    
    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "${dir}" ]]; then
            echo "[PATH_RESOLVER] ERROR: Required directory not found: ${dir}" >&2
            ((errors++))
        fi
    done
    
    if [[ ${errors} -gt 0 ]]; then
        echo "[PATH_RESOLVER] ERROR: Test environment validation failed with ${errors} error(s)" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Get the path to the test's temporary directory
# Creates it if it doesn't exist
# Returns: Path to temp directory
#######################################
vrooli_test_tmpdir() {
    if [[ -z "${BATS_TEST_TMPDIR:-}" ]]; then
        # Performance optimization: use in-memory temp directory when available
        if [[ -d "/dev/shm" ]] && [[ -w "/dev/shm" ]]; then
            export BATS_TEST_TMPDIR="/dev/shm/vrooli-tests-$$-${RANDOM}"
        else
            export BATS_TEST_TMPDIR="$(mktemp -d)"
        fi
    fi
    
    mkdir -p "${BATS_TEST_TMPDIR}"
    echo "${BATS_TEST_TMPDIR}"
}

#######################################
# Print debug information about the current path configuration
# Useful for troubleshooting path issues
#######################################
vrooli_debug_paths() {
    echo "[PATH_RESOLVER] Current path configuration:"
    echo "  VROOLI_TEST_FIXTURES_DIR: ${VROOLI_TEST_FIXTURES_DIR:-<not set>}"
    echo "  VROOLI_TEST_ROOT: ${VROOLI_TEST_ROOT:-<not set>}"
    echo "  VROOLI_TEST_HELPERS_DIR: ${VROOLI_TEST_HELPERS_DIR:-<not set>}"
    echo "  VROOLI_TEST_RESOURCES_DIR: ${VROOLI_TEST_RESOURCES_DIR:-<not set>}"
    echo "  BATS_TEST_TMPDIR: ${BATS_TEST_TMPDIR:-<not set>}"
    echo "  Script location: ${BASH_SOURCE[0]}"
    echo "  Called from: ${BASH_SOURCE[1]:-<direct>}"
}

# Initialize path resolution on load
vrooli_resolve_fixtures_path

# Export functions for use in tests
export -f vrooli_resolve_fixtures_path
export -f vrooli_fixture_path
export -f vrooli_source_fixture
export -f vrooli_validate_test_environment
export -f vrooli_test_tmpdir
export -f vrooli_debug_paths

# Validate environment on load (can be disabled by setting SKIP_PATH_VALIDATION=true)
if [[ "${SKIP_PATH_VALIDATION:-false}" != "true" ]]; then
    if ! vrooli_validate_test_environment; then
        echo "[PATH_RESOLVER] WARNING: Test environment validation failed" >&2
        # Don't fail here - let the test decide how to handle this
    fi
fi