#!/usr/bin/env bash
# Generic workflow runner for executing Browser Automation Studio workflows from any scenario test.
set -euo pipefail
# This runner allows any scenario to define BAS workflow JSONs and have them automatically executed
# for UI/integration testing via the browser-automation-studio API.

TESTING_PLAYBOOKS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTING_PLAYBOOKS_APP_ROOT="$(cd "${TESTING_PLAYBOOKS_DIR}/../../../../" && pwd)"

declare -g _TESTING_PLAYBOOKS_SEEDS_APPLIED=false
declare -g _TESTING_PLAYBOOKS_SEEDS_DIR=""
declare -g _TESTING_PLAYBOOKS_SEEDS_SCENARIO_DIR=""
declare -ga TESTING_PLAYBOOKS_LAST_PROGRESS_LINES=()

declare -g TESTING_PLAYBOOKS_LAST_WORKFLOW_ID=""
declare -g TESTING_PLAYBOOKS_LAST_EXECUTION_ID=""
declare -g TESTING_PLAYBOOKS_LAST_SCENARIO=""
declare -g TESTING_PLAYBOOKS_LAST_FIXTURE_REQUIREMENTS=""
declare -g TESTING_PLAYBOOKS_LAST_WORKFLOW_PATH=""
declare -g TESTING_PLAYBOOKS_LAST_ARTIFACT_NOTE=""

_testing_playbooks__require_tools() {
    local missing=()
    for tool in jq curl; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            missing+=("$tool")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        echo "❌ Missing required tools: ${missing[*]}. Install them to run workflows." >&2
        return 1
    fi
}

_testing_playbooks__read_workflow_definition() {
    local workflow_path="$1"
    local scenario_dir="$2"
    if [ ! -f "$workflow_path" ]; then
        echo "Workflow definition not found: $workflow_path" >&2
        return 1
    fi
    local resolver_script="${TESTING_PLAYBOOKS_APP_ROOT}/scripts/scenarios/testing/playbooks/resolve-workflow.py"
    if [ -f "$resolver_script" ]; then
        local python_cmd=""
        if command -v python3 >/dev/null 2>&1; then
            python_cmd="python3"
        elif command -v python >/dev/null 2>&1; then
            python_cmd="python"
        fi
        if [ -n "$python_cmd" ]; then
            "$python_cmd" "$resolver_script" --workflow "$workflow_path" --scenario "$scenario_dir"
            return $?
        fi
        echo "⚠️  python is not available; skipping workflow fixture resolution" >&2
    fi
    cat "$workflow_path"
}

_testing_playbooks__apply_seeds_if_needed() {
    local scenario_dir="$1"
    local seeds_dir="$scenario_dir/test/playbooks/__seeds"
    if [ ! -d "$seeds_dir" ]; then
        return 0
    fi
    if [ "${_TESTING_PLAYBOOKS_SEEDS_APPLIED:-false}" = true ]; then
        return 0
    fi
    local apply_script="$seeds_dir/apply.sh"
    if [ -f "$apply_script" ]; then
        echo "🌱 Applying BAS seed data from ${apply_script}"
        (cd "$scenario_dir" && bash "$apply_script") || return 1
    fi
    _TESTING_PLAYBOOKS_SEEDS_APPLIED=true
    _TESTING_PLAYBOOKS_SEEDS_DIR="$seeds_dir"
    _TESTING_PLAYBOOKS_SEEDS_SCENARIO_DIR="$scenario_dir"
    return 0
}

_testing_playbooks__cleanup_seeds() {
    if [ "${_TESTING_PLAYBOOKS_SEEDS_APPLIED:-false}" != true ]; then
        return 0
    fi
    local cleanup_script="${_TESTING_PLAYBOOKS_SEEDS_DIR}/cleanup.sh"
    if [ -f "$cleanup_script" ]; then
        echo "🧹 Cleaning BAS seed data via ${cleanup_script}"
        (cd "${_TESTING_PLAYBOOKS_SEEDS_SCENARIO_DIR}" && bash "$cleanup_script") || \
            echo "⚠️  Seed cleanup script failed" >&2
    fi
    _TESTING_PLAYBOOKS_SEEDS_APPLIED=false
    _TESTING_PLAYBOOKS_SEEDS_DIR=""
    _TESTING_PLAYBOOKS_SEEDS_SCENARIO_DIR=""
}

