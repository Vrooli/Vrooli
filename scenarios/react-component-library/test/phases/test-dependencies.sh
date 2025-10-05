#!/bin/bash
# Dependency testing phase for react-component-library scenario

set -e

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🔍 Testing React Component Library dependencies..."
echo "=================================================="

# Test Go dependencies
echo ""
echo "📦 Checking Go dependencies..."
cd api

if ! go mod verify; then
    echo "❌ Go module verification failed"
    exit 1
fi

if ! go mod tidy -diff; then
    echo "⚠️  WARNING: go.mod needs tidying"
fi

echo "✅ Go dependencies verified"

# Test Go build
echo ""
echo "🔨 Testing Go build..."
if ! go build -o /tmp/react-component-library-api-test .; then
    echo "❌ Go build failed"
    exit 1
fi
rm -f /tmp/react-component-library-api-test
echo "✅ Go build successful"

# Test Node.js dependencies (UI)
echo ""
echo "📦 Checking Node.js dependencies..."
cd ../ui

if [ -f "package.json" ]; then
    if [ ! -d "node_modules" ]; then
        echo "⚠️  WARNING: node_modules not found, skipping detailed check"
    else
        # Check for known vulnerabilities
        if command -v npm &> /dev/null; then
            echo "🔒 Checking for security vulnerabilities..."
            npm audit --audit-level=moderate || echo "⚠️  Found security advisories"
        fi
        echo "✅ Node.js dependencies checked"
    fi
else
    echo "ℹ️  No package.json found, skipping Node.js checks"
fi

cd ..

# Test required binaries
echo ""
echo "🔧 Checking required binaries..."

required_binaries=(
    "go"
    "psql"
)

missing_binaries=()

for binary in "${required_binaries[@]}"; do
    if ! command -v "$binary" &> /dev/null; then
        missing_binaries+=("$binary")
        echo "❌ Missing: $binary"
    else
        version=$($binary --version 2>&1 | head -n1)
        echo "✅ Found: $binary ($version)"
    fi
done

if [ ${#missing_binaries[@]} -gt 0 ]; then
    echo ""
    echo "⚠️  WARNING: Missing required binaries: ${missing_binaries[*]}"
    echo "   Tests may fail without these dependencies"
fi

# Test resource CLI availability
echo ""
echo "📡 Checking resource CLI availability..."

resource_clis=(
    "resource-postgres"
    "resource-qdrant"
    "resource-minio"
)

available_resources=()
unavailable_resources=()

for cli in "${resource_clis[@]}"; do
    if command -v "$cli" &> /dev/null; then
        available_resources+=("$cli")
        echo "✅ Available: $cli"
    else
        unavailable_resources+=("$cli")
        echo "⚠️  Not available: $cli"
    fi
done

if [ ${#unavailable_resources[@]} -gt 0 ]; then
    echo ""
    echo "ℹ️  Some resource CLIs are not available"
    echo "   This is expected if resources are not running"
fi

echo ""
echo "✅ Dependency tests completed"
echo ""
echo "Summary:"
echo "  - Go dependencies: verified"
echo "  - Required binaries: ${#required_binaries[@]} checked, ${#missing_binaries[@]} missing"
echo "  - Resource CLIs: ${#available_resources[@]}/${#resource_clis[@]} available"

testing::phase::end_with_summary "Dependency tests completed"
