#!/usr/bin/env bash
################################################################################
# SQLite Resource - Unit Test Phase
#
# Test individual library functions
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Run unit test via test runner
"${RESOURCE_DIR}/test/run-tests.sh" unit