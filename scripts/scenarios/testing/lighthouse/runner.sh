#!/usr/bin/env bash
# Lighthouse test runner for Vrooli scenarios
# Integrates with phase-based testing system and requirements tracking
set -euo pipefail

# This script assumes TESTING_PHASE_* variables are set by phase-helpers.sh
# It provides a single entry point for running Lighthouse audits on scenario UIs

# Main entry point for Lighthouse audits
# Returns 0 if all audits pass, 1 if any fail
lighthouse::run_audits() {
  local lighthouse_only="${1:-false}"

  local config_path="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/lighthouse.json"
  local scenario_name="${TESTING_PHASE_SCENARIO_NAME}"
  local app_root="${TESTING_PHASE_APP_ROOT}"

  if [ ! -f "$config_path" ]; then
    log::warning "No Lighthouse config found at $config_path"
    log::info "To enable Lighthouse testing, create .vrooli/lighthouse.json in your scenario"
    log::info "See docs/testing/guides/lighthouse-integration.md for details"
    testing::phase::add_test skipped
    return 0
  fi

  # Check if Lighthouse testing is enabled in config
  local enabled
  if command -v jq >/dev/null 2>&1; then
    enabled=$(jq -r '.enabled // true' "$config_path")
  else
    log::warning "jq not available; assuming Lighthouse enabled"
    enabled="true"
  fi

  if [ "$enabled" != "true" ]; then
    log::info "Lighthouse testing disabled in config"
    testing::phase::add_test skipped
    return 0
  fi

  # Get UI port for scenario
  local ui_port
  if ! ui_port=$(vrooli scenario port "$scenario_name" UI_PORT 2>/dev/null); then
    log::error "Unable to determine UI_PORT for $scenario_name"
    log::error "Ensure scenario is running with: vrooli scenario start $scenario_name"
    testing::phase::add_test failed
    return 1
  fi

  if [ -z "$ui_port" ] || [ "$ui_port" = "Error" ]; then
    log::error "Invalid UI_PORT returned for $scenario_name: '$ui_port'"
    testing::phase::add_test failed
    return 1
  fi

  local base_url="http://localhost:${ui_port}"
  log::info "Running Lighthouse audits for $scenario_name on $base_url"

  # Prepare output directory
  local output_dir="${TESTING_PHASE_SCENARIO_DIR}/test/artifacts/lighthouse"
  mkdir -p "$output_dir"

  # Check if Node.js is available
  if ! command -v node >/dev/null 2>&1; then
    log::error "Node.js not found; Lighthouse requires Node.js 16+"
    log::error "Install Node.js or skip Lighthouse testing"
    testing::phase::add_test failed
    return 1
  fi

  # Check if Lighthouse dependencies are installed
  local lighthouse_dir="${app_root}/scripts/scenarios/testing/lighthouse"
  if [ ! -d "${lighthouse_dir}/node_modules" ]; then
    log::warning "Lighthouse dependencies not installed"
    log::info "Installing Lighthouse and chrome-launcher..."
    (
      cd "$lighthouse_dir"
      npm install --silent 2>&1 | grep -v "^npm WARN" || true
    )
  fi

  # Run Node.js orchestrator (handles Chrome launch, Lighthouse execution, threshold checking)
  log::info "Launching Lighthouse runner (this may take 1-3 minutes)..."

  local runner_args=(
    --config "$config_path"
    --base-url "$base_url"
    --output-dir "$output_dir"
    --scenario "$scenario_name"
    --phase-results-dir "${TESTING_PHASE_RESULTS_DIR}"
  )

  if [ "$lighthouse_only" = "true" ]; then
    runner_args+=(--lighthouse-only)
  fi

  if node "${lighthouse_dir}/runner.js" "${runner_args[@]}"; then
    log::success "✅ Lighthouse audits passed"
    testing::phase::add_test passed
    return 0
  else
    log::error "❌ Lighthouse audits failed (see $output_dir/)"
    echo ""
    echo "   View HTML reports:"
    # List HTML reports with absolute paths
    local html_files
    html_files=$(find "$output_dir" -name "*.html" -type f 2>/dev/null | sort)
    if [ -n "$html_files" ]; then
      echo "$html_files" | while IFS= read -r html_file; do
        local page_name=$(basename "$html_file" .html | sed 's/_[0-9]\+$//')
        echo "     - $page_name: $html_file"
      done
    fi
    testing::phase::add_test failed
    return 1
  fi
}

# Check if Lighthouse config exists for a scenario
# Returns 0 if config exists, 1 otherwise
lighthouse::has_config() {
  local scenario_dir="${1:-$TESTING_PHASE_SCENARIO_DIR}"
  [ -f "${scenario_dir}/.vrooli/lighthouse.json" ]
}

# Validate Lighthouse config file
# Returns 0 if valid, 1 if invalid
lighthouse::validate_config() {
  local config_path="${1}"

  if [ ! -f "$config_path" ]; then
    echo "Config file not found: $config_path" >&2
    return 1
  fi

  if ! command -v jq >/dev/null 2>&1; then
    echo "jq not available; skipping config validation" >&2
    return 0
  fi

  # Basic validation - check for required fields
  if ! jq -e '.pages' "$config_path" >/dev/null 2>&1; then
    echo "Config missing required 'pages' array" >&2
    return 1
  fi

  # Validate each page has required fields
  local page_count
  page_count=$(jq '.pages | length' "$config_path")

  for ((i=0; i<page_count; i++)); do
    if ! jq -e ".pages[$i].id" "$config_path" >/dev/null 2>&1; then
      echo "Page at index $i missing required 'id' field" >&2
      return 1
    fi
    if ! jq -e ".pages[$i].path" "$config_path" >/dev/null 2>&1; then
      echo "Page at index $i missing required 'path' field" >&2
      return 1
    fi
  done

  return 0
}

# Get list of pages to audit from config
lighthouse::list_pages() {
  local config_path="${1:-${TESTING_PHASE_SCENARIO_DIR}/.vrooli/lighthouse.json}"

  if [ ! -f "$config_path" ]; then
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    jq -r '.pages[] | "\(.id): \(.label // .path)"' "$config_path"
  fi
}

# Export functions for use in phase scripts
if [ "${BASH_SOURCE[0]}" != "${0}" ]; then
  # Script is being sourced, not executed directly
  export -f lighthouse::run_audits 2>/dev/null || true
  export -f lighthouse::has_config 2>/dev/null || true
  export -f lighthouse::validate_config 2>/dev/null || true
  export -f lighthouse::list_pages 2>/dev/null || true
fi
