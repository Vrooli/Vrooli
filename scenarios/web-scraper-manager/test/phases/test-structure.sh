#!/bin/bash
# Structure and standards tests for Web Scraper Manager

set -e

echo "Running structure tests for Web Scraper Manager..."

# Test required files exist
echo "✓ Testing required files..."
REQUIRED_FILES=(
    "PRD.md"
    "README.md"
    "Makefile"
    ".vrooli/service.json"
    "api/main.go"
    "cli/web-scraper-manager"
    "ui/server.js"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ Required file missing: $file"
        exit 1
    fi
done
echo "✓ All required files present"

# Test directory structure
echo "✓ Testing directory structure..."
REQUIRED_DIRS=(
    "api"
    "cli"
    "ui"
    "test"
    "initialization/storage/postgres"
    "scripts/lib"
)

for dir in "${REQUIRED_DIRS[@]}"; do
    if [ ! -d "$dir" ]; then
        echo "❌ Required directory missing: $dir"
        exit 1
    fi
done
echo "✓ All required directories present"

# Test API binary exists
echo "✓ Testing API binary..."
if [ ! -f "api/web-scraper-manager-api" ]; then
    echo "❌ API binary not built: api/web-scraper-manager-api"
    exit 1
fi
echo "✓ API binary present"

# Test Go code compiles
echo "✓ Testing Go compilation..."
cd api
if ! go build -o test-build . > /dev/null 2>&1; then
    echo "❌ Go code does not compile"
    exit 1
fi
rm -f test-build
cd ..
echo "✓ Go code compiles successfully"

# Test CLI is executable
echo "✓ Testing CLI executable..."
if [ ! -x "cli/web-scraper-manager" ]; then
    echo "❌ CLI is not executable"
    exit 1
fi
echo "✓ CLI is executable"

# Test service.json validity
echo "✓ Testing service.json validity..."
if ! jq empty .vrooli/service.json 2>/dev/null; then
    echo "❌ service.json is not valid JSON"
    exit 1
fi
echo "✓ service.json is valid"

echo "✅ Structure tests passed"
