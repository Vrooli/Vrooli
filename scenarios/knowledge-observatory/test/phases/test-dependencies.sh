#!/usr/bin/env bash
#
# Dependencies Test Phase for knowledge-observatory
# Tests resource dependencies and build requirements
#

set -eo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

echo "📦 Running knowledge-observatory dependencies tests..."

# Test 1: Go module dependencies
echo "  Checking Go dependencies..."
if [ -f "api/go.mod" ]; then
    cd api
    if go mod verify >/dev/null 2>&1; then
        echo "  ✓ Go dependencies verified"
    else
        echo "  ✗ Go module verification failed"
        exit 1
    fi

    # Check for major outdated dependencies
    if go list -m -u all 2>/dev/null | grep -q '\['; then
        echo "    Some Go dependencies have updates available"
    fi

    cd ..
else
    echo "  ✗ No go.mod found in api/"
    exit 1
fi

# Test 2: Required resource dependencies
echo "  Checking required resource dependencies..."

required_resources=("qdrant" "postgres")
for resource in "${required_resources[@]}"; do
    if vrooli resource status "$resource" --json >/dev/null 2>&1; then
        status=$(vrooli resource status "$resource" --json 2>/dev/null | jq -r '.status // "unknown"')
        echo "    Resource $resource status: $status"

        if [ "$status" = "running" ] || [ "$status" = "healthy" ]; then
            echo "  ✓ Required resource $resource is available"
        else
            echo "  ⚠️  Required resource $resource status: $status"
        fi
    else
        echo "  ⚠️  Unable to check status of required resource: $resource"
    fi
done

# Test 3: Optional resource dependencies
echo "  Checking optional resource dependencies..."

optional_resources=("ollama" "n8n")
for resource in "${optional_resources[@]}"; do
    if vrooli resource status "$resource" --json >/dev/null 2>&1; then
        status=$(vrooli resource status "$resource" --json 2>/dev/null | jq -r '.status // "unknown"')
        echo "    Optional resource $resource status: $status"
    else
        echo "    Optional resource $resource not available (acceptable)"
    fi
done

# Test 4: Qdrant collections
echo "  Checking Qdrant collections..."
qdrant_url="http://localhost:6333"

if curl -sf "$qdrant_url/collections" >/dev/null 2>&1; then
    collections=$(curl -sf "$qdrant_url/collections" | jq -r '.result.collections | length' 2>/dev/null || echo "0")
    echo "    Found $collections Qdrant collections"

    if [ "$collections" -gt 0 ]; then
        echo "  ✓ Qdrant has collections available"
    else
        echo "  ⚠️  No Qdrant collections found"
    fi
else
    echo "  ⚠️  Unable to connect to Qdrant API"
fi

# Test 5: PostgreSQL connectivity
echo "  Checking PostgreSQL connectivity..."

# Try to connect using environment variables or defaults
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-vrooli}"

if command -v psql >/dev/null 2>&1; then
    # Use environment variable or skip test if not configured
    if [ -n "${POSTGRES_PASSWORD}" ]; then
        export PGPASSWORD="${POSTGRES_PASSWORD}"
        if psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "${POSTGRES_USER:-postgres}" -d "$POSTGRES_DB" -c "SELECT 1;" >/dev/null 2>&1; then
            echo "  ✓ PostgreSQL connection successful"
        else
            echo "  ⚠️  Unable to connect to PostgreSQL (may be auth config)"
        fi
        unset PGPASSWORD
    else
        echo "    POSTGRES_PASSWORD not set, skipping direct database test"
    fi
else
    echo "    psql not available, skipping direct database test"
fi

# Test 6: Makefile targets
echo "  Checking Makefile targets..."

if [ -f "Makefile" ]; then
    required_targets=("run" "test" "stop" "status")
    missing_targets=()

    for target in "${required_targets[@]}"; do
        if grep -q "^${target}:" Makefile; then
            : # Target exists
        else
            missing_targets+=("$target")
        fi
    done

    if [ ${#missing_targets[@]} -gt 0 ]; then
        echo "  ⚠️  Missing Makefile targets: ${missing_targets[*]}"
    else
        echo "  ✓ All required Makefile targets present"
    fi
else
    echo "  ✗ No Makefile found"
    exit 1
fi

# Test 7: CLI binary
echo "  Checking CLI binary..."

if [ -f "cli/knowledge-observatory" ]; then
    if [ -x "cli/knowledge-observatory" ]; then
        echo "  ✓ CLI binary exists and is executable"

        # Test basic CLI functionality
        if ./cli/knowledge-observatory --version >/dev/null 2>&1; then
            echo "    CLI --version works"
        else
            echo "  ⚠️  CLI --version failed"
        fi
    else
        echo "  ✗ CLI binary exists but is not executable"
        exit 1
    fi
else
    echo "  ✗ CLI binary not found at cli/knowledge-observatory"
    exit 1
fi

# Test 8: service.json configuration
echo "  Checking service.json configuration..."

if [ -f ".vrooli/service.json" ]; then
    echo "  ✓ service.json exists"

    # Validate JSON structure
    if jq empty .vrooli/service.json 2>/dev/null; then
        echo "    service.json has valid JSON"

        # Check for required fields
        required_fields=("name" "version" "description" "lifecycle")
        missing_fields=()

        for field in "${required_fields[@]}"; do
            if ! jq -e ".$field" .vrooli/service.json >/dev/null 2>&1; then
                missing_fields+=("$field")
            fi
        done

        if [ ${#missing_fields[@]} -gt 0 ]; then
            echo "  ⚠️  service.json missing fields: ${missing_fields[*]}"
        else
            echo "  ✓ service.json has all required fields"
        fi

        # Check resource dependencies
        if jq -e '.resources.required[]' .vrooli/service.json >/dev/null 2>&1; then
            resource_count=$(jq '.resources.required | length' .vrooli/service.json)
            echo "    service.json declares $resource_count required resources"
        fi
    else
        echo "  ✗ service.json has invalid JSON"
        exit 1
    fi
else
    echo "  ✗ service.json not found at .vrooli/service.json"
    exit 1
fi

# Test 9: Build capability
echo "  Testing build capability..."

if [ -f "api/main.go" ]; then
    cd api
    if go build -o /tmp/knowledge-observatory-test-build ./... >/dev/null 2>&1; then
        echo "  ✓ Go code compiles successfully"
        rm -f /tmp/knowledge-observatory-test-build
    else
        echo "  ✗ Go build failed"
        exit 1
    fi
    cd ..
else
    echo "  ✗ No main.go found in api/"
    exit 1
fi

echo "✅ All dependencies tests passed"
