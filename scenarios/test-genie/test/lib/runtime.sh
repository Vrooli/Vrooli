#!/usr/bin/env bash
# Shared helpers for Test Genie scenario-local test orchestration.
# Provides consistent logging utilities and assertions used by each phase.

if [[ -n "${TEST_GENIE_RUNTIME_LOADED:-}" ]]; then
  return
fi
TEST_GENIE_RUNTIME_LOADED=1

readonly TEST_GENIE_RUNTIME_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export TEST_GENIE_SCENARIO_DIR="${TEST_GENIE_SCENARIO_DIR:-$(cd "${TEST_GENIE_RUNTIME_DIR}/../.." && pwd)}"
export TEST_GENIE_APP_ROOT="${TEST_GENIE_APP_ROOT:-$(cd "${TEST_GENIE_SCENARIO_DIR}/../.." && pwd)}"

readonly TG_COLOR_INFO="\033[34m"
readonly TG_COLOR_WARN="\033[33m"
readonly TG_COLOR_SUCCESS="\033[32m"
readonly TG_COLOR_ERROR="\033[31m"
readonly TG_COLOR_RESET="\033[0m"

tg::log() {
  local level="$1"
  shift
  local color="$TG_COLOR_INFO"
  case "$level" in
    INFO) color="$TG_COLOR_INFO" ;;
    WARN) color="$TG_COLOR_WARN" ;;
    OK) color="$TG_COLOR_SUCCESS" ;;
    ERROR) color="$TG_COLOR_ERROR" ;;
  esac
  printf "%b[%s]%b %s\n" "$color" "$level" "$TG_COLOR_RESET" "$*"
}

tg::log_info() {
  tg::log "INFO" "$@"
}

tg::log_warn() {
  tg::log "WARN" "$@"
}

tg::log_success() {
  tg::log "OK" "$@"
}

tg::log_error() {
  tg::log "ERROR" "$@"
}

tg::fail() {
  tg::log_error "$@"
  exit 1
}

tg::require_command() {
  local command_name="$1"
  local description="${2:-Command '${command_name}' is required}"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    tg::fail "$description"
  fi
  tg::log_success "Command '${command_name}' detected"
}

tg::require_dir() {
  local dir_path="$1"
  local message="${2:-Directory '${dir_path}' must exist}"
  if [[ ! -d "$dir_path" ]]; then
    tg::fail "$message"
  fi
  tg::log_success "Directory '${dir_path}' verified"
}

tg::require_file() {
  local file_path="$1"
  local message="${2:-File '${file_path}' must exist}"
  if [[ ! -f "$file_path" ]]; then
    tg::fail "$message"
  fi
  tg::log_success "File '${file_path}' verified"
}

tg::require_executable() {
  local file_path="$1"
  local message="${2:-Executable '${file_path}' is required}"
  if [[ ! -x "$file_path" ]]; then
    tg::fail "$message"
  fi
  tg::log_success "Executable '${file_path}' verified"
}

tg::run_in_dir() {
  local dir="$1"
  shift
  if [[ ! -d "$dir" ]]; then
    tg::fail "Cannot run command in missing directory '${dir}'"
  fi
  (cd "$dir" && "$@")
}

tg::read_json_value() {
  local file="$1"
  local jq_expr="$2"
  if ! command -v jq >/dev/null 2>&1; then
    tg::log_warn "jq is not installed; skipping JSON read for ${file}"
    return 1
  fi
  if [[ ! -f "$file" ]]; then
    tg::log_warn "File '${file}' does not exist; cannot read JSON value"
    return 1
  fi
  jq -r "${jq_expr}" "$file" 2>/dev/null
}
