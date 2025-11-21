#!/usr/bin/env bash
# Structure validation helper with convention-over-configuration
# Standard scenario structure is tested by default, .vrooli/testing.json defines exceptions
set -euo pipefail

# Source required helpers
SHELL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SHELL_DIR/config.sh"
source "$SHELL_DIR/phase-helpers.sh"
source "$SHELL_DIR/ui-smoke.sh"

_testing_structure_get_mtime() {
  local target="$1"
  local ts

  if ts=$(stat -c %Y "$target" 2>/dev/null) && [[ "$ts" =~ ^[0-9]+$ ]]; then
    printf '%s' "$ts"
    return 0
  fi

  if ts=$(stat -f %m "$target" 2>/dev/null) && [[ "$ts" =~ ^[0-9]+$ ]]; then
    printf '%s' "$ts"
    return 0
  fi

  if command -v python3 >/dev/null 2>&1; then
    python3 - "$target" <<'PY'
import os
import sys

path = sys.argv[1]
try:
    print(int(os.stat(path).st_mtime))
except FileNotFoundError:
    print(0)
PY
    return 0
  fi

  printf '0'
}

# Standard scenario structure (convention)
# All scenarios MUST have these files/directories unless explicitly excluded
declare -ga TESTING_STRUCTURE_STANDARD_FILES=(
  "api/main.go"
  "cli/install.sh"
  "test/run-tests.sh"
  "test/phases/test-structure.sh"
  "test/phases/test-dependencies.sh"
  "test/phases/test-unit.sh"
  "test/phases/test-integration.sh"
  "test/phases/test-business.sh"
  "test/phases/test-performance.sh"
  ".vrooli/service.json"
  ".vrooli/testing.json"
  "README.md"
  "PRD.md"
  "Makefile"
)

declare -ga TESTING_STRUCTURE_STANDARD_DIRS=(
  "api"
  "cli"
  "test"
  "test/phases"
  "requirements"
  "docs"
)

