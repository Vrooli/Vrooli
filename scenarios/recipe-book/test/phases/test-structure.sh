#!/bin/bash
set -euo pipefail

# Recipe Book Structure and Dependencies Test Runner

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Starting Recipe Book structure and dependencies tests"

# Test 1: Verify required files exist
testing::phase::log "Verifying project structure..."

REQUIRED_FILES=(
    "api/main.go"
    "ui/app.js"
    ".vrooli/service.json"
    "test/phases/test-unit.sh"
    "test/phases/test-integration.sh"
    "test/phases/test-performance.sh"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        testing::phase::log "✓ Found: $file"
    else
        testing::phase::log "✗ Missing: $file"
    fi
done

# Test 2: Verify API structure
testing::phase::log "Verifying API structure..."

API_FILES=(
    "api/main.go"
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
    "api/comprehensive_test.go"
    "api/coverage_test.go"
)

for file in "${API_FILES[@]}"; do
    if [ -f "$file" ]; then
        testing::phase::log "✓ Found API file: $file"
    else
        testing::phase::log "⚠ Missing API file: $file"
    fi
done

# Test 3: Verify Go dependencies
testing::phase::log "Checking Go dependencies..."

cd api

if [ -f "go.mod" ]; then
    testing::phase::log "✓ go.mod exists"

    # Check for required dependencies
    REQUIRED_DEPS=(
        "github.com/gorilla/mux"
        "github.com/lib/pq"
        "github.com/google/uuid"
    )

    for dep in "${REQUIRED_DEPS[@]}"; do
        if grep -q "$dep" go.mod; then
            testing::phase::log "✓ Dependency: $dep"
        else
            testing::phase::log "⚠ Missing dependency: $dep"
        fi
    done
else
    testing::phase::log "✗ go.mod not found"
fi

# Test 4: Verify Go code compiles
testing::phase::log "Verifying Go code compiles..."

if go build -o /tmp/recipe-book-test . 2>&1 | grep -v "warning"; then
    testing::phase::log "✓ Go code compiles successfully"
    rm -f /tmp/recipe-book-test
else
    testing::phase::log "✗ Go code has compilation errors"
fi

cd ..

# Test 5: Verify test infrastructure
testing::phase::log "Verifying test infrastructure..."

TEST_FILES=(
    "api/test_helpers.go"
    "api/test_patterns.go"
)

for file in "${TEST_FILES[@]}"; do
    if [ -f "$file" ]; then
        # Check for key test helper functions
        if grep -q "setupTestLogger" "$file" && \
           grep -q "setupTestEnvironment" "$file" && \
           grep -q "makeHTTPRequest" "$file"; then
            testing::phase::log "✓ Test helpers complete in $file"
        else
            testing::phase::log "⚠ Test helpers incomplete in $file"
        fi
    fi
done

# Test 6: Verify service configuration
testing::phase::log "Verifying service configuration..."

if [ -f ".vrooli/service.json" ]; then
    testing::phase::log "✓ service.json exists"

    # Validate JSON
    if command -v jq &> /dev/null; then
        if jq empty < .vrooli/service.json 2>/dev/null; then
            testing::phase::log "✓ service.json is valid JSON"

            # Check for required fields
            if jq -e '.name' < .vrooli/service.json > /dev/null 2>&1; then
                testing::phase::log "✓ service.json has name field"
            fi

            if jq -e '.dependencies' < .vrooli/service.json > /dev/null 2>&1; then
                DEPS=$(jq -r '.dependencies[] // empty' < .vrooli/service.json)
                if [ -n "$DEPS" ]; then
                    testing::phase::log "✓ service.json declares dependencies"
                fi
            fi
        else
            testing::phase::log "✗ service.json is invalid JSON"
        fi
    else
        testing::phase::log "⚠ jq not available, skipping JSON validation"
    fi
else
    testing::phase::log "✗ service.json not found"
fi

# Test 7: Verify API endpoints are defined
testing::phase::log "Verifying API endpoints..."

if [ -f "api/main.go" ]; then
    ENDPOINTS=(
        "HandleFunc.*health"
        "HandleFunc.*recipes.*GET"
        "HandleFunc.*recipes.*POST"
        "HandleFunc.*recipes.*PUT"
        "HandleFunc.*recipes.*DELETE"
        "HandleFunc.*search"
        "HandleFunc.*generate"
        "HandleFunc.*shopping-list"
    )

    for endpoint in "${ENDPOINTS[@]}"; do
        if grep -q "$endpoint" api/main.go; then
            testing::phase::log "✓ Endpoint defined: $endpoint"
        else
            testing::phase::log "⚠ Endpoint not found: $endpoint"
        fi
    done
fi

# Test 8: Verify database schema setup
testing::phase::log "Verifying database schema..."

if [ -d "initialization/postgres" ]; then
    testing::phase::log "✓ Database initialization directory exists"

    if [ -f "initialization/postgres/schema.sql" ]; then
        testing::phase::log "✓ schema.sql exists"

        # Check for required tables
        TABLES=(
            "CREATE TABLE.*recipes"
            "CREATE TABLE.*recipe_ratings"
        )

        for table in "${TABLES[@]}"; do
            if grep -q "$table" initialization/postgres/schema.sql; then
                testing::phase::log "✓ Table definition: $table"
            else
                testing::phase::log "⚠ Missing table: $table"
            fi
        done
    else
        testing::phase::log "⚠ schema.sql not found"
    fi
else
    testing::phase::log "⚠ initialization/postgres directory not found"
fi

# Test 9: Verify UI files
testing::phase::log "Verifying UI files..."

if [ -f "ui/app.js" ]; then
    testing::phase::log "✓ UI JavaScript file exists"
fi

if [ -f "ui/index.html" ]; then
    testing::phase::log "✓ UI HTML file exists"
else
    testing::phase::log "⚠ UI HTML file not found"
fi

# Test 10: Verify test coverage tools
testing::phase::log "Verifying test coverage tools..."

cd api

if go test -tags testing -coverprofile=/tmp/recipe-coverage-check.out . > /dev/null 2>&1; then
    COVERAGE=$(go tool cover -func=/tmp/recipe-coverage-check.out 2>/dev/null | tail -1 | awk '{print $NF}' | sed 's/%//')
    testing::phase::log "✓ Coverage tools working (current: ${COVERAGE}%)"
    rm -f /tmp/recipe-coverage-check.out
else
    testing::phase::log "⚠ Coverage tools not working properly"
fi

cd ..

testing::phase::end_with_summary "Recipe Book structure and dependencies tests completed"
