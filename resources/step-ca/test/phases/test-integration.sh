#!/bin/bash
# Step-CA Integration Tests - Full functionality validation

set -euo pipefail

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Source libraries
source "${RESOURCE_DIR}/lib/test.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Run integration tests
echo "🧪 Step-CA Integration Tests"
test_integration