_testing_playbooks__persist_failure_artifacts() {
    local scenario_dir="$1"
    local workflow_rel="$2"
    local execution_id="$3"
    local api_base="$4"
    local timeline_payload="$5"

    if [ -z "$scenario_dir" ] || [ -z "$workflow_rel" ] || [ -z "$execution_id" ]; then
        return 0
    fi

    local artifact_root="$scenario_dir/coverage/automation"
    local sanitized="${workflow_rel#./}"
    if [ -z "$sanitized" ]; then
        sanitized="$(basename "$workflow_rel")"
    fi
    local base_no_ext="${sanitized%.*}"
    local base_path="$artifact_root/${base_no_ext}"
    local target_dir
    target_dir=$(dirname "$base_path")
    mkdir -p "$target_dir"

    # Try folder export first (includes markdown, screenshots, etc.)
    local folder_export_dir="${base_path}-${execution_id}"
    local folder_export_resp
    folder_export_resp=$(curl -s --max-time 30 -X POST \
        -H 'Content-Type: application/json' \
        -d "{\"format\":\"folder\",\"output_dir\":\"${folder_export_dir}\"}" \
        "$api_base/executions/${execution_id}/export" 2>&1)

    local folder_export_success=false
    if printf '%s' "$folder_export_resp" | grep -q "Execution exported successfully"; then
        folder_export_success=true
    fi

    # Legacy: Always write timeline.json at old location for backward compatibility
    if [ -n "$timeline_payload" ]; then
        printf '%s\n' "$timeline_payload" >"${base_path}.timeline.json"
    fi

    if command -v jq >/dev/null 2>&1; then
        local screenshot_resp
        screenshot_resp=$(curl -s --max-time 10 "$api_base/executions/${execution_id}/screenshots" || true)
        local screenshot_url
        screenshot_url=$(printf '%s' "$screenshot_resp" | jq -r '.screenshots | last | (.storage_url // .thumbnail_url // empty)' 2>/dev/null || echo '')
        if [ -n "$screenshot_url" ]; then
            local base_host="${api_base%/api/v1}"
            local resolved_url="$screenshot_url"
            if [[ "$resolved_url" != http*://* ]]; then
                resolved_url="$base_host$resolved_url"
            fi
            curl -s --max-time 30 "$resolved_url" -o "${base_path}.png" >/dev/null 2>&1 || true
        fi
    fi

    # Prefer folder export with README.md if available and non-trivial
    local folder_readme="${folder_export_dir}/README.md"
    if [ "$folder_export_success" = true ] && [ -f "$folder_readme" ] && [ -s "$folder_readme" ]; then
        # Check if README has actual content (not just template)
        local readme_lines=$(wc -l < "$folder_readme" 2>/dev/null || echo "0")
        if [ "$readme_lines" -gt 10 ]; then
            local rel_readme="$folder_readme"
            if [[ "$rel_readme" == "$scenario_dir/"* ]]; then
                rel_readme="${rel_readme#$scenario_dir/}"
            fi
            TESTING_PLAYBOOKS_LAST_ARTIFACT_NOTE="Read $rel_readme"
        else
            # README exists but is mostly empty - treat as failed export
            TESTING_PLAYBOOKS_LAST_ARTIFACT_NOTE=""
        fi
    else
        # Fallback to legacy artifact listing
        local recorded_paths=()
        if [ -s "${base_path}.timeline.json" ]; then
            recorded_paths+=("${base_path}.timeline.json")
        fi
        if [ -s "${base_path}.png" ]; then
            recorded_paths+=("${base_path}.png")
        fi

        if [ ${#recorded_paths[@]} -gt 0 ]; then
            local rel_paths=()
            local path_entry
            for path_entry in "${recorded_paths[@]}"; do
                if [[ "$path_entry" == "$scenario_dir/"* ]]; then
                    rel_paths+=("${path_entry#$scenario_dir/}")
                else
                    rel_paths+=("$path_entry")
                fi
            done
            TESTING_PLAYBOOKS_LAST_ARTIFACT_NOTE=$(IFS=', '; echo "${rel_paths[*]}")
        else
            TESTING_PLAYBOOKS_LAST_ARTIFACT_NOTE=""
        fi
    fi
}

_testing_playbooks__resolve_api_port() {
    local scenario_name="$1"
    local resolved
    resolved=$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null || true)
    if [ -z "$resolved" ]; then
        echo "" && return 1
    fi

    if printf '%s' "$resolved" | grep -q '='; then
        resolved=$(printf '%s' "$resolved" | awk -F= '/API_PORT/{gsub(/ /,"");print $2}')
    fi

    printf '%s' "$resolved" | tr -d '\r\n '
}

_testing_playbooks__api_base() {
    local port="$1"
    printf 'http://localhost:%s/api/v1' "$port"
}

# Legacy function removed - adhoc execution doesn't need project folders

# Legacy functions removed - adhoc execution only

_testing_playbooks__wait_for_execution() {
    local api_base="$1"
    local execution_id="$2"
    local timeout="${3:-240}"
    local start_ts
    start_ts=$(date +%s)
    local last_status=""
    local last_progress=-1
    local poll_count=0
    local verbose_logging="${TESTING_PLAYBOOKS_VERBOSE:-0}"
    TESTING_PLAYBOOKS_LAST_PROGRESS_LINES=()

    while true; do
        local poll_start
        poll_start=$(date +%s)
        local exec_resp
        exec_resp=$(curl -s --max-time 10 "$api_base/executions/${execution_id}" || true)
        local status
        status=$(printf '%s\n' "$exec_resp" | jq -r '.status // "unknown"')
        local progress
        progress=$(printf '%s\n' "$exec_resp" | jq -r '.progress // 0')
        local current_step
        current_step=$(printf '%s\n' "$exec_resp" | jq -r '.current_step // ""')

        # Log progress updates (every 5 polls or when status/progress changes)
        poll_count=$((poll_count + 1))
        if [ "$status" != "$last_status" ] || [ "$progress" != "$last_progress" ] || [ $((poll_count % 5)) -eq 0 ]; then
            local message
            if [ -n "$current_step" ] && [ "$current_step" != "null" ]; then
                message="   [${status}] ${progress}% - ${current_step}"
            else
                message="   [${status}] ${progress}%"
            fi
            if [ "$verbose_logging" = "1" ]; then
                echo "$message" >&2
            else
                TESTING_PLAYBOOKS_LAST_PROGRESS_LINES+=("$message")
            fi
            last_status="$status"
            last_progress="$progress"
        fi

        case "$status" in
            completed|success)
                printf '%s\n' "$exec_resp"
                return 0
                ;;
            failed|errored|error)
                printf '%s\n' "$exec_resp"
                return 1
                ;;
            running|pending|queued)
                :
                ;;
            *)
                :
                ;;
        esac

        local now
        now=$(date +%s)
        if [ $((now - start_ts)) -ge "$timeout" ]; then
            echo "❌ Execution ${execution_id} did not complete within ${timeout}s" >&2
            printf '%s\n' "$exec_resp"
            return 2
        fi

        # Adaptive polling: faster when running, slower when queued
        local poll_interval=1
        case "$status" in
            running)
                poll_interval=0.5  # Fast polling during active execution
                ;;
            pending|queued)
                poll_interval=2    # Slow polling while waiting to start
                ;;
            *)
                poll_interval=1    # Default for unknown/transitional states
                ;;
        esac

        # Sleep for remaining interval time (avoid overlapping requests if API was slow)
        local poll_duration=$((now - poll_start))
        if command -v bc >/dev/null 2>&1; then
            local sleep_time
            sleep_time=$(echo "$poll_interval - $poll_duration" | bc -l)
            if [ "$(echo "$sleep_time > 0" | bc -l)" -eq 1 ]; then
                sleep "$sleep_time"
            fi
        else
            # Fallback without bc: use integer math with 500ms minimum
            if [ "$poll_interval" = "0.5" ]; then
                # For 500ms interval, only sleep if poll took <1s
                if [ "$poll_duration" -lt 1 ]; then
                    sleep 0.5
                fi
            else
                local sleep_time=$((poll_interval - poll_duration))
                if [ "$sleep_time" -gt 0 ]; then
                    sleep "$sleep_time"
                fi
            fi
        fi
    done
}

