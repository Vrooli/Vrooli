#!/usr/bin/env bash
# Vault Verification Script - Tests that Vault is working correctly

set -euo pipefail

# Source var.sh for directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get script directory
VAULT_LIB_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
cd "$VAULT_LIB_DIR"

echo "=== Vault Verification Script ==="
echo

# Function to print status
print_status() {
    local status=$1
    local message=$2
    if [[ $status -eq 0 ]]; then
        echo -e "${GREEN}✓${NC} $message"
    else
        echo -e "${RED}✗${NC} $message"
        return 1
    fi
}

# Function to print info
print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Track overall success
VERIFICATION_PASSED=true

# 1. Check if Vault is installed
echo "1. Checking Vault installation..."
if resource-vault status >/dev/null 2>&1; then
    STATUS=$(./manage.sh --action status 2>&1 | grep -oP 'Status: \K\w+' || echo "unknown")
    if [[ "$STATUS" == "healthy" ]]; then
        print_status 0 "Vault is installed and healthy"
    else
        print_status 1 "Vault status: $STATUS (not healthy)"
        VERIFICATION_PASSED=false
    fi
else
    print_status 1 "Vault is not installed"
    print_info "Run: resource-vault manage install && resource-vault content init-dev"
    exit 1
fi
echo

# 2. Check Docker container
echo "2. Checking Docker container..."
if docker ps --format "table {{.Names}}" | grep -q "vault"; then
    print_status 0 "Vault container is running"
    CONTAINER_ID=$(docker ps -q -f name=vault)
    print_info "Container ID: ${CONTAINER_ID:0:12}"
else
    print_status 1 "Vault container is not running"
    VERIFICATION_PASSED=false
fi
echo

# 3. Check API accessibility
echo "3. Checking API accessibility..."
VAULT_ADDR="http://localhost:8200"
if curl -s -o /dev/null -w "%{http_code}" "$VAULT_ADDR/v1/sys/health" | grep -q "200"; then
    print_status 0 "Vault API is accessible at $VAULT_ADDR"
else
    print_status 1 "Cannot reach Vault API at $VAULT_ADDR"
    VERIFICATION_PASSED=false
fi
echo

# 4. Check token file
echo "4. Checking authentication..."
TOKEN_FILE="/tmp/vault-token"
if [[ -f "$TOKEN_FILE" ]]; then
    print_status 0 "Token file exists at $TOKEN_FILE"
    TOKEN=$(cat "$TOKEN_FILE" 2>/dev/null || echo "")
    if [[ -n "$TOKEN" ]]; then
        print_info "Token length: ${#TOKEN} characters"
    else
        print_status 1 "Token file is empty"
        VERIFICATION_PASSED=false
    fi
else
    print_status 1 "Token file not found at $TOKEN_FILE"
    VERIFICATION_PASSED=false
fi
echo

# 5. Test secret operations
echo "5. Testing secret operations..."
TEST_PATH="test/verification/test-secret"
TEST_VALUE="test-value-$(date +%s)"

# Store a secret
echo -n "   Storing test secret... "
if resource-vault content add --path "$TEST_PATH" --value "$TEST_VALUE" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    VERIFICATION_PASSED=false
fi

# Retrieve the secret
echo -n "   Retrieving test secret... "
RETRIEVED=$(./manage.sh --action get-secret --path "$TEST_PATH" 2>/dev/null || echo "")
if [[ "$RETRIEVED" == "$TEST_VALUE" ]]; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    print_info "Expected: $TEST_VALUE"
    print_info "Got: $RETRIEVED"
    VERIFICATION_PASSED=false
fi

# List secrets
echo -n "   Listing secrets... "
if resource-vault content list --path "test/" 2>&1 | grep -q "verification"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    VERIFICATION_PASSED=false
fi

# Delete the secret
echo -n "   Deleting test secret... "
if resource-vault content remove --path "$TEST_PATH" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    VERIFICATION_PASSED=false
fi

# Verify deletion
echo -n "   Verifying deletion... "
if ! resource-vault content get --path "$TEST_PATH" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    VERIFICATION_PASSED=false
fi
echo

# 6. Check default paths
echo "6. Checking default namespace paths..."
for path in "environments/dev/" "environments/staging/" "environments/prod/" "resources/"; do
    echo -n "   Checking $path... "
    if ./manage.sh --action list-secrets --path "$path" >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        VERIFICATION_PASSED=false
    fi
done
echo

# 7. Test bulk operations
echo "7. Testing bulk operations..."
BULK_JSON='{"key1": "value1", "key2": "value2", "key3": "value3"}'
echo "$BULK_JSON" > /tmp/vault-test-bulk.json

echo -n "   Bulk storing secrets... "
if resource-vault content add --json-file /tmp/vault-test-bulk.json --base-path "test/bulk" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
    
    # Verify each key
    echo -n "   Verifying bulk secrets... "
    ALL_FOUND=true
    for key in key1 key2 key3; do
        if ! ./manage.sh --action get-secret --path "test/bulk/$key" >/dev/null 2>&1; then
            ALL_FOUND=false
            break
        fi
    done
    
    if $ALL_FOUND; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        VERIFICATION_PASSED=false
    fi
    
    # Cleanup
    for key in key1 key2 key3; do
        ./manage.sh --action delete-secret --path "test/bulk/$key" >/dev/null 2>&1
    done
else
    echo -e "${RED}✗${NC}"
    VERIFICATION_PASSED=false
fi

trash::safe_remove /tmp/vault-test-bulk.json --temp
echo

# 8. Display summary
echo "=== Verification Summary ==="
if $VERIFICATION_PASSED; then
    echo -e "${GREEN}All verification tests passed!${NC}"
    echo
    echo "Vault is working correctly. You can now:"
    echo "  - Store secrets: ./manage.sh --action put-secret --path \"path/to/secret\" --value \"secret-value\""
    echo "  - Retrieve secrets: ./manage.sh --action get-secret --path \"path/to/secret\""
    echo "  - Access Vault UI: http://localhost:8200 (use token from $TOKEN_FILE)"
    exit 0
else
    echo -e "${RED}Some verification tests failed!${NC}"
    echo
    echo "Troubleshooting steps:"
    echo "  1. Check logs: resource-vault logs --tail 50"
    echo "  2. Run diagnostics: resource-vault test integration"
    echo "  3. Restart Vault: resource-vault manage restart"
    echo "  4. Reinstall if needed: resource-vault manage uninstall && resource-vault manage install"
    exit 1
fi