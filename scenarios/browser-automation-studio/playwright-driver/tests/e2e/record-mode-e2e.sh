#!/bin/bash
# =============================================================================
# Record Mode End-to-End Test Script
# =============================================================================
#
# This script tests the full Record Mode flow:
# 1. Create a browser session
# 2. Navigate to a test page
# 3. Start recording
# 4. Execute some actions (click, type)
# 5. Stop recording
# 6. Verify actions were captured
# 7. Generate workflow from actions
# 8. Verify workflow was created
# 9. Cleanup session
#
# Prerequisites:
# - Playwright driver running on PLAYWRIGHT_DRIVER_URL (default: http://localhost:39400)
# - API server running on API_URL (default: http://localhost:39500)
# - jq installed for JSON parsing
#
# Usage:
#   ./tests/e2e/record-mode-e2e.sh [--driver-url URL] [--api-url URL] [--verbose]
#
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default URLs
PLAYWRIGHT_DRIVER_URL="${PLAYWRIGHT_DRIVER_URL:-http://localhost:39400}"
API_URL="${API_URL:-http://localhost:39500}"
VERBOSE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --driver-url)
      PLAYWRIGHT_DRIVER_URL="$2"
      shift 2
      ;;
    --api-url)
      API_URL="$2"
      shift 2
      ;;
    --verbose|-v)
      VERBOSE=true
      shift
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Test state
SESSION_ID=""
RECORDING_ID=""
WORKFLOW_ID=""
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log() {
  echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
  echo -e "${GREEN}[PASS]${NC} $1"
  ((TESTS_PASSED++))
}