_testing_playbooks__execute_adhoc_workflow() {
    local api_base="$1"
    local workflow_json="$2"

    # Extract the inner flow_definition from the workflow JSON
    # Workflow files can have two structures:
    # 1. Direct: {metadata: {...}, nodes: [...], edges: [...]} (test playbooks)
    # 2. Wrapped: {flow_definition: {metadata: {...}, nodes: [...], edges: [...]}} (saved workflows)
    # The API expects just the flow_definition part, without extra playbook metadata
    local flow_def
    flow_def=$(printf '%s' "$workflow_json" | jq '(.flow_definition // .) | del(.metadata, .description, .requirements, .cleanup, .fixtures)')

    # Extract metadata for the request
    local name
    name=$(printf '%s' "$workflow_json" | jq -r '.flow_definition.metadata.name // .metadata.name // .flow_definition.metadata.description // .metadata.description // empty')
    if [ -z "$name" ]; then
        name=$(basename "$workflow_path" .json)
    fi

    # Build request payload for adhoc execution
    local payload
    payload=$(jq -n \
        --argjson flow "$flow_def" \
        --arg name "$name" \
        '{
            flow_definition: $flow,
            parameters: {},
            wait_for_completion: false,
            metadata: {name: $name}
        }')

    # Execute workflow via adhoc endpoint
    local resp
    resp=$(curl -s --max-time 30 -X POST \
        -H 'Content-Type: application/json' \
        -d "$payload" \
        "$api_base/workflows/execute-adhoc")

    local execution_id
    execution_id=$(printf '%s' "$resp" | jq -r '.execution_id // empty')

    if [ -z "$execution_id" ]; then
        echo "❌ Adhoc workflow execution did not return execution_id" >&2
        printf '%s\n' "$resp" >&2
        return 1
    fi

    printf '%s' "$execution_id"
}

