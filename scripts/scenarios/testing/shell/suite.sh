#!/usr/bin/env bash
# Shared scenario test suite orchestrator
set -euo pipefail

SHELL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SHELL_DIR/runner.sh"

# Default phase metadata
declare -Ag TESTING_SUITE_DEFAULT_TIMEOUTS=(
    [structure]=30
    [dependencies]=60
    [unit]=120
    [integration]=600
    [business]=120
    [performance]=60
)

declare -Ag TESTING_SUITE_DEFAULT_RUNTIME=(
    [integration]=true
    [performance]=true
)

TESTING_SUITE_PRESET_ORDER=(quick smoke comprehensive)

declare -Ag TESTING_SUITE_DEFAULT_PRESETS=(
    [quick]="structure unit"
    [smoke]="structure integration"
    [comprehensive]="structure dependencies unit integration business performance"
)

declare -Ag TESTING_SUITE_PHASE_SCRIPTS=()
TESTING_SUITE_LINT_PERFORMED=false

_testing_suite_realpath() {
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

_testing_suite_detect_selector_root() {
    local scenario_dir="$1"
    if [ -n "${WORKFLOW_LINT_SELECTOR_ROOT:-}" ]; then
        printf '%s\n' "${WORKFLOW_LINT_SELECTOR_ROOT}"
        return 0
    fi
    if [ -z "$scenario_dir" ]; then
        printf '%s\n' ""
        return 0
    fi
    local candidate="$scenario_dir/ui/src"
    if [ -d "$candidate" ]; then
        _testing_suite_realpath "$candidate"
        return 0
    fi
    printf '%s\n' ""
}

_testing_suite_lint_workflows() {
    if [ "$TESTING_SUITE_LINT_PERFORMED" = true ]; then
        return 0
    fi
    TESTING_SUITE_LINT_PERFORMED=true

    local scenario_dir="$1"
    local scenario_name="$2"
    local app_root="$3"

    if [[ "${WORKFLOW_LINT_PRECHECK:-1}" =~ ^(0|false|no)$ ]]; then
        return 0
    fi

    local playbook_dir="$scenario_dir/test/playbooks"
    if [ ! -d "$playbook_dir" ]; then
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        log::warning "⚠️  jq not available; skipping workflow lint"
        return 0
    fi

    local invalid_found=0
    while IFS= read -r -d '' file; do
        if ! jq empty "$file" >/dev/null 2>&1; then
            log::error "❌ Invalid JSON in workflow playbook: ${file#$scenario_dir/}"
            invalid_found=1
        fi
    done < <(find "$playbook_dir" -type f -name '*.json' -print0)

    if [ "$invalid_found" -ne 0 ]; then
        log::error "Fix invalid workflow JSON before rerunning tests."
        return 1
    fi

    mapfile -t workflow_files < <(find "$playbook_dir" -type f -name '*.json' | sort)
    if [ ${#workflow_files[@]} -eq 0 ]; then
        return 0
    fi

    if ! command -v go >/dev/null 2>&1; then
        log::warning "⚠️  Go toolchain not available; skipping workflow lint"
        return 0
    fi

    local bas_api_dir="$app_root/scenarios/browser-automation-studio/api"
    if [ ! -d "$bas_api_dir" ]; then
        log::warning "⚠️  Browser Automation Studio validator not found; skipping workflow lint"
        return 0
    fi

    local selector_root
    selector_root=$(_testing_suite_detect_selector_root "$scenario_dir")

    local strict_env="${WORKFLOW_LINT_STRICT:-}" 
    local -a lint_cmd=(go run ./cmd/workflow-lint)
    if [[ "$strict_env" =~ ^(1|true|yes)$ ]]; then
        lint_cmd+=(--strict)
    fi
    if [ -n "$selector_root" ]; then
        lint_cmd+=(--selector-root "$selector_root")
    fi
    lint_cmd+=(--json)

    local lint_artifact_dir="$scenario_dir/test/artifacts/lint"
    mkdir -p "$lint_artifact_dir"
    local lint_artifact="$lint_artifact_dir/workflow-lint.json"

    local -a abs_files=()
    local file
    for file in "${workflow_files[@]}"; do
        abs_files+=("$(_testing_suite_realpath "$file")")
    done

    log::info "🔎 Linting ${#abs_files[@]} workflow playbooks (${scenario_name})"

    if ! (
        cd "$bas_api_dir"
        "${lint_cmd[@]}" "${abs_files[@]}"
    ) >"$lint_artifact"; then
        log::error "Workflow lint failed (see ${lint_artifact#$scenario_dir/})"
        _testing_suite_render_lint_summary "$lint_artifact"
        return 1
    fi

    _testing_suite_render_lint_summary "$lint_artifact"
    return 0
}

_testing_suite_render_lint_summary() {
    local artifact="$1"
    if [ ! -s "$artifact" ]; then
        return 0
    fi
    if ! command -v jq >/dev/null 2>&1; then
        cat "$artifact"
        return 0
    fi
    jq -r '
        .results[] as $res |
        if ($res.error // "") != "" then
            "✗ \($res.file)" + "\n  \($res.error)"
        else
            (if $res.result.valid then "✓" else "✗" end)
            + " \($res.file) (\($res.result.stats.node_count // 0) nodes, \($res.result.stats.edge_count // 0) edges)"
            + (if ($res.result.errors | length) > 0 then
                "\n" + ($res.result.errors[] |
                    "      [" + (.code // "error") + "] " + (.message // "")
                )
            else "" end)
            + (if ($res.result.warnings | length) > 0 then
                "\n" + ($res.result.warnings[] |
                    "      [warn:" + (.code // "warning") + "] " + (.message // "")
                )
            else "" end)
        end
    ' "$artifact"
}

_testing_suite_read_json_value() {
    local file="$1"
    local jq_expr="$2"
    local default_value="$3"

    if [ ! -f "$file" ] || ! command -v jq >/dev/null 2>&1; then
        printf '%s\n' "$default_value"
        return 0
    fi

    local value
    value=$(jq -r "$jq_expr // empty" "$file" 2>/dev/null || echo "")
    if [ -z "$value" ] || [ "$value" = "null" ]; then
        printf '%s\n' "$default_value"
    else
        printf '%s\n' "$value"
    fi
}

_testing_suite_phase_enabled() {
    local config_file="$1"
    local phase="$2"
    local default_value="true"

    if [ ! -f "$config_file" ] || ! command -v jq >/dev/null 2>&1; then
        printf '%s\n' "$default_value"
        return 0
    fi

    local enabled
    enabled=$(jq -r ".phases.${phase}.enabled" "$config_file" 2>/dev/null || echo "null")
    if [ "$enabled" = "false" ]; then
        printf 'false\n'
    else
        printf '%s\n' "$default_value"
    fi
}

_testing_suite_phase_timeout() {
    local config_file="$1"
    local phase="$2"
    _testing_suite_read_json_value "$config_file" ".phases.${phase}.timeout" ""
}

_testing_suite_phase_require_runtime() {
    local config_file="$1"
    local phase="$2"
    local fallback="${TESTING_SUITE_DEFAULT_RUNTIME[$phase]:-false}"
    _testing_suite_read_json_value "$config_file" ".phases.${phase}.require_runtime" "$fallback"
}

_testing_suite_time_to_seconds() {
    local value="$1"
    if [ -z "$value" ]; then
        echo ""
        return 0
    fi

    if [[ "$value" =~ ^[0-9]+$ ]]; then
        echo "$value"
        return 0
    fi

    if [[ "$value" =~ ^([0-9]+)([smh])$ ]]; then
        local number="${BASH_REMATCH[1]}"
        local unit="${BASH_REMATCH[2]}"
        case "$unit" in
            s) echo "$number" ;;
            m) echo $((number * 60)) ;;
            h) echo $((number * 3600)) ;;
        esac
        return 0
    fi

    echo ""
}

_testing_suite_find_service_name() {
    local scenario_dir="$1"
    local service_file="$scenario_dir/.vrooli/service.json"
    local fallback="$(basename "$scenario_dir")"
    _testing_suite_read_json_value "$service_file" '.service.name' "$fallback"
}

_testing_suite_sync_requirements() {
    local scenario_dir="$1"
    local scenario_name="$2"
    local app_root="$3"

    local registry_file="$scenario_dir/docs/requirements.json"
    local registry_dir="$scenario_dir/requirements"

    if [ "${TESTING_REQUIREMENTS_SYNC:-1}" != "1" ]; then
        return 0
    fi

    if [ ! -f "$registry_file" ] && [ ! -d "$registry_dir" ]; then
        log::warning "⚠️  Skipping requirements sync (no registry found)"
        return 0
    fi

    if ! command -v node >/dev/null 2>&1; then
        log::warning "⚠️  Skipping requirements sync (Node.js not available)"
        return 0
    fi

    if node "$app_root/scripts/requirements/report.js" --scenario "$scenario_name" --mode sync >/dev/null; then
        log::info "📋 Requirements registry synced after test run"
    else
        log::warning "⚠️  Failed to sync requirements registry; leaving requirement files unchanged"
    fi
}

_testing_suite_register_presets() {
    local config_file="$1"
    shift
    local -a available_phases=("$@")
    declare -A available_map=()
    for phase in "${available_phases[@]}"; do
        available_map["$phase"]=1
    done

    local custom_registered=false
    if [ -f "$config_file" ] && command -v jq >/dev/null 2>&1; then
        while IFS=$'\t' read -r preset phase_list; do
            [ -z "$preset" ] && continue
            local -a filtered=()
            for phase in $phase_list; do
                if [ -n "${available_map[$phase]+x}" ]; then
                    filtered+=("$phase")
                fi
            done
            if [ ${#filtered[@]} -gt 0 ]; then
                testing::runner::define_preset "$preset" "${filtered[*]}"
                custom_registered=true
            fi
        done < <(jq -r '.presets // {} | to_entries[] | "\(.key)\t\(.value | join(\" \"))"' "$config_file" 2>/dev/null || true)
    fi

    if [ "$custom_registered" = false ]; then
        for preset in "${TESTING_SUITE_PRESET_ORDER[@]}"; do
            local phases="${TESTING_SUITE_DEFAULT_PRESETS[$preset]}"
            local -a filtered=()
            for phase in $phases; do
                if [ -n "${available_map[$phase]+x}" ]; then
                    filtered+=("$phase")
                fi
            done
            if [ ${#filtered[@]} -gt 0 ]; then
                testing::runner::define_preset "$preset" "${filtered[*]}"
            fi
        done
    fi
}

declare -Ag TESTING_SUITE_PHASE_SCRIPTS=()

_testing_suite_collect_phases() {
    local test_dir="$1"
    local phase_name
    TESTING_SUITE_PHASE_SCRIPTS=()

    if [ ! -d "$test_dir/phases" ]; then
        return 0
    fi

    while IFS= read -r script_path; do
        local filename
        filename="$(basename "$script_path")"
        phase_name="${filename#test-}"
        phase_name="${phase_name%.sh}"
        TESTING_SUITE_PHASE_SCRIPTS["$phase_name"]="$script_path"
    done < <(find "$test_dir/phases" -maxdepth 1 -name 'test-*.sh' -type f | sort)
}

testing::suite::run() {
    local scenario_dir=""
    local scenario_name=""
    local -a runner_args=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --scenario-dir)
                scenario_dir="$2"
                shift 2
                ;;
            --scenario-name)
                scenario_name="$2"
                shift 2
                ;;
            --)
                shift
                runner_args=("$@")
                break
                ;;
            *)
                runner_args+=("$1")
                shift
                ;;
        esac
    done

    if [ -z "$scenario_dir" ]; then
        scenario_dir="$(pwd)"
    fi

    local test_dir="$scenario_dir/test"
    local app_root="${APP_ROOT:-$(cd "$scenario_dir/../.." && pwd)}"
    export APP_ROOT="$app_root"

    source "$APP_ROOT/scripts/lib/utils/log.sh"

    local config_file="$scenario_dir/.vrooli/testing.json"

    if [ -z "$scenario_name" ]; then
        scenario_name=$(_testing_suite_find_service_name "$scenario_dir")
    fi

    export TESTING_REQUIREMENTS_ENFORCE=0
    export TESTING_REQUIREMENTS_SYNC=1

    if [ -f "$config_file" ]; then
        local enforce_flag
        enforce_flag=$(_testing_suite_read_json_value "$config_file" '.requirements.enforce' "false")
        if [ "$enforce_flag" = "true" ]; then
            export TESTING_REQUIREMENTS_ENFORCE=1
        fi

        local sync_flag
        sync_flag=$(_testing_suite_read_json_value "$config_file" '.requirements.sync' "true")
        if [ "$sync_flag" != "true" ]; then
            export TESTING_REQUIREMENTS_SYNC=0
        fi
    fi

    local skip_lint=false
    local arg
    for arg in "${runner_args[@]}"; do
        case "$arg" in
            -h|--help|help|--usage)
                skip_lint=true
                break
                ;;
        esac
    done

    if [ "$skip_lint" = false ]; then
        if ! _testing_suite_lint_workflows "$scenario_dir" "$scenario_name" "$app_root"; then
            return 1
        fi
    fi

    local log_dir="$test_dir/artifacts"

    testing::runner::init \
        --scenario-name "$scenario_name" \
        --scenario-dir "$scenario_dir" \
        --test-dir "$test_dir" \
        --log-dir "$log_dir" \
        --default-manage-runtime true

    _testing_suite_collect_phases "$test_dir"

    declare -a register_order=(structure dependencies unit integration business performance)
    declare -a extra_phases=()

    for phase in "${!TESTING_SUITE_PHASE_SCRIPTS[@]}"; do
        local known=false
        for default_phase in "${register_order[@]}"; do
            if [ "$phase" = "$default_phase" ]; then
                known=true
                break
            fi
        done
        if [ "$known" = false ]; then
            extra_phases+=("$phase")
        fi
    done

    IFS=$'\n' extra_phases=($(sort <<<"${extra_phases[*]-}"))
    unset IFS

    register_order+=("${extra_phases[@]}")

    declare -a registered_phases=()

    for phase in "${register_order[@]}"; do
        local script="${TESTING_SUITE_PHASE_SCRIPTS[$phase]:-}"
        [ -z "$script" ] && continue

        local enabled
        enabled=$(_testing_suite_phase_enabled "$config_file" "$phase")
        if [ "$enabled" = "false" ]; then
            continue
        fi

        local timeout_config
        timeout_config=$(_testing_suite_phase_timeout "$config_file" "$phase")
        local timeout_seconds
        timeout_seconds=$(_testing_suite_time_to_seconds "$timeout_config")
        if [ -z "$timeout_seconds" ]; then
            timeout_seconds="${TESTING_SUITE_DEFAULT_TIMEOUTS[$phase]:-120}"
        fi

        local require_runtime
        require_runtime=$(_testing_suite_phase_require_runtime "$config_file" "$phase")

        testing::runner::register_phase \
            --name "$phase" \
            --script "$script" \
            --timeout "$timeout_seconds" \
            --requires-runtime "$require_runtime"

        registered_phases+=("$phase")
    done

    if [ ${#registered_phases[@]} -eq 0 ]; then
        log::error "No test phases detected; ensure test/phases contains test-*.sh scripts"
        return 1
    fi

    _testing_suite_register_presets "$config_file" "${registered_phases[@]}"

    local result=0
    testing::runner::execute "${runner_args[@]}" || result=$?

    _testing_suite_sync_requirements "$scenario_dir" "$scenario_name" "$app_root"

    return $result
}

export -f testing::suite::run
