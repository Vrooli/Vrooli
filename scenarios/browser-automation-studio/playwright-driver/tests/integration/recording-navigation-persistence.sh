#!/usr/bin/env bash
#
# Integration test: Recording Pipeline Navigation Persistence
#
# This test verifies that the recording pipeline works correctly AFTER navigation.
# It specifically tests the fix for rebrowser-playwright's page.route() loss after navigation.
#
# The bug: page.route() handlers don't persist across navigation with rebrowser-playwright,
# causing recording events to silently fail after the first page navigation.
#
# The fix: Navigation listeners re-register page.route() after each page load.
#
# Usage:
#   ./tests/integration/recording-navigation-persistence.sh [API_BASE_URL]
#
# Requirements:
#   - browser-automation-studio scenario must be running
#   - curl and jq must be installed
#
# Exit codes:
#   0 - Test passed
#   1 - Test failed
#   2 - Prerequisites not met

set -euo pipefail

# Configuration
API_BASE_URL="${1:-http://localhost:37955}"
TIMEOUT_MS=30000

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Recording Pipeline Navigation Persistence Test                  ║${NC}"
echo -e "${BLUE}╠══════════════════════════════════════════════════════════════════╣${NC}"
echo -e "${BLUE}║  Verifies that recording works after page navigation             ║${NC}"
echo -e "${BLUE}║  (regression test for rebrowser-playwright route loss fix)       ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check prerequisites
if ! command -v curl &> /dev/null; then
    echo -e "${RED}Error: curl is required but not installed${NC}"
    exit 2
fi

if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is required but not installed${NC}"
    exit 2
fi

# Check if API is reachable
echo -e "${YELLOW}Checking API availability at ${API_BASE_URL}...${NC}"
if ! curl -s "${API_BASE_URL}/api/v1/observability" > /dev/null 2>&1; then
    echo -e "${RED}Error: API not reachable at ${API_BASE_URL}${NC}"
    echo "Make sure the browser-automation-studio scenario is running:"
    echo "  vrooli scenario start browser-automation-studio"
    exit 2
fi
echo -e "${GREEN}✓ API is reachable${NC}"
echo ""

# Run pipeline test
echo -e "${YELLOW}Running pipeline test...${NC}"
RESULT=$(curl -s -X POST "${API_BASE_URL}/api/v1/observability/pipeline-test" \
    -H "Content-Type: application/json" \
    -d "{\"timeout_ms\": ${TIMEOUT_MS}}")

# Parse results
SUCCESS=$(echo "$RESULT" | jq -r '.success')
USED_TEMP_SESSION=$(echo "$RESULT" | jq -r '.used_temp_session // false')
FAILURE_POINT=$(echo "$RESULT" | jq -r '.failure_point // "none"')
FAILURE_MESSAGE=$(echo "$RESULT" | jq -r '.failure_message // "none"')
DURATION=$(echo "$RESULT" | jq -r '.duration_ms // 0')

echo ""
echo -e "${BLUE}Test Results:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Success:          $SUCCESS"
echo "  Used temp session: $USED_TEMP_SESSION"
echo "  Duration:         ${DURATION}ms"

if [ "$SUCCESS" == "true" ]; then
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  ✓ TEST PASSED                                                   ║${NC}"
    echo -e "${GREEN}║  Recording pipeline works correctly after navigation             ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════════╝${NC}"
    exit 0
else
    echo ""
    echo "  Failure point:    $FAILURE_POINT"
    echo "  Failure message:  $FAILURE_MESSAGE"
    echo ""

    # Show step results
    echo -e "${YELLOW}Step Results:${NC}"
    echo "$RESULT" | jq -r '.steps[] | "  [\(if .passed then "✓" else "✗" end)] \(.name): \(.error // "ok")"'

    echo ""
    echo -e "${RED}╔══════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║  ✗ TEST FAILED                                                   ║${NC}"
    echo -e "${RED}║  Recording pipeline failed - possible regression                 ║${NC}"
    echo -e "${RED}╚══════════════════════════════════════════════════════════════════╝${NC}"

    # Show suggestions if available
    SUGGESTIONS=$(echo "$RESULT" | jq -r '.suggestions // []')
    if [ "$SUGGESTIONS" != "[]" ]; then
        echo ""
        echo -e "${YELLOW}Suggestions:${NC}"
        echo "$RESULT" | jq -r '.suggestions[]? | "  • \(.)"'
    fi

    exit 1
fi
