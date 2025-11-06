#!/usr/bin/env bash
# Shared helpers for executing Browser Automation Studio workflows from scenario tests.
set -euo pipefail

declare -g TESTING_PLAYBOOKS_BAS_LAST_WORKFLOW_ID=""
declare -g TESTING_PLAYBOOKS_BAS_LAST_EXECUTION_ID=""
declare -g TESTING_PLAYBOOKS_BAS_LAST_SCENARIO=""

# Utility: generate unique workflow name based on file + timestamp
_testing_playbooks_bas__name_from_path() {
    local workflow_path="$1"
    local basename
    basename=$(basename "$workflow_path" .json)
    printf '%s-%s' "$basename" "$(date +%s%N | tail -c 6)"
}

_testing_playbooks_bas__require_tools() {
    local missing=()
    for tool in jq curl; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            missing+=("$tool")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        echo "❌ Missing required tools: ${missing[*]}. Install them to run BAS workflows." >&2
        return 1
    fi
}

_testing_playbooks_bas__resolve_api_port() {
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

_testing_playbooks_bas__api_base() {
    local port="$1"
    printf 'http://localhost:%s/api/v1' "$port"
}

_testing_playbooks_bas__project_root() {
    local scenario_dir="${TESTING_PHASE_SCENARIO_DIR:-${TESTING_PHASE_SCENARIO_NAME:-}}"
    if [ -n "$scenario_dir" ] && [ -d "$scenario_dir" ]; then
        printf '%s/data/projects/testing-harness' "$scenario_dir"
    else
        printf '%s/data/projects/testing-harness' "${PWD}"
    fi
}

_testing_playbooks_bas__ensure_project() {
    local api_base="$1"
    local project_name="$2"
    local folder_path="$3"

    mkdir -p "$folder_path"

    local projects_json
    projects_json=$(curl -s "$api_base/projects" || true)
    local project_id=""
    if [ -n "$projects_json" ]; then
        project_id=$(printf '%s\n' "$projects_json" | jq -r --arg name "$project_name" '.projects[]? | select(.name == $name) | .id' | head -n1)
    fi

    if [ -z "$project_id" ]; then
        local payload
        payload=$(jq -n --arg name "$project_name" --arg folder "$folder_path" '{name:$name, folder_path:$folder}')
        local create_resp
        create_resp=$(curl -s -X POST -H 'Content-Type: application/json' -d "$payload" "$api_base/projects")
        project_id=$(printf '%s\n' "$create_resp" | jq -r '.project_id // .project.id // empty')
    fi

    if [ -z "$project_id" ]; then
        echo "❌ Unable to resolve project ID for BAS automation" >&2
        return 1
    fi

    printf '%s\n' "$project_id"
}

_testing_playbooks_bas__import_workflow() {
    local api_base="$1"
    local project_id="$2"
    local folder_path="$3"
    local workflow_path="$4"
    local workflow_name="$5"

    local payload
    payload=$(jq -n --arg pid "$project_id" --arg name "$workflow_name" --arg folder "$folder_path" --slurpfile flow "$workflow_path" '{project_id:$pid, name:$name, folder_path:$folder, flow_definition:$flow[0]}')

    local resp
    resp=$(curl -s -X POST -H 'Content-Type: application/json' -d "$payload" "$api_base/workflows/create")
    local workflow_id
    workflow_id=$(printf '%s\n' "$resp" | jq -r '.workflow_id // .id // empty')

    if [ -z "$workflow_id" ]; then
        echo "❌ Failed to import BAS workflow" >&2
        printf '%s\n' "$resp" >&2
        return 1
    fi

    printf '%s\n' "$workflow_id"
}

_testing_playbooks_bas__delete_workflow() {
    local api_base="$1"
    local project_id="$2"
    local workflow_id="$3"

    local payload
    payload=$(jq -n --arg id "$workflow_id" '{workflow_ids:[$id]}')
    curl -s -X POST -H 'Content-Type: application/json' -d "$payload" "$api_base/projects/${project_id}/workflows/bulk-delete" >/dev/null 2>&1 || true
}

_testing_playbooks_bas__execute_workflow() {
    local api_base="$1"
    local workflow_id="$2"
    local payload='{"wait_for_completion":false}'
    local resp
    resp=$(curl -s -X POST -H 'Content-Type: application/json' -d "$payload" "$api_base/workflows/${workflow_id}/execute")
    local execution_id
    execution_id=$(printf '%s\n' "$resp" | jq -r '.execution_id // empty')

    if [ -z "$execution_id" ]; then
        echo "❌ Workflow execution did not return an execution_id" >&2
        printf '%s\n' "$resp" >&2
        return 1
    fi

    printf '%s\n' "$execution_id"
}

_testing_playbooks_bas__wait_for_execution() {
    local api_base="$1"
    local execution_id="$2"
    local timeout="${3:-240}"
    local start_ts
    start_ts=$(date +%s)

    while true; do
        local exec_resp
        exec_resp=$(curl -s "$api_base/executions/${execution_id}" || true)
        local status
        status=$(printf '%s\n' "$exec_resp" | jq -r '.status // "unknown"')

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

_testing_playbooks_bas__execute_adhoc_workflow() {
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
    resp=$(curl -s -X POST \
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

# Run a Browser Automation Studio workflow JSON via the public CLI/API.
# Options:
#   --file PATH                 JSON workflow definition (required)
#   --scenario NAME             Scenario to start/manage (default: browser-automation-studio)
#   --workflow-id ID            Execute an existing workflow instead of importing from file
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
    local workflow_id_input=""
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
            --workflow-id)
                workflow_id_input="$2"
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

    TESTING_PLAYBOOKS_BAS_LAST_WORKFLOW_ID=""
    TESTING_PLAYBOOKS_BAS_LAST_EXECUTION_ID=""
    TESTING_PLAYBOOKS_BAS_LAST_SCENARIO=""

    if [ -z "$workflow_path" ] && [ -z "$workflow_id_input" ]; then
        echo "testing::playbooks::bas::run_workflow requires --file or --workflow-id" >&2
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
            echo "⚠️  BAS workflow not found: $workflow_path" >&2
            return 210
        fi
        echo "❌ BAS workflow not found: $workflow_path" >&2
        return 1
    fi

    if [ -n "$workflow_path" ] && [ ! -s "$workflow_path" ]; then
        echo "❌ BAS workflow is empty: $workflow_path" >&2
        return 1
    fi

    _testing_playbooks_bas__require_tools

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
            if [ "$started_scenario" = true ]; then
                vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
            fi
            return 1
        fi
    fi

    local api_port
    api_port=$(_testing_playbooks_bas__resolve_api_port "$scenario_name") || {
        echo "❌ Unable to resolve API_PORT for $scenario_name" >&2
        if [ "$started_scenario" = true ]; then
            vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
        fi
        return 1
    }

    local api_base
    api_base=$(_testing_playbooks_bas__api_base "$api_port")

    # Try adhoc execution first if workflow file is provided
    # This avoids database pollution with test workflows
    if [ -n "$workflow_path" ] && [ -z "$workflow_id_input" ]; then
        echo "✨ Attempting adhoc workflow execution (no DB pollution)" >&2

        local execution_id
        if execution_id=$(_testing_playbooks_bas__execute_adhoc_workflow "$api_base" "$workflow_path" 2>&1); then
            TESTING_PLAYBOOKS_BAS_LAST_EXECUTION_ID="$execution_id"
            TESTING_PLAYBOOKS_BAS_LAST_WORKFLOW_ID=""  # No workflow persisted
            TESTING_PLAYBOOKS_BAS_LAST_SCENARIO="$scenario_name"

            local execution_summary
            if execution_summary=$(_testing_playbooks_bas__wait_for_execution "$api_base" "$execution_id" 360); then
                if [ "$started_scenario" = true ]; then
                    vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
                fi
                return 0
            else
                echo "❌ Adhoc workflow execution failed" >&2
                printf '%s\n' "$execution_summary" >&2
                if [ "$started_scenario" = true ]; then
                    vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
                fi
                return 1
            fi
        else
            # Adhoc execution failed, fall back to legacy import flow
            echo "⚠️  Adhoc execution not available or failed, using legacy import flow" >&2
        fi
    fi

    local workflow_id="$workflow_id_input"
    local created_workflow=false
    local workflow_name=""
    local imported_project_id=""

    if [ -n "$workflow_path" ]; then
        workflow_name=$(_testing_playbooks_bas__name_from_path "$workflow_path")
        local project_folder
        project_folder=$(_testing_playbooks_bas__project_root)
        local project_id
        project_id=$(_testing_playbooks_bas__ensure_project "$api_base" "Testing Harness" "$project_folder") || {
            if [ "$started_scenario" = true ]; then
                vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
            fi
            return 1
        }

        imported_project_id="$project_id"

        workflow_id=$(_testing_playbooks_bas__import_workflow "$api_base" "$project_id" "$folder_path" "$workflow_path" "$workflow_name") || {
            if [ "$started_scenario" = true ]; then
                vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
            fi
            return 1
        }
        created_workflow=true
    fi

    if [ -z "$workflow_id" ]; then
        echo "❌ No workflow available to execute (provide --file or --workflow-id)" >&2
        if [ "$started_scenario" = true ]; then
            vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
        fi
        return 1
    fi

    if [ -z "$workflow_name" ]; then
        workflow_name="$workflow_id"
    fi

    local execution_id
    execution_id=$(_testing_playbooks_bas__execute_workflow "$api_base" "$workflow_id") || {
        if [ "$started_scenario" = true ]; then
            vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
        fi
        return 1
    }

    local execution_summary
    if ! execution_summary=$(_testing_playbooks_bas__wait_for_execution "$api_base" "$execution_id" 360); then
        echo "❌ Workflow execution failed for $workflow_id" >&2
        printf '%s\n' "$execution_summary" >&2
        if [ "$keep_workflow" = false ] && [ "$created_workflow" = true ] && [ -n "$imported_project_id" ]; then
            _testing_playbooks_bas__delete_workflow "$api_base" "$imported_project_id" "$workflow_id"
        fi
        if [ "$started_scenario" = true ]; then
            vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
        fi
        return 1
    fi

    if [ "$keep_workflow" = false ] && [ "$created_workflow" = true ] && [ -n "$imported_project_id" ]; then
        _testing_playbooks_bas__delete_workflow "$api_base" "$imported_project_id" "$workflow_id"
    fi

    if [ "$started_scenario" = true ]; then
        vrooli scenario stop "$scenario_name" >/dev/null 2>&1 || true
    fi

    TESTING_PLAYBOOKS_BAS_LAST_WORKFLOW_ID="$workflow_id"
    TESTING_PLAYBOOKS_BAS_LAST_EXECUTION_ID="$execution_id"
    TESTING_PLAYBOOKS_BAS_LAST_SCENARIO="$scenario_name"

    return 0
}

export -f testing::playbooks::bas::run_workflow
