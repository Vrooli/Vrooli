#!/bin/bash
# Integration tests for file-tools - Real-world workflows demonstrating cross-scenario value
set -euo pipefail

# Get API configuration from environment
API_BASE="${API_BASE:-http://localhost:${API_PORT:-15468}}"
API_TOKEN="${API_TOKEN:-API_TOKEN_PLACEHOLDER}"

echo "=== File Tools Integration Tests ==="
echo "Testing cross-scenario workflows at: $API_BASE"
echo ""

TESTS_RUN=0
TESTS_PASSED=0

# Test 1: Duplicate Detection for Storage Optimization
echo "[1/4] Duplicate Detection Workflow..."
TEST_DIR="/tmp/file-tools-int-$$"
mkdir -p "$TEST_DIR/docs"
echo "Document 1" > "$TEST_DIR/docs/doc1.txt"
echo "Document 2" > "$TEST_DIR/docs/doc2.txt"
echo "Document 1" > "$TEST_DIR/docs/doc1-dup.txt"

if curl -sf -X POST "$API_BASE/api/v1/files/duplicates/detect" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $API_TOKEN" \
    -d "{\"scan_paths\":[\"$TEST_DIR/docs\"],\"detection_method\":\"hash\"}" | grep -q "scan_id"; then
    echo "  ✓ Duplicate detection"
    TESTS_PASSED=$((TESTS_PASSED + 1))
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Test 2: File Compression for Backups
echo "[2/4] Compression Workflow..."
if curl -sf -X POST "$API_BASE/api/v1/files/compress" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $API_TOKEN" \
    -d "{\"files\":[\"$TEST_DIR/docs/doc1.txt\",\"$TEST_DIR/docs/doc2.txt\"],\"archive_format\":\"zip\",\"output_path\":\"$TEST_DIR/backup.zip\"}" | grep -q "archive_path"; then
    echo "  ✓ File compression"
    TESTS_PASSED=$((TESTS_PASSED + 1))

    # Test 3: Checksum Verification for Integrity (depends on Test 2)
    echo "[3/4] Checksum Verification..."
    if curl -sf -X POST "$API_BASE/api/v1/files/checksum" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $API_TOKEN" \
        -d "{\"files\":[\"$TEST_DIR/backup.zip\"],\"algorithm\":\"sha256\"}" | grep -q "checksum"; then
        echo "  ✓ Checksum verification"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    fi
    TESTS_RUN=$((TESTS_RUN + 1))
else
    TESTS_RUN=$((TESTS_RUN + 1))  # Compression failed, skip checksum test
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Test 4: Smart Organization
echo "[4/4] Smart Organization Workflow..."
if curl -sf -X POST "$API_BASE/api/v1/files/organize" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $API_TOKEN" \
    -d "{\"source_path\":\"$TEST_DIR/docs\",\"destination_path\":\"$TEST_DIR/organized\",\"organization_rules\":[{\"rule_type\":\"by_type\"}],\"options\":{\"dry_run\":true}}" | grep -q "organization_plan"; then
    echo "  ✓ Smart organization"
    TESTS_PASSED=$((TESTS_PASSED + 1))
fi
TESTS_RUN=$((TESTS_RUN + 1))

# Cleanup
rm -rf "$TEST_DIR"

# Summary
echo ""
echo "═══════════════════════════════════"
echo "Results: $TESTS_PASSED/$TESTS_RUN workflows validated"
echo "═══════════════════════════════════"

if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
    echo "✅ All integration workflows passed"
    echo ""
    echo "These workflows demonstrate file-tools value for:"
    echo "  • document-manager: Duplicate detection + organization"
    echo "  • backup-automation: Compression + integrity verification"
    echo "  • storage-optimizer: Duplicate detection + checksums"
    exit 0
else
    echo "⚠️  $((TESTS_RUN - TESTS_PASSED)) workflows failed"
    exit 1
fi