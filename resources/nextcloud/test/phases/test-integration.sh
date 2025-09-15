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

# Test 8: Collabora Office integration (if available)
echo -n "  Testing Collabora Office integration... "
if docker ps --format "{{.Names}}" | grep -q "nextcloud_collabora"; then
    if timeout 5 curl -sf "http://localhost:9980/hosting/discovery" &>/dev/null; then
        # Check if richdocuments app is enabled
        if docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ app:list | grep -q "richdocuments:"; then
            echo "✓"
        else
            echo "(not configured)"
        fi
    else
        echo "(Collabora not ready)"
    fi
else
    echo "(Collabora not running)"
fi

# Test Talk/Spreed integration
echo -n "  Testing Talk video conferencing... "
if docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ app:list | grep -q "spreed:"; then
    # Check if Talk is properly configured
    if docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:get spreed signaling_mode &>/dev/null; then
        echo "✓"
    else
        echo "(not configured)"
    fi
else
    echo "(Talk not installed)"
fi

echo "All integration tests passed!"
exit 0
