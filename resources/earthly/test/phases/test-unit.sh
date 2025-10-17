#!/bin/bash
# Earthly Resource - Unit Test Phase
# Library function validation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

# Execute unit tests
test_unit "$@"