#!/usr/bin/env bash
# Integration phase helper utilities
set -euo pipefail

# Resolve workflow selector root for the active scenario (override via WORKFLOW_LINT_SELECTOR_ROOT)
testing::integration::detect_selector_root() {
    local scenario_dir="$1"
    if [ -n "${WORKFLOW_LINT_SELECTOR_ROOT:-}" ]; then
        printf '%s\n' "${WORKFLOW_LINT_SELECTOR_ROOT}"
        return 0
    fi
    if [ -z "$scenario_dir" ]; then
        printf '\n'
        return 0
    fi
    local candidate="$scenario_dir/ui/src"
    if [ -d "$candidate" ]; then
        testing::integration::realpath "$candidate"
        return 0
    fi
    printf '\n'
}

testing::integration::realpath() {
    local target="$1"
    if command -v realpath >/dev/null 2>&1; then
        realpath "$target"
        return 0
    fi
    if command -v python3 >/dev/null 2>&1; then
        python3 - "$target" <<'PY'
import os, sys
print(os.path.abspath(sys.argv[1]))
PY
        return 0
    fi
    printf '%s\n' "$target"
}

# Check whether the running API exposes /workflows/validate so we can safely lint via HTTP.
testing::integration::ensure_validate_endpoint() {
    local api_base="$1"
    if [ -z "$api_base" ]; then
        return 2
    fi

    local probe_payload
    probe_payload='{"workflow":{"metadata":{"description":"lint probe"},"nodes":[{"id":"probe-navigate","type":"navigate","position":{"x":0,"y":0},"data":{"destinationType":"url","url":"https://example.com"}},{"id":"probe-wait","type":"wait","position":{"x":120,"y":0},"data":{"waitType":"duration","durationMs":10}}],"edges":[{"id":"probe-edge","source":"probe-navigate","target":"probe-wait"}]},"strict":false}'

    local probe_file
    probe_file=$(mktemp)
    local status
    local curl_rc=0
    status=$(curl -s -o "$probe_file" -w "%{http_code}" -X POST "$api_base/workflows/validate" \
        -H 'Content-Type: application/json' -d "$probe_payload") || curl_rc=$?

    rm -f "$probe_file"

    if [ "$curl_rc" -ne 0 ]; then
        echo "❌ Failed to contact ${api_base}/workflows/validate (curl exit $curl_rc)"
        return 2
    fi

    case "$status" in
        200|400|422)
            return 0
            ;;
        404)
            echo "⚠️  ${api_base}/workflows/validate not available (404)"
            return 1
            ;;
        *)
            echo "❌ Unexpected status ${status} probing ${api_base}/workflows/validate"
            return 2
            ;;
    esac
}

