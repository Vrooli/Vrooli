#!/bin/bash
# Eclipse Ditto Unit Tests - Library function validation

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/twins.sh"

echo "Eclipse Ditto Unit Tests"
echo "========================"

# Track failures
failed=0

# Test 1: Configuration variables
echo -n "1. Configuration loading... "
if [[ -n "${DITTO_GATEWAY_PORT:-}" ]] && \
   [[ -n "${DITTO_USERNAME:-}" ]] && \
   [[ -n "${DITTO_PASSWORD:-}" ]] && \
   [[ -n "${DITTO_DOCKER_TAG:-}" ]]; then
    echo "✓"
else
    echo "✗ Configuration not properly loaded"
    failed=$((failed + 1))
fi

# Test 2: Function availability - core
echo -n "2. Core functions... "
if type -t cmd_help &>/dev/null && \
   type -t cmd_info &>/dev/null && \
   type -t cmd_manage &>/dev/null && \
   type -t cmd_status &>/dev/null; then
    echo "✓"
else
    echo "✗ Core functions missing"
    failed=$((failed + 1))
fi

# Test 3: Function availability - lifecycle
echo -n "3. Lifecycle functions... "
if type -t manage_install &>/dev/null && \
   type -t manage_start &>/dev/null && \
   type -t manage_stop &>/dev/null && \
   type -t manage_uninstall &>/dev/null; then
    echo "✓"
else
    echo "✗ Lifecycle functions missing"
    failed=$((failed + 1))
fi

# Test 4: Function availability - twins
echo -n "4. Twin management functions... "
if type -t twin_create &>/dev/null && \
   type -t twin_update &>/dev/null && \
   type -t twin_query &>/dev/null && \
   type -t twin_send_command &>/dev/null; then
    echo "✓"
else
    echo "✗ Twin functions missing"
    failed=$((failed + 1))
fi

# Test 5: Directory structure
echo -n "5. Directory structure... "
if [[ -d "${RESOURCE_DIR}/lib" ]] && \
   [[ -d "${RESOURCE_DIR}/config" ]] && \
   [[ -d "${RESOURCE_DIR}/test" ]] && \
   [[ -f "${RESOURCE_DIR}/cli.sh" ]]; then
    echo "✓"
else
    echo "✗ Invalid directory structure"
    failed=$((failed + 1))
fi

# Test 6: Port registry integration
echo -n "6. Port registry... "
REGISTRY_PORT=$(ports::get_resource_port 'eclipse-ditto' 2>/dev/null || echo "")
if [[ "${REGISTRY_PORT}" == "${DITTO_GATEWAY_PORT}" ]]; then
    echo "✓"
else
    echo "✗ Port registry mismatch"
    failed=$((failed + 1))
fi

# Test 7: Docker compose generation
echo -n "7. Docker compose generation... "
if type -t create_docker_compose &>/dev/null; then
    # Test function exists and can generate valid YAML
    temp_compose=$(mktemp)
    (
        DOCKER_COMPOSE_FILE="$temp_compose"
        create_docker_compose
    )
    if [[ -f "$temp_compose" ]] && grep -q "version:" "$temp_compose"; then
        echo "✓"
        rm -f "$temp_compose"
    else
        echo "✗ Invalid docker-compose generation"
        failed=$((failed + 1))
        rm -f "$temp_compose"
    fi
else
    echo "✗ Docker compose function missing"
    failed=$((failed + 1))
fi

# Test 8: Logging functions
echo -n "8. Logging functions... "
if type -t log_info &>/dev/null && \
   type -t log_error &>/dev/null && \
   type -t log_debug &>/dev/null; then
    # Test they actually work
    log_info "Test" &>/dev/null
    log_error "Test" &>/dev/null
    echo "✓"
else
    echo "✗ Logging functions missing"
    failed=$((failed + 1))
fi

# Report results
echo ""
if [[ $failed -gt 0 ]]; then
    echo "❌ Unit tests failed: $failed test(s) failed"
    exit 1
else
    echo "✅ All unit tests passed"
fi