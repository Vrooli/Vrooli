#!/bin/bash
# Test file-tools API endpoints

set -euo pipefail

API_BASE="${API_BASE:-http://localhost:15468}"
API_TOKEN="${API_TOKEN:-API_TOKEN_PLACEHOLDER}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
test_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="${4:-}"
    local expected_status="${5:-200}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    echo -n "Testing $name... "
    
    local url="${API_BASE}${endpoint}"
    local curl_opts=(-s -w "\n%{http_code}" -X "$method")
    
    if [[ "$endpoint" != "/health" ]]; then
        curl_opts+=(-H "Authorization: Bearer $API_TOKEN")
    fi
    
    if [[ -n "$data" ]]; then
        curl_opts+=(-H "Content-Type: application/json" -d "$data")
    fi
    
    response=$(curl "${curl_opts[@]}" "$url" 2>/dev/null)
    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [[ "$status_code" == "$expected_status" ]]; then
        echo -e "${GREEN}✓${NC} (${status_code})"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗${NC} (Expected: ${expected_status}, Got: ${status_code})"
        echo "  Response: $body"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Create test files
echo "Creating test files..."
echo "Test content 1" > /tmp/test1.txt
echo "Test content 2" > /tmp/test2.txt
mkdir -p /tmp/test-dir
echo "Test content 3" > /tmp/test-dir/test3.txt

echo -e "\n${BLUE}Testing P0 Endpoints${NC}"
echo "========================"

# Health check
test_endpoint "Health Check" "GET" "/health" "" "200"

# File compression
test_endpoint "Compress ZIP" "POST" "/api/v1/files/compress" \
    '{"files":["/tmp/test1.txt","/tmp/test2.txt"],"archive_format":"zip","output_path":"/tmp/test.zip"}' "200"

test_endpoint "Compress TAR" "POST" "/api/v1/files/compress" \
    '{"files":["/tmp/test1.txt","/tmp/test2.txt"],"archive_format":"tar","output_path":"/tmp/test.tar"}' "200"

test_endpoint "Compress GZIP" "POST" "/api/v1/files/compress" \
    '{"files":["/tmp/test1.txt"],"archive_format":"gzip","output_path":"/tmp/test.tar.gz"}' "200"

# File extraction
test_endpoint "Extract ZIP" "POST" "/api/v1/files/extract" \
    '{"archive_path":"/tmp/test.zip","destination_path":"/tmp/extracted"}' "200"

# File operations
test_endpoint "Copy File" "POST" "/api/v1/files/operation" \
    '{"operation":"copy","source":"/tmp/test1.txt","target":"/tmp/test1-copy.txt"}' "200"

test_endpoint "Move File" "POST" "/api/v1/files/operation" \
    '{"operation":"move","source":"/tmp/test1-copy.txt","target":"/tmp/test1-moved.txt"}' "200"

test_endpoint "Delete File" "POST" "/api/v1/files/operation" \
    '{"operation":"delete","source":"/tmp/test1-moved.txt"}' "200"

# Checksum
test_endpoint "Checksum MD5" "POST" "/api/v1/files/checksum" \
    '{"files":["/tmp/test1.txt"],"algorithm":"md5"}' "200"

test_endpoint "Checksum SHA256" "POST" "/api/v1/files/checksum" \
    '{"files":["/tmp/test1.txt","/tmp/test2.txt"],"algorithm":"sha256"}' "200"

# File split
test_endpoint "Split File" "POST" "/api/v1/files/split" \
    '{"file":"/tmp/test.zip","size":"1K"}' "200"

# File merge
test_endpoint "Merge Files" "POST" "/api/v1/files/merge" \
    '{"pattern":"/tmp/test.zip.part*","output":"/tmp/test-merged.zip"}' "200"

# Metadata
test_endpoint "Get Metadata" "GET" "/api/v1/files/metadata/tmp/test1.txt" "" "200"

echo -e "\n${BLUE}Test Summary${NC}"
echo "==================="
echo -e "Tests Run:    ${TESTS_RUN}"
echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"

# Cleanup
echo -e "\n${BLUE}Cleaning up test files...${NC}"
rm -f /tmp/test*.txt /tmp/test*.zip /tmp/test*.tar /tmp/test*.tar.gz
rm -rf /tmp/test-dir /tmp/extracted
rm -f /tmp/test*.part* /tmp/test-merged.*

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed!${NC}"
    exit 1
fi