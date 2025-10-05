#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running structure tests for token-economy..."

# Check required files exist
echo "Checking required files..."

required_files=(
    "api/main.go"
    "api/go.mod"
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✓ $file exists"
    else
        echo "✗ Missing required file: $file"
        testing::phase::fail "Missing required file: $file"
    fi
done

# Check API structure
echo "Checking API structure..."

if [ -d "api" ]; then
    # Check for main.go
    if [ -f "api/main.go" ]; then
        echo "✓ API main.go exists"

        # Check for package main
        if grep -q "^package main" api/main.go; then
            echo "✓ main.go has package main"
        else
            echo "✗ main.go missing package main declaration"
            testing::phase::fail "Invalid main.go package declaration"
        fi

        # Check for main function
        if grep -q "func main()" api/main.go; then
            echo "✓ main() function exists"
        else
            echo "✗ main() function not found"
            testing::phase::fail "Missing main() function"
        fi
    fi

    # Check for test files
    test_files=(
        "api/main_test.go"
        "api/test_helpers.go"
        "api/test_patterns.go"
        "api/integration_test.go"
        "api/business_test.go"
        "api/performance_comprehensive_test.go"
    )

    for test_file in "${test_files[@]}"; do
        if [ -f "$test_file" ]; then
            echo "✓ $test_file exists"
        else
            echo "⚠ Optional test file missing: $test_file"
        fi
    done
fi

# Check test phase structure
echo "Checking test phase structure..."

test_phases=(
    "test/phases/test-unit.sh"
    "test/phases/test-integration.sh"
    "test/phases/test-performance.sh"
    "test/phases/test-dependencies.sh"
    "test/phases/test-structure.sh"
)

for phase in "${test_phases[@]}"; do
    if [ -f "$phase" ]; then
        echo "✓ $phase exists"

        # Check if executable
        if [ -x "$phase" ]; then
            echo "✓ $phase is executable"
        else
            echo "⚠ $phase is not executable, fixing..."
            chmod +x "$phase"
        fi

        # Check for proper shebang
        if head -n 1 "$phase" | grep -q "^#!/bin/bash"; then
            echo "✓ $phase has proper shebang"
        else
            echo "✗ $phase missing proper shebang"
        fi
    else
        echo "⚠ Test phase missing: $phase"
    fi
done

# Check database initialization structure
echo "Checking database initialization..."

if [ -d "initialization/storage/postgres" ]; then
    echo "✓ PostgreSQL initialization directory exists"

    if [ -f "initialization/storage/postgres/schema.sql" ]; then
        echo "✓ Database schema file exists"

        # Check schema file size
        schema_size=$(wc -l < "initialization/storage/postgres/schema.sql")
        if [ "$schema_size" -gt 10 ]; then
            echo "✓ Schema file has content ($schema_size lines)"
        else
            echo "✗ Schema file appears empty or incomplete"
        fi
    else
        echo "✗ Database schema file missing"
    fi
fi

# Check service.json structure
echo "Checking service.json structure..."

if [ -f ".vrooli/service.json" ]; then
    if command -v jq &> /dev/null; then
        # Validate JSON syntax
        if jq empty .vrooli/service.json 2>/dev/null; then
            echo "✓ service.json is valid JSON"

            # Check for required fields
            required_fields=("name" "version" "lifecycle")
            for field in "${required_fields[@]}"; do
                if jq -e ".${field}" .vrooli/service.json &> /dev/null; then
                    echo "✓ service.json has '${field}' field"
                else
                    echo "⚠ service.json missing '${field}' field"
                fi
            done
        else
            echo "✗ service.json has invalid JSON"
            testing::phase::fail "Invalid service.json"
        fi
    else
        echo "⚠ jq not available, skipping JSON validation"
    fi
fi

# Check code organization
echo "Checking code organization..."

if [ -f "api/main.go" ]; then
    # Check for proper imports
    if grep -q "^import" api/main.go; then
        echo "✓ main.go has imports"
    fi

    # Check for handler functions
    handler_count=$(grep -c "func.*Handler" api/main.go)
    if [ "$handler_count" -gt 5 ]; then
        echo "✓ Found $handler_count handler functions"
    else
        echo "⚠ Only found $handler_count handler functions"
    fi

    # Check for proper error handling
    if grep -q "if err != nil" api/main.go; then
        echo "✓ Code includes error handling"
    else
        echo "⚠ Limited error handling found"
    fi
fi

# Check documentation structure
echo "Checking documentation..."

if [ -f "PRD.md" ]; then
    prd_size=$(wc -l < "PRD.md")
    if [ "$prd_size" -gt 20 ]; then
        echo "✓ PRD.md has substantial content ($prd_size lines)"
    else
        echo "⚠ PRD.md appears minimal"
    fi
fi

if [ -f "README.md" ]; then
    readme_size=$(wc -l < "README.md")
    if [ "$readme_size" -gt 10 ]; then
        echo "✓ README.md has content ($readme_size lines)"
    else
        echo "⚠ README.md appears minimal"
    fi
fi

# Check for API endpoints documentation
echo "Checking API endpoint definitions..."

if grep -q "HandleFunc" api/main.go; then
    endpoint_count=$(grep -c "HandleFunc" api/main.go)
    echo "✓ Found $endpoint_count API endpoints"

    # List endpoints
    echo "API Endpoints:"
    grep "HandleFunc" api/main.go | sed 's/.*HandleFunc("\([^"]*\)".*/  - \1/' | sort | uniq
else
    echo "✗ No API endpoints found"
fi

# Check Makefile if it exists
if [ -f "Makefile" ]; then
    echo "✓ Makefile exists"

    # Check for standard targets
    standard_targets=("test" "start" "stop" "logs")
    for target in "${standard_targets[@]}"; do
        if grep -q "^${target}:" Makefile; then
            echo "✓ Makefile has '${target}' target"
        else
            echo "⚠ Makefile missing '${target}' target"
        fi
    done
fi

# Check CLI if it exists
if [ -d "cli" ]; then
    echo "✓ CLI directory exists"

    if [ -f "cli/$(basename $PWD)" ]; then
        echo "✓ CLI binary/script exists"
    else
        echo "⚠ CLI binary/script not found"
    fi
fi

# Summary
echo ""
echo "Structure validation complete!"
echo "Required files: $(echo "${required_files[@]}" | wc -w) checked"
echo "Test phases: $(echo "${test_phases[@]}" | wc -w) checked"

testing::phase::end_with_summary "Structure tests completed"
