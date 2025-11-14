#!/usr/bin/env bash
# Generic workflow runner for executing Browser Automation Studio workflows from any scenario test.
set -euo pipefail
# This runner allows any scenario to define BAS workflow JSONs and have them automatically executed
# for UI/integration testing via the browser-automation-studio API.

declare -g TESTING_PLAYBOOKS_LAST_WORKFLOW_ID=""
declare -g TESTING_PLAYBOOKS_LAST_EXECUTION_ID=""
declare -g TESTING_PLAYBOOKS_LAST_SCENARIO=""

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

    while true; do
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
            if [ -n "$current_step" ] && [ "$current_step" != "null" ]; then
                echo "   [${status}] ${progress}% - ${current_step}" >&2
            else
                echo "   [${status}] ${progress}%" >&2
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

        sleep 2
    done
}

_testing_playbooks__execute_adhoc_workflow() {
    local api_base="$1"
    local workflow_path="$2"

    # Read workflow JSON and extract metadata
    local flow_def
    flow_def=$(cat "$workflow_path")

    local name
    name=$(printf '%s' "$flow_def" | jq -r '.metadata.name // .metadata.description // empty')
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

    if [ -z "$workflow_path" ]; then
        echo "testing::playbooks::run_workflow requires --file" >&2
        return 1
    fi

    # Resolve workflow path relative to scenario directory when possible
    if [ -n "$workflow_path" ] && [ ! -f "$workflow_path" ] && [ -n "${TESTING_PHASE_SCENARIO_DIR:-}" ]; then
        local candidate="${TESTING_PHASE_SCENARIO_DIR}/${workflow_path#./}"
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

    local api_port
    api_port=$(_testing_playbooks__resolve_api_port "$scenario_name") || {
        echo "❌ Unable to resolve API_PORT for $scenario_name" >&2
        # Leave scenario running for debugging
        return 1
    }

    local api_base
    api_base=$(_testing_playbooks__api_base "$api_port")

    # Execute workflow via adhoc endpoint (no DB persistence)
    local workflow_name
    workflow_name=$(basename "$workflow_path" .json)
    echo "🚀 Starting workflow: ${workflow_name}" >&2

    local execution_id
    if ! execution_id=$(_testing_playbooks__execute_adhoc_workflow "$api_base" "$workflow_path"); then
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
    local workflow_timeout=${TESTING_PLAYBOOKS_WORKFLOW_TIMEOUT:-90}
    if execution_summary=$(_testing_playbooks__wait_for_execution "$api_base" "$execution_id" "$workflow_timeout"); then
        wait_rc=0
    else
        wait_rc=$?
    fi

    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    if [ $wait_rc -eq 0 ]; then
        echo "✅ Workflow ${workflow_name} completed in ${duration}s" >&2
        return 0
    fi

    echo "❌ Workflow ${workflow_name} failed after ${duration}s" >&2
    printf '%s\n' "$execution_summary" >&2

    if [ $wait_rc -eq 2 ] && [ "$duration" -ge "$workflow_timeout" ]; then
        echo "⚠️ [WF_RUNTIME_SLOW] Workflow ${workflow_name} timed out after ${duration}s (execution stalled)" >&2
    fi

    return 1
}

export -f testing::playbooks::run_workflow
