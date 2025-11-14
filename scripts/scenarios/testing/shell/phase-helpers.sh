#!/usr/bin/env bash
# Phase lifecycle helpers - Eliminates boilerplate from test phase scripts
# Usage: source this file at the beginning of your phase script, then use the helpers
set -euo pipefail

# Global variables for phase state
TESTING_PHASE_NAME=""
TESTING_PHASE_TARGET_TIME=""
TESTING_PHASE_START_TIME=""
TESTING_PHASE_ERROR_COUNT=0
TESTING_PHASE_TEST_COUNT=0
TESTING_PHASE_WARNING_COUNT=0
TESTING_PHASE_SKIPPED_COUNT=0
TESTING_PHASE_SCENARIO_DIR=""
TESTING_PHASE_SCENARIO_NAME=""
TESTING_PHASE_APP_ROOT=""
TESTING_PHASE_CLEANUP_FUNCTIONS=()
TESTING_PHASE_RESULTS_DIR=""
TESTING_PHASE_REQUIREMENTS=()
declare -A TESTING_PHASE_REQUIREMENT_STATUS=()
declare -A TESTING_PHASE_REQUIREMENT_EVIDENCE=()
TESTING_PHASE_EXPECTED_REQUIREMENTS=()
declare -A TESTING_PHASE_EXPECTED_CRITICALITY=()
declare -A TESTING_PHASE_EXPECTED_VALIDATIONS=()

TESTING_PHASE_INITIALIZED=false
TESTING_PHASE_AUTO_MANAGED=false
TESTING_PHASE_AUTO_SUMMARY=""

# Initialize phase environment
# Usage: testing::phase::init [options]
# Options:
#   --phase-name NAME       Override auto-detected phase name
#   --target-time TIME      Target execution time (e.g., "15s", "2m")
#   --skip-runtime-check    Skip runtime availability check
#   --require-runtime       Require runtime to be available
testing::phase::init() {
    local phase_name=""
    local target_time=""
    local skip_runtime_check=false
    local require_runtime=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --phase-name)
                phase_name="$2"
                shift 2
                ;;
            --target-time)
                target_time="$2"
                shift 2
                ;;
            --skip-runtime-check)
                skip_runtime_check=true
                shift
                ;;
            --require-runtime)
                require_runtime=true
                shift
                ;;
            *)
                echo "Unknown option to testing::phase::init: $1" >&2
                return 1
                ;;
        esac
    done
    
    # Auto-detect directories
    local phase_script_path
    phase_script_path="$(testing::phase::_detect_phase_script_path)"
    local phase_script_dir="$(cd "$(dirname "$phase_script_path")" && pwd)"
    TESTING_PHASE_SCENARIO_DIR="$(cd "$phase_script_dir/../.." && pwd)"
    TESTING_PHASE_APP_ROOT="${APP_ROOT:-$(builtin cd "${TESTING_PHASE_SCENARIO_DIR}/../.." && builtin pwd)}"
    export APP_ROOT="$TESTING_PHASE_APP_ROOT"
    TESTING_PHASE_SCENARIO_NAME="$(basename "$TESTING_PHASE_SCENARIO_DIR")"

    # Auto-detect phase name from script filename if not provided
    if [ -z "$phase_name" ]; then
        local script_name="$(basename "${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}")"
        phase_name="${script_name#test-}"
        phase_name="${phase_name%.sh}"
    fi
    TESTING_PHASE_NAME="$phase_name"
    TESTING_PHASE_TARGET_TIME="$target_time"

    TESTING_PHASE_REQUIREMENTS=()
    TESTING_PHASE_REQUIREMENT_STATUS=()
    TESTING_PHASE_REQUIREMENT_EVIDENCE=()
    TESTING_PHASE_EXPECTED_REQUIREMENTS=()
    TESTING_PHASE_EXPECTED_CRITICALITY=()
    TESTING_PHASE_EXPECTED_VALIDATIONS=()
    TESTING_PHASE_EXPECTED_VALIDATIONS=()

    # Source required libraries
    source "${TESTING_PHASE_APP_ROOT}/scripts/lib/utils/log.sh"
    source "${TESTING_PHASE_APP_ROOT}/scripts/scenarios/testing/shell/core.sh"
    source "${TESTING_PHASE_APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"
    if [ -f "${TESTING_PHASE_APP_ROOT}/scripts/scenarios/testing/playbooks/workflow-runner.sh" ]; then
        source "${TESTING_PHASE_APP_ROOT}/scripts/scenarios/testing/playbooks/workflow-runner.sh"
    fi
    
    # Source scenario-specific utilities if they exist
    if [ -f "${TESTING_PHASE_SCENARIO_DIR}/test/utils/cli.sh" ]; then
        source "${TESTING_PHASE_SCENARIO_DIR}/test/utils/cli.sh"
    fi

    testing::phase::_load_expected_requirements "$phase_name"
    
    # Display phase header
    local header="=== ${TESTING_PHASE_NAME^} Phase"
    if [ -n "$TESTING_PHASE_TARGET_TIME" ]; then
        header="$header (Target: <${TESTING_PHASE_TARGET_TIME})"
    fi
    header="$header ==="
    echo "$header"
    
    # Start timing
    TESTING_PHASE_START_TIME=$(date +%s)
    
    # Handle runtime requirements
    if [ "$require_runtime" = true ] && [ "$skip_runtime_check" = false ]; then
        if ! testing::core::ensure_runtime_or_skip "$TESTING_PHASE_SCENARIO_NAME" "${TESTING_PHASE_NAME} tests"; then
            local status=$?
            if [ "$status" -eq 200 ]; then
                exit 200  # Skipped
            else
                exit 1    # Failed
            fi
        fi
    fi
    
    # Change to scenario directory by default
    cd "$TESTING_PHASE_SCENARIO_DIR"

    TESTING_PHASE_RESULTS_DIR="${TESTING_PHASE_SCENARIO_DIR}/coverage/phase-results"
    mkdir -p "$TESTING_PHASE_RESULTS_DIR"

    # Set up cleanup trap
    trap 'testing::phase::cleanup' EXIT
    TESTING_PHASE_INITIALIZED=true
    TESTING_PHASE_AUTO_MANAGED=false
}

# Add a test result
# Usage: testing::phase::add_test [passed|failed|skipped]
testing::phase::add_test() {
    local result="${1:-passed}"
    TESTING_PHASE_TEST_COUNT=$((TESTING_PHASE_TEST_COUNT + 1))
    
    case "$result" in
        failed)
            TESTING_PHASE_ERROR_COUNT=$((TESTING_PHASE_ERROR_COUNT + 1))
            ;;
        skipped)
            TESTING_PHASE_SKIPPED_COUNT=$((TESTING_PHASE_SKIPPED_COUNT + 1))
            ;;
    esac
}

# Add an error (convenience function)
# Usage: testing::phase::add_error [message]
testing::phase::add_error() {
    local message="${1:-}"
    TESTING_PHASE_ERROR_COUNT=$((TESTING_PHASE_ERROR_COUNT + 1))
    if [ -n "$message" ]; then
        log::error "$message"
    fi
}

# Add a warning
# Usage: testing::phase::add_warning [message]
testing::phase::add_warning() {
    local message="${1:-}"
    TESTING_PHASE_WARNING_COUNT=$((TESTING_PHASE_WARNING_COUNT + 1))
    if [ -n "$message" ]; then
        log::warning "$message"
    fi
}

# Associate a requirement outcome with this phase for downstream reporting
# Usage: testing::phase::add_requirement --id REQUIREMENT_ID [--status passed] [--evidence "context"]
testing::phase::add_requirement() {
    local requirement_id=""
    local status="passed"
    local evidence=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id)
                requirement_id="$2"
                shift 2
                ;;
            --status)
                status="$2"
                shift 2
                ;;
            --evidence)
                evidence="$2"
                shift 2
                ;;
            *)
                echo "Unknown option to testing::phase::add_requirement: $1" >&2
                return 1
                ;;
        esac
    done

    if [ -z "$requirement_id" ]; then
        echo "testing::phase::add_requirement requires --id" >&2
        return 1
    fi

    local seen=false
    for existing in "${TESTING_PHASE_REQUIREMENTS[@]}"; do
        if [ "$existing" = "$requirement_id" ]; then
            seen=true
            break
        fi
    done

    if [ "$seen" = false ]; then
        TESTING_PHASE_REQUIREMENTS+=("$requirement_id")
    fi

    TESTING_PHASE_REQUIREMENT_STATUS["$requirement_id"]="$status"
    TESTING_PHASE_REQUIREMENT_EVIDENCE["$requirement_id"]="$evidence"
}

# Check a condition and report result
# Usage: testing::phase::check "description" command [args...]
# Example: testing::phase::check "API health endpoint" curl -sf "$API_URL/health"
testing::phase::check() {
    local description="$1"
    shift
    
    echo -n "🔍 Checking $description... "
    if "$@" >/dev/null 2>&1; then
        log::success "✅ Passed"
        testing::phase::add_test passed
        return 0
    else
        log::error "❌ Failed"
        testing::phase::add_test failed
        return 1
    fi
}

# Check for required files
# Usage: testing::phase::check_files file1 file2 ...
testing::phase::check_files() {
    local files=("$@")
    local missing_files=()
    
    echo "🔍 Checking required files..."
    for file in "${files[@]}"; do
        if [ ! -f "$file" ]; then
            missing_files+=("$file")
            TESTING_PHASE_ERROR_COUNT=$((TESTING_PHASE_ERROR_COUNT + 1))
        fi
    done
    
    if [ ${#missing_files[@]} -gt 0 ]; then
        log::error "❌ Missing required files:"
        printf "   - %s\n" "${missing_files[@]}"
        return 1
    else
        log::success "✅ All required files present"
        return 0
    fi
}

# Check for required directories
# Usage: testing::phase::check_directories dir1 dir2 ...
testing::phase::check_directories() {
    local dirs=("$@")
    local missing_dirs=()
    
    echo "🔍 Checking required directories..."
    for dir in "${dirs[@]}"; do
        if [ ! -d "$dir" ]; then
            missing_dirs+=("$dir")
            TESTING_PHASE_ERROR_COUNT=$((TESTING_PHASE_ERROR_COUNT + 1))
        fi
    done
    
    if [ ${#missing_dirs[@]} -gt 0 ]; then
        log::error "❌ Missing required directories:"
        printf "   - %s\n" "${missing_dirs[@]}"
        return 1
    else
        log::success "✅ All required directories present"
        return 0
    fi
}

# Register a cleanup function
# Usage: testing::phase::register_cleanup function_name
testing::phase::register_cleanup() {
    local func="$1"
    TESTING_PHASE_CLEANUP_FUNCTIONS+=("$func")
}

# Run cleanup functions
testing::phase::cleanup() {
    for func in "${TESTING_PHASE_CLEANUP_FUNCTIONS[@]}"; do
        if declare -f "$func" >/dev/null; then
            "$func" || true
        fi
    done
}

# End phase with summary and exit
# Usage: testing::phase::end_with_summary [custom_message]
testing::phase::end_with_summary() {
    local custom_message="${1:-}"
    local end_time=$(date +%s)
    local duration=$((end_time - TESTING_PHASE_START_TIME))
    
    echo ""
    
    # Build summary message
    local summary_parts=()
    if [ $TESTING_PHASE_TEST_COUNT -gt 0 ]; then
        summary_parts+=("$TESTING_PHASE_TEST_COUNT tests")
    fi
    if [ $TESTING_PHASE_ERROR_COUNT -gt 0 ]; then
        summary_parts+=("$TESTING_PHASE_ERROR_COUNT errors")
    fi
    if [ $TESTING_PHASE_WARNING_COUNT -gt 0 ]; then
        summary_parts+=("$TESTING_PHASE_WARNING_COUNT warnings")  
    fi
    if [ $TESTING_PHASE_SKIPPED_COUNT -gt 0 ]; then
        summary_parts+=("$TESTING_PHASE_SKIPPED_COUNT skipped")
    fi
    if [ ${#TESTING_PHASE_REQUIREMENTS[@]} -gt 0 ]; then
        summary_parts+=("${#TESTING_PHASE_REQUIREMENTS[@]} requirements")
    fi
    
    local summary=""
    if [ ${#summary_parts[@]} -gt 0 ]; then
        summary=" ($(IFS=', '; echo "${summary_parts[*]}"))"
    fi

    local missing_requirements=()
    if [ ${#TESTING_PHASE_EXPECTED_REQUIREMENTS[@]} -gt 0 ]; then
        for expected_id in "${TESTING_PHASE_EXPECTED_REQUIREMENTS[@]}"; do
            if [ -z "${TESTING_PHASE_REQUIREMENT_STATUS["$expected_id"]+x}" ]; then
                missing_requirements+=("$expected_id")
                TESTING_PHASE_REQUIREMENT_STATUS["$expected_id"]="not_run"
                TESTING_PHASE_REQUIREMENT_EVIDENCE["$expected_id"]="No results recorded for phase ${TESTING_PHASE_NAME}"

                local already_listed=false
                for existing_id in "${TESTING_PHASE_REQUIREMENTS[@]}"; do
                    if [ "$existing_id" = "$expected_id" ]; then
                        already_listed=true
                        break
                    fi
                done
                if [ "$already_listed" = false ]; then
                    TESTING_PHASE_REQUIREMENTS+=("$expected_id")
                fi
            fi
        done
    fi

    if [ ${#missing_requirements[@]} -gt 0 ]; then
        local enforce_requirements="${TESTING_REQUIREMENTS_ENFORCE:-${VROOLI_REQUIREMENTS_ENFORCE:-0}}"
        local formatted_list
        formatted_list=$(IFS=', '; echo "${missing_requirements[*]}")
        if [ "$enforce_requirements" = "1" ]; then
            testing::phase::add_error "Expected requirements not covered in phase ${TESTING_PHASE_NAME}: ${formatted_list}"
        else
            testing::phase::add_warning "Expected requirements missing coverage in phase ${TESTING_PHASE_NAME}: ${formatted_list}"
        fi
    fi

    # Check if we exceeded target time
    local time_warning=""
    if [ -n "$TESTING_PHASE_TARGET_TIME" ]; then
        local target_seconds
        if [[ "$TESTING_PHASE_TARGET_TIME" =~ ^([0-9]+)s$ ]]; then
            target_seconds="${BASH_REMATCH[1]}"
        elif [[ "$TESTING_PHASE_TARGET_TIME" =~ ^([0-9]+)m$ ]]; then
            target_seconds=$((BASH_REMATCH[1] * 60))
        else
            target_seconds=0
        fi
        
        if [ $target_seconds -gt 0 ] && [ $duration -gt $target_seconds ]; then
            time_warning=" ⚠️  Exceeded ${TESTING_PHASE_TARGET_TIME} target"
        fi
    fi
    
    local phase_status="passed"
    if [ $TESTING_PHASE_ERROR_COUNT -gt 0 ]; then
        phase_status="failed"
    fi

    if [ -n "$TESTING_PHASE_RESULTS_DIR" ]; then
        local results_file="${TESTING_PHASE_RESULTS_DIR}/${TESTING_PHASE_NAME}.json"
        local iso_timestamp=$(date -Iseconds)
        mkdir -p "$(dirname "$results_file")"
        local requirements_json="[]"
        if [ ${#TESTING_PHASE_REQUIREMENTS[@]} -gt 0 ]; then
            local first=true
            requirements_json="["
            for requirement_id in "${TESTING_PHASE_REQUIREMENTS[@]}"; do
                local requirement_status="${TESTING_PHASE_REQUIREMENT_STATUS["$requirement_id"]:-unknown}"
                local requirement_evidence="${TESTING_PHASE_REQUIREMENT_EVIDENCE["$requirement_id"]:-}"
                local requirement_criticality="${TESTING_PHASE_EXPECTED_CRITICALITY["$requirement_id"]:-}"
                local escaped_id="$(testing::phase::_json_escape "$requirement_id")"
                local escaped_status="$(testing::phase::_json_escape "$requirement_status")"
                local entry="{\"id\":\"${escaped_id}\",\"status\":\"${escaped_status}\""
                if [ -n "$requirement_criticality" ]; then
                    local escaped_criticality="$(testing::phase::_json_escape "$requirement_criticality")"
                    entry="${entry},\"criticality\":\"${escaped_criticality}\""
                fi
                if [ -n "$requirement_evidence" ]; then
                    local escaped_evidence="$(testing::phase::_json_escape "$requirement_evidence")"
                    entry="${entry},\"evidence\":\"${escaped_evidence}\""
                fi
                entry="${entry}}"
                if [ "$first" = true ]; then
                    requirements_json="${requirements_json}${entry}"
                    first=false
                else
                    requirements_json="${requirements_json},${entry}"
                fi
            done
            requirements_json="${requirements_json}]"
        fi
        cat <<JSON >"$results_file"
{
  "phase": "${TESTING_PHASE_NAME}",
  "scenario": "${TESTING_PHASE_SCENARIO_NAME}",
  "status": "${phase_status}",
  "tests": ${TESTING_PHASE_TEST_COUNT},
  "errors": ${TESTING_PHASE_ERROR_COUNT},
  "warnings": ${TESTING_PHASE_WARNING_COUNT},
  "skipped": ${TESTING_PHASE_SKIPPED_COUNT},
  "duration_seconds": ${duration},
  "target": "${TESTING_PHASE_TARGET_TIME}",
  "updated_at": "${iso_timestamp}",
  "requirements": ${requirements_json}
}
JSON
    fi

    # Display final status
    if [ "$phase_status" = "passed" ]; then
        if [ -n "$custom_message" ]; then
            log::success "✅ $custom_message in ${duration}s${summary}${time_warning}"
        else
            log::success "✅ ${TESTING_PHASE_NAME^} phase completed successfully in ${duration}s${summary}${time_warning}"
        fi
        TESTING_PHASE_INITIALIZED=false
        TESTING_PHASE_AUTO_MANAGED=false
        TESTING_PHASE_AUTO_SUMMARY=""
        exit 0
    else
        if [ -n "$custom_message" ]; then
            log::error "❌ $custom_message in ${duration}s${summary}${time_warning}"
        else
            log::error "❌ ${TESTING_PHASE_NAME^} phase failed in ${duration}s${summary}${time_warning}"
        fi
        TESTING_PHASE_INITIALIZED=false
        TESTING_PHASE_AUTO_MANAGED=false
        TESTING_PHASE_AUTO_SUMMARY=""
        exit 1
    fi
}

# Measure and report execution time for a command
# Usage: testing::phase::timed_exec "description" command [args...]
testing::phase::timed_exec() {
    local description="$1"
    shift
    
    echo "⏱️  $description..."
    local start=$(date +%s)
    
    if "$@"; then
        local end=$(date +%s)
        local duration=$((end - start))
        log::success "✅ Completed in ${duration}s"
        return 0
    else
        local end=$(date +%s)
        local duration=$((end - start))
        log::error "❌ Failed after ${duration}s"
        return 1
    fi
}

testing::phase::_load_expected_requirements() {
    local phase_name="${1:-}"
    TESTING_PHASE_EXPECTED_REQUIREMENTS=()
    TESTING_PHASE_EXPECTED_CRITICALITY=()

    local reporter_path="${TESTING_PHASE_APP_ROOT}/scripts/requirements/report.js"
    if [ ! -f "$reporter_path" ]; then
        return
    fi

    if ! command -v node >/dev/null 2>&1; then
        log::warning "Node.js not available; skipping requirement inspection for phase ${phase_name}"
        return
    fi

    local inspect_output=""
    if ! inspect_output=$(cd "$TESTING_PHASE_SCENARIO_DIR" && node "$reporter_path" --scenario "$TESTING_PHASE_SCENARIO_NAME" --mode phase-inspect --phase "$phase_name" 2>/dev/null); then
        log::warning "Unable to inspect requirements for phase ${phase_name}" >&2
        return
    fi

    if ! command -v jq >/dev/null 2>&1; then
        log::warning "jq not available; skipping requirement expectation checks for phase ${phase_name}"
        return
    fi

    mapfile -t TESTING_PHASE_EXPECTED_REQUIREMENTS < <(printf '%s\n' "$inspect_output" | jq -r '.requirements[].id' | sort -u)

    if [ ${#TESTING_PHASE_EXPECTED_REQUIREMENTS[@]} -eq 0 ]; then
        return
    fi

    local entry
    while IFS= read -r entry; do
        local req_id
        local criticality
        req_id=$(printf '%s\n' "$entry" | jq -r '.id')
        criticality=$(printf '%s\n' "$entry" | jq -r '.criticality // ""')
        TESTING_PHASE_EXPECTED_CRITICALITY["$req_id"]="$criticality"
        local validations_json
        validations_json=$(printf '%s\n' "$entry" | jq -c '.validations // []')
        TESTING_PHASE_EXPECTED_VALIDATIONS["$req_id"]="$validations_json"
    done < <(printf '%s\n' "$inspect_output" | jq -c '.requirements[]')

    local missing_refs
    missing_refs=$(printf '%s\n' "$inspect_output" | jq -r '.missingReferences[]?' 2>/dev/null || true)
    if [ -n "$missing_refs" ]; then
        log::warning "Requirement references declared for phase ${phase_name} are missing on disk:"
        while IFS= read -r missing_ref; do
            [ -n "$missing_ref" ] && printf '   - %s\n' "$missing_ref"
        done <<< "$missing_refs"
    fi
}

testing::phase::expected_validations_for() {
    local requirement_id="$1"
    if [ -z "$requirement_id" ]; then
        echo "[]"
        return 0
    fi
    local payload="${TESTING_PHASE_EXPECTED_VALIDATIONS["$requirement_id"]:-[]}" 
    if [ -z "$payload" ]; then
        echo "[]"
    else
        printf '%s\n' "$payload"
    fi
}

# Execute workflow validations declared for this phase using browser-automation-studio.
# This function is generic and can be used by any scenario to run BAS workflow JSONs for testing.
# Options:
#   --scenario NAME         Target scenario to test (default: browser-automation-studio)
#   --manage-runtime MODE   Pass-through to workflow runner (auto|start|skip)
#   --disallow-missing      Treat missing workflow files as failures instead of skips
testing::phase::run_workflow_validations() {
    local default_scenario="browser-automation-studio"
    local manage_runtime="auto"
    local allow_missing=true

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --scenario)
                default_scenario="$2"
                shift 2
                ;;
            --manage-runtime)
                manage_runtime="$2"
                shift 2
                ;;
            --disallow-missing)
                allow_missing=false
                shift
                ;;
            *)
                echo "Unknown option to testing::phase::run_workflow_validations: $1" >&2
                return 1
                ;;
        esac
    done

    if ! declare -F testing::playbooks::run_workflow >/dev/null 2>&1; then
        log::warning "Workflow runner not available; skip workflow validations"
        return 200
    fi

    if ! command -v jq >/dev/null 2>&1; then
        log::warning "jq missing; cannot parse automation validation metadata"
        return 200
    fi

    local bas_cli_available=true
    if ! command -v browser-automation-studio >/dev/null 2>&1; then
        bas_cli_available=false
        log::warning "browser-automation-studio CLI missing; automation validations will be skipped"
    fi

    local overall_status=0
    local req_id
    for req_id in "${TESTING_PHASE_EXPECTED_REQUIREMENTS[@]}"; do
        local validations_json
        validations_json=$(testing::phase::expected_validations_for "$req_id")
        if [ -z "$validations_json" ] || [ "$validations_json" = "[]" ]; then
            continue
        fi

        local automation_entries
        automation_entries=$(printf '%s\n' "$validations_json" | jq -c '.[] | select((.type // "") == "automation")' 2>/dev/null || true)
        if [ -z "$automation_entries" ]; then
            continue
        fi

        while IFS= read -r entry; do
            [ -z "$entry" ] && continue

            local ref
            ref=$(printf '%s\n' "$entry" | jq -r '.ref // ""')
            if [ -z "$ref" ] || [ "$ref" = "null" ]; then
                log::warning "Automation validation for $req_id is missing a ref"
                continue
            fi

            local target_scenario
            target_scenario=$(printf '%s\n' "$entry" | jq -r '.scenario // ""')
            if [ -z "$target_scenario" ] || [ "$target_scenario" = "null" ]; then
                target_scenario="$default_scenario"
            fi

            local workflow_label="$ref"
            local folder
            folder=$(printf '%s\n' "$entry" | jq -r '.folder // ""')
            if [ "$folder" = "null" ]; then
                folder=""
            fi

            local workflow_id
            workflow_id=$(printf '%s\n' "$entry" | jq -r '.workflow_id // ""')
            if [ "$workflow_id" = "null" ]; then
                workflow_id=""
            fi

            local validation_timeout
            validation_timeout=$(printf '%s\n' "$entry" | jq -r '.timeout_seconds // ""')
            if [ "$validation_timeout" = "null" ]; then
                validation_timeout=""
            fi
            local validation_timeout_int=0
            if [[ "$validation_timeout" =~ ^[0-9]+$ ]]; then
                validation_timeout_int="$validation_timeout"
            fi

            local validation_manage_runtime
            validation_manage_runtime=$(printf '%s\n' "$entry" | jq -r '.manage_runtime // ""')
            if [ "$validation_manage_runtime" = "null" ]; then
                validation_manage_runtime=""
            fi

            local validation_keep_workflow
            validation_keep_workflow=$(printf '%s\n' "$entry" | jq -r '.keep_workflow // ""')
            if [ "$validation_keep_workflow" = "null" ]; then
                validation_keep_workflow=""
            fi

            local validation_allow_missing
            validation_allow_missing=$(printf '%s\n' "$entry" | jq -r '.allow_missing // ""')
            if [ "$validation_allow_missing" = "null" ]; then
                validation_allow_missing=""
            fi

            if [ "$bas_cli_available" = false ]; then
                testing::phase::add_warning "BAS CLI unavailable; skipping $workflow_label"
                testing::phase::add_test skipped
                testing::phase::add_requirement --id "$req_id" --status skipped --evidence "BAS CLI unavailable for $workflow_label"
                continue
            fi

            local args=(--scenario "$target_scenario")
            local runtime_mode="$manage_runtime"
            if [ -n "$validation_manage_runtime" ]; then
                runtime_mode="$validation_manage_runtime"
            fi
            args+=(--manage-runtime "$runtime_mode")

            local effective_allow_missing="$allow_missing"
            if [ -n "$validation_allow_missing" ]; then
                if [[ "$validation_allow_missing" =~ ^(true|1)$ ]]; then
                    effective_allow_missing=true
                else
                    effective_allow_missing=false
                fi
            fi

            if [ -n "$workflow_id" ]; then
                args+=(--workflow-id "$workflow_id")
                workflow_label="${workflow_label:-$workflow_id}"
            elif [ -n "$ref" ]; then
                args+=(--file "$ref")
                if [ "$effective_allow_missing" = true ]; then
                    args+=(--allow-missing)
                fi
            fi

            if [ -n "$folder" ]; then
                args+=(--folder "$folder")
            fi
            if [ -n "$validation_timeout" ]; then
                args+=(--timeout "$validation_timeout")
            fi
            if [[ "$validation_keep_workflow" =~ ^(true|1)$ ]]; then
                args+=(--keep-workflow)
            fi

            if [ "$validation_timeout_int" -gt 90 ]; then
                if command -v log::warning >/dev/null 2>&1; then
                    log::warning "⚠️ [WF_TIMEOUT_HIGH] Workflow ${workflow_label} configured with timeout ${validation_timeout_int}s (>90s)"
                else
                    echo "⚠️ [WF_TIMEOUT_HIGH] Workflow ${workflow_label} configured with timeout ${validation_timeout_int}s (>90s)"
                fi
            fi

            if testing::playbooks::run_workflow "${args[@]}"; then
                testing::phase::add_test passed
                local evidence="Workflow ${workflow_label} executed"
                if [ -n "${TESTING_PLAYBOOKS_LAST_EXECUTION_ID:-}" ]; then
                    evidence+=" (execution ${TESTING_PLAYBOOKS_LAST_EXECUTION_ID})"
                fi
                if [ -n "${TESTING_PLAYBOOKS_LAST_SCENARIO:-}" ]; then
                    evidence+=" via ${TESTING_PLAYBOOKS_LAST_SCENARIO}"
                fi
                testing::phase::add_requirement --id "$req_id" --status passed --evidence "$evidence"
            else
                local rc=$?
                if [ "$effective_allow_missing" = true ] && [ "$rc" -eq 210 ]; then
                    testing::phase::add_warning "Workflow ${workflow_label} not found; export pending"
                    testing::phase::add_test skipped
                    testing::phase::add_requirement --id "$req_id" --status skipped --evidence "Workflow ${workflow_label} missing"
                else
                    testing::phase::add_test failed
                    testing::phase::add_requirement --id "$req_id" --status failed --evidence "Workflow ${workflow_label} failed"
                    overall_status=1
                fi
            fi
        done <<< "$automation_entries"
    done

    return $overall_status
}

testing::phase::_json_escape() {
    local value="$1"
    value="${value//\\/\\\\}"
    value="${value//\"/\\\"}"
    value="${value//$'\n'/\\n}"
    value="${value//$'\r'/\\r}"
    value="${value//$'\t'/\\t}"
    value="${value//$'\f'/\\f}"
    value="${value//$'\b'/\\b}"
    printf '%s' "$value"
}

testing::phase::run_workflow_yaml() {
    local workflow_path=""
    local label=""
    local requirement_id=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                workflow_path="$2"
                shift 2
                ;;
            --label)
                label="$2"
                shift 2
                ;;
            --requirement)
                requirement_id="$2"
                shift 2
                ;;
            *)
                echo "Unknown option to testing::phase::run_workflow_yaml: $1" >&2
                return 1
                ;;
        esac
    done

    if [ -z "$workflow_path" ]; then
        echo "testing::phase::run_workflow_yaml requires --file" >&2
        return 1
    fi

    local absolute_path
    absolute_path="$workflow_path"
    if [ ! -f "$absolute_path" ]; then
        absolute_path="${TESTING_PHASE_SCENARIO_DIR}/${workflow_path}"
    fi

    if [ ! -f "$absolute_path" ] && [ -n "${TESTING_PHASE_APP_ROOT:-}" ]; then
        absolute_path="${TESTING_PHASE_APP_ROOT}/${workflow_path#./}"
    fi

    if [ ! -f "$absolute_path" ]; then
        log::error "Workflow definition not found: $workflow_path"
        return 1
    fi

    local workflow_name
    workflow_name=$(grep -E '^[[:space:]]*name:' "$absolute_path" | head -1 | sed -E 's/^[[:space:]]*name:[[:space:]]*//')
    if [ -z "$workflow_name" ]; then
        workflow_name="$(basename "$workflow_path" .yaml)"
    fi

    local workflow_command
    workflow_command=$(grep -E '^[[:space:]]*command:' "$absolute_path" | head -1 | sed -E 's/^[[:space:]]*command:[[:space:]]*//')
    if [ -z "$workflow_command" ]; then
        log::error "Workflow ${workflow_name} does not define a command"
        return 1
    fi

    local workflow_timeout
    workflow_timeout=$(grep -E '^[[:space:]]*timeout:' "$absolute_path" | head -1 | sed -E 's/^[[:space:]]*timeout:[[:space:]]*//')

    local success_pattern
    success_pattern=$(grep -E '^[[:space:]]*pattern:' "$absolute_path" | head -1 | sed -E 's/^[[:space:]]*pattern:[[:space:]]*//; s/^"//; s/"$//')

    local display_label="$label"
    if [ -z "$display_label" ]; then
        display_label="$workflow_name"
    fi

    local log_dir="${TESTING_PHASE_SCENARIO_DIR}/coverage/automation"
    mkdir -p "$log_dir"
    local log_file="$log_dir/$(basename "$workflow_path" .yaml).log"

    echo "▶ Running workflow ${display_label}"

    local timeout_cmd=()
    if [ -n "$workflow_timeout" ] && command -v timeout >/dev/null 2>&1; then
        timeout_cmd=(timeout "$workflow_timeout")
    fi

    local command_exit=0
    (
        cd "$TESTING_PHASE_SCENARIO_DIR"
        set +e
        "${timeout_cmd[@]}" bash -lc "$workflow_command" >"$log_file" 2>&1
        command_exit=$?
        set -e
        exit $command_exit
    )
    command_exit=$?

    if [ $command_exit -ne 0 ]; then
        log::error "Workflow ${display_label} failed (exit code $command_exit). Log: $log_file"
        if [ -n "$requirement_id" ]; then
            testing::phase::add_requirement --id "$requirement_id" --status failed --evidence "Workflow ${display_label}"
        fi
        return $command_exit
    fi

    if [ -n "$success_pattern" ]; then
        if ! grep -Fq "$success_pattern" "$log_file"; then
            log::error "Workflow ${display_label} did not emit expected pattern '$success_pattern'. Log: $log_file"
            if [ -n "$requirement_id" ]; then
                testing::phase::add_requirement --id "$requirement_id" --status failed --evidence "Workflow ${display_label}"
            fi
            return 1
        fi
    fi

    log::success "Workflow ${display_label} completed (${log_file})"

    if [ -n "$requirement_id" ]; then
        testing::phase::add_requirement --id "$requirement_id" --status passed --evidence "Workflow ${display_label}"
    fi

    return 0
}

# Export functions for use by phase scripts
export -f testing::phase::init
export -f testing::phase::add_test
export -f testing::phase::add_error
export -f testing::phase::add_warning
export -f testing::phase::add_requirement
export -f testing::phase::check
export -f testing::phase::check_files
export -f testing::phase::check_directories
export -f testing::phase::register_cleanup
export -f testing::phase::cleanup
export -f testing::phase::end_with_summary
export -f testing::phase::timed_exec
export -f testing::phase::expected_validations_for
export -f testing::phase::run_workflow_validations
# Backward compatibility alias
testing::phase::run_bas_automation_validations() { testing::phase::run_workflow_validations "$@"; }
export -f testing::phase::run_bas_automation_validations

# --- Lifecycle automation helpers ---

testing::phase::_detect_phase_script_path() {
    local idx
    for ((idx=${#BASH_SOURCE[@]}-1; idx>=0; idx--)); do
        local candidate="${BASH_SOURCE[$idx]}"
        if [[ "$candidate" == */test/phases/test-* ]]; then
            printf '%s\n' "$candidate"
            return 0
        fi
    done

    if [ ${#BASH_SOURCE[@]} -gt 0 ]; then
        printf '%s\n' "${BASH_SOURCE[${#BASH_SOURCE[@]}-1]}"
    fi
}

testing::phase::_detect_scenario_dir() {
    local script_path
    script_path="$(testing::phase::_detect_phase_script_path)"
    if [ -z "$script_path" ]; then
        printf '%s\n' "$(pwd)"
        return 0
    fi

    local script_dir
    script_dir="$(cd "$(dirname "$script_path")" && pwd)"
    if [ -z "$script_dir" ]; then
        printf '%s\n' "$(pwd)"
        return 0
    fi

    (
        cd "$script_dir/../.." >/dev/null 2>&1 && pwd
    )
}

testing::phase::_read_phase_timeout() {
    local phase_key="${1:-}"
    if [ -z "$phase_key" ]; then
        return 0
    fi

    local scenario_dir
    scenario_dir="$(testing::phase::_detect_scenario_dir)"
    if [ -z "$scenario_dir" ]; then
        return 0
    fi

    local config_file="$scenario_dir/.vrooli/testing.json"
    if [ ! -f "$config_file" ]; then
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        return 0
    fi

    local configured
    configured=$(jq -r ".phases.${phase_key}.timeout // empty" "$config_file" 2>/dev/null || echo "")
    if [ -n "$configured" ] && [ "$configured" != "null" ]; then
        printf '%s\n' "$configured"
    fi
}

testing::phase::auto_lifecycle_start() {
    local phase_name=""
    local default_target=""
    local summary=""
    local require_runtime=false
    local skip_runtime_check=false
    local config_phase_key=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --phase-name)
                phase_name="$2"
                shift 2
                ;;
            --default-target-time)
                default_target="$2"
                shift 2
                ;;
            --summary)
                summary="$2"
                shift 2
                ;;
            --require-runtime)
                require_runtime=true
                shift
                ;;
            --skip-runtime-check)
                skip_runtime_check=true
                shift
                ;;
            --config-phase-key)
                config_phase_key="$2"
                shift 2
                ;;
            *)
                echo "Unknown option to testing::phase::auto_lifecycle_start: $1" >&2
                return 1
                ;;
        esac
    done

    if [ "$TESTING_PHASE_INITIALIZED" = true ]; then
        TESTING_PHASE_AUTO_MANAGED=false
        return 1
    fi

    local resolved_target="$default_target"
    local configured_target=""
    if [ -n "$config_phase_key" ]; then
        configured_target="$(testing::phase::_read_phase_timeout "$config_phase_key")"
    fi
    if [ -n "$configured_target" ]; then
        resolved_target="$configured_target"
    fi

    local init_args=()
    if [ -n "$phase_name" ]; then
        init_args+=(--phase-name "$phase_name")
    fi
    if [ -n "$resolved_target" ]; then
        init_args+=(--target-time "$resolved_target")
    fi
    if [ "$require_runtime" = true ]; then
        init_args+=(--require-runtime)
    fi
    if [ "$skip_runtime_check" = true ]; then
        init_args+=(--skip-runtime-check)
    fi

    testing::phase::init "${init_args[@]}"

    TESTING_PHASE_AUTO_MANAGED=true
    TESTING_PHASE_AUTO_SUMMARY="$summary"
    return 0
}

testing::phase::auto_lifecycle_end() {
    local summary_override="${1:-}"
    if [ "$TESTING_PHASE_AUTO_MANAGED" = true ]; then
        local summary_to_use="$summary_override"
        if [ -z "$summary_to_use" ]; then
            summary_to_use="$TESTING_PHASE_AUTO_SUMMARY"
        fi
        TESTING_PHASE_AUTO_MANAGED=false
        testing::phase::end_with_summary "$summary_to_use"
    fi
}

export -f testing::phase::auto_lifecycle_start
export -f testing::phase::auto_lifecycle_end
