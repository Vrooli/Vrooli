#!/bin/bash
# Traccar Unit Tests - Library function validation (60s max)

set -euo pipefail

# Get directories
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "${TEST_DIR}")")"
LIB_DIR="${RESOURCE_DIR}/lib"

echo "Running Traccar unit tests..."

# Test 1: Verify library files exist
echo -n "1. Checking library files... "
MISSING_FILES=()
for file in core.sh test.sh content.sh device.sh track.sh; do
    if [[ ! -f "${LIB_DIR}/${file}" ]]; then
        MISSING_FILES+=("${file}")
    fi
done

if [[ ${#MISSING_FILES[@]} -eq 0 ]]; then
    echo "✓"
else
    echo "✗"
    echo "   Missing files: ${MISSING_FILES[*]}"
    exit 1
fi

# Test 2: Verify configuration files
echo -n "2. Checking configuration files... "
CONFIG_DIR="${RESOURCE_DIR}/config"
MISSING_CONFIGS=()
for file in defaults.sh runtime.json schema.json; do
    if [[ ! -f "${CONFIG_DIR}/${file}" ]]; then
        MISSING_CONFIGS+=("${file}")
    fi
done

if [[ ${#MISSING_CONFIGS[@]} -eq 0 ]]; then
    echo "✓"
else
    echo "✗"
    echo "   Missing configs: ${MISSING_CONFIGS[*]}"
    exit 1
fi

# Test 3: Validate runtime.json structure
echo -n "3. Validating runtime.json... "
RUNTIME_FILE="${CONFIG_DIR}/runtime.json"
if [[ -f "$RUNTIME_FILE" ]]; then
    # Check required fields
    REQUIRED_FIELDS=("startup_order" "dependencies" "startup_timeout" "startup_time_estimate" "recovery_attempts" "priority")
    MISSING_FIELDS=()
    
    for field in "${REQUIRED_FIELDS[@]}"; do
        if ! jq -e ".${field}" "$RUNTIME_FILE" &>/dev/null; then
            MISSING_FIELDS+=("${field}")
        fi
    done
    
    if [[ ${#MISSING_FIELDS[@]} -eq 0 ]]; then
        echo "✓"
    else
        echo "✗"
        echo "   Missing fields: ${MISSING_FIELDS[*]}"
        exit 1
    fi
else
    echo "✗"
    echo "   runtime.json not found"
    exit 1
fi

# Test 4: Validate schema.json
echo -n "4. Validating schema.json... "
SCHEMA_FILE="${CONFIG_DIR}/schema.json"
if [[ -f "$SCHEMA_FILE" ]]; then
    if jq -e '."$schema"' "$SCHEMA_FILE" &>/dev/null; then
        echo "✓"
    else
        echo "✗"
        echo "   Invalid JSON schema"
        exit 1
    fi
else
    echo "✗"
    echo "   schema.json not found"
    exit 1
fi

# Test 5: Test CLI script syntax
echo -n "5. Checking CLI script syntax... "
CLI_SCRIPT="${RESOURCE_DIR}/cli.sh"
if [[ -f "$CLI_SCRIPT" ]]; then
    if bash -n "$CLI_SCRIPT" 2>/dev/null; then
        echo "✓"
    else
        echo "✗"
        echo "   Syntax errors in cli.sh"
        exit 1
    fi
else
    echo "✗"
    echo "   cli.sh not found"
    exit 1
fi

# Test 6: Test library syntax
echo -n "6. Checking library syntax... "
SYNTAX_ERRORS=()
for file in "${LIB_DIR}"/*.sh; do
    if [[ -f "$file" ]]; then
        if ! bash -n "$file" 2>/dev/null; then
            SYNTAX_ERRORS+=("$(basename "$file")")
        fi
    fi
done

if [[ ${#SYNTAX_ERRORS[@]} -eq 0 ]]; then
    echo "✓"
else
    echo "✗"
    echo "   Syntax errors in: ${SYNTAX_ERRORS[*]}"
    exit 1
fi

# Test 7: Check environment variables
echo -n "7. Checking environment variables... "
source "${CONFIG_DIR}/defaults.sh"
REQUIRED_VARS=("TRACCAR_PORT" "TRACCAR_HOST" "TRACCAR_VERSION" "TRACCAR_CONTAINER_NAME")
MISSING_VARS=()

for var in "${REQUIRED_VARS[@]}"; do
    if [[ -z "${!var:-}" ]]; then
        MISSING_VARS+=("${var}")
    fi
done

if [[ ${#MISSING_VARS[@]} -eq 0 ]]; then
    echo "✓"
else
    echo "✗"
    echo "   Missing variables: ${MISSING_VARS[*]}"
    exit 1
fi

# Test 8: Verify Docker image name
echo -n "8. Validating Docker configuration... "
if [[ -n "${TRACCAR_DOCKER_IMAGE:-}" ]]; then
    if [[ "$TRACCAR_DOCKER_IMAGE" =~ ^traccar/traccar: ]]; then
        echo "✓"
    else
        echo "✗"
        echo "   Invalid Docker image: $TRACCAR_DOCKER_IMAGE"
        exit 1
    fi
else
    echo "✗"
    echo "   TRACCAR_DOCKER_IMAGE not defined"
    exit 1
fi

echo ""
echo "Unit tests completed successfully!"
exit 0