#!/usr/bin/env bash
# Shared helpers for executing Browser Automation Studio workflows from scenario tests.
set -euo pipefail

# Utility: generate unique workflow name based on file + timestamp
_testing_playbooks_bas__name_from_path() {
    local workflow_path="$1"
    local basename
    basename=$(basename "$workflow_path" .json)
    printf '%s-%s' "$basename" "$(date +%s%N | tail -c 6)"
}

# Run a Browser Automation Studio workflow JSON via the public CLI/API.
# Options:
#   --file PATH                 JSON workflow definition (required)
#   --scenario NAME             Scenario to start/manage (default: browser-automation-studio)
#   --folder PATH               Folder to import workflow into (default: /testing)
#   --manage-runtime [auto|1|0] Auto start/stop scenario (auto: start if not running)
#   --keep-workflow             Do not delete workflow after execution
#   --timeout SECONDS           Max wait for scenario readiness (default: 120)
#   --allow-missing             Return special exit code 210 when workflow file missing
# Returns:
#   0 on success, >0 otherwise. Exit code 210 indicates workflow file missing when
#   --allow-missing is supplied (callers may treat as skip).

testing::playbooks::bas::run_workflow() {
    local workflow_path=""
    local scenario_name="browser-automation-studio"
    local folder_path="/testing"
    local manage_runtime="auto"
    local keep_workflow=false
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
            --folder)
                folder_path="$2"
                shift 2
                ;;
            --manage-runtime)
                manage_runtime="$2"
                shift 2
                ;;
            --keep-workflow)
                keep_workflow=true
                shift
                ;;
            --timeout)
                readiness_timeout="$2"
                shift 2
                ;;
            --allow-missing)
                allow_missing=true
                shift
                ;;
            *)
                echo "Unknown option to testing::playbooks::bas::run_workflow: $1" >&2
                return 1
                ;;
        esac
    done

    if [ -z "$workflow_path" ]; then
        echo "testing::playbooks::bas::run_workflow requires --file" >&2
        return 1
    fi

    # Resolve workflow path relative to scenario directory when possible
    if [ ! -f "$workflow_path" ] && [ -n "${TESTING_PHASE_SCENARIO_DIR:-}" ]; then
        local candidate="${TESTING_PHASE_SCENARIO_DIR}/${workflow_path#./}"
        if [ -f "$candidate" ]; then
            workflow_path="$candidate"
        fi
    fi

    if [ ! -f "$workflow_path" ]; then
        if [ "$allow_missing" = true ]; then
            echo "⚠️  BAS workflow not found: $workflow_path" >&2
            return 210
        fi
        echo "❌ BAS workflow not found: $workflow_path" >&2
        return 1
    fi

    if [ ! -s "$workflow_path" ]; then
        echo "❌ BAS workflow is empty: $workflow_path" >&2
        return 1
    fi

    local started_scenario=false

    # Ensure scenario is running when manage_runtime dictates.
    local runtime_running=false
    if testing::core::is_scenario_running "$scenario_name"; then
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

    if ! testing::core::wait_for_scenario "$scenario_name" "$readiness_timeout" >/dev/null 2>&1; then
        echo "❌ Scenario $scenario_name did not become ready" >&2
        if [ "$started_scenario" = true ]; then
            vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
        fi
        return 1
    fi

    local api_port
    api_port=$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null | awk -F= '/API_PORT/{gsub(/ /,"");print $2}')
    if [ -z "$api_port" ]; then
        echo "❌ Unable to resolve API_PORT for $scenario_name" >&2
        if [ "$started_scenario" = true ]; then
            vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
        fi
        return 1
    fi

    local workflow_name
    workflow_name=$(_testing_playbooks_bas__name_from_path "$workflow_path")

    local create_output
    if ! create_output=$(API_PORT="$api_port" browser-automation-studio workflow create "$workflow_name" --folder "$folder_path" --from-file "$workflow_path" 2>&1); then
        echo "❌ Failed to create BAS workflow from $workflow_path" >&2
        echo "$create_output" >&2
        if [ "$started_scenario" = true ]; then
            vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
        fi
        return 1
    fi

    local workflow_id
    workflow_id=$(printf '%s\n' "$create_output" | awk -F': ' '/Workflow ID/{print $2}' | tr -d '\r')
    if [ -z "$workflow_id" ]; then
        echo "❌ Unable to parse workflow ID from create output" >&2
        printf '%s\n' "$create_output" >&2
        if [ "$started_scenario" = true ]; then
            vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
        fi
        return 1
    fi

    local execution_output
    if ! execution_output=$(API_PORT="$api_port" browser-automation-studio workflow execute "$workflow_id" --wait 2>&1); then
        echo "❌ Workflow execution failed for $workflow_id" >&2
        echo "$execution_output" >&2
        if [ "$keep_workflow" = false ]; then
            API_PORT="$api_port" browser-automation-studio workflow delete "$workflow_id" >/dev/null 2>&1 || true
        fi
        if [ "$started_scenario" = true ]; then
            vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
        fi
        return 1
    fi

    local execution_id
    execution_id=$(printf '%s\n' "$execution_output" | awk -F': ' '/Execution ID/{print $2}' | tr -d '\r')
    if [ -z "$execution_id" ]; then
        echo "⚠️  Unable to determine execution ID; assuming success" >&2
    else
        local final_status
        final_status=$(curl -s "http://localhost:${api_port}/api/v1/executions/${execution_id}" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")
        if [ "$final_status" != "completed" ]; then
            echo "❌ Workflow execution status: $final_status" >&2
            if [ "$keep_workflow" = false ]; then
                API_PORT="$api_port" browser-automation-studio workflow delete "$workflow_id" >/dev/null 2>&1 || true
            fi
            if [ "$started_scenario" = true ]; then
                vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
            fi
            return 1
        fi
    fi

    if [ "$keep_workflow" = false ]; then
        API_PORT="$api_port" browser-automation-studio workflow delete "$workflow_id" >/dev/null 2>&1 || true
    fi

    if [ "$started_scenario" = true ]; then
        vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
    fi

    return 0
}

export -f testing::playbooks::bas::run_workflow
