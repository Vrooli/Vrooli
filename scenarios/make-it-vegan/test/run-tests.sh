#!/bin/bash
# Main test runner for Make It Vegan

set -e

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCENARIO_DIR"

echo "========================================="
echo "Running Make It Vegan Test Suite"
echo "========================================="

# Make test scripts executable
chmod +x test/phases/*.sh

# Run unit tests
echo ""
echo "Phase 1: Unit Tests"
echo "========================================="
./test/phases/test-unit.sh

# Run API tests
echo ""
echo "Phase 2: API Tests"
echo "========================================="
./test/phases/test-api.sh

# Run UI tests
echo ""
echo "Phase 3: UI Tests"
echo "========================================="
./test/phases/test-ui.sh

echo ""
echo "========================================="
echo "✅ All tests passed!"
echo "========================================="
