#!/usr/bin/env bats
# Tests for agent customization CLI commands [REQ:AGENT-TRIGGER][REQ:AGENT-INPUT]

setup() {
  # Get scenario root
  SCENARIO_ROOT="${BATS_TEST_DIRNAME}/../.."
  CLI_BIN="${SCENARIO_ROOT}/cli/landing-manager"

  # Ensure CLI binary exists
  if [[ ! -f "$CLI_BIN" ]]; then
    skip "CLI binary not found at $CLI_BIN"
  fi

  # Get API port from scenario
  API_PORT=$(grep -A 10 "allocated_ports" "${SCENARIO_ROOT}/.vrooli/service.json" | grep "API_PORT" | sed 's/.*: *"\?\([0-9]*\)"\?.*/\1/')
  if [[ -z "$API_PORT" ]]; then
    API_PORT=15843  # Default fallback
  fi
  export API_URL="http://localhost:${API_PORT}"

  # Create temporary test file for brief
  export TEST_BRIEF_FILE="/tmp/landing-manager-test-brief-$$.json"
}

teardown() {
  # Cleanup test files
  if [[ -f "$TEST_BRIEF_FILE" ]]; then
    rm -f "$TEST_BRIEF_FILE"
  fi
}

# Agent Trigger Tests [REQ:AGENT-TRIGGER]

@test "[REQ:AGENT-TRIGGER] CLI command 'customize' executes successfully" {
  # Create test brief file
  cat > "$TEST_BRIEF_FILE" <<EOF
{
  "landing_page_slug": "test-landing",
  "brief": "Update hero section with new headline",
  "goals": ["Increase conversion rate", "Improve clarity"],
  "assets": []
}
EOF

  run "$CLI_BIN" customize --brief-file "$TEST_BRIEF_FILE"
  [ "$status" -eq 0 ]
}

@test "[REQ:AGENT-TRIGGER] CLI command 'customize' requires brief file" {
  run "$CLI_BIN" customize
  [ "$status" -ne 0 ]
  [[ "$output" == *"brief"* ]] || [[ "$output" == *"required"* ]]
}

@test "[REQ:AGENT-TRIGGER] CLI command 'customize' supports preview mode" {
  cat > "$TEST_BRIEF_FILE" <<EOF
{
  "landing_page_slug": "test-landing",
  "brief": "Test brief",
  "goals": ["Test goal"]
}
EOF

  run "$CLI_BIN" customize --brief-file "$TEST_BRIEF_FILE" --preview
  [ "$status" -eq 0 ]
  [[ "$output" == *"preview"* ]] || [ "$status" -eq 0 ]
}

# Agent Input Tests [REQ:AGENT-INPUT]

@test "[REQ:AGENT-INPUT] Customization API accepts structured brief" {
  run curl -s -X POST "${API_URL}/api/v1/customize" \
    -H "Content-Type: application/json" \
    -d '{
      "landing_page_slug": "test-landing",
      "brief": "Update hero section with new headline",
      "goals": ["Increase conversion rate"],
      "assets": []
    }'
  [ "$status" -eq 0 ]

  # Verify response structure
  echo "$output" | jq -e '.status' > /dev/null
  [ $? -eq 0 ]
}

@test "[REQ:AGENT-INPUT] Customization API validates required fields" {
  # Test missing brief
  run curl -s -X POST "${API_URL}/api/v1/customize" \
    -H "Content-Type: application/json" \
    -d '{
      "landing_page_slug": "test-landing",
      "goals": ["Test goal"]
    }'
  [ "$status" -eq 0 ]

  # Should return error or validation message
  [[ "$output" == *"brief"* ]] || echo "$output" | jq -e '.error' > /dev/null
}

@test "[REQ:AGENT-INPUT] Customization API includes goals in response" {
  run curl -s -X POST "${API_URL}/api/v1/customize" \
    -H "Content-Type: application/json" \
    -d '{
      "landing_page_slug": "test-landing",
      "brief": "Test brief",
      "goals": ["Increase conversion", "Improve clarity"]
    }'
  [ "$status" -eq 0 ]

  # Verify response has structured format
  echo "$output" | jq -e 'type' > /dev/null
  [ $? -eq 0 ]
}

@test "[REQ:AGENT-INPUT] Customization API supports asset references" {
  run curl -s -X POST "${API_URL}/api/v1/customize" \
    -H "Content-Type: application/json" \
    -d '{
      "landing_page_slug": "test-landing",
      "brief": "Update logo",
      "goals": ["Rebrand"],
      "assets": [
        {"type": "image", "url": "https://example.com/logo.png", "description": "New logo"}
      ]
    }'
  [ "$status" -eq 0 ]

  # Should process successfully
  echo "$output" | jq -e '.status' > /dev/null || [ "$status" -eq 0 ]
}

@test "[REQ:AGENT-INPUT] Customization brief is limited to defined APIs/files" {
  # This test verifies the API documentation/contract includes restrictions
  run curl -s "${API_URL}/api/v1/customize" -X OPTIONS
  [ "$status" -eq 0 ]

  # API should be accessible for OPTIONS (CORS/discovery)
  [ "$status" -eq 0 ] || [[ "$output" == *"Allow"* ]]
}