# Run a Browser Automation Studio workflow JSON via adhoc execution endpoint.
# Options:
#   --file PATH                 JSON workflow definition (required)
#   --scenario NAME             Scenario to start/manage (default: browser-automation-studio)
#   --manage-runtime [auto|1|0] Auto start/stop scenario (auto: start if not running)
#   --timeout SECONDS           Max wait for scenario readiness (default: 120)
#   --allow-missing             Return special exit code 210 when workflow file missing
# Returns:
#   0 on success, >0 otherwise. Exit code 210 indicates workflow file missing when
#   --allow-missing is supplied (callers may treat as skip).

testing::playbooks::run_workflow() {
    local workflow_path=""
    local scenario_name="browser-automation-studio"
    local manage_runtime="auto"
    local readiness_timeout=120
    local allow_missing=false
    local verbose_logging="${TESTING_PLAYBOOKS_VERBOSE:-0}"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                workflow_path="$2"
                shift 2
                ;;
            --scenario)
                scenario_name="$2"
                shift 2
                ;;
            --manage-runtime)
                manage_runtime="$2"
                shift 2
                ;;
            --timeout)
                readiness_timeout="$2"
                shift 2
                ;;
            --allow-missing)
                allow_missing=true
                shift
                ;;
            --workflow-id|--folder|--keep-workflow)
                echo "❌ Deprecated option: $1 (adhoc execution only)" >&2
                return 1
                ;;
            *)
                echo "Unknown option to testing::playbooks::run_workflow: $1" >&2
                return 1
                ;;
        esac
    done

    TESTING_PLAYBOOKS_LAST_WORKFLOW_ID=""
    TESTING_PLAYBOOKS_LAST_EXECUTION_ID=""
    TESTING_PLAYBOOKS_LAST_SCENARIO=""
    TESTING_PLAYBOOKS_LAST_FIXTURE_REQUIREMENTS=""
    TESTING_PLAYBOOKS_LAST_ARTIFACT_NOTE=""

    local scenario_dir="${TESTING_PHASE_SCENARIO_DIR:-$(pwd)}"

    if [ -z "$workflow_path" ]; then
        echo "testing::playbooks::run_workflow requires --file" >&2
        return 1
    fi

    # Resolve workflow path relative to scenario directory when possible
    if [ -n "$workflow_path" ] && [ ! -f "$workflow_path" ] && [ -n "$scenario_dir" ]; then
        local candidate="${scenario_dir}/${workflow_path#./}"
        if [ -f "$candidate" ]; then
            workflow_path="$candidate"
        fi
    fi

    if [ -n "$workflow_path" ] && [ ! -f "$workflow_path" ]; then
        if [ "$allow_missing" = true ]; then
            echo "⚠️  workflow not found: $workflow_path" >&2
            return 210
        fi
        echo "❌ workflow not found: $workflow_path" >&2
        return 1
    fi

    if [ -n "$workflow_path" ] && [ ! -s "$workflow_path" ]; then
        echo "❌ workflow is empty: $workflow_path" >&2
        return 1
    fi

    local artifact_rel_path="$workflow_path"
    if [[ "$artifact_rel_path" == "$scenario_dir/"* ]]; then
        artifact_rel_path="${artifact_rel_path#$scenario_dir/}"
    fi
    artifact_rel_path="${artifact_rel_path#./}"
    if [ -z "$artifact_rel_path" ]; then
        artifact_rel_path="$(basename "$workflow_path")"
    fi
    TESTING_PLAYBOOKS_LAST_WORKFLOW_PATH="$artifact_rel_path"

    _testing_playbooks__require_tools

    local started_scenario=false

    # Ensure scenario is running when manage_runtime dictates.
    local runtime_running=false
    if declare -F testing::core::is_scenario_running >/dev/null 2>&1; then
        if testing::core::is_scenario_running "$scenario_name"; then
            runtime_running=true
        fi
    else
        runtime_running=true
    fi

    case "$manage_runtime" in
        1|true|start)
            if [ "$runtime_running" = false ]; then
                vrooli scenario start "$scenario_name" --clean-stale >/dev/null
                started_scenario=true
            fi
            ;;
        0|false|skip)
            :
            ;;
        auto)
            if [ "$runtime_running" = false ]; then
                vrooli scenario start "$scenario_name" --clean-stale >/dev/null
                started_scenario=true
            fi
            ;;
        *)
            echo "Unknown --manage-runtime value: $manage_runtime" >&2
            return 1
            ;;
    esac

    if declare -F testing::core::wait_for_scenario >/dev/null 2>&1; then
        if ! testing::core::wait_for_scenario "$scenario_name" "$readiness_timeout" >/dev/null 2>&1; then
            echo "❌ Scenario $scenario_name did not become ready" >&2
            # Leave scenario running for debugging
            return 1
        fi
    fi

    # NOTE: Seeding is now handled by the phase loop (phase-helpers.sh) before calling run_workflow.
    # This ensures seeds are applied once at the start and cleaned/reapplied only when needed.

    local api_port
    api_port=$(_testing_playbooks__resolve_api_port "$scenario_name") || {
        echo "❌ Unable to resolve API_PORT for $scenario_name" >&2
        # Leave scenario running for debugging
        return 1
    }

    local api_base
    api_base=$(_testing_playbooks__api_base "$api_port")

    local workflow_json
    if ! workflow_json=$(_testing_playbooks__read_workflow_definition "$workflow_path" "$scenario_dir"); then
        echo "❌ Failed to load workflow definition $workflow_path" >&2
        return 1
    fi

    # Substitute environment variables in workflow JSON (e.g., ${BASE_URL}, {{UI_PORT}})
    # Determine the scenario being tested (not the BAS scenario that executes workflows)
    local tested_scenario_name
    tested_scenario_name=$(basename "$scenario_dir")

    # Get UI_PORT for the scenario being tested
    local ui_port
    ui_port=$(vrooli scenario port "$tested_scenario_name" UI_PORT 2>/dev/null || true)
    if [ -n "$ui_port" ]; then
        # Extract port number from "UI_PORT=12345" format if needed
        if printf '%s' "$ui_port" | grep -q '='; then
            ui_port=$(printf '%s' "$ui_port" | awk -F= '/UI_PORT/{gsub(/ /,"");print $2}')
        fi
        ui_port=$(printf '%s' "$ui_port" | tr -d '\r\n ')

        # Substitute ${BASE_URL} with actual UI URL and {{UI_PORT}} with port number
        local base_url="http://localhost:${ui_port}"
        workflow_json=$(printf '%s' "$workflow_json" | jq --arg url "$base_url" --arg port "$ui_port" '
            walk(if type == "string" then gsub("\\$\\{BASE_URL\\}"; $url) | gsub("\\{\\{UI_PORT\\}\\}"; $port) else . end)
        ')
    fi

    local fixture_requirements
    fixture_requirements=$(printf '%s' "$workflow_json" | jq -r '.metadata.requirementsFromFixtures[]?' 2>/dev/null || true)
    TESTING_PLAYBOOKS_LAST_FIXTURE_REQUIREMENTS="$fixture_requirements"

    # Execute workflow via adhoc endpoint (no DB persistence)
    local workflow_name
    workflow_name=$(basename "$workflow_path" .json)
    echo "🚀 Starting workflow: ${workflow_name}" >&2

    local execution_id
    if ! execution_id=$(_testing_playbooks__execute_adhoc_workflow "$api_base" "$workflow_json"); then
        echo "❌ Failed to start adhoc workflow execution" >&2
        # Leave scenario running for debugging
        return 1
    fi

    echo "   Execution ID: ${execution_id}" >&2

    TESTING_PLAYBOOKS_LAST_EXECUTION_ID="$execution_id"
    TESTING_PLAYBOOKS_LAST_WORKFLOW_ID=""  # No workflow persisted
    TESTING_PLAYBOOKS_LAST_SCENARIO="$scenario_name"

    local start_time
    start_time=$(date +%s)
    local execution_summary
    local wait_rc=0
    local workflow_timeout=${TESTING_PLAYBOOKS_WORKFLOW_TIMEOUT:-45}
    if execution_summary=$(_testing_playbooks__wait_for_execution "$api_base" "$execution_id" "$workflow_timeout"); then
        wait_rc=0
    else
        wait_rc=$?
    fi

    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    # Fetch assertion and step statistics from timeline
    local stats_output=""
    local timeline_json=""
    if command -v jq >/dev/null 2>&1; then
        timeline_json=$(curl -s --max-time 5 "$api_base/executions/${execution_id}/timeline" 2>/dev/null || echo '{"frames":[]}')
        local total_steps=0
        local assert_total=0
        local assert_passed=0
        local assert_failed=0

        # Count total steps (all frames are steps)
        total_steps=$(printf '%s' "$timeline_json" | jq '.frames | length' 2>/dev/null || echo "0")

        # Count assertions by status (step_type == "assert")
        assert_total=$(printf '%s' "$timeline_json" | jq '[.frames[] | select(.step_type == "assert")] | length' 2>/dev/null || echo "0")
        assert_passed=$(printf '%s' "$timeline_json" | jq '[.frames[] | select(.step_type == "assert" and .status == "completed")] | length' 2>/dev/null || echo "0")
        assert_failed=$(printf '%s' "$timeline_json" | jq '[.frames[] | select(.step_type == "assert" and .status == "failed")] | length' 2>/dev/null || echo "0")

        # Build stats output
        if [ "$assert_total" -gt 0 ]; then
            stats_output=" (${total_steps} steps, ${assert_passed}/${assert_total} assertions passed)"
        elif [ "$total_steps" -gt 0 ]; then
            stats_output=" (${total_steps} steps)"
        fi
    fi

    if [ $wait_rc -eq 0 ]; then
        echo "✅ Workflow ${workflow_name} completed in ${duration}s${stats_output}" >&2
        return 0
    fi

    if [ "$verbose_logging" != "1" ]; then
        for line in "${TESTING_PLAYBOOKS_LAST_PROGRESS_LINES[@]}"; do
            echo "$line" >&2
        done
    fi

    echo "❌ Workflow ${workflow_name} failed after ${duration}s${stats_output}" >&2

    _testing_playbooks__persist_failure_artifacts "$scenario_dir" "$artifact_rel_path" "$execution_id" "$api_base" "$timeline_json"

    # Only print JSON summary if folder export failed (fallback to legacy format)
    if [ -n "${TESTING_PLAYBOOKS_LAST_ARTIFACT_NOTE:-}" ]; then
        if [[ "${TESTING_PLAYBOOKS_LAST_ARTIFACT_NOTE}" == Read* ]]; then
            # Folder export succeeded - skip JSON, just show clean README pointer
            echo "   ↳ ${TESTING_PLAYBOOKS_LAST_ARTIFACT_NOTE}" >&2
        else
            # Folder export failed - show JSON and legacy artifacts
            printf '%s\n' "$execution_summary" >&2
            echo "   ↳ Failure artifacts: ${TESTING_PLAYBOOKS_LAST_ARTIFACT_NOTE}" >&2
        fi
    else
        # No artifacts at all - still show JSON for debugging
        printf '%s\n' "$execution_summary" >&2
    fi

    if [ $wait_rc -eq 2 ] && [ "$duration" -ge "$workflow_timeout" ]; then
        echo "⚠️ [WF_RUNTIME_SLOW] Workflow ${workflow_name} timed out after ${duration}s (execution stalled)" >&2
    fi

    return 1
}

export -f testing::playbooks::run_workflow

testing::playbooks::reset_seed_state() {
    local reset_mode="none"
    local scenario_dir="${TESTING_PHASE_SCENARIO_DIR:-$(pwd)}"
    local scenario_name="${TESTING_PLAYBOOKS_LAST_SCENARIO:-${TESTING_PHASE_SCENARIO_NAME:-browser-automation-studio}}"
    local readiness_timeout="${TESTING_PLAYBOOKS_RESET_TIMEOUT:-150}"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --mode)
                reset_mode="$2"
                shift 2
                ;;
            --scenario-dir)
                scenario_dir="$2"
                shift 2
                ;;
            --scenario-name)
                scenario_name="$2"
                shift 2
                ;;
            --timeout)
                readiness_timeout="$2"
                shift 2
                ;;
            *)
                echo "Unknown option to testing::playbooks::reset_seed_state: $1" >&2
                return 1
                ;;
        esac
    done

    if [ "$reset_mode" = "none" ]; then
        return 0
    fi

    # DEPRECATED: This function now does nothing. Cleanup and seeding are handled by phase loop.
    # Return success immediately for backward compatibility.
    return 0
}
export -f testing::playbooks::reset_seed_state
