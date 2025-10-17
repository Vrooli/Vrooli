#!/usr/bin/env bash
#
# Dependencies Test: Check required dependencies for PRD Control Tower
#

set -eo pipefail

echo "🔍 Testing PRD Control Tower dependencies..."

# Check Go
if ! command -v go &> /dev/null; then
    echo "  ✗ Go not installed"
    exit 1
fi
echo "  ✓ Go: $(go version)"

# Check Node.js
if ! command -v node &> /dev/null; then
    echo "  ✗ Node.js not installed"
    exit 1
fi
echo "  ✓ Node.js: $(node --version)"

# Check npm
if ! command -v npm &> /dev/null; then
    echo "  ✗ npm not installed"
    exit 1
fi
echo "  ✓ npm: $(npm --version)"

# Check PostgreSQL (optional - might be in container)
if command -v psql &> /dev/null; then
    echo "  ✓ PostgreSQL client: $(psql --version)"
else
    echo "  ℹ PostgreSQL client not locally installed (may be containerized)"
fi

# Check jq (required by CLI)
if ! command -v jq &> /dev/null; then
    echo "  ✗ jq not installed (required for CLI)"
    exit 1
fi
echo "  ✓ jq: $(jq --version)"

echo "✅ Dependencies test passed"