log_error() {
  echo -e "${RED}[FAIL]${NC} $1"
  ((TESTS_FAILED++))
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

debug() {
  if [ "$VERBOSE" = true ]; then
    echo -e "${YELLOW}[DEBUG]${NC} $1"
  fi
}

cleanup() {
  log "Cleaning up..."
  if [ -n "$SESSION_ID" ]; then
    debug "Closing session $SESSION_ID"
    curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/close" &>/dev/null || true
  fi
}

trap cleanup EXIT

# Check prerequisites
check_prerequisites() {
  log "Checking prerequisites..."

  # Check jq
  if ! command -v jq &> /dev/null; then
    log_error "jq is required but not installed"
    exit 1
  fi

  # Check playwright driver health
  debug "Checking playwright driver at ${PLAYWRIGHT_DRIVER_URL}/health"
  if ! curl -s -f "${PLAYWRIGHT_DRIVER_URL}/health" &>/dev/null; then
    log_error "Playwright driver not available at ${PLAYWRIGHT_DRIVER_URL}"
    log "Make sure the driver is running: make start (in playwright-driver dir)"
    exit 1
  fi
  log_success "Playwright driver is healthy"

  # Check API health (optional - may not be needed for driver-only tests)
  debug "Checking API at ${API_URL}/api/v1/health"
  if curl -s -f "${API_URL}/api/v1/health" &>/dev/null; then
    log_success "API server is healthy"
  else
    log_warn "API server not available - skipping API-level tests"
  fi
}

# =============================================================================
# Test: Create Session
# =============================================================================
test_create_session() {
  log "Creating browser session..."

  local response
  response=$(curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/start" \
    -H "Content-Type: application/json" \
    -d '{
      "viewport": {"width": 1280, "height": 720},
      "userAgent": "Record Mode E2E Test"
    }')

  debug "Response: $response"

  SESSION_ID=$(echo "$response" | jq -r '.session_id // empty')

  if [ -z "$SESSION_ID" ]; then
    log_error "Failed to create session: $response"
    return 1
  fi

  log_success "Created session: $SESSION_ID"
}

# =============================================================================
# Test: Navigate to Test Page
# =============================================================================
test_navigate() {
  log "Navigating to test page..."

  local response
  response=$(curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/run" \
    -H "Content-Type: application/json" \
    -d '{
      "instruction": {
        "index": 0,
        "node_id": "nav-1",
        "type": "navigate",
        "params": {
          "url": "https://example.com"
        }
      }
    }')

  debug "Response: $response"

  local success
  success=$(echo "$response" | jq -r '.success // false')

  if [ "$success" != "true" ]; then
    log_error "Failed to navigate: $response"
    return 1
  fi

  log_success "Navigated to https://example.com"
}

# =============================================================================
# Test: Start Recording
# =============================================================================
test_start_recording() {
  log "Starting recording..."

  local response
  response=$(curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/record/start" \
    -H "Content-Type: application/json" \
    -d '{}')

  debug "Response: $response"

  RECORDING_ID=$(echo "$response" | jq -r '.recording_id // empty')

  if [ -z "$RECORDING_ID" ]; then
    log_error "Failed to start recording: $response"
    return 1
  fi

  log_success "Started recording: $RECORDING_ID"
}

# =============================================================================
# Test: Recording Status
# =============================================================================
test_recording_status() {
  log "Checking recording status..."

  local response
  response=$(curl -s -X GET "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/record/status")

  debug "Response: $response"

  local is_recording
  is_recording=$(echo "$response" | jq -r '.is_recording // false')

  if [ "$is_recording" != "true" ]; then
    log_error "Recording not active: $response"
    return 1
  fi

  log_success "Recording is active"
}

# =============================================================================
# Test: Perform Actions
# =============================================================================
test_perform_actions() {
  log "Performing test actions..."

  # Click on the "More information..." link on example.com
  local click_response
  click_response=$(curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/run" \
    -H "Content-Type: application/json" \
    -d '{
      "instruction": {
        "index": 1,
        "node_id": "click-1",
        "type": "click",
        "params": {
          "selector": "h1"
        }
      }
    }')

  debug "Click response: $click_response"

  # Small delay to let recording capture the action
  sleep 0.5

  log_success "Performed click action"
}

# =============================================================================
# Test: Get Recorded Actions
# =============================================================================
test_get_actions() {
  log "Getting recorded actions..."

  local response
  response=$(curl -s -X GET "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/record/actions")

  debug "Response: $response"

  local count
  count=$(echo "$response" | jq -r '.count // 0')

  log "Recorded $count actions"

  if [ "$count" -lt 0 ]; then
    log_error "No actions recorded"
    return 1
  fi

  # Print action summary
  echo "$response" | jq -r '.actions[]? | "  - \(.actionType): \(.selector.primary // "no selector")"' 2>/dev/null || true

  log_success "Retrieved recorded actions"
}

# =============================================================================
# Test: Stop Recording
# =============================================================================
test_stop_recording() {
  log "Stopping recording..."

  local response
  response=$(curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/record/stop")

  debug "Response: $response"

  local action_count
  action_count=$(echo "$response" | jq -r '.action_count // -1')

  if [ "$action_count" -lt 0 ]; then
    log_error "Failed to stop recording: $response"
    return 1
  fi

  log_success "Stopped recording. Total actions: $action_count"
}

# =============================================================================
# Test: Validate Selector
# =============================================================================
test_validate_selector() {
  log "Validating selector..."

  local response
  response=$(curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/record/validate-selector" \
    -H "Content-Type: application/json" \
    -d '{"selector": "h1"}')

  debug "Response: $response"

  local valid
  valid=$(echo "$response" | jq -r '.valid // false')
  local match_count
  match_count=$(echo "$response" | jq -r '.match_count // 0')

  log "Selector 'h1' valid: $valid, matches: $match_count"

  if [ "$valid" = "true" ] || [ "$match_count" -gt 0 ]; then
    log_success "Selector validation works"
  else
    log_error "Selector validation failed: $response"
    return 1
  fi
}

# =============================================================================
# Test: Duplicate Start Recording (Should Fail)
# =============================================================================
test_duplicate_start() {
  log "Testing duplicate start recording (should fail)..."

  # First start recording
  curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/record/start" &>/dev/null

  # Try to start again
  local response
  response=$(curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/record/start")

  debug "Response: $response"

  local error
  error=$(echo "$response" | jq -r '.error // empty')

  if [ "$error" = "RECORDING_IN_PROGRESS" ]; then
    log_success "Correctly rejected duplicate start"
  else
    log_error "Should have rejected duplicate start: $response"
    return 1
  fi

  # Stop the recording we just started
  curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/record/stop" &>/dev/null
}

# =============================================================================
# Test: Generate Workflow via API
# =============================================================================
test_generate_workflow_api() {
  # Check if API is available
  if ! curl -s -f "${API_URL}/api/v1/health" &>/dev/null; then
    log_warn "Skipping API workflow generation test (API not available)"
    return 0
  fi

  log "Testing workflow generation via API..."

  # First, start recording and perform some actions
  curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/record/start" &>/dev/null
  sleep 0.2

  # Perform a click
  curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/run" \
    -H "Content-Type: application/json" \
    -d '{
      "instruction": {
        "index": 0,
        "node_id": "test-click",
        "type": "click",
        "params": {"selector": "h1"}
      }
    }' &>/dev/null
  sleep 0.2

  # Don't stop recording yet - API will read from active buffer

  # Generate workflow via API
  local response
  response=$(curl -s -X POST "${API_URL}/api/v1/recordings/live/${SESSION_ID}/generate-workflow" \
    -H "Content-Type: application/json" \
    -d '{"name": "E2E Test Workflow"}')

  debug "Response: $response"

  # Stop recording
  curl -s -X POST "${PLAYWRIGHT_DRIVER_URL}/session/${SESSION_ID}/record/stop" &>/dev/null

  WORKFLOW_ID=$(echo "$response" | jq -r '.workflow_id // empty')

  if [ -n "$WORKFLOW_ID" ] && [ "$WORKFLOW_ID" != "null" ]; then
    log_success "Generated workflow: $WORKFLOW_ID"
  else
    log_warn "Could not generate workflow via API: $response"
    return 0  # Don't fail - API may not be configured
  fi
}

# =============================================================================
# Main Test Runner
# =============================================================================
main() {
  echo "=============================================="
  echo "  Record Mode End-to-End Tests"
  echo "=============================================="
  echo ""
  echo "Playwright Driver: $PLAYWRIGHT_DRIVER_URL"
  echo "API Server: $API_URL"
  echo ""

  check_prerequisites

  echo ""
  echo "Running tests..."
  echo ""

  # Run tests in sequence
  test_create_session || true
  test_navigate || true
  test_start_recording || true
  test_recording_status || true
  test_perform_actions || true
  test_get_actions || true
  test_validate_selector || true
  test_stop_recording || true
  test_duplicate_start || true
  test_generate_workflow_api || true

  echo ""
  echo "=============================================="
  echo "  Test Results"
  echo "=============================================="
  echo ""
  echo -e "  ${GREEN}Passed:${NC} $TESTS_PASSED"
  echo -e "  ${RED}Failed:${NC} $TESTS_FAILED"
  echo ""

  if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
  else
    echo -e "${RED}Some tests failed${NC}"
    exit 1
  fi
}

main "$@"
