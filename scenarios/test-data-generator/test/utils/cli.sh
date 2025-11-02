#!/bin/bash
# Scenario-specific testing helpers for test-data-generator
set -euo pipefail

# Normalise logging so existing phase scripts integrate with the shared helpers.
testing::phase::log() {
  local level="${1:-INFO}"
  shift || true
  local message="$*"

  case "${level^^}" in
    ERROR)
      log::error "$message"
      testing::phase::add_error
      ;;
    WARN|WARNING)
      log::warning "$message"
      testing::phase::add_warning
      ;;
    SUCCESS)
      log::success "$message"
      ;;
    DEBUG)
      log::info "$message"
      ;;
    *)
      log::info "$message"
      ;;
  esac
}

# Ensure the helper is available to child scripts
export -f testing::phase::log
