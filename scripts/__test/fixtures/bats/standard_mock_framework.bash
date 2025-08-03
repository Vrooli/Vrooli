#!/usr/bin/env bash
# Legacy Standard Mock Framework - DEPRECATED
# This file is deprecated. Please use core/common_setup.bash directly in new tests.
#
# This file now just redirects to the new unified testing infrastructure
# for backward compatibility with existing tests.

echo "[DEPRECATED] standard_mock_framework.bash is deprecated. Use core/common_setup.bash instead." >&2

# Redirect to the new unified infrastructure
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core/common_setup.bash"

# Provide legacy aliases for backward compatibility
alias setup_standard_mock_framework='setup_standard_mocks'
alias cleanup_standard_mock_framework='cleanup_mocks'

# Export legacy functions
export -f setup_standard_mocks cleanup_mocks