# Run workflow linting against the BAS API before executing playbooks
testing::integration::lint_workflows_via_api() {
    if [[ "${WORKFLOW_LINT_API:-1}" =~ ^(0|false|no)$ ]]; then
        return 0
    fi
    local scenario_name="$1"
    local scenario_dir="${TESTING_PHASE_SCENARIO_DIR:-$(pwd)}"
    local playbook_dir="$scenario_dir/test/playbooks"

    if [ -z "$scenario_name" ] || [ ! -d "$playbook_dir" ]; then
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        echo "⚠️  jq not available; skipping workflow lint"
        return 0
    fi

    mapfile -t lint_files < <(find "$playbook_dir" -type f -name '*.json' | sort)
    if [ ${#lint_files[@]} -eq 0 ]; then
        return 0
    fi

    local strict_flag=false
    if [[ "${WORKFLOW_LINT_STRICT:-}" =~ ^(1|true|yes)$ ]]; then
        strict_flag=true
    fi

    if ! testing::core::is_scenario_running "$scenario_name"; then
        vrooli scenario start "$scenario_name" --clean-stale >/dev/null 2>&1 || true
    fi

    local readiness_timeout="${WORKFLOW_LINT_API_WAIT:-120}"
    if ! testing::core::wait_for_scenario "$scenario_name" "$readiness_timeout" >/dev/null 2>&1; then
        echo "❌ Scenario $scenario_name did not become ready for linting"
        return 1
    fi

    local api_port
    api_port=$(vrooli scenario port "$scenario_name" API_PORT 2>/dev/null || echo "")
    if [ -z "$api_port" ]; then
        echo "❌ Unable to resolve API_PORT for $scenario_name; cannot lint workflows"
        return 1
    fi

    local api_base="http://localhost:${api_port}/api/v1"
    local lint_failed=0
    local selector_root
    selector_root=$(testing::integration::detect_selector_root "$scenario_dir")

    local probe_rc=0
    testing::integration::ensure_validate_endpoint "$api_base" || probe_rc=$?
    if [ "$probe_rc" -eq 1 ]; then
        echo "⚠️  Scenario API does not expose /workflows/validate; skipping API lint"
        return 0
    fi
    if [ "$probe_rc" -ne 0 ]; then
        return $probe_rc
    fi

    for file_path in "${lint_files[@]}"; do
        local rel_path
        rel_path="${file_path#$scenario_dir/}"
        echo "🔍 Linting workflow ${rel_path}"

        local payload
        if ! payload=$(jq -c --argjson strict "$strict_flag" --arg selector "$selector_root" '{workflow: ., strict: $strict, selector_root: ($selector // "")}' "$file_path" 2>/dev/null); then
            echo "❌ Failed to build lint payload for $rel_path"
            lint_failed=1
            continue
        fi

        local response_file
        response_file=$(mktemp)
        local status
        local curl_rc=0
        status=$(curl -s -o "$response_file" -w "%{http_code}" -X POST "$api_base/workflows/validate" \
            -H 'Content-Type: application/json' -d "$payload") || curl_rc=$?

        if [ "$curl_rc" -ne 0 ]; then
            echo "❌ Lint request failed for $rel_path (curl exit $curl_rc)"
            lint_failed=1
            rm -f "$response_file"
            continue
        fi

        if [ "$status" != "200" ]; then
            local message
            message=$(jq -r '.message // .error // empty' "$response_file" 2>/dev/null || cat "$response_file")
            echo "❌ Lint request failed for $rel_path: ${message:-unknown error}"
            lint_failed=1
            rm -f "$response_file"
            continue
        fi

        local valid
        valid=$(jq -r '.valid // false' "$response_file" 2>/dev/null)
        local node_count edge_count
        node_count=$(jq -r '.stats.node_count // 0' "$response_file" 2>/dev/null)
        edge_count=$(jq -r '.stats.edge_count // 0' "$response_file" 2>/dev/null)

        if [ "$valid" = "true" ]; then
            echo "   ✅ Passed (${node_count} nodes, ${edge_count} edges)"
            local warning_count
            warning_count=$(jq '.warnings | length' "$response_file" 2>/dev/null || echo 0)
            if [ "$warning_count" -gt 0 ]; then
                echo "   ⚠️  ${warning_count} warning(s):"
                jq -r '.warnings[] | "      [warn:\(.code // "warning")] \(.message // "")"' "$response_file"
            fi
        else
            echo "   ❌ Failed (${node_count} nodes, ${edge_count} edges)"
            jq -r '.errors[]? | "      [\(.code // "error")] \(.message // "")"' "$response_file"
            jq -r '.warnings[]? | "      [warn:\(.code // "warning")] \(.message // "")"' "$response_file"
            lint_failed=1
        fi

        rm -f "$response_file"
    done

    return $lint_failed
}

# Run Browser Automation Studio workflow validations (or other workflow-based checks)
testing::integration::validate_all() {
    local summary="Integration tests completed"
    local force_end=false

    # Set up signal handler to ensure completion logic runs on timeout
    _integration_cleanup() {
        if [ "$force_end" = true ]; then
            return 0  # Already cleaned up
        fi
        force_end=true

        if [ "$TESTING_PHASE_AUTO_MANAGED" = true ]; then
            # Add note that phase was interrupted
            testing::phase::add_error "Phase timed out - recording partial results"
            testing::phase::auto_lifecycle_end "Integration tests timed out (partial results)"
        fi
    }
    trap _integration_cleanup SIGTERM SIGINT EXIT

    testing::phase::auto_lifecycle_start \
        --phase-name "integration" \
        --default-target-time "240s" \
        --summary "$summary" \
        --config-phase-key "integration" \
        || true

    local scenario_dir="${TESTING_PHASE_SCENARIO_DIR:-$(pwd)}"
    local scenario_name="$(basename "$scenario_dir")"

    if ! testing::integration::lint_workflows_via_api "$scenario_name"; then
        testing::phase::add_error "Workflow linting failed"
        force_end=true
        testing::phase::auto_lifecycle_end "Workflow linting failed"
        return 1
    fi

    if ! testing::phase::run_bas_automation_validations --scenario "$scenario_name" --manage-runtime auto; then
        local bas_rc=$?
        if [ "$bas_rc" -ne 0 ] && [ "$bas_rc" -ne 200 ]; then
            testing::phase::add_error "Browser Automation Studio workflow validations failed"
        fi
    fi

    force_end=true  # Mark that we're doing normal completion
    testing::phase::auto_lifecycle_end "$summary"
}

export -f testing::integration::validate_all
