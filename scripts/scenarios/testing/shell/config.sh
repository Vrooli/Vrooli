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

  # Go configuration
  local go_enabled=$(testing::config::get "unit.languages.go.enabled")
  if [ "$go_enabled" = "false" ]; then
    args+=("--skip-go")
  else
    local go_dir=$(testing::config::get "unit.languages.go.dir")
    [ -n "$go_dir" ] && args+=("--go-dir" "$go_dir")

    # Go coverage thresholds
    local go_warn=$(testing::config::get "unit.languages.go.coverage.warn_threshold")
    local go_error=$(testing::config::get "unit.languages.go.coverage.error_threshold")
    [ -n "$go_warn" ] && args+=("--coverage-warn" "$go_warn")
    [ -n "$go_error" ] && args+=("--coverage-error" "$go_error")
  fi

  # Node configuration
  local node_enabled=$(testing::config::get "unit.languages.node.enabled")
  if [ "$node_enabled" = "false" ]; then
    args+=("--skip-node")
  else
    local node_dir=$(testing::config::get "unit.languages.node.dir")
    [ -n "$node_dir" ] && args+=("--node-dir" "$node_dir")
  fi

  # Python configuration
  local python_enabled=$(testing::config::get "unit.languages.python.enabled")
  if [ "$python_enabled" = "false" ]; then
    args+=("--skip-python")
  else
    local python_dir=$(testing::config::get "unit.languages.python.dir")
    [ -n "$python_dir" ] && args+=("--python-dir" "$python_dir")
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
