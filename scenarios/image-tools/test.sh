#!/bin/bash
# Image Tools Test Runner
# Runs comprehensive tests for the image-tools scenario

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
cd "$SCRIPT_DIR"

echo "🧪 Running Image Tools Test Suite"
echo "================================="

# Phase 1: Structure validation
echo ""
echo "📁 Phase 1: Structure Validation"
./test/phases/test-structure.sh

# Phase 2: Dependency checks
echo ""
echo "📦 Phase 2: Dependency Checks"
./test/phases/test-dependencies.sh

# Phase 3: Unit tests
echo ""
echo "🔬 Phase 3: Unit Tests"
./test/phases/test-unit.sh

# Phase 4: Integration tests
echo ""
echo "🔗 Phase 4: Integration Tests"
./test/phases/test-integration.sh

# Phase 5: API tests
echo ""
echo "🌐 Phase 5: API Tests"
./test/phases/test-api.sh

# Phase 6: CLI tests
echo ""
echo "⌨️  Phase 6: CLI Tests"
./test/phases/test-cli.sh

# Phase 7: Performance tests
echo ""
echo "⚡ Phase 7: Performance Tests"
./test/phases/test-performance.sh

echo ""
echo "✅ All tests passed!"