# Main validation function
testing::structure::validate_all() {
  local scenario_name=""
  local summary="Structure validation completed"

  testing::phase::auto_lifecycle_start \
    --phase-name "structure" \
    --default-target-time "120s" \
    --summary "$summary" \
    --config-phase-key "structure" \
    --require-runtime \
    || true

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --scenario)
        scenario_name="$2"
        shift 2
        ;;
      *)
        echo "Unknown option to testing::structure::validate_all: $1" >&2
        testing::phase::auto_lifecycle_end "$summary"
        return 1
        ;;
    esac
  done

  if [ -z "$scenario_name" ]; then
    scenario_name="${TESTING_PHASE_SCENARIO_NAME:-$(basename "$PWD")}"
  fi

  local scenario_dir="${TESTING_PHASE_SCENARIO_DIR:-$PWD}"
  local config_file="$scenario_dir/.vrooli/testing.json"

  echo "🏗️  Validating scenario structure..."
  echo ""

  # Read configuration
  local exclude_files=()
  local exclude_dirs=()
  local additional_files=()
  local additional_dirs=()
  local validate_service_json_name=true
  local check_json_validity=true

  if [ -f "$config_file" ]; then
    # Read exclusions
    if command -v jq >/dev/null 2>&1; then
      mapfile -t exclude_files < <(jq -r '.structure.exclude_files[]? // empty' "$config_file" 2>/dev/null || true)
      mapfile -t exclude_dirs < <(jq -r '.structure.exclude_dirs[]? // empty' "$config_file" 2>/dev/null || true)

      # Read additional checks (simple strings or objects)
      mapfile -t additional_files < <(jq -r '.structure.additional_files[]? | if type == "string" then . else .path end' "$config_file" 2>/dev/null || true)
      mapfile -t additional_dirs < <(jq -r '.structure.additional_dirs[]? | if type == "string" then . else .path end' "$config_file" 2>/dev/null || true)

      # Read validation flags
      local svc_json_check=$(jq -r '.structure.validations.service_json_name_matches_directory // true' "$config_file" 2>/dev/null)
      [ "$svc_json_check" = "false" ] && validate_service_json_name=false

      local json_check=$(jq -r '.structure.validations.check_json_validity // true' "$config_file" 2>/dev/null)
      [ "$json_check" = "false" ] && check_json_validity=false
    fi
  fi

  # Build final file/dir lists
  local files_to_check=()
  local dirs_to_check=()

  # Add standard files, excluding any in exclude list
  for file in "${TESTING_STRUCTURE_STANDARD_FILES[@]}"; do
    local excluded=false
    for excl in "${exclude_files[@]}"; do
      if [ "$file" = "$excl" ]; then
        excluded=true
        break
      fi
    done
    if [ "$excluded" = false ]; then
      # Add scenario name to cli binary path
      if [ "$file" = "cli/install.sh" ]; then
        files_to_check+=("$file")
        files_to_check+=("cli/$scenario_name")
      else
        files_to_check+=("$file")
      fi
    fi
  done

  # Add standard dirs, excluding any in exclude list
  for dir in "${TESTING_STRUCTURE_STANDARD_DIRS[@]}"; do
    local excluded=false
    for excl in "${exclude_dirs[@]}"; do
      if [ "$dir" = "$excl" ]; then
        excluded=true
        break
      fi
    done
    if [ "$excluded" = false ]; then
      dirs_to_check+=("$dir")
    fi
  done

  # Add additional files/dirs
  files_to_check+=("${additional_files[@]}")
  dirs_to_check+=("${additional_dirs[@]}")

  # Validate directories
  if [ ${#dirs_to_check[@]} -gt 0 ]; then
    if testing::phase::check_directories "${dirs_to_check[@]}"; then
      testing::phase::add_test passed
    else
      testing::phase::add_test failed
    fi
  fi

  # Validate files
  if [ ${#files_to_check[@]} -gt 0 ]; then
    if testing::phase::check_files "${files_to_check[@]}"; then
      testing::phase::add_test passed
    else
      testing::phase::add_test failed
    fi
  fi

  # Additional validations
  if [ "$validate_service_json_name" = true ]; then
    _validate_service_json_name "$scenario_name" "$scenario_dir"
  fi

  if [ "$check_json_validity" = true ]; then
    _validate_json_files "$scenario_dir"
  fi

  # Validate playbook-requirement linkage
  _validate_playbook_linkage "$scenario_dir"

  # Ensure playbooks declare reset metadata for reseed orchestration
  _validate_playbook_reset_metadata "$scenario_dir"

  # Reject legacy metadata.requirement fields (BAS playbooks only)
  _validate_playbook_requirement_metadata "$scenario_dir"

  # Validate selector registry wiring when present
  _validate_selector_registry "$scenario_dir"

  # UI smoke harness (auto-skips for non-UI scenarios)
  testing::structure::run_ui_smoke "$scenario_name" "$scenario_dir"

  echo ""
  local total_checks=$((${#dirs_to_check[@]} + ${#files_to_check[@]} + 4))
  echo "✅ Structure validation completed ($total_checks checks)"

  testing::phase::auto_lifecycle_end "$summary"
}

testing::structure::run_ui_smoke() {
  local scenario_name="$1"
  local scenario_dir="$2"

  local helper="${APP_ROOT}/scripts/scenarios/testing/shell/ui-smoke.sh"
  if [[ ! -f "$helper" ]]; then
    testing::phase::add_warning "UI smoke helper missing; skipping"
    return 0
  fi

  if ! command -v jq >/dev/null 2>&1; then
    testing::phase::add_warning "jq not available; skipping UI smoke"
    return 0
  fi

  local summary_json
  local smoke_exit_code=0
  summary_json=$(bash "$helper" --scenario "$scenario_name" --scenario-dir "$scenario_dir" --json) || smoke_exit_code=$?

  if [[ $smoke_exit_code -ne 0 ]]; then
    testing::structure::_log_ui_smoke "$summary_json"

    # Exit code 60 = bundle stale (message already includes restart guidance from ui-smoke.sh)
    # Just mark as failed; the detailed message is already shown
    testing::phase::add_test failed
    return 1
  fi

  local status
  status=$(echo "$summary_json" | jq -r '.status // "failed"' 2>/dev/null || echo "failed")
  testing::structure::_log_ui_smoke "$summary_json"

  case "$status" in
    passed)
      testing::phase::add_test passed
      ;;
    skipped)
      testing::phase::add_test skipped
      ;;
    *)
      testing::phase::add_test failed
      return 1
      ;;
  esac
}

testing::structure::_log_ui_smoke() {
  local summary_json="$1"

  if ! command -v jq >/dev/null 2>&1 || [[ -z "$summary_json" ]]; then
    log::info "UI smoke: summary unavailable"
    return
  fi

  local status message duration screenshot_path
  status=$(echo "$summary_json" | jq -r '.status // "unknown"')
  message=$(echo "$summary_json" | jq -r '.message // ""')
  duration=$(echo "$summary_json" | jq -r '.duration_ms // 0')
  screenshot_path=$(echo "$summary_json" | jq -r '.artifacts.screenshot // empty')

  log::info "UI smoke: ${status} (${duration}ms)"
  if [[ -n "$message" && "$message" != "null" ]]; then
    # Format multiline messages with proper indentation for log::info prefix
    local formatted_message=$(echo "$message" | sed '2,$s/^/      /')
    log::info "  ↳ $formatted_message"
  fi
  if [[ -n "$screenshot_path" && "$screenshot_path" != "null" ]]; then
    log::info "  ↳ Screenshot: $screenshot_path"
  fi
}

# Private helper: Validate service.json name matches directory
_validate_service_json_name() {
  local scenario_name="$1"
  local scenario_dir="$2"
  local service_json="$scenario_dir/.vrooli/service.json"

  if [ ! -f "$service_json" ]; then
    testing::phase::add_warning "service.json not found, skipping name validation"
    return 0
  fi

  if ! command -v jq >/dev/null 2>&1; then
    testing::phase::add_warning "jq not available, skipping service.json name validation"
    return 0
  fi

  local service_name
  service_name=$(jq -r '.service.name // empty' "$service_json" 2>/dev/null)

  if [ -z "$service_name" ]; then
    testing::phase::add_warning "service.json missing service.name field"
    testing::phase::add_test failed
    return 1
  fi

  if [ "$service_name" != "$scenario_name" ]; then
    testing::phase::add_error "service.json name mismatch: expected '$scenario_name', got '$service_name'"
    testing::phase::add_test failed
    return 1
  fi

  log::success "✅ service.json name matches scenario directory"
  testing::phase::add_test passed
  return 0
}

# Private helper: Validate JSON files are valid
_validate_json_files() {
  local scenario_dir="$1"

  if ! command -v jq >/dev/null 2>&1; then
    testing::phase::add_warning "jq not available, skipping JSON validity checks"
    return 0
  fi

  local json_files=()
  local json_file

  # Find common JSON files
  while IFS= read -r json_file; do
    json_files+=("$json_file")
  done < <(find "$scenario_dir" -maxdepth 3 -name "*.json" -type f \
    ! -path "*/node_modules/*" \
    ! -path "*/dist/*" \
    ! -path "*/coverage/*" \
    ! -path "*/.git/*" 2>/dev/null || true)

  if [ ${#json_files[@]} -eq 0 ]; then
    return 0
  fi

  local invalid_count=0
  for json_file in "${json_files[@]}"; do
    local rel_path="${json_file#$scenario_dir/}"
    if ! jq empty < "$json_file" >/dev/null 2>&1; then
      testing::phase::add_error "Invalid JSON: $rel_path"
      invalid_count=$((invalid_count + 1))
    fi
  done

  if [ $invalid_count -gt 0 ]; then
    testing::phase::add_test failed
    return 1
  fi

  log::success "✅ All JSON files are valid (${#json_files[@]} checked)"
  testing::phase::add_test passed
  return 0
}

# Private helper: Validate playbook-requirement linkage
_validate_playbook_linkage() {
  local scenario_dir="$1"
  local app_root="${TESTING_PHASE_APP_ROOT:-${APP_ROOT:-$(cd "$scenario_dir/../.." && pwd)}}"
  local linkage_script="$app_root/scripts/scenarios/testing/shell/validate-playbook-linkage.sh"

  # Skip if playbooks directory doesn't exist
  if [ ! -d "$scenario_dir/test/playbooks" ]; then
    log::success "✅ No playbooks directory (linkage validation skipped)"
    testing::phase::add_test passed
    return 0
  fi

  # Skip if requirements directory doesn't exist
  if [ ! -d "$scenario_dir/requirements" ]; then
    testing::phase::add_warning "Requirements directory not found, but playbooks exist"
    testing::phase::add_test failed
    return 1
  fi

  # Skip if validation script doesn't exist
  if [ ! -f "$linkage_script" ]; then
    testing::phase::add_warning "Playbook linkage validation script not found at $linkage_script"
    testing::phase::add_test passed
    return 0
  fi

  echo ""
  echo "🔗 Validating playbook-requirement linkage..."

  # Run the validation script (suppress output, we'll show summary)
  local validation_output
  local validation_exit_code=0

  if validation_output=$(bash "$linkage_script" "$scenario_dir" 2>&1); then
    # Extract summary from output
    local playbook_count=$(echo "$validation_output" | grep -oP 'Total playbooks: \K\d+' || echo "unknown")
    local requirement_count=$(echo "$validation_output" | grep -oP 'Total requirements: \K\d+' || echo "unknown")
    log::success "✅ All playbooks properly linked ($playbook_count playbooks, $requirement_count requirements)"
    testing::phase::add_test passed
    return 0
  else
    validation_exit_code=$?
    # Show the full validation output on failure
    echo "$validation_output"
    echo ""
    testing::phase::add_error "Playbook-requirement linkage validation failed (exit code: $validation_exit_code)"
    testing::phase::add_test failed
    return 1
  fi
}

_validate_playbook_reset_metadata() {
  local scenario_dir="$1"
  local playbook_root="$scenario_dir/test/playbooks"

  if [ ! -d "$playbook_root" ]; then
    return 0
  fi

  if ! command -v jq >/dev/null 2>&1; then
    testing::phase::add_warning "jq not available; skipping playbook reset validation"
    return 0
  fi

  local -a missing=()
  local -a invalid=()
  while IFS= read -r file_path; do
    [ -z "$file_path" ] && continue
    local reset_value
    reset_value=$(jq -r '.metadata.reset // empty' "$file_path" 2>/dev/null || echo "")
    local rel_path="${file_path#$scenario_dir/}"
    case "$reset_value" in
      none|full)
        :
        ;;
      "")
        missing+=("$rel_path")
        ;;
      *)
        invalid+=("$rel_path:$reset_value")
        ;;
    esac
  done < <(find "$playbook_root" -type f -name '*.json' ! -name 'registry.json' ! -path '*/__seeds/*' | sort)

  if [ ${#missing[@]} -eq 0 ] && [ ${#invalid[@]} -eq 0 ]; then
    log::success "✅ All playbooks declare metadata.reset"
    testing::phase::add_test passed
    return 0
  fi

  if [ ${#missing[@]} -gt 0 ]; then
    log::error "Playbooks missing metadata.reset (${#missing[@]})"
    for rel in "${missing[@]}"; do
      echo "   • ${rel}" >&2
    done
    testing::phase::add_error "${#missing[@]} playbook(s) missing metadata.reset"
  fi

  if [ ${#invalid[@]} -gt 0 ]; then
    log::error "Playbooks with invalid metadata.reset values (${#invalid[@]})"
    for entry in "${invalid[@]}"; do
      echo "   • ${entry}" >&2
    done
    testing::phase::add_error "${#invalid[@]} playbook(s) use invalid reset values"
  fi

  testing::phase::add_test failed
  return 1
}

_validate_playbook_requirement_metadata() {
  local scenario_dir="$1"
  local scenario_name="${TESTING_PHASE_SCENARIO_NAME:-$(basename "$scenario_dir")}"

  # Only Browser Automation Studio has fully migrated requirement metadata
  if [ "$scenario_name" != "browser-automation-studio" ]; then
    return 0
  fi

  local playbook_root="$scenario_dir/test/playbooks"
  if [ ! -d "$playbook_root" ]; then
    return 0
  fi

  if ! command -v jq >/dev/null 2>&1; then
    testing::phase::add_warning "jq not available; skipping metadata.requirement validation"
    return 0
  fi

  local -a flagged=()
  while IFS= read -r file_path; do
    [ -z "$file_path" ] && continue
    local value
    value=$(jq -r '.metadata.requirement // empty' "$file_path" 2>/dev/null || echo "")
    [ -z "$value" ] && continue
    local rel_path="${file_path#$scenario_dir/}"
    flagged+=("$rel_path -> $value")
  done < <(find "$playbook_root" -type f -name '*.json' ! -name 'registry.json' | sort)

  if [ ${#flagged[@]} -eq 0 ]; then
    log::success "✅ metadata.requirement removed from BAS playbooks"
    testing::phase::add_test passed
    return 0
  fi

  log::error "Playbooks still declare deprecated metadata.requirement (${#flagged[@]})"
  for entry in "${flagged[@]}"; do
    echo "   • ${entry}" >&2
  done
  testing::phase::add_error "Remove metadata.requirement from BAS playbooks (use requirements/*.json instead)"
  testing::phase::add_test failed
  return 1
}

_validate_selector_registry() {
  local scenario_dir="$1"
  local selectors_file="$scenario_dir/ui/src/consts/selectors.ts"
  local manifest_file="$scenario_dir/ui/src/consts/selectors.manifest.json"

  if [ ! -f "$selectors_file" ]; then
    log::success "✅ Selector registry not found (skipping registry validation)"
    testing::phase::add_test passed
    return 0
  fi

  if [ ! -f "$manifest_file" ]; then
    testing::phase::add_error "Selector manifest missing at $manifest_file"
    testing::phase::add_test failed
    return 1
  fi

  local selectors_mtime manifest_mtime
  selectors_mtime=$(_testing_structure_get_mtime "$selectors_file")
  manifest_mtime=$(_testing_structure_get_mtime "$manifest_file")
  if [ "$manifest_mtime" -lt "$selectors_mtime" ]; then
    testing::phase::add_error "Selector manifest is outdated. Re-run build-selector-manifest.js"
    testing::phase::add_test failed
    return 1
  fi

  if ! command -v node >/dev/null 2>&1; then
    testing::phase::add_warning "Node.js not available; skipping selector manifest validation"
    testing::phase::add_test passed
    return 0
  fi

  local app_root="${TESTING_PHASE_APP_ROOT:-${APP_ROOT:-$(cd "$scenario_dir/../.." && pwd)}}"
  local registry_script="$app_root/scripts/scenarios/testing/playbooks/selector-registry.js"
  if [ ! -f "$registry_script" ]; then
    testing::phase::add_warning "Selector registry helper missing at $registry_script"
    testing::phase::add_test passed
    return 0
  fi

  if node "$registry_script" --scenario "$scenario_dir" >/dev/null 2>&1; then
    log::success "✅ Selector manifest parsed successfully"
    testing::phase::add_test passed
  else
    testing::phase::add_error "Selector manifest validation failed"
    testing::phase::add_test failed
  fi
}

# Export functions
export -f testing::structure::validate_all
