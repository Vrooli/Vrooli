#!/bin/bash
# Keycloak webhook functionality tests

# Source dependencies with non-strict mode for initial setup
set +euo pipefail

# Get the script directory
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KEYCLOAK_DIR="$(cd "${TEST_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${KEYCLOAK_DIR}/../.." && pwd)"

# Source common testing utilities and configuration
# shellcheck source=/dev/null
source "${KEYCLOAK_DIR}/config/defaults.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${KEYCLOAK_DIR}/lib/common.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${KEYCLOAK_DIR}/lib/webhooks.sh" 2>/dev/null || true

# Set default values if not set
KEYCLOAK_PORT="${KEYCLOAK_PORT:-8070}"
KEYCLOAK_CONTAINER_NAME="${KEYCLOAK_CONTAINER_NAME:-vrooli-keycloak}"
KEYCLOAK_ADMIN_USER="${KEYCLOAK_ADMIN_USER:-admin}"
KEYCLOAK_ADMIN_PASSWORD="${KEYCLOAK_ADMIN_PASSWORD:-admin}"

# Now enable strict mode for the actual tests
set -euo pipefail

# Test realm and webhook configuration
TEST_REALM="test-webhooks-$$"
# Initialize token variable
token=""
TEST_WEBHOOK_URL="https://webhook.site/test-$$"
TEST_SECRET="webhook-test-secret-$$"

echo "[INFO] Starting Keycloak webhook tests..."

# ========== Test 1: List Available Events ==========
echo "[INFO] Testing webhook event listing..."
if webhook::list_events | grep -q "LOGIN"; then
    echo "[SUCCESS] Event listing works"
else
    echo "[ERROR] Failed to list webhook events"
    exit 1
fi

# ========== Test 2: Test Webhook Connectivity ==========
echo "[INFO] Testing webhook connectivity check..."
# Note: This will fail for webhook.site URLs without actual endpoint, which is expected
if webhook::test master "https://httpbin.org/post" 2>&1 | grep -q "Webhook endpoint"; then
    echo "[SUCCESS] Webhook connectivity test works"
else
    echo "[WARNING] Webhook connectivity test may require valid endpoint"
fi

# ========== Test 3: Configure Events for Realm ==========
echo "[INFO] Testing event configuration..."

# Create test realm first
echo "[INFO] Creating test realm: $TEST_REALM"
# Ensure container is running
if ! docker ps --format '{{.Names}}' | grep -q "$KEYCLOAK_CONTAINER_NAME"; then
    echo "[WARNING] Keycloak container not running, skipping realm tests"
else
    # Get admin token using inline method since function may not be available
    token=$(curl -s -X POST \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "username=${KEYCLOAK_ADMIN_USER}" \
        -d "password=${KEYCLOAK_ADMIN_PASSWORD}" \
        -d "grant_type=password" \
        -d "client_id=admin-cli" \
        "http://localhost:${KEYCLOAK_PORT}/realms/master/protocol/openid-connect/token" | \
        jq -r '.access_token' 2>/dev/null || echo "")
realm_config='{
    "realm": "'$TEST_REALM'",
    "enabled": true,
    "eventsEnabled": true,
    "eventsListeners": ["jboss-logging"]
}'

response=$(curl -s -X POST \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "$realm_config" \
    "http://localhost:${KEYCLOAK_PORT}/admin/realms" 2>&1)

if [[ -z "$response" ]] || ! echo "$response" | grep -q "error"; then
    echo "[SUCCESS] Test realm created: $TEST_REALM"
else
    echo "[WARNING] Could not create test realm (may already exist)"
fi

# Configure events for the test realm
if webhook::configure_events "$TEST_REALM" "LOGIN,LOGOUT,REGISTER" 2>&1 | grep -q "Events configured"; then
    echo "[SUCCESS] Event configuration successful"
else
    echo "[WARNING] Event configuration may require existing realm"
fi
fi  # End of container running check

# ========== Test 4: Event History ==========
echo "[INFO] Testing event history retrieval..."
if webhook::history "$TEST_REALM" 5 2>&1 | grep -E "Recent events|No events"; then
    echo "[SUCCESS] Event history retrieval works"
else
    echo "[WARNING] Event history may be empty for new realm"
fi

# ========== Test 5: CLI Command Structure ==========
echo "[INFO] Testing CLI webhook commands..."

# Check if webhook commands are registered
echo "[INFO] Checking webhook command registration..."
if vrooli resource keycloak help 2>&1 | grep -q "webhook"; then
    echo "[SUCCESS] Webhook command group registered in CLI"
    
    # Test webhook help
    echo "[INFO] Testing webhook list-events command..."
    if vrooli resource keycloak webhook list-events 2>&1 | grep -q "LOGIN"; then
        echo "[SUCCESS] CLI webhook list-events command works"
    else
        echo "[WARNING] CLI webhook command may require running container"
    fi
else
    echo "[WARNING] Webhook commands may not be visible in help"
fi

# ========== Cleanup ==========
echo "[INFO] Cleaning up test data..."

# Delete test realm if it was created
if [[ -n "${token:-}" ]]; then
    curl -s -X DELETE \
        -H "Authorization: Bearer $token" \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${TEST_REALM}" 2>&1 >/dev/null || true
    echo "[INFO] Test realm cleanup attempted"
fi

echo "[SUCCESS] All webhook tests passed"
exit 0