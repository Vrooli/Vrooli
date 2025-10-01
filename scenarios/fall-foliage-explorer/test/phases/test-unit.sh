#!/usr/bin/env bash
# Unit tests for Fall Foliage Explorer
# Tests individual components and functions

set -e

echo "🧪 Running unit tests..."

# Test 1: API compilation
echo "  [1/4] Testing Go API compilation..."
cd api
if go build -o test-build . &>/dev/null; then
    rm test-build
    echo "    ✅ API compiles successfully"
else
    echo "    ❌ API compilation failed"
    exit 1
fi
cd ..

# Test 2: UI dependencies
echo "  [2/4] Testing UI dependencies..."
if [[ -f "ui/package.json" ]] && [[ -d "ui/node_modules" ]]; then
    echo "    ✅ UI dependencies installed"
else
    echo "    ⚠️  UI dependencies may need installation"
fi

# Test 3: Database schema
echo "  [3/4] Testing database schema..."
if [[ -f "initialization/postgres/schema.sql" ]]; then
    # Check for required tables
    if grep -q "CREATE TABLE.*regions" initialization/postgres/schema.sql && \
       grep -q "CREATE TABLE.*foliage_observations" initialization/postgres/schema.sql; then
        echo "    ✅ Database schema includes required tables"
    else
        echo "    ❌ Database schema missing required tables"
        exit 1
    fi
else
    echo "    ❌ Database schema file not found"
    exit 1
fi

# Test 4: Configuration files
echo "  [4/4] Testing configuration files..."
if [[ -f ".vrooli/service.json" ]]; then
    echo "    ✅ Service configuration exists"
else
    echo "    ❌ Service configuration missing"
    exit 1
fi

echo "✅ All unit tests passed!"
