#!/bin/bash
# Step-CA Unit Tests - Library function validation

set -euo pipefail

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Source libraries
source "${RESOURCE_DIR}/lib/test.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Run unit tests
echo "🧪 Step-CA Unit Tests"
test_unit