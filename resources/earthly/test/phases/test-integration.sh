#!/bin/bash
# Earthly Resource - Integration Test Phase
# End-to-end functionality testing

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

# Execute integration tests
test_integration "$@"