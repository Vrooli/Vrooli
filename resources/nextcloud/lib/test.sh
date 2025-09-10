#!/usr/bin/env bash
# Nextcloud Resource - Test Functions

set -euo pipefail

# ============================================================================
# Test Orchestration
# ============================================================================

test_all() {
    echo "Running all Nextcloud tests..."
    
    local test_dir="${SCRIPT_DIR}/test"
    
    if [[ ! -f "${test_dir}/run-tests.sh" ]]; then
        echo "Creating test runner..."
        create_test_runner
    fi
    
    # Delegate to test runner
    bash "${test_dir}/run-tests.sh" all
    return $?
}

test_smoke() {
    echo "Running Nextcloud smoke tests..."
    
    local test_dir="${SCRIPT_DIR}/test/phases"
    
    if [[ ! -f "${test_dir}/test-smoke.sh" ]]; then
        echo "Creating smoke test..."
        create_smoke_test
    fi
    
    # Run smoke test
    bash "${test_dir}/test-smoke.sh"
    return $?
}

test_integration() {
    echo "Running Nextcloud integration tests..."
    
    local test_dir="${SCRIPT_DIR}/test/phases"
    
    if [[ ! -f "${test_dir}/test-integration.sh" ]]; then
        echo "Creating integration test..."
        create_integration_test
    fi
    
    # Run integration test
    bash "${test_dir}/test-integration.sh"
    return $?
}

test_unit() {
    echo "Running Nextcloud unit tests..."
    
    local test_dir="${SCRIPT_DIR}/test/phases"
    
    if [[ ! -f "${test_dir}/test-unit.sh" ]]; then
        echo "Creating unit test..."
        create_unit_test
    fi
    
    # Run unit test
    bash "${test_dir}/test-unit.sh"
    return $?
}

# ============================================================================
# Test Creation Functions
# ============================================================================

create_test_runner() {
    local test_dir="${SCRIPT_DIR}/test"
    mkdir -p "$test_dir"
    
    cat > "${test_dir}/run-tests.sh" << 'EOF'
#!/usr/bin/env bash
# Nextcloud Test Runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="${SCRIPT_DIR}/phases"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test statistics
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

run_test_phase() {
    local phase="$1"
    local test_file="${PHASES_DIR}/test-${phase}.sh"
    
    if [[ ! -f "$test_file" ]]; then
        echo -e "${YELLOW}⚠ Skipping $phase tests (not found)${NC}"
        return 0
    fi
    
    echo -e "\n${GREEN}▶ Running $phase tests...${NC}"
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if bash "$test_file"; then
        echo -e "${GREEN}✓ $phase tests passed${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ $phase tests failed${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

main() {
    local test_type="${1:-all}"
    local exit_code=0
    
    echo "=============================="
    echo "Nextcloud Resource Test Suite"
    echo "=============================="
    
    case "$test_type" in
        all)
            run_test_phase "smoke" || exit_code=1
            run_test_phase "unit" || exit_code=1
            run_test_phase "integration" || exit_code=1
            ;;
        smoke|unit|integration)
            run_test_phase "$test_type" || exit_code=1
            ;;
        *)
            echo "Error: Unknown test type: $test_type"
            echo "Usage: $0 [all|smoke|unit|integration]"
            exit 1
            ;;
    esac
    
    # Print summary
    echo ""
    echo "=============================="
    echo "Test Summary"
    echo "=============================="
    echo "Tests Run: $TESTS_RUN"
    echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
    else
        echo -e "Tests Failed: $TESTS_FAILED"
    fi
    echo ""
    
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
    else
        echo -e "${RED}✗ Some tests failed${NC}"
    fi
    
    exit $exit_code
}

main "$@"
EOF
    
    chmod +x "${test_dir}/run-tests.sh"
}

create_smoke_test() {
    local test_dir="${SCRIPT_DIR}/test/phases"
    mkdir -p "$test_dir"
    
    cat > "${test_dir}/test-smoke.sh" << 'EOF'
#!/usr/bin/env bash
# Nextcloud Smoke Tests - Quick health validation

set -euo pipefail

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Running Nextcloud smoke tests..."

# Test 1: Check if container is running
echo -n "  Checking container status... "
if docker ps --format "{{.Names}}" | grep -q "^${NEXTCLOUD_CONTAINER_NAME}$"; then
    echo "✓"
else
    echo "✗"
    echo "Error: Nextcloud container is not running"
    exit 1
fi

# Test 2: Check health endpoint
echo -n "  Checking health endpoint... "
if timeout 5 curl -sf "http://localhost:${NEXTCLOUD_PORT}/status.php" > /dev/null; then
    echo "✓"
else
    echo "✗"
    echo "Error: Health endpoint not responding"
    exit 1
fi

# Test 3: Check database connectivity
echo -n "  Checking database connection... "
if docker exec nextcloud_postgres pg_isready -U "${NEXTCLOUD_DB_USER}" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "Error: Database not ready"
    exit 1
fi

# Test 4: Check Redis connectivity
echo -n "  Checking Redis connection... "
if docker exec nextcloud_redis redis-cli ping | grep -q PONG; then
    echo "✓"
else
    echo "✗"
    echo "Error: Redis not responding"
    exit 1
fi

# Test 5: Check WebDAV endpoint
echo -n "  Checking WebDAV endpoint... "
if curl -sf -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
        -X PROPFIND \
        "http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/" > /dev/null; then
    echo "✓"
else
    echo "✗"
    echo "Error: WebDAV endpoint not accessible"
    exit 1
fi

echo "All smoke tests passed!"
exit 0
EOF
    
    chmod +x "${test_dir}/test-smoke.sh"
}

create_integration_test() {
    local test_dir="${SCRIPT_DIR}/test/phases"
    mkdir -p "$test_dir"
    
    cat > "${test_dir}/test-integration.sh" << 'EOF'
#!/usr/bin/env bash
# Nextcloud Integration Tests - Full functionality validation

set -euo pipefail

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Running Nextcloud integration tests..."

# Test setup
TEST_FILE="/tmp/nextcloud_test_$(date +%s).txt"
TEST_CONTENT="Integration test content $(date)"
echo "$TEST_CONTENT" > "$TEST_FILE"

cleanup() {
    rm -f "$TEST_FILE" "/tmp/downloaded_test.txt"
}
trap cleanup EXIT

# Test 1: Upload file
echo -n "  Testing file upload... "
if curl -sf -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
        -T "$TEST_FILE" \
        "http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/files/${NEXTCLOUD_ADMIN_USER}/$(basename "$TEST_FILE")"; then
    echo "✓"
else
    echo "✗"
    echo "Error: Failed to upload file"
    exit 1
fi

# Test 2: List files
echo -n "  Testing file listing... "
if curl -sf -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
        -X PROPFIND \
        "http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/files/${NEXTCLOUD_ADMIN_USER}/" | \
        grep -q "$(basename "$TEST_FILE")"; then
    echo "✓"
else
    echo "✗"
    echo "Error: Uploaded file not found in listing"
    exit 1
fi

# Test 3: Download file
echo -n "  Testing file download... "
if curl -sf -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
        -o "/tmp/downloaded_test.txt" \
        "http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/files/${NEXTCLOUD_ADMIN_USER}/$(basename "$TEST_FILE")"; then
    if grep -q "$TEST_CONTENT" "/tmp/downloaded_test.txt"; then
        echo "✓"
    else
        echo "✗"
        echo "Error: Downloaded file content mismatch"
        exit 1
    fi
else
    echo "✗"
    echo "Error: Failed to download file"
    exit 1
fi

# Test 4: Create share
echo -n "  Testing share creation... "
SHARE_RESPONSE=$(curl -sf -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
        -X POST \
        -H "OCS-APIRequest: true" \
        -d "path=/$(basename "$TEST_FILE")&shareType=3" \
        "http://localhost:${NEXTCLOUD_PORT}/ocs/v2.php/apps/files_sharing/api/v1/shares?format=json")

if echo "$SHARE_RESPONSE" | grep -q '"statuscode":200'; then
    echo "✓"
else
    echo "✗"
    echo "Error: Failed to create share"
    echo "$SHARE_RESPONSE"
    exit 1
fi

# Test 5: Delete file
echo -n "  Testing file deletion... "
if curl -sf -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
        -X DELETE \
        "http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/files/${NEXTCLOUD_ADMIN_USER}/$(basename "$TEST_FILE")"; then
    echo "✓"
else
    echo "✗"
    echo "Error: Failed to delete file"
    exit 1
fi

# Test 6: Verify deletion
echo -n "  Verifying file deletion... "
if ! curl -sf -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
        -X PROPFIND \
        "http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/files/${NEXTCLOUD_ADMIN_USER}/" | \
        grep -q "$(basename "$TEST_FILE")"; then
    echo "✓"
else
    echo "✗"
    echo "Error: File still exists after deletion"
    exit 1
fi

# Test 7: OCC command execution
echo -n "  Testing OCC commands... "
if docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ status --output=json | grep -q '"installed":true'; then
    echo "✓"
else
    echo "✗"
    echo "Error: OCC commands not working"
    exit 1
fi

echo "All integration tests passed!"
exit 0
EOF
    
    chmod +x "${test_dir}/test-integration.sh"
}

create_unit_test() {
    local test_dir="${SCRIPT_DIR}/test/phases"
    mkdir -p "$test_dir"
    
    cat > "${test_dir}/test-unit.sh" << 'EOF'
#!/usr/bin/env bash
# Nextcloud Unit Tests - Library function validation

set -euo pipefail

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${RESOURCE_DIR}/config/defaults.sh"

echo "Running Nextcloud unit tests..."

# Test 1: Configuration loading
echo -n "  Testing configuration loading... "
if [[ -n "${NEXTCLOUD_PORT}" ]] && [[ "${NEXTCLOUD_PORT}" -eq 8086 ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: Configuration not loaded correctly"
    exit 1
fi

# Test 2: Runtime.json exists and is valid
echo -n "  Testing runtime.json validity... "
if jq -e . "${RESOURCE_DIR}/config/runtime.json" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "Error: runtime.json is not valid JSON"
    exit 1
fi

# Test 3: Schema.json exists and is valid
echo -n "  Testing schema.json validity... "
if jq -e . "${RESOURCE_DIR}/config/schema.json" > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "Error: schema.json is not valid JSON"
    exit 1
fi

# Test 4: CLI script is executable
echo -n "  Testing CLI script permissions... "
if [[ -x "${RESOURCE_DIR}/cli.sh" ]]; then
    echo "✓"
else
    echo "✗"
    echo "Error: cli.sh is not executable"
    exit 1
fi

# Test 5: Help command works
echo -n "  Testing help command... "
if "${RESOURCE_DIR}/cli.sh" help | grep -q "Nextcloud Resource Management"; then
    echo "✓"
else
    echo "✗"
    echo "Error: Help command not working"
    exit 1
fi

# Test 6: Info command returns valid JSON
echo -n "  Testing info command... "
if "${RESOURCE_DIR}/cli.sh" info --json | jq -e . > /dev/null 2>&1; then
    echo "✓"
else
    echo "✗"
    echo "Error: Info command not returning valid JSON"
    exit 1
fi

echo "All unit tests passed!"
exit 0
EOF
    
    chmod +x "${test_dir}/test-unit.sh"
}