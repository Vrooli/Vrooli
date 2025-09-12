#!/usr/bin/env bash
# Strapi Integration Test Phase
# End-to-end functionality testing (<120s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

# Run integration tests
test::run_integration