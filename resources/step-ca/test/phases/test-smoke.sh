#!/bin/bash
# Step-CA Smoke Tests - Quick health validation

set -euo pipefail

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Source libraries
source "${RESOURCE_DIR}/lib/test.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Run smoke tests
echo "🧪 Step-CA Smoke Tests"
test_smoke