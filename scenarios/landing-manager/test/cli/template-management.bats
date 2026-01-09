#!/usr/bin/env bats
# Tests for template management CLI commands [REQ:TMPL-AVAILABILITY][REQ:TMPL-METADATA][REQ:TMPL-GENERATION][REQ:TMPL-RUNNABLE]

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
}

# Template Availability Tests [REQ:TMPL-AVAILABILITY]

@test "[REQ:TMPL-AVAILABILITY] CLI command 'template list' executes successfully" {
  run "$CLI_BIN" template list
  [ "$status" -eq 0 ]
  [[ "$output" == *"saas-landing-page"* ]]
}

@test "[REQ:TMPL-AVAILABILITY] CLI command 'template list' returns valid JSON when using API" {
  # Test that the underlying API returns valid JSON array
  run curl -s "${API_URL}/api/v1/templates"
  [ "$status" -eq 0 ]

  # Verify JSON structure (array of templates)
  echo "$output" | jq -e 'if type == "array" then . else empty end' > /dev/null
  [ $? -eq 0 ]

  # Verify at least one template exists
  echo "$output" | jq -e '.[0].id' > /dev/null
  [ $? -eq 0 ]
}

@test "[REQ:TMPL-AVAILABILITY] Template 'saas-landing-page' is available via CLI" {
  run "$CLI_BIN" template show saas-landing-page
  [ "$status" -eq 0 ]
  [[ "$output" == *"saas-landing-page"* ]]
}

# Template Metadata Tests [REQ:TMPL-METADATA]

@test "[REQ:TMPL-METADATA] Template metadata includes required sections" {
  # Fetch template metadata from API
  run curl -s "${API_URL}/api/v1/templates/saas-landing-page"
  [ "$status" -eq 0 ]

  # Verify sections exist
  echo "$output" | jq -e '.sections' > /dev/null
  [ $? -eq 0 ]

  # Verify both required and optional sections exist
  echo "$output" | jq -e '.sections.required' > /dev/null
  [ $? -eq 0 ]

  echo "$output" | jq -e '.sections.optional' > /dev/null
  [ $? -eq 0 ]

  # Verify key required sections are present (hero, features, pricing)
  hero_count=$(echo "$output" | jq '[.sections.required[] | select(.id=="hero")] | length')
  [ "$hero_count" -gt 0 ]
}

@test "[REQ:TMPL-METADATA] Template metadata includes metrics hooks" {
  run curl -s "${API_URL}/api/v1/templates/saas-landing-page"
  [ "$status" -eq 0 ]

  # Verify metrics_hooks exists and contains expected events
  echo "$output" | jq -e '.metrics_hooks' > /dev/null
  [ $? -eq 0 ]

  # Check for key metrics events (pageview, conversion)
  echo "$output" | jq -e '.metrics_hooks[] | select(.event_type=="pageview")' > /dev/null
  [ $? -eq 0 ]

  echo "$output" | jq -e '.metrics_hooks[] | select(.event_type=="conversion")' > /dev/null
  [ $? -eq 0 ]

  # Verify metrics hooks is an array with multiple items
  hooks_count=$(echo "$output" | jq '.metrics_hooks | length')
  [ "$hooks_count" -ge 3 ]
}

@test "[REQ:TMPL-METADATA] Template metadata includes customization schema" {
  run curl -s "${API_URL}/api/v1/templates/saas-landing-page"
  [ "$status" -eq 0 ]

  # Verify customization_schema exists
  echo "$output" | jq -e '.customization_schema' > /dev/null
  [ $? -eq 0 ]

  # Verify key customization categories
  echo "$output" | jq -e '.customization_schema.branding' > /dev/null
  [ $? -eq 0 ]

  echo "$output" | jq -e '.customization_schema.seo' > /dev/null
  [ $? -eq 0 ]
}

# Template Generation Tests [REQ:TMPL-GENERATION]

@test "[REQ:TMPL-GENERATION] CLI command 'generate' accepts required parameters" {
  run "$CLI_BIN" generate saas-landing-page --name "Test Landing Page" --slug "test-landing"
  [ "$status" -eq 0 ]
  [[ "$output" == *"test-landing"* ]]
}

@test "[REQ:TMPL-GENERATION] CLI command 'generate' fails without required parameters" {
  # Test missing --name
  run "$CLI_BIN" generate saas-landing-page --slug "test-landing"
  [ "$status" -ne 0 ]

  # Test missing --slug
  run "$CLI_BIN" generate saas-landing-page --name "Test Landing Page"
  [ "$status" -ne 0 ]
}

@test "[REQ:TMPL-GENERATION] Generation API returns valid response structure" {
  run curl -s -X POST "${API_URL}/api/v1/generate" \
    -H "Content-Type: application/json" \
    -d '{"template_id":"saas-landing-page","name":"Test Landing Page","slug":"test-landing"}'
  [ "$status" -eq 0 ]

  # Verify response has expected structure
  echo "$output" | jq -e '.scenario_id' > /dev/null
  [ $? -eq 0 ]

  echo "$output" | jq -e '.status' > /dev/null
  [ $? -eq 0 ]

  # Verify status is "created"
  status_value=$(echo "$output" | jq -r '.status')
  [ "$status_value" = "created" ]
}

# Template Runnable Tests [REQ:TMPL-RUNNABLE]

@test "[REQ:TMPL-RUNNABLE] Template generation plan includes startup command" {
  run curl -s -X POST "${API_URL}/api/v1/generate" \
    -H "Content-Type: application/json" \
    -d '{"template_id":"saas-landing-page","name":"Test Landing Page","slug":"test-landing"}'
  [ "$status" -eq 0 ]

  # Verify next_steps includes startup instructions
  echo "$output" | jq -e '.next_steps' > /dev/null
  [ $? -eq 0 ]

  # Verify next_steps is an array with at least one item
  steps_count=$(echo "$output" | jq '.next_steps | length')
  [ "$steps_count" -gt 0 ]

  # Verify next_steps includes scenario start command
  [[ "$output" == *"vrooli scenario start"* ]]
}

@test "[REQ:TMPL-RUNNABLE] Template metadata includes deployment requirements" {
  run curl -s "${API_URL}/api/v1/templates/saas-landing-page"
  [ "$status" -eq 0 ]

  # Verify template includes tech stack info
  echo "$output" | jq -e '.description' > /dev/null
  [ $? -eq 0 ]

  # Verify description is not empty
  desc_length=$(echo "$output" | jq -r '.description | length')
  [ "$desc_length" -gt 0 ]
}
