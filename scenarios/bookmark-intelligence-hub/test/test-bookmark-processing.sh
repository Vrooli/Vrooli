#!/bin/bash

# Test script for Bookmark Intelligence Hub bookmark processing functionality
set -e

API_BASE_URL="${API_BASE_URL:-http://localhost:15200}"
TEST_PROFILE_ID="test-profile-$(date +%s)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🧪 Testing Bookmark Intelligence Hub - Bookmark Processing${NC}"
echo "API Base URL: $API_BASE_URL"
echo ""

# Function to test API endpoint
test_endpoint() {
    local method="$1"
    local endpoint="$2"
    local expected_status="$3"
    local description="$4"
    local data="$5"
    
    echo -n "Testing $description... "
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
                       -H "Content-Type: application/json" \
                       -d "$data" \
                       "$API_BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
                       -H "Content-Type: application/json" \
                       "$API_BASE_URL$endpoint")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" -eq "$expected_status" ]; then
        echo -e "${GREEN}✅ PASS${NC} (HTTP $http_code)"
        if [ -n "$body" ] && command -v jq >/dev/null 2>&1; then
            echo "$body" | jq . | head -5
        fi
    else
        echo -e "${RED}❌ FAIL${NC} (Expected HTTP $expected_status, got $http_code)"
        echo "Response body: $body"
    fi
    
    echo ""
}

# Test 1: Health Check
echo -e "${YELLOW}Test 1: Health Check${NC}"
test_endpoint "GET" "/health" 200 "API health check"

# Test 2: Get Profiles
echo -e "${YELLOW}Test 2: Profile Management${NC}"
test_endpoint "GET" "/api/v1/profiles" 200 "Get profiles list"

# Test 3: Get Profile Stats
echo -e "${YELLOW}Test 3: Profile Stats${NC}"
test_endpoint "GET" "/api/v1/profiles/demo-profile/stats" 200 "Get profile statistics"

# Test 4: Process Bookmarks
echo -e "${YELLOW}Test 4: Bookmark Processing${NC}"
bookmark_data='{
  "profile_id": "demo-profile",
  "bookmarks": [
    {
      "platform": "reddit",
      "url": "https://reddit.com/r/programming/comments/test",
      "title": "Test Programming Post",
      "content": "This is a test post about programming and JavaScript frameworks.",
      "author": "testuser",
      "metadata": {
        "subreddit": "programming",
        "score": 100,
        "comments": 25
      }
    }
  ]
}'

test_endpoint "POST" "/api/v1/bookmarks/process" 200 "Process test bookmarks" "$bookmark_data"

# Test 5: Query Bookmarks
echo -e "${YELLOW}Test 5: Query Bookmarks${NC}"
test_endpoint "GET" "/api/v1/bookmarks/query?profile_id=demo-profile&limit=10" 200 "Query bookmarks"

# Test 6: Sync Bookmarks
echo -e "${YELLOW}Test 6: Sync Bookmarks${NC}"
sync_data='{"profile_id": "demo-profile"}'
test_endpoint "POST" "/api/v1/bookmarks/sync" 200 "Sync bookmarks" "$sync_data"

# Test 7: Get Categories
echo -e "${YELLOW}Test 7: Category Management${NC}"
test_endpoint "GET" "/api/v1/categories?profile_id=demo-profile" 200 "Get categories"

# Test 8: Get Actions
echo -e "${YELLOW}Test 8: Action Management${NC}"
test_endpoint "GET" "/api/v1/actions?profile_id=demo-profile" 200 "Get pending actions"

# Test 9: Approve Actions
echo -e "${YELLOW}Test 9: Approve Actions${NC}"
approve_data='{
  "profile_id": "demo-profile",
  "action_ids": ["test-action-1"],
  "approval_type": "approve"
}'
test_endpoint "POST" "/api/v1/actions/approve" 200 "Approve actions" "$approve_data"

# Test 10: Get Platforms
echo -e "${YELLOW}Test 10: Platform Management${NC}"
test_endpoint "GET" "/api/v1/platforms" 200 "Get platforms list"

# Test 11: Get Platform Status
echo -e "${YELLOW}Test 11: Platform Status${NC}"
test_endpoint "GET" "/api/v1/platforms/status?profile_id=demo-profile" 200 "Get platform status"

# Test 12: Platform Sync
echo -e "${YELLOW}Test 12: Platform Sync${NC}"
test_endpoint "POST" "/api/v1/platforms/reddit/sync" 200 "Sync Reddit platform"

# Test 13: Analytics
echo -e "${YELLOW}Test 13: Analytics${NC}"
test_endpoint "GET" "/api/v1/analytics/metrics?profile_id=demo-profile" 200 "Get analytics metrics"

echo -e "${BLUE}🎉 Bookmark processing tests completed!${NC}"
echo ""
echo "Note: These tests check API endpoint availability and basic response structure."
echo "Full integration tests would require:"
echo "  - Running database with test data"
echo "  - Configured platform integrations (Reddit, Twitter, TikTok)"
echo "  - Huginn and Browserless services"
echo "  - End-to-end bookmark processing workflow"
echo ""
echo "Next steps:"
echo "  1. Run 'vrooli scenario run bookmark-intelligence-hub' to start all services"
echo "  2. Configure platform credentials in the UI"
echo "  3. Test actual bookmark sync from social media platforms"