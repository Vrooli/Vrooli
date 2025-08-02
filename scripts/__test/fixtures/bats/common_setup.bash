#!/bin/bash
# Common BATS Test Setup
# This file provides standard setup that all BATS tests can use

# Prevent duplicate sourcing
if [[ "${COMMON_SETUP_LOADED:-}" == "true" ]]; then
    return 0
fi
export COMMON_SETUP_LOADED="true"

# Source the standard mock framework
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/standard_mock_framework.bash"

#######################################
# Standard setup function for BATS tests
# Globals: MOCK_RESPONSES_DIR, BATS_TEST_TMPDIR
# Arguments: None
# Returns: 0 on success
#######################################
setup_standard_mocks() {
    # Create mock responses directory
    export MOCK_RESPONSES_DIR="${BATS_TEST_TMPDIR:-/tmp}/mock_responses"
    mkdir -p "$MOCK_RESPONSES_DIR"
    
    # Set standard environment variables
    export FORCE="${FORCE:-no}"
    export YES="${YES:-no}"
    export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
    export VERBOSE="${VERBOSE:-no}"
    export DEBUG="${DEBUG:-no}"
    
    # Setup the mock framework
    setup_standard_mock_framework
    
    return 0
}

#######################################
# Standard cleanup function for BATS tests
# Globals: MOCK_RESPONSES_DIR
# Arguments: None
# Returns: 0 on success
#######################################
cleanup_mocks() {
    # Clean up mock framework
    cleanup_standard_mock_framework
    
    # Remove mock responses directory
    rm -rf "$MOCK_RESPONSES_DIR" 2>/dev/null || true
    
    # Clear loaded flag
    unset COMMON_SETUP_LOADED
    
    return 0
}

# Auto-run setup if this file is sourced directly
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    setup_standard_mocks
fi