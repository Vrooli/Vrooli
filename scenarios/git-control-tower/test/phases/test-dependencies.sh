#!/usr/bin/env bash
# Test Dependencies Phase - Check required dependencies

set -e

echo "Testing Git Control Tower dependencies..."

# Check Go
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed"
    exit 1
fi

# Check Git
if ! command -v git &> /dev/null; then
    echo "❌ Git is not installed"
    exit 1
fi

# Check jq (for CLI JSON parsing)
if ! command -v jq &> /dev/null; then
    echo "⚠️  Warning: jq not installed (required for CLI JSON output)"
fi

# Check PostgreSQL connection (optional)
if command -v psql &> /dev/null; then
    echo "✅ PostgreSQL client available"
else
    echo "⚠️  Warning: psql not available (database initialization will be manual)"
fi

echo "✅ Dependencies validation passed"
