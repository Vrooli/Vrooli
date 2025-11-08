#!/usr/bin/env bash
# Configuration reader for .vrooli/testing.json
set -euo pipefail

# Read testing config value using jq path notation
# Usage: testing::config::get "unit.languages.go.coverage.warn_threshold"
# Returns: Value from config, or empty string if not found
testing::config::get() {
  local key="$1"
  local config_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/testing.json"

  if [ ! -f "$config_file" ]; then
    echo ""
    return 0
  fi

  jq -r ".$key // empty" "$config_file" 2>/dev/null || echo ""
}

# Convert testing.json config to testing::unit::run_all_tests arguments
# Uses convention-over-configuration:
# - Only languages present in config are tested (absence = skip)
# - Default dirs: go→api, node→ui, python→api
# - Default coverage: warn=80, error=70
# Returns: Array of arguments (one per line) to pass to testing::unit::run_all_tests
testing::config::get_unit_test_args() {
  local config_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/testing.json"

  if [ ! -f "$config_file" ]; then
    echo "❌ ERROR: .vrooli/testing.json not found" >&2
    echo "   Location: $config_file" >&2
    echo "   Create this file to configure unit tests" >&2
    echo "   See: scripts/scenarios/testing/schemas/testing.schema.json" >&2
    return 1
  fi

  local args=()

  # Go configuration - Convention: present = enabled, absent = skip
  local go_present=$(testing::config::get "unit.languages.go")
  if [ -z "$go_present" ] || [ "$go_present" = "null" ]; then
    # No go key = skip
    args+=("--skip-go")
  else
    # Check explicit enabled flag (default true)
    local go_enabled=$(testing::config::get "unit.languages.go.enabled")
    if [ "$go_enabled" = "false" ]; then
      args+=("--skip-go")
    else
      # Apply convention defaults
      local go_dir=$(testing::config::get "unit.languages.go.dir")
      if [ -z "$go_dir" ] || [ "$go_dir" = "null" ]; then
        go_dir="api"  # Convention default
      fi
      args+=("--go-dir" "$go_dir")

      # Go coverage thresholds with convention defaults
      local go_warn=$(testing::config::get "unit.languages.go.coverage.warn_threshold")
      local go_error=$(testing::config::get "unit.languages.go.coverage.error_threshold")
      [ -z "$go_warn" ] || [ "$go_warn" = "null" ] && go_warn="80"  # Convention default
      [ -z "$go_error" ] || [ "$go_error" = "null" ] && go_error="70"  # Convention default
      args+=("--coverage-warn" "$go_warn")
      args+=("--coverage-error" "$go_error")
    fi
  fi

  # Node configuration - Convention: present = enabled, absent = skip
  local node_present=$(testing::config::get "unit.languages.node")
  if [ -z "$node_present" ] || [ "$node_present" = "null" ]; then
    # No node key = skip
    args+=("--skip-node")
  else
    # Check explicit enabled flag (default true)
    local node_enabled=$(testing::config::get "unit.languages.node.enabled")
    if [ "$node_enabled" = "false" ]; then
      args+=("--skip-node")
    else
      # Apply convention default
      local node_dir=$(testing::config::get "unit.languages.node.dir")
      if [ -z "$node_dir" ] || [ "$node_dir" = "null" ]; then
        node_dir="ui"  # Convention default
      fi
      args+=("--node-dir" "$node_dir")
    fi
  fi

  # Python configuration - Convention: present = enabled, absent = skip
  local python_present=$(testing::config::get "unit.languages.python")
  if [ -z "$python_present" ] || [ "$python_present" = "null" ]; then
    # No python key = skip
    args+=("--skip-python")
  else
    # Check explicit enabled flag (default true)
    local python_enabled=$(testing::config::get "unit.languages.python.enabled")
    if [ "$python_enabled" = "false" ]; then
      args+=("--skip-python")
    else
      # Apply convention default
      local python_dir=$(testing::config::get "unit.languages.python.dir")
      if [ -z "$python_dir" ] || [ "$python_dir" = "null" ]; then
        python_dir="api"  # Convention default
      fi
      args+=("--python-dir" "$python_dir")
    fi
  fi

  # Global options
  local verbose=$(testing::config::get "unit.options.verbose")
  [ "$verbose" = "true" ] && args+=("--verbose")

  local fail_fast=$(testing::config::get "unit.options.fail_fast")
  [ "$fail_fast" = "true" ] && args+=("--fail-fast")

  # Output one argument per line for mapfile compatibility
  printf '%s\n' "${args[@]}"
}

export -f testing::config::get
export -f testing::config::get_unit_test_args

testing::config::get_phase_timeout() {
  local phase_name="$1"
  local default_value="${2:-}"
  local config_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/testing.json"

  if [ ! -f "$config_file" ]; then
    printf '%s\n' "$default_value"
    return 0
  fi

  if ! command -v jq >/dev/null 2>&1; then
    printf '%s\n' "$default_value"
    return 0
  fi

  local configured
  configured=$(jq -r ".phases.${phase_name}.timeout // empty" "$config_file" 2>/dev/null || echo "")
  if [ -n "$configured" ] && [ "$configured" != "null" ]; then
    printf '%s\n' "$configured"
  else
    printf '%s\n' "$default_value"
  fi
}

export -f testing::config::get_phase_timeout
