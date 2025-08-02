#!/usr/bin/env bash
# Legacy Common BATS Test Setup
# This file redirects to the new unified testing infrastructure for backward compatibility
#
# NOTE: This file is deprecated. Please use core/common_setup.bash directly in new tests.

# Redirect to the new unified infrastructure
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core/common_setup.bash"