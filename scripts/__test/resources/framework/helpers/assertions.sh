#!/bin/bash
# ====================================================================
# Test Assertion Functions - Module Loader
# ====================================================================
#
# This file now loads modular assertion functions from the assertions/
# subdirectory. The original monolithic file has been split into:
#
# - assertions/basic.sh       - Core assertions (equals, contains, empty)
# - assertions/http.sh        - HTTP-specific assertions
# - assertions/json.sh        - JSON validation assertions
# - assertions/file-system.sh - File and command assertions
# - assertions/utilities.sh   - Test utilities and requirements
#
# All original functionality is preserved while improving organization
# and maintainability.
#
# ====================================================================

# Get the directory of this script
HELPERS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source the modular assertions index
if [[ -f "$HELPERS_DIR/assertions/index.sh" ]]; then
    source "$HELPERS_DIR/assertions/index.sh"
else
    # Fallback error if modules are missing
    echo "ERROR: Assertion modules not found in $HELPERS_DIR/assertions/" >&2
    echo "The assertion library has been modularized. Please ensure all assertion modules are present." >&2
    exit 1
fi

# Backward compatibility notice
# This file maintains the same interface as before, so existing tests
# will continue to work without modification.