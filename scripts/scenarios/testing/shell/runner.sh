#!/usr/bin/env bash
# Generic phased test runner orchestrator shared by scenarios
set -euo pipefail

SHELL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SHELL_DIR/runtime.sh"
source "$SHELL_DIR/reporting.sh"
source "$SHELL_DIR/artifacts.sh"
source "$SHELL_DIR/parallel.sh"
source "$SHELL_DIR/cache.sh"

# -----------------------------------------------------------------------------
# Global state
# -----------------------------------------------------------------------------

TESTING_RUNNER_SCENARIO_NAME=""
TESTING_RUNNER_TEST_DIR=""
TESTING_RUNNER_SCENARIO_DIR=""
TESTING_RUNNER_LOG_DIR=""
TESTING_RUNNER_DEFAULT_TIMEOUT_MULTIPLIER=1
TESTING_RUNNER_DEFAULT_MANAGE_RUNTIME=false

TESTING_RUNNER_PHASES=()
declare -A TESTING_RUNNER_PHASE_SCRIPT=()
declare -A TESTING_RUNNER_PHASE_TIMEOUT=()
declare -A TESTING_RUNNER_PHASE_OPTIONAL=()
declare -A TESTING_RUNNER_PHASE_RUNTIME=()
declare -A TESTING_RUNNER_PHASE_DISPLAY=()
declare -A TESTING_RUNNER_PHASE_CACHE_TTL=()
declare -A TESTING_RUNNER_PHASE_CACHE_KEYS=()

TESTING_RUNNER_TEST_TYPES=()
declare -A TESTING_RUNNER_TEST_HANDLER=()
declare -A TESTING_RUNNER_TEST_KIND=()
declare -A TESTING_RUNNER_TEST_RUNTIME=()
declare -A TESTING_RUNNER_TEST_DISPLAY=()

declare -A TESTING_RUNNER_PRESETS=()

TESTING_RUNNER_EXECUTION_ITEMS=()
declare -A TESTING_RUNNER_ITEM_TYPE=()
declare -A TESTING_RUNNER_ITEM_DISPLAY=()
declare -A TESTING_RUNNER_STATUS=()
declare -A TESTING_RUNNER_DURATION=()
declare -A TESTING_RUNNER_LOG_PATH=()

# -----------------------------------------------------------------------------
# Registration
# -----------------------------------------------------------------------------

testing::runner::init() {
    local scenario_name=""
    local test_dir=""
    local log_dir=""
    local default_manage="false"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --scenario-name)
                scenario_name="$2"
                shift 2
                ;;
            --scenario-dir)
                TESTING_RUNNER_SCENARIO_DIR="$2"
                shift 2
                ;;
            --test-dir)
                test_dir="$2"
                shift 2
                ;;
            --log-dir)
                log_dir="$2"
                shift 2
                ;;
            --default-timeout-multiplier)
                TESTING_RUNNER_DEFAULT_TIMEOUT_MULTIPLIER="$2"
                shift 2
                ;;
            --default-manage-runtime)
                default_manage="$2"
                shift 2
                ;;
            *)
                echo "Unknown option to testing::runner::init: $1" >&2
                return 1
                ;;
        esac
    done

    if [ -z "$scenario_name" ] || [ -z "$test_dir" ]; then
        echo "testing::runner::init requires --scenario-name and --test-dir" >&2
        return 1
    fi

    TESTING_RUNNER_SCENARIO_NAME="$scenario_name"
    TESTING_RUNNER_TEST_DIR="$test_dir"
    if [ -z "$TESTING_RUNNER_SCENARIO_DIR" ]; then
        TESTING_RUNNER_SCENARIO_DIR="$(cd "$test_dir/.." && pwd)"
    fi
    TESTING_RUNNER_LOG_DIR="${log_dir:-$test_dir/artifacts}"
    mkdir -p "$TESTING_RUNNER_LOG_DIR"

    TESTING_RUNNER_DEFAULT_MANAGE_RUNTIME="$default_manage"
    testing::runtime::configure "$scenario_name" "$default_manage"
    
    # Configure artifact management
    testing::artifacts::configure \
        --dir "$TESTING_RUNNER_LOG_DIR" \
        --max-logs 10 \
        --compress-old true \
        --retention-days 7 \
        --auto-cleanup true
    
    # Configure parallel execution
    testing::parallel::configure \
        --enabled true \
        --max-workers 4
    
    # Configure test result caching
    testing::cache::configure \
        --dir "$TESTING_RUNNER_LOG_DIR" \
        --enabled true \
        --default-ttl 3600 \
        --check-git true
}

# Register a phase script
testing::runner::register_phase() {
    local name=""
    local script=""
    local timeout="60"
    local requires_runtime="false"
    local optional="false"
    local display=""
    local cache_ttl="0"
    local cache_key_from=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --script)
                script="$2"
                shift 2
                ;;
            --timeout)
                timeout="$2"
                shift 2
                ;;
            --requires-runtime)
                requires_runtime="$2"
                shift 2
                ;;
            --optional)
                optional="$2"
                shift 2
                ;;
            --display)
                display="$2"
                shift 2
                ;;
            --cache-ttl)
                cache_ttl="$2"
                shift 2
                ;;
            --cache-key-from)
                cache_key_from="$2"
                shift 2
                ;;
            *)
                echo "Unknown option to testing::runner::register_phase: $1" >&2
                return 1
                ;;
        esac
    done

    if [ -z "$name" ] || [ -z "$script" ]; then
        echo "register_phase requires --name and --script" >&2
        return 1
    fi

    TESTING_RUNNER_PHASES+=("$name")
    TESTING_RUNNER_PHASE_SCRIPT["$name"]="$script"
    TESTING_RUNNER_PHASE_TIMEOUT["$name"]="$timeout"
    TESTING_RUNNER_PHASE_OPTIONAL["$name"]="$optional"
    TESTING_RUNNER_PHASE_RUNTIME["$name"]="$requires_runtime"
    TESTING_RUNNER_PHASE_DISPLAY["$name"]="${display:-$name}"
    TESTING_RUNNER_PHASE_CACHE_TTL["$name"]="$cache_ttl"
    TESTING_RUNNER_PHASE_CACHE_KEYS["$name"]="$cache_key_from"
    
    # Configure phase-specific caching
    if [ "$cache_ttl" -gt 0 ]; then
        testing::cache::configure_phase "$name" \
            --ttl "$cache_ttl" \
            --key-from "$cache_key_from" \
            --enabled true
    fi
}

# Register a test type handler
testing::runner::register_test_type() {
    local name=""
    local handler=""
    local kind="script"
    local requires_runtime="false"
    local display=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --handler)
                handler="$2"
                shift 2
                ;;
            --kind)
                kind="$2"
                shift 2
                ;;
            --requires-runtime)
                requires_runtime="$2"
                shift 2
                ;;
            --display)
                display="$2"
                shift 2
                ;;
            *)
                echo "Unknown option to testing::runner::register_test_type: $1" >&2
                return 1
                ;;
        esac
    done

    if [ -z "$name" ] || [ -z "$handler" ]; then
        echo "register_test_type requires --name and --handler" >&2
        return 1
    fi

    TESTING_RUNNER_TEST_TYPES+=("$name")
    TESTING_RUNNER_TEST_HANDLER["$name"]="$handler"
    TESTING_RUNNER_TEST_KIND["$name"]="$kind"
    TESTING_RUNNER_TEST_RUNTIME["$name"]="$requires_runtime"
    TESTING_RUNNER_TEST_DISPLAY["$name"]="${display:-$name}"
}

# Define presets (quick, smoke, etc.)
testing::runner::define_preset() {
    local preset="$1"
    shift
    TESTING_RUNNER_PRESETS["$preset"]="$*"
}

# Define a parallel group of phases
testing::runner::define_parallel_group() {
    local group_name="$1"
    shift
    testing::parallel::define_group "$group_name" "$@"
}

# -----------------------------------------------------------------------------
# Helpers
# -----------------------------------------------------------------------------

testing::runner::usage() {
    cat <<EOF
🧪 Scenario Test Runner

USAGE:
    run-tests.sh [phases/tests] [options]

OPTIONS:
    --verbose, -v          Verbose output
    --parallel, -p         Execute selections in parallel (no runtime management)
    --timeout N            Multiply phase timeouts by N
    --dry-run, -n          Show commands without executing
    --continue, -c         Continue after failures
    --junit                Generate junit-results.xml
    --coverage             Emit coverage artifact hints
    --manage-runtime       Auto-start/stop scenario runtime
    --no-manage-runtime    Disable auto runtime management
    --help, -h             Show this help message
EOF

    if [ ${#TESTING_RUNNER_PRESETS[@]} -gt 0 ]; then
        echo ""
        echo "PRESETS:"
        for preset in "${!TESTING_RUNNER_PRESETS[@]}"; do
            printf "    %-10s %s\n" "$preset" "${TESTING_RUNNER_PRESETS[$preset]}"
        done
    fi

    if [ ${#TESTING_RUNNER_PHASES[@]} -gt 0 ]; then
        echo ""
        echo "PHASES:"
        for phase in "${TESTING_RUNNER_PHASES[@]}"; do
            printf "    %-12s %ss\n" "$phase" "${TESTING_RUNNER_PHASE_TIMEOUT[$phase]}"
        done
    fi

    if [ ${#TESTING_RUNNER_TEST_TYPES[@]} -gt 0 ]; then
        echo ""
        echo "TEST TYPES: ${TESTING_RUNNER_TEST_TYPES[*]}"
    fi
}

_testing_runner_is_phase() {
    local item="$1"
    for phase in "${TESTING_RUNNER_PHASES[@]}"; do
        if [ "$phase" = "$item" ]; then
            return 0
        fi
    done
    return 1
}

_testing_runner_is_test() {
    local item="$1"
    for t in "${TESTING_RUNNER_TEST_TYPES[@]}"; do
        if [ "$t" = "$item" ]; then
            return 0
        fi
    done
    return 1
}

_testing_runner_record() {
    local key="$1"
    local type="$2"
    local display="$3"
    local status="$4"
    local duration="$5"
    local log_path="${6:-}"

    TESTING_RUNNER_STATUS["$key"]="$status"
    TESTING_RUNNER_DURATION["$key"]="$duration"
    if [ -n "$log_path" ]; then
        TESTING_RUNNER_LOG_PATH["$key"]="$log_path"
    fi
    TESTING_RUNNER_ITEM_TYPE["$key"]="$type"
    TESTING_RUNNER_ITEM_DISPLAY["$key"]="$display"
    TESTING_RUNNER_EXECUTION_ITEMS+=("$key")
}

_testing_runner_join_list() {
    local separator="$1"
    shift || true
    local result=""
    local first=true
    for item in "$@"; do
        if [ "$first" = true ]; then
            result="$item"
            first=false
        else
            result+="$separator$item"
        fi
    done
    echo "$result"
}

_testing_runner_friendly_label() {
    local key="$1"
    local type="${TESTING_RUNNER_ITEM_TYPE[$key]:-}"
    local display="${TESTING_RUNNER_ITEM_DISPLAY[$key]:-}" 

    case "$type" in
        phase)
            echo "phase '${display#phase-}'"
            ;;
        test)
            echo "test '${display#test-}'"
            ;;
        *)
            echo "$display"
            ;;
    esac
}

testing::runner::print_contextual_tips() {
    local overall_duration="${1:-0}"
    if ! [[ "$overall_duration" =~ ^[0-9]+$ ]]; then
        overall_duration=0
    fi

    local -a tips=()
    local -a timed_out=()
    local -a slow_timeouts=()
    local -a timed_out_stalled=()
    local -a missing=()
    local -a not_exec=()
    local -a failed_items=()
    local -a log_refs=()
    local -a high_timeout_workflows=()
    local -a wait_saturation_info=()
    local stuck_timeout_threshold=30
    local wait_saturation_threshold=5
    local api_port="${API_PORT:-17695}"
    local -A timed_out_stalled_lookup=()
    local -A scanned_timeout_logs=()
    local -A scanned_wait_logs=()
    local -A scanned_missing_logs=()
    local -A execution_lookup=()
    local -A missing_req_lookup=()
    local python_available=true
    if ! command -v python3 >/dev/null 2>&1; then
        python_available=false
    fi

    for key in "${TESTING_RUNNER_EXECUTION_ITEMS[@]}"; do
        local status="${TESTING_RUNNER_STATUS[$key]:-}"
        if [ -z "$status" ]; then
            continue
        fi
        local label=$(_testing_runner_friendly_label "$key")
        local log_path="${TESTING_RUNNER_LOG_PATH[$key]:-}"
        local duration="${TESTING_RUNNER_DURATION[$key]:-0}"
        if ! [[ "$duration" =~ ^[0-9]+$ ]]; then
            duration=0
        fi

        if [ -n "$log_path" ] && [ -f "$log_path" ] && [ -z "${scanned_timeout_logs[$log_path]:-}" ]; then
            scanned_timeout_logs[$log_path]=1

            local execution_id
            execution_id=$(awk '/Execution ID:/ {print $3}' "$log_path" | tail -n1)
            if [ -n "$execution_id" ]; then
                execution_lookup["$label"]="$execution_id"
            fi

            while IFS= read -r line; do
                [ -z "$line" ] && continue
                local detail="${line#*\[WF_TIMEOUT_HIGH\] }"
                if [ -z "$detail" ] || [ "$detail" = "$line" ]; then
                    detail="$(basename "$log_path") reports workflow timeout >90s"
                fi
                high_timeout_workflows+=("$detail")
            done < <(grep -F "[WF_TIMEOUT_HIGH]" "$log_path" 2>/dev/null || true)

            while IFS= read -r line; do
                [ -z "$line" ] && continue
                local detail="${line#*\[WF_RUNTIME_SLOW\] }"
                if [ -z "$detail" ] || [ "$detail" = "$line" ]; then
                    detail="$(basename "$log_path") reports workflow runtime >90s"
                fi
                slow_timeouts+=("$detail")
            done < <(grep -F "[WF_RUNTIME_SLOW]" "$log_path" 2>/dev/null || true)

            if [ "$python_available" = true ] && [ -z "${scanned_wait_logs[$log_path]:-}" ]; then
                scanned_wait_logs[$log_path]=1
                local saturation_output
                saturation_output=$(python3 - "$log_path" "$wait_saturation_threshold" <<'PY'
import re, sys
path = sys.argv[1]
threshold = int(sys.argv[2])
results = {}
try:
    with open(path, 'r', errors='ignore') as fh:
        prev = None
        streak = 0
        for line in fh:
            match = re.search(r"\[running\]\s+\d+% - wait \(([^)]+)\)", line)
            if not match:
                prev = None
                streak = 0
                continue
            step = match.group(1)
            if step == prev:
                streak += 1
            else:
                prev = step
                streak = 1
            if streak >= threshold:
                results[step] = max(results.get(step, threshold), streak)
except FileNotFoundError:
    pass
for step, count in results.items():
    print(f"{step}|{count}")
PY
)
                if [ -n "$saturation_output" ]; then
                    while IFS= read -r saturation_line; do
                        [ -z "$saturation_line" ] && continue
                        wait_saturation_info+=("$label|$saturation_line")
                    done <<< "$saturation_output"
                fi
            fi

            if [ -z "${scanned_missing_logs[$log_path]:-}" ]; then
                scanned_missing_logs[$log_path]=1
                while IFS= read -r missing_line; do
                    [ -z "$missing_line" ] && continue
                    local ids
                    ids=$(printf '%s' "$missing_line" | sed 's/.*://')
                    ids=${ids// /}
                    IFS=',' read -r -a req_array <<< "$ids"
                    for req in "${req_array[@]}"; do
                        [ -z "$req" ] && continue
                        missing_req_lookup["$req"]=1
                    done
                done < <(grep -F "Expected requirements missing coverage" "$log_path" 2>/dev/null || true)
            fi
        fi

        case "$status" in
            timed_out)
                timed_out+=("$label")
                if [ -n "$log_path" ]; then
                    log_refs+=("$label log: $log_path")
                fi
                if [ -n "$log_path" ] && [ "$duration" -ge "$stuck_timeout_threshold" ] && \
                   grep -Fq "Expected requirements missing coverage" "$log_path" 2>/dev/null; then
                    timed_out_stalled+=("$label")
                    timed_out_stalled_lookup["$label"]=1
                fi
                ;;
            failed)
                failed_items+=("$label")
                if [ -n "$log_path" ]; then
                    log_refs+=("$label log: $log_path")
                fi
                ;;
            missing)
                missing+=("$label")
                ;;
            not_executable)
                not_exec+=("$label")
                ;;
        esac
    done

    if [ ${#timed_out[@]} -gt 0 ]; then
        if [ ${#timed_out_stalled[@]} -gt 0 ]; then
            local stalled_joined=$(_testing_runner_join_list ", " "${timed_out_stalled[@]}")
            tips+=("• $stalled_joined ran past ${stuck_timeout_threshold}s and still left workflows uncovered. Consider lowering per-workflow wait/timeout inside the playbooks or raising the overall phase timeout so later workflows can run.")
        fi

        local -a regular_timeouts=()
        for item in "${timed_out[@]}"; do
            if [ -z "${timed_out_stalled_lookup["$item"]+x}" ]; then
                regular_timeouts+=("$item")
            fi
        done

        if [ ${#regular_timeouts[@]} -gt 0 ]; then
            local joined=$(_testing_runner_join_list ", " "${regular_timeouts[@]}")
            tips+=("• Timed out: $joined. Inspect the scenario logs or rerun with '--timeout 2' to grant more time once the root cause is understood.")
        fi
    fi

    if [ ${#slow_timeouts[@]} -gt 0 ]; then
        local joined=$(_testing_runner_join_list "; " "${slow_timeouts[@]}")
        tips+=("• Workflows exceeded 90s ($joined). BAS flows should finish quickly—trim waits/selectors or lower per-node timeouts so executions never block the test suite.")
    fi

    if [ ${#wait_saturation_info[@]} -gt 0 ]; then
        local -a saturation_descriptions=()
        for info in "${wait_saturation_info[@]}"; do
            local friendly="${info%%|*}"
            local remainder="${info#*|}"
            local step="${remainder%%|*}"
            local count="${remainder##*|}"
            saturation_descriptions+=("$friendly stuck on '$step' (~${count} waits)")
        done
        local joined=$(_testing_runner_join_list "; " "${saturation_descriptions[@]}")
        tips+=("• Wait saturation detected: $joined. This usually means the UI state never appeared or the selector is brittle—tag elements with stable data-testids and tighten the BAS selectors described in the UI automation guides below.")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        local joined=$(_testing_runner_join_list ", " "${missing[@]}")
        tips+=("• Missing scripts for $joined. Ensure the phase/test scripts exist in the scenario test directory or update the runner configuration.")
    fi

    if [ ${#not_exec[@]} -gt 0 ]; then
        local joined=$(_testing_runner_join_list ", " "${not_exec[@]}")
        tips+=("• Non-executable scripts for $joined. Run 'chmod +x' on the corresponding files so they can be invoked.")
    fi

    if [ ${#failed_items[@]} -gt 0 ]; then
        local joined=$(_testing_runner_join_list ", " "${failed_items[@]}")
        tips+=("• Failed items: $joined. Review their logs for assertion errors or stack traces.")
    fi

    if [ ${#log_refs[@]} -gt 0 ]; then
        local joined=$(_testing_runner_join_list "; " "${log_refs[@]}")
        tips+=("• Logs to inspect: $joined")
    fi

    if [ ${#missing_req_lookup[@]} -gt 0 ]; then
        local -a sorted_missing=()
        local req
        for req in "${!missing_req_lookup[@]}"; do
            sorted_missing+=("$req")
        done
        IFS=$'\n' sorted_missing=($(printf '%s\n' "${sorted_missing[@]}" | sort))
        unset IFS
        local joined=$(_testing_runner_join_list ", " "${sorted_missing[@]}")
        if [ ${#timed_out[@]} -gt 0 ]; then
            tips+=("• Requirements still uncovered ($joined) because integration exited early. Fix the blocking telemetry/seed workflows, rerun, then sync the registry using the requirement tracking process described below.")
        else
            tips+=("• Requirements still uncovered ($joined). Link each requirement to a BAS workflow validation or author a workflow following the requirement and UI automation guides listed below.")
        fi
    fi

    local -A execution_tip_lookup=()
    for item in "${timed_out[@]}"; do
        if [ -n "${execution_lookup[$item]:-}" ]; then
            execution_tip_lookup["$item:${execution_lookup[$item]}"]=1
        fi
    done
    for item in "${failed_items[@]}"; do
        if [ -n "${execution_lookup[$item]:-}" ]; then
            execution_tip_lookup["$item:${execution_lookup[$item]}"]=1
        fi
    done

    if [ ${#execution_tip_lookup[@]} -gt 0 ]; then
        local -a execution_entries=()
        local entry
        for entry in "${!execution_tip_lookup[@]}"; do
            local friendly="${entry%%:*}"
            local exec_id="${entry#*:}"
            execution_entries+=("$friendly → $exec_id")
        done
        local joined=$(_testing_runner_join_list "; " "${execution_entries[@]}")
        tips+=("• Execution artifacts: $joined. Fetch screenshots & DOM snapshots via curl http://localhost:${api_port}/api/v1/executions/<ID>/artifacts or load them in the BAS Execution Viewer as explained in the docs below.")
    fi

    if [ ${#high_timeout_workflows[@]} -gt 0 ]; then
        local joined=$(_testing_runner_join_list "; " "${high_timeout_workflows[@]}")
        tips+=("• Workflow timeouts too high (>90s): $joined. Lower BAS workflow timeouts so each run completes faster and avoids starving other workflows.")
    fi

    if [ "$overall_duration" -gt 600 ]; then
        tips+=("• Entire test selection ran ${overall_duration}s (>600s). Consider trimming per-workflow waits or splitting phases so runs stay under 10 minutes.")
    fi

    # Check for UI bundle issues in all failed phase logs
    local bundle_issue_detected=false
    for item in "${failed_items[@]}"; do
        if [[ "$item" == "phase"* ]]; then
            # Extract phase name from label like "phase 'structure'"
            local phase_name=$(echo "$item" | sed "s/^phase[: ]*['\"]*//" | sed "s/['\"].*$//")
            # Find the most recent log for this phase using artifacts directory
            local artifacts_dir="${TESTING_RUNNER_SCENARIO_DIR}/test/artifacts"
            if [[ -d "$artifacts_dir" ]]; then
                local phase_log=$(ls -t "${artifacts_dir}/${phase_name}"-*.log 2>/dev/null | head -1)
                if [[ -n "$phase_log" && -f "$phase_log" ]]; then
                    if grep -qE "UI bundle (outdated|missing|stale)|bundle.*stale|Bundle Status:.*stale" "$phase_log" 2>/dev/null; then
                        bundle_issue_detected=true
                        break
                    fi
                fi
            fi
        fi
    done

    if [ "$bundle_issue_detected" = true ]; then
        tips+=("• UI bundle outdated. Restart scenario to auto-rebuild: vrooli scenario restart ${TESTING_RUNNER_SCENARIO_NAME}")
    fi

    echo "💡 Troubleshooting tips:"
    if [ ${#tips[@]} -eq 0 ]; then
        echo "   • Ensure $TESTING_RUNNER_SCENARIO_NAME scenario is running: vrooli scenario start $TESTING_RUNNER_SCENARIO_NAME"
        echo "   • Check dependencies: vrooli resource start postgres"
        echo "   • Install CLI: cd cli && ./install.sh"
        echo "   • Run individual phases for detailed debugging"
        return
    fi

    for tip in "${tips[@]}"; do
        echo "   $tip"
    done
}

# -----------------------------------------------------------------------------
# Phase & test execution
# -----------------------------------------------------------------------------

testing::runner::run_phase() {
    local phase="$1"
    local timeout_multiplier="$2"
    local dry_run="$3"

    local script="${TESTING_RUNNER_PHASE_SCRIPT[$phase]}"
    local timeout="${TESTING_RUNNER_PHASE_TIMEOUT[$phase]}"
    local requires_runtime="${TESTING_RUNNER_PHASE_RUNTIME[$phase]:-false}"
    local optional="${TESTING_RUNNER_PHASE_OPTIONAL[$phase]:-false}"
    local display="phase-$phase"

    local id="phase:$phase"
    local actual_timeout=$((timeout * timeout_multiplier))
    local log_file=$(testing::artifacts::get_log_path "$phase")
    
    # Check cache first
    local cache_ttl="${TESTING_RUNNER_PHASE_CACHE_TTL[$phase]:-0}"
    local cache_key=""
    if [ "$cache_ttl" -gt 0 ] && [ "$dry_run" != "true" ]; then
        cache_key=$(testing::cache::generate_key "$phase" "$script")
        
        if testing::cache::is_valid "$phase" "$cache_key"; then
            if testing::cache::get "$phase" "$cache_key"; then
                if command -v log::info >/dev/null 2>&1; then
                    log::info "📦 Using cached result for phase: $phase"
                    log::info "   Cached ${CACHE_DURATION}s ago, exit code: $CACHE_EXIT_CODE"
                else
                    echo "📦 Using cached result for phase: $phase"
                    echo "   Cached ${CACHE_DURATION}s ago, exit code: $CACHE_EXIT_CODE"
                fi
                
                # Copy cached log to new location
                if [ -f "$CACHE_LOG_FILE" ]; then
                    cp "$CACHE_LOG_FILE" "$log_file"
                    cat "$CACHE_LOG_FILE"
                fi
                
                local status="passed"
                if [ "$CACHE_EXIT_CODE" -ne 0 ]; then
                    if [ "$CACHE_EXIT_CODE" -eq 200 ]; then
                        status="skipped"
                    else
                        status="failed"
                    fi
                fi
                
                _testing_runner_record "$id" "phase" "$display" "$status" "$CACHE_DURATION" "$log_file"
                echo ""
                return $CACHE_EXIT_CODE
            fi
        fi
    fi
    
    # Rotate logs for this phase before starting
    testing::artifacts::rotate_phase_logs "$phase"

    if command -v log::info >/dev/null 2>&1; then
        log::info "🚀 Running Phase: $phase"
        log::info "   Script: $script"
        log::info "   Timeout: ${actual_timeout}s"
    else
        echo "🚀 Running Phase: $phase"
        echo "   Script: $script"
        echo "   Timeout: ${actual_timeout}s"
    fi

    if ! testing::runtime::ensure_available "$phase" "$requires_runtime"; then
        _testing_runner_record "$id" "phase" "$display" "failed" 0 "$log_file"
        return 1
    fi

    if [ "$dry_run" = "true" ]; then
        if command -v log::warning >/dev/null 2>&1; then
            log::warning "   [DRY RUN] Would execute: $script"
        else
            echo "   [DRY RUN] Would execute: $script"
        fi
        _testing_runner_record "$id" "phase" "$display" "skipped" 0
        echo ""
        return 0
    fi

    if [ ! -f "$script" ]; then
        if [ "$optional" = "true" ]; then
            if command -v log::warning >/dev/null 2>&1; then
                log::warning "⚠️  Optional phase script missing: $script"
            else
                echo "⚠️  Optional phase script missing: $script"
            fi
            _testing_runner_record "$id" "phase" "$display" "skipped" 0
            echo ""
            return 0
        fi
        if command -v log::error >/dev/null 2>&1; then
            log::error "❌ Phase script not found: $script"
        else
            echo "❌ Phase script not found: $script" >&2
        fi
        _testing_runner_record "$id" "phase" "$display" "missing" 0
        echo ""
        return 1
    fi

    if [ ! -x "$script" ]; then
        if command -v log::error >/dev/null 2>&1; then
            log::error "❌ Phase script not executable: $script"
        else
            echo "❌ Phase script not executable: $script" >&2
        fi
        _testing_runner_record "$id" "phase" "$display" "not_executable" 0
        echo ""
        return 1
    fi

    : > "$log_file"
    local start_ts=$(date +%s)
    local result=0
    timeout "${actual_timeout}s" "$script" > >(tee "$log_file") 2>&1 || result=$?
    local end_ts=$(date +%s)
    local duration=$((end_ts - start_ts))

    case $result in
        0)
            if command -v log::success >/dev/null 2>&1; then
                log::success "✅ Phase '$phase' passed in ${duration}s"
            else
                echo "✅ Phase '$phase' passed in ${duration}s"
            fi
            _testing_runner_record "$id" "phase" "$display" "passed" "$duration" "$log_file"
            # Store in cache if enabled
            if [ "$cache_ttl" -gt 0 ] && [ "$dry_run" != "true" ]; then
                testing::cache::store "$phase" "$cache_key" 0 "$duration" "$log_file"
            fi
            ;;
        200)
            if command -v log::warning >/dev/null 2>&1; then
                log::warning "⚠️  Phase '$phase' skipped"
            else
                echo "⚠️  Phase '$phase' skipped"
            fi
            _testing_runner_record "$id" "phase" "$display" "skipped" "$duration" "$log_file"
            # Store in cache if enabled
            if [ "$cache_ttl" -gt 0 ] && [ "$dry_run" != "true" ]; then
                testing::cache::store "$phase" "$cache_key" 200 "$duration" "$log_file"
            fi
            result=0
            ;;
        124)
            if command -v log::error >/dev/null 2>&1; then
                log::error "❌ Phase '$phase' timed out after ${actual_timeout}s"
            else
                echo "❌ Phase '$phase' timed out after ${actual_timeout}s" >&2
            fi
            _testing_runner_record "$id" "phase" "$display" "timed_out" "$duration" "$log_file"
            ;;
        *)
            if command -v log::error >/dev/null 2>&1; then
                log::error "❌ Phase '$phase' failed with exit code $result in ${duration}s"
            else
                echo "❌ Phase '$phase' failed with exit code $result in ${duration}s" >&2
            fi
            _testing_runner_record "$id" "phase" "$display" "failed" "$duration" "$log_file"
            # Store in cache if enabled (might cache failures for debugging)
            if [ "$cache_ttl" -gt 0 ] && [ "$dry_run" != "true" ]; then
                testing::cache::store "$phase" "$cache_key" "$result" "$duration" "$log_file"
            fi
            ;;
    esac

    # Clean up artifacts after phase completion to prevent accumulation
    testing::artifacts::rotate_phase_logs "$phase"

    echo ""
    return $result
}

testing::runner::run_test_type() {
    local test_type="$1"
    local dry_run="$2"

    local handler="${TESTING_RUNNER_TEST_HANDLER[$test_type]}"
    local kind="${TESTING_RUNNER_TEST_KIND[$test_type]}"
    local requires_runtime="${TESTING_RUNNER_TEST_RUNTIME[$test_type]:-false}"
    local display="test-$test_type"
    local id="test:$test_type"

    if command -v log::info >/dev/null 2>&1; then
        log::info "🎯 Running Test Type: $test_type"
    else
        echo "🎯 Running Test Type: $test_type"
    fi

    if ! testing::runtime::ensure_available "$test_type" "$requires_runtime"; then
        _testing_runner_record "$id" "test" "$display" "failed" 0
        echo ""
        return 1
    fi

    if [ "$dry_run" = "true" ]; then
        if command -v log::warning >/dev/null 2>&1; then
            log::warning "   [DRY RUN] Would execute handler: $handler"
        else
            echo "   [DRY RUN] Would execute handler: $handler"
        fi
        _testing_runner_record "$id" "test" "$display" "skipped" 0
        echo ""
        return 0
    fi

    local start_ts=$(date +%s)
    local result=0
    case "$kind" in
        script)
            if [ ! -x "$handler" ]; then
                if command -v log::error >/dev/null 2>&1; then
                    log::error "❌ Test handler not executable: $handler"
                else
                    echo "❌ Test handler not executable: $handler" >&2
                fi
                result=1
            else
                "$handler"
                result=$?
            fi
            ;;
        function)
            "$handler"
            result=$?
            ;;
        command)
            eval "$handler"
            result=$?
            ;;
        *)
            if command -v log::error >/dev/null 2>&1; then
                log::error "❌ Unknown handler kind '$kind' for test type '$test_type'"
            else
                echo "❌ Unknown handler kind '$kind' for test type '$test_type'" >&2
            fi
            result=1
            ;;
    esac
    local end_ts=$(date +%s)
    local duration=$((end_ts - start_ts))

    if [ $result -eq 0 ]; then
        if command -v log::success >/dev/null 2>&1; then
            log::success "✅ Test type '$test_type' passed"
        else
            echo "✅ Test type '$test_type' passed"
        fi
        _testing_runner_record "$id" "test" "$display" "passed" "$duration"
    else
        if command -v log::error >/dev/null 2>&1; then
            log::error "❌ Test type '$test_type' failed"
        else
            echo "❌ Test type '$test_type' failed" >&2
        fi
        _testing_runner_record "$id" "test" "$display" "failed" "$duration"
    fi

    echo ""
    return $result
}

_testing_runner_print_header() {
    local selection_ref_name="$1"
    local verbose="$2"
    local parallel="$3"
    local dry_run="$4"
    local manage_runtime="$5"
    local -n selection_ref="$selection_ref_name"

    if command -v log::header >/dev/null 2>&1; then
        log::header "🧪 ${TESTING_RUNNER_SCENARIO_NAME^} Comprehensive Test Suite"
        log::info "   Test directory: $TESTING_RUNNER_TEST_DIR"
        log::info "   Scenario directory: $TESTING_RUNNER_SCENARIO_DIR"
        log::info "   Selected phases/types: ${selection_ref[*]}"
        log::info "   Options: verbose=$verbose, parallel=$parallel, dry-run=$dry_run, manage-runtime=$manage_runtime"
        log::info "   Ports: API=$API_PORT UI=$UI_PORT"
        echo ""
    else
        echo "🧪 ${TESTING_RUNNER_SCENARIO_NAME^} Comprehensive Test Suite"
        echo "Test directory: $TESTING_RUNNER_TEST_DIR"
        echo "Scenario directory: $TESTING_RUNNER_SCENARIO_DIR"
        echo "Selected phases/types: ${selection_ref[*]}"
        echo "Options: verbose=$verbose, parallel=$parallel, dry-run=$dry_run, manage-runtime=$manage_runtime"
        echo "Ports: API=$API_PORT UI=$UI_PORT"
        echo ""
    fi
}

_testing_runner_parallel() {
    local selection_ref_name="$1"
    local timeout_multiplier="$2"
    local dry_run="$3"
    local -n selection_ref="$selection_ref_name"

    # Try to use optimized parallel execution if available
    if testing::parallel::can_optimize "${selection_ref[@]}"; then
        # Override the start_phase function for parallel execution
        testing::parallel::start_phase() {
            local phase="$1"
            local log_dir="$2"
            local tm="$3"
            testing::runner::run_phase "$phase" "$tm" "$dry_run"
        }
        
        if testing::parallel::execute_phases selection_ref "$TESTING_RUNNER_LOG_DIR" "$timeout_multiplier"; then
            return 0
        else
            return 1
        fi
    fi
    
    # Fallback to simple parallel execution
    local pids=()
    local overall_result=0

    for item in "${selection_ref[@]}"; do
        if _testing_runner_is_phase "$item"; then
            ( testing::runner::run_phase "$item" "$timeout_multiplier" "$dry_run" ) &
        else
            ( testing::runner::run_test_type "$item" "$dry_run" ) &
        fi
        pids+=($!)
    done

    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            overall_result=1
        fi
    done

    return $overall_result
}

_testing_runner_print_summary() {
    local total_duration="$1"
    local coverage_requested="$2"

    local total=0
    local passed=0
    local failed=0
    local skipped=0

    for item in "${TESTING_RUNNER_EXECUTION_ITEMS[@]}"; do
        ((total++)) || true
        case "${TESTING_RUNNER_STATUS[$item]}" in
            passed)
                ((passed++)) || true
                ;;
            skipped)
                ((skipped++)) || true
                ;;
            "")
                ;;
            *)
                ((failed++)) || true
                ;;
        esac
    done

    echo ""
    echo "📊 Test Execution Summary"
    echo "═══════════════════════════════════════════"
    echo "   Items run: $total"
    echo "   Passed:    $passed"
    echo "   Failed:    $failed"
    echo "   Skipped:   $skipped"
    echo "   Duration:  ${total_duration}s"
    echo ""

    if [ "$coverage_requested" = "true" ]; then
        echo "📈 Coverage reports may be available in:"
        echo "   • Go: api/coverage.html"
        echo "   • Node.js: ui/coverage/lcov-report/index.html"
        echo "   • Python: htmlcov/index.html"
        echo ""
    fi

    # Show artifact summary
    testing::artifacts::summary
    echo ""
    
    # Show cache statistics
    testing::cache::stats
    echo ""
    
    if [ $failed -eq 0 ] && [ $skipped -eq 0 ]; then
        log::success "🎉 All tests passed successfully!"
        log::success "✅ ${TESTING_RUNNER_SCENARIO_NAME^} testing infrastructure is working correctly"
    elif [ $failed -eq 0 ]; then
        log::success "✅ Test execution completed (with some skipped)"
    else
        log::error "❌ Test execution completed with failures"
        log::error "   $failed of $total items failed"
        echo ""
        testing::runner::print_contextual_tips "$total_duration"
    fi

    echo ""
    echo "📚 For more information, see:"
    echo "   • docs/testing/architecture/PHASED_TESTING.md"
    echo "   • docs/testing/guides/writing-testable-uis.md"
    echo "   • docs/testing/guides/ui-automation-with-bas.md"
    echo "   • docs/testing/guides/requirement-tracking-quick-start.md"
    echo "   • Test files in: $TESTING_RUNNER_TEST_DIR"
    echo ""
}

# -----------------------------------------------------------------------------
# Entry point
# -----------------------------------------------------------------------------

testing::runner::execute() {
    local VERBOSE=false
    local PARALLEL=false
    local DRY_RUN=false
    local CONTINUE_ON_FAILURE=true
    local JUNIT_OUTPUT=false
    local COVERAGE=false
    local TIMEOUT_MULTIPLIER="$TESTING_RUNNER_DEFAULT_TIMEOUT_MULTIPLIER"
    local MANAGE_RUNTIME="$TESTING_RUNNER_DEFAULT_MANAGE_RUNTIME"

    local args=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --help|-h)
                testing::runner::usage
                return 0
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --parallel|-p)
                PARALLEL=true
                shift
                ;;
            --timeout)
                TIMEOUT_MULTIPLIER="$2"
                shift 2
                ;;
            --dry-run|-n)
                DRY_RUN=true
                shift
                ;;
            --continue|-c)
                CONTINUE_ON_FAILURE=true
                shift
                ;;
            --junit)
                JUNIT_OUTPUT=true
                shift
                ;;
            --coverage)
                COVERAGE=true
                shift
                ;;
            --manage-runtime)
                MANAGE_RUNTIME=true
                shift
                ;;
            --no-manage-runtime)
                MANAGE_RUNTIME=false
                shift
                ;;
            quick|smoke|core|all)
                args+=("$1")
                shift
                ;;
            *)
                args+=("$1")
                shift
                ;;
        esac
    done

    if [ "${TEST_MANAGE_RUNTIME:-}" = "true" ]; then
        MANAGE_RUNTIME=true
    fi

    if [ "$DRY_RUN" = "true" ]; then
        testing::runtime::set_dry_run
    fi

    testing::runtime::set_enabled "$MANAGE_RUNTIME"

    if [ "$PARALLEL" = "true" ] && [ "$MANAGE_RUNTIME" = "true" ]; then
        log::error "❌ Parallel execution is not supported with --manage-runtime"
        return 1
    fi

    local selections=()
    if [ ${#args[@]} -eq 0 ]; then
        selections=("${TESTING_RUNNER_PHASES[@]}")
    else
        for arg in "${args[@]}"; do
            if [[ -n "${TESTING_RUNNER_PRESETS[$arg]:-}" ]]; then
                read -r -a preset_items <<< "${TESTING_RUNNER_PRESETS[$arg]}"
                selections+=("${preset_items[@]}")
            elif [ "$arg" = "all" ]; then
                selections+=("${TESTING_RUNNER_PHASES[@]}")
            else
                selections+=("$arg")
            fi
        done
    fi

    testing::runtime::discover_ports "$TESTING_RUNNER_SCENARIO_NAME"
    _testing_runner_print_header selections "$VERBOSE" "$PARALLEL" "$DRY_RUN" "$MANAGE_RUNTIME"
    trap testing::runtime::cleanup EXIT

    local overall_start=$(date +%s)
    local execution_result=0

    if [ "$PARALLEL" = "true" ]; then
        if ! _testing_runner_parallel selections "$TIMEOUT_MULTIPLIER" "$DRY_RUN"; then
            execution_result=1
        fi
    else
        for item in "${selections[@]}"; do
            local status=0
            if _testing_runner_is_phase "$item"; then
                testing::runner::run_phase "$item" "$TIMEOUT_MULTIPLIER" "$DRY_RUN" || status=$?
            elif _testing_runner_is_test "$item"; then
                testing::runner::run_test_type "$item" "$DRY_RUN" || status=$?
            else
                log::error "❌ Unknown phase/test type: $item"
                _testing_runner_record "unknown:$item" "unknown" "$item" "failed" 0
                status=1
            fi

            if [ $status -ne 0 ]; then
                execution_result=1
                if [ "$CONTINUE_ON_FAILURE" != "true" ]; then
                    log::error "❌ Stopping execution due to failure in: $item"
                    break
                else
                    log::warning "⚠️  Continuing despite failure in: $item"
                fi
            fi
        done
    fi

    local overall_end=$(date +%s)
    local overall_duration=$((overall_end - overall_start))

    _testing_runner_print_summary "$overall_duration" "$COVERAGE"

    if [ "$JUNIT_OUTPUT" = "true" ]; then
        local junit_file="$TESTING_RUNNER_TEST_DIR/junit-results.xml"
        testing::reporting::generate_junit \
            "$TESTING_RUNNER_SCENARIO_NAME-tests" \
            "$junit_file" \
            "$overall_duration" \
            TESTING_RUNNER_STATUS \
            TESTING_RUNNER_DURATION \
            TESTING_RUNNER_LOG_PATH \
            TESTING_RUNNER_EXECUTION_ITEMS \
            TESTING_RUNNER_ITEM_DISPLAY \
            TESTING_RUNNER_ITEM_TYPE
        log::info "📄 JUnit results written to: $junit_file"
    fi

    if declare -F testing::artifacts::finalize_workspace >/dev/null 2>&1; then
        testing::artifacts::finalize_workspace
    fi

    return $execution_result
}

# -----------------------------------------------------------------------------
# Exports
# -----------------------------------------------------------------------------

export -f testing::runner::init
export -f testing::runner::register_phase
export -f testing::runner::register_test_type
export -f testing::runner::define_preset
export -f testing::runner::usage
export -f testing::runner::execute
export -f testing::runner::run_phase
export -f testing::runner::run_test_type
