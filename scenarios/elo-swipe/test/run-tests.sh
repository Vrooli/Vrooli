#!/bin/bash
# run-tests.sh - Main test runner for phased testing

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PHASES_DIR="$SCRIPT_DIR/phases"

echo "🧪 Elo Swipe Test Suite"
echo "======================="
echo ""

# Phase 1: Smoke Tests
if [ -f "$PHASES_DIR/test-smoke.sh" ]; then
    echo "Phase 1: Smoke Tests"
    bash "$PHASES_DIR/test-smoke.sh"
    echo ""
fi

# Phase 2: Unit Tests  
if [ -f "$PHASES_DIR/test-unit.sh" ]; then
    echo "Phase 2: Unit Tests"
    bash "$PHASES_DIR/test-unit.sh"
    echo ""
fi

# Phase 3: Integration Tests
if [ -f "$PHASES_DIR/test-integration.sh" ]; then
    echo "Phase 3: Integration Tests"
    bash "$PHASES_DIR/test-integration.sh"
    echo ""
fi

echo "═══════════════════════════════════"
echo "✅ All test phases completed!"
echo "═══════════════════════════════════"
