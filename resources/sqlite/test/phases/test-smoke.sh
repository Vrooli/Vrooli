#!/usr/bin/env bash
################################################################################
# SQLite Resource - Smoke Test Phase
#
# Quick validation that SQLite is available and functional
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Run smoke test via test runner
"${RESOURCE_DIR}/test/run-tests.sh" smoke