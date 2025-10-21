#!/usr/bin/env bash
# Test Phase 2: Dependencies Validation
# Validates that all required dependencies are available and properly configured

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "================================================"
echo "🔗 Phase 2: Dependencies Validation"
echo "================================================"

TESTS_PASSED=0
TESTS_FAILED=0

# Helper function
check_command() {
    local cmd="$1"
    local description="$2"

    if command -v "$cmd" &> /dev/null; then
        local version=$(eval "$cmd --version 2>&1 | head -1" || echo "unknown")
        echo "  ✅ $description: $cmd ($version)"
        ((TESTS_PASSED++))
    else
        echo "  ❌ $description: $cmd (not found)"
        ((TESTS_FAILED++))
    fi
}

# System Dependencies
echo ""
echo "🛠️ System Dependencies:"
check_command "go" "Go compiler"
check_command "node" "Node.js runtime"
check_command "curl" "HTTP client"
check_command "jq" "JSON processor"

# Optional Tools
echo ""
echo "🔧 Optional Tools:"
if command -v bats &> /dev/null; then
    echo "  ✅ BATS testing framework available"
    ((TESTS_PASSED++))
else
    echo "  ⚠️  BATS testing framework not found (optional)"
fi

# Go Module Dependencies
echo ""
echo "📦 Go Module Dependencies:"
if [ -f "$SCENARIO_DIR/api/go.mod" ]; then
    cd "$SCENARIO_DIR/api"
    if go mod verify &> /dev/null; then
        echo "  ✅ Go modules verified"
        ((TESTS_PASSED++))
    else
        echo "  ❌ Go modules verification failed"
        ((TESTS_FAILED++))
    fi

    # Check key dependencies
    if grep -q "github.com/gorilla/mux" go.mod; then
        echo "  ✅ gorilla/mux present"
        ((TESTS_PASSED++))
    else
        echo "  ❌ gorilla/mux missing"
        ((TESTS_FAILED++))
    fi

    if grep -q "github.com/lib/pq" go.mod; then
        echo "  ✅ lib/pq (PostgreSQL driver) present"
        ((TESTS_PASSED++))
    else
        echo "  ❌ lib/pq missing"
        ((TESTS_FAILED++))
    fi
else
    echo "  ❌ go.mod not found"
    ((TESTS_FAILED++))
fi

# Node Dependencies
echo ""
echo "📦 Node.js Dependencies:"
if [ -f "$SCENARIO_DIR/ui/package.json" ]; then
    cd "$SCENARIO_DIR/ui"
    if [ -d "node_modules" ]; then
        echo "  ✅ node_modules directory exists"
        ((TESTS_PASSED++))
    else
        echo "  ⚠️  node_modules not found (may need npm install)"
    fi
else
    echo "  ✅ UI uses vanilla HTML/CSS/JS (no package.json needed)"
    ((TESTS_PASSED++))
fi

# Resource Dependencies
echo ""
echo "🗄️ Resource Dependencies:"

# PostgreSQL
if vrooli resource status postgres --json &> /dev/null 2>&1; then
    POSTGRES_STATUS=$(vrooli resource status postgres --json 2>/dev/null | jq -r '.status' || echo "unknown")
    if [ "$POSTGRES_STATUS" = "running" ]; then
        echo "  ✅ PostgreSQL is running"
        ((TESTS_PASSED++))
    else
        echo "  ⚠️  PostgreSQL status: $POSTGRES_STATUS (required resource)"
        ((TESTS_FAILED++))
    fi
else
    echo "  ⚠️  Cannot check PostgreSQL status"
    ((TESTS_FAILED++))
fi

# Redis
if vrooli resource status redis --json &> /dev/null 2>&1; then
    REDIS_STATUS=$(vrooli resource status redis --json 2>/dev/null | jq -r '.status' || echo "unknown")
    if [ "$REDIS_STATUS" = "running" ]; then
        echo "  ✅ Redis is running"
        ((TESTS_PASSED++))
    else
        echo "  ⚠️  Redis status: $REDIS_STATUS (required resource)"
        ((TESTS_FAILED++))
    fi
else
    echo "  ⚠️  Cannot check Redis status"
    ((TESTS_FAILED++))
fi

# Summary
echo ""
echo "================================================"
echo "📊 Dependencies Validation Summary"
echo "================================================"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    echo "✅ All dependency validation tests passed!"
    exit 0
else
    echo ""
    echo "❌ Some dependency validation tests failed"
    exit 1
fi
