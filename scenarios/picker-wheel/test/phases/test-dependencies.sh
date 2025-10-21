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

# Test 2: Check Node.js dependencies
echo "Checking Node.js dependencies..."
if [ -f "ui/package.json" ]; then
    echo "  - Validating package.json..."
    if command -v node &> /dev/null; then
        echo "  ✓ Node.js is available"

        # Check if dependencies are installed
        if [ -d "ui/node_modules" ]; then
            echo "  ✓ node_modules directory exists"
        else
            echo "  ⚠ node_modules not found (run npm install)"
        fi
    else
        echo "  ✗ Node.js not found"
        exit 1
    fi
else
    echo "  ✗ ui/package.json not found"
    exit 1
fi

# Test 3: Check for required resource connections
echo "Checking resource dependencies..."

# PostgreSQL (optional but recommended)
echo "  - PostgreSQL (optional)..."
if command -v resource-postgres &> /dev/null; then
    if resource-postgres status &> /dev/null; then
        echo "  ✓ PostgreSQL is available"
    else
        echo "  ℹ PostgreSQL not running (will use in-memory storage)"
    fi
else
    echo "  ℹ PostgreSQL resource command not found (optional)"
fi

# N8n (optional)
echo "  - N8n (optional)..."
if command -v resource-n8n &> /dev/null; then
    if resource-n8n status &> /dev/null; then
        echo "  ✓ N8n is available"
    else
        echo "  ℹ N8n not running (optional for workflow automation)"
    fi
else
    echo "  ℹ N8n resource command not found (optional)"
fi

# Ollama (optional)
echo "  - Ollama (optional)..."
if command -v resource-ollama &> /dev/null; then
    if resource-ollama status &> /dev/null; then
        echo "  ✓ Ollama is available"
    else
        echo "  ℹ Ollama not running (optional for AI suggestions)"
    fi
else
    echo "  ℹ Ollama resource command not found (optional)"
fi

# Test 4: Check database schema (if PostgreSQL is available)
echo "Checking database schema..."
if command -v psql &> /dev/null && command -v resource-postgres &> /dev/null; then
    if resource-postgres status &> /dev/null; then
        echo "  - Checking for picker-wheel tables..."

        # Check if wheels table exists
        if psql -U postgres -d vrooli_db -c "\dt wheels" &> /dev/null; then
            echo "  ✓ wheels table exists"
        else
            echo "  ℹ wheels table not found (will be created on first run)"
        fi

        # Check if spin_history table exists
        if psql -U postgres -d vrooli_db -c "\dt spin_history" &> /dev/null; then
            echo "  ✓ spin_history table exists"
        else
            echo "  ℹ spin_history table not found (will be created on first run)"
        fi
    else
        echo "  ℹ PostgreSQL not running (skipping table checks)"
    fi
else
    echo "  ℹ psql not available (skipping database checks)"
fi

# Test 5: Check CLI installation
echo "Checking CLI installation..."
if [ -x "cli/picker-wheel" ]; then
    echo "  ✓ CLI binary exists and is executable"

    # Check if CLI is in PATH (from install.sh)
    if command -v picker-wheel &> /dev/null; then
        echo "  ✓ CLI is installed in PATH"
    else
        echo "  ℹ CLI not in PATH (run cli/install.sh to install globally)"
    fi
else
    echo "  ⚠ CLI binary not found or not executable"
fi

# Test 6: Check initialization scripts
echo "Checking initialization files..."
required_init_files=(
    "initialization/postgres/schema.sql"
)

for file in "${required_init_files[@]}"; do
    if [ -f "$file" ]; then
        echo "  ✓ $file exists"
    else
        echo "  ℹ $file not found (optional, tables created dynamically)"
    fi
done

# Test 7: Check N8n workflows (optional)
echo "Checking N8n workflow files..."
if [ -d "initialization/n8n" ]; then
    workflow_count=$(find initialization/n8n -name "*.json" | wc -l)
    echo "  ✓ Found $workflow_count N8n workflow files"
else
    echo "  ℹ N8n workflow directory not found (optional)"
fi

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
                name=$(jq -r '.service.name' .vrooli/service.json)
                if [ "$name" = "picker-wheel" ]; then
                    echo "  ✓ service.name is correct"
                else
                    echo "  ✗ service.name mismatch (expected: picker-wheel, got: $name)"
                    exit 1
                fi
            else
                echo "  ✗ service.name not found"
                exit 1
            fi

            # Check for ports configuration
            if jq -e '.ports.api' .vrooli/service.json > /dev/null 2>&1; then
                echo "  ✓ API port configuration exists"
            else
                echo "  ✗ API port configuration missing"
                exit 1
            fi

            if jq -e '.ports.ui' .vrooli/service.json > /dev/null 2>&1; then
                echo "  ✓ UI port configuration exists"
            else
                echo "  ✗ UI port configuration missing"
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

# Test 9: Check API binary
echo "Checking API binary..."
if [ -f "api/picker-wheel-api" ]; then
    echo "  ✓ API binary exists"

    if [ -x "api/picker-wheel-api" ]; then
        echo "  ✓ API binary is executable"
    else
        echo "  ⚠ API binary is not executable (run chmod +x)"
    fi
else
    echo "  ℹ API binary not found (will be built during setup)"
fi

# Test 10: Check system requirements
echo "Checking system requirements..."

# Check Go version
if command -v go &> /dev/null; then
    go_version=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' || echo "0.0")
    echo "  ✓ Go installed (version: $go_version)"

    # Minimum Go version 1.19
    if [ "$(echo "$go_version >= 1.19" | bc -l 2>/dev/null || echo "1")" = "1" ]; then
        echo "  ✓ Go version meets requirements (>= 1.19)"
    else
        echo "  ⚠ Go version may be too old (< 1.19)"
    fi
else
    echo "  ✗ Go not installed"
    exit 1
fi

# Check Node version
if command -v node &> /dev/null; then
    node_version=$(node --version | grep -oP 'v\K[0-9]+' || echo "0")
    echo "  ✓ Node.js installed (version: $(node --version))"

    # Minimum Node version 16
    if [ "$node_version" -ge 16 ]; then
        echo "  ✓ Node.js version meets requirements (>= 16)"
    else
        echo "  ⚠ Node.js version may be too old (< 16)"
    fi
else
    echo "  ✗ Node.js not installed"
    exit 1
fi

# Check curl availability
if command -v curl &> /dev/null; then
    echo "  ✓ curl is available"
else
    echo "  ✗ curl not found (required for CLI and testing)"
    exit 1
fi

testing::phase::end_with_summary "Dependency validation completed"
