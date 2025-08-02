#!/bin/bash
# ====================================================================
# Assertion Module Index
# ====================================================================
#
# Sources all assertion modules to provide a complete assertion library
# for integration tests.
#
# Modules:
#   - basic.sh       - Core equality and string assertions
#   - http.sh        - HTTP request and response assertions
#   - json.sh        - JSON validation and field assertions
#   - file-system.sh - File and directory assertions
#   - utilities.sh   - Test utilities and requirements
#
# ====================================================================

# Get the directory of this script
ASSERTIONS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source all assertion modules in the correct order
source "$ASSERTIONS_DIR/basic.sh"
source "$ASSERTIONS_DIR/http.sh"
source "$ASSERTIONS_DIR/json.sh"
source "$ASSERTIONS_DIR/file-system.sh"
source "$ASSERTIONS_DIR/utilities.sh"

# Export assertion counters so they're available globally
export TEST_ASSERTIONS
export FAILED_ASSERTIONS
export PASSED_ASSERTIONS