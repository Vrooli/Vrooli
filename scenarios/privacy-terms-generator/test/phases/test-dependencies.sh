#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running dependency validation tests..."

# Test 1: Check Go dependencies
echo "Checking Go module dependencies..."
if [ -f "api/go.mod" ]; then
    cd api

    echo "  - Validating go.mod..."
    if go list -m all &> /dev/null; then
        echo "  ✓ Go modules are valid"
    else
        echo "  ✗ Go modules validation failed"
        exit 1
    fi

    echo "  - Checking for known vulnerabilities..."
    if command -v govulncheck &> /dev/null; then
        if govulncheck ./... &> /dev/null; then
            echo "  ✓ No known vulnerabilities found"
        else
            echo "  ⚠ Potential vulnerabilities detected (non-blocking)"
        fi
    else
        echo "  ⚠ govulncheck not installed (skipping vulnerability scan)"
    fi

    cd ..
else
    echo "  ✗ api/go.mod not found"
    exit 1
fi

# Test 2: Check for required resource connections
echo "Checking resource dependencies..."

# PostgreSQL
echo "  - PostgreSQL..."
if command -v resource-postgres &> /dev/null; then
    if resource-postgres status &> /dev/null; then
        echo "  ✓ PostgreSQL is available"
    else
        echo "  ⚠ PostgreSQL not running (required for full functionality)"
    fi
else
    echo "  ⚠ PostgreSQL resource command not found"
fi

# Ollama
echo "  - Ollama..."
if command -v resource-ollama &> /dev/null; then
    if resource-ollama status &> /dev/null; then
        echo "  ✓ Ollama is available"
    else
        echo "  ⚠ Ollama not running (required for custom clause generation)"
    fi
else
    echo "  ⚠ Ollama resource command not found"
fi

# Qdrant (optional)
echo "  - Qdrant (optional)..."
if command -v resource-qdrant &> /dev/null; then
    if resource-qdrant status &> /dev/null; then
        echo "  ✓ Qdrant is available"
    else
        echo "  ℹ Qdrant not running (optional, will use PostgreSQL fallback)"
    fi
else
    echo "  ℹ Qdrant resource command not found (optional)"
fi

# Browserless (optional)
echo "  - Browserless (optional)..."
if command -v resource-browserless &> /dev/null; then
    if resource-browserless status &> /dev/null; then
        echo "  ✓ Browserless is available"
    else
        echo "  ℹ Browserless not running (optional, PDF generation unavailable)"
    fi
else
    echo "  ℹ Browserless resource command not found (optional)"
fi

# Test 3: Check database schema
echo "Checking database schema..."
if command -v psql &> /dev/null && command -v resource-postgres &> /dev/null; then
    if resource-postgres status &> /dev/null; then
        echo "  - Checking for required tables..."

        # Check if legal_templates table exists
        if psql -U postgres -d vrooli_db -c "\dt legal_templates" &> /dev/null; then
            echo "  ✓ legal_templates table exists"
        else
            echo "  ⚠ legal_templates table not found (run setup to initialize)"
        fi

        # Check if document_history table exists
        if psql -U postgres -d vrooli_db -c "\dt document_history" &> /dev/null; then
            echo "  ✓ document_history table exists"
        else
            echo "  ⚠ document_history table not found"
        fi
    else
        echo "  ⚠ PostgreSQL not running (skipping table checks)"
    fi
else
    echo "  ⚠ psql not available (skipping database checks)"
fi

# Test 4: Check CLI installation
echo "Checking CLI installation..."
if [ -x "cli/privacy-terms-generator" ]; then
    echo "  ✓ CLI binary exists and is executable"

    # Check if CLI is in PATH (from install.sh)
    if command -v privacy-terms-generator &> /dev/null; then
        echo "  ✓ CLI is installed in PATH"
    else
        echo "  ℹ CLI not in PATH (run cli/install.sh to install globally)"
    fi
else
    echo "  ⚠ CLI binary not found or not executable"
fi

# Test 5: Check initialization scripts
echo "Checking initialization files..."
required_init_files=(
    "initialization/postgres/schema.sql"
    "initialization/postgres/seed-templates.sql"
)

for file in "${required_init_files[@]}"; do
    if [ -f "$file" ]; then
        echo "  ✓ $file exists"
    else
        echo "  ✗ $file not found"
        exit 1
    fi
done

# Test 6: Check library scripts
echo "Checking library scripts..."
if [ -f "lib/generator.sh" ]; then
    echo "  ✓ lib/generator.sh exists"

    # Check if it's executable
    if [ -x "lib/generator.sh" ]; then
        echo "  ✓ lib/generator.sh is executable"
    else
        echo "  ⚠ lib/generator.sh is not executable"
    fi
else
    echo "  ✗ lib/generator.sh not found"
    exit 1
fi

# Test 7: Check for template freshness script
echo "Checking additional library scripts..."
optional_lib_files=(
    "lib/fetch-templates.sh"
    "lib/web_updater.sh"
    "lib/pdf_export.sh"
    "lib/semantic_search.sh"
)

for file in "${optional_lib_files[@]}"; do
    if [ -f "$file" ]; then
        echo "  ✓ $file exists"
    else
        echo "  ℹ $file not found (optional)"
    fi
done

# Test 8: Validate configuration
echo "Checking service configuration..."
if [ -f ".vrooli/service.json" ]; then
    echo "  ✓ service.json exists"

    # Validate JSON syntax
    if command -v jq &> /dev/null; then
        if jq empty .vrooli/service.json 2> /dev/null; then
            echo "  ✓ service.json is valid JSON"

            # Check for required fields
            if jq -e '.service.name' .vrooli/service.json > /dev/null 2>&1; then
                echo "  ✓ service.name is defined"
            else
                echo "  ✗ service.name not found"
                exit 1
            fi
        else
            echo "  ✗ service.json is invalid JSON"
            exit 1
        fi
    else
        echo "  ⚠ jq not installed (skipping JSON validation)"
    fi
else
    echo "  ✗ .vrooli/service.json not found"
    exit 1
fi

testing::phase::end_with_summary "Dependency validation completed"
