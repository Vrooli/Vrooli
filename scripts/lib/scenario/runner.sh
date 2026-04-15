#!/usr/bin/env bash
set -euo pipefail

# Get script directory for sourcing utils
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
source "${SCRIPT_DIR}/../utils/var.sh"
source "${SCRIPT_DIR}/../utils/log.sh"
source "${SCRIPT_DIR}/dependencies.sh"

# =============================================================================
# Sandbox-aware path resolution helpers
#
# When an AI agent runs inside an overlayfs sandbox (managed by workspace-sandbox),
# its file changes are captured in the overlay's upper/ layer — the real repo is
# untouched. The agent-manager injects three environment variables into the agent's
# process so the Vrooli CLI can transparently redirect scenario paths to the sandbox:
#
#   VROOLI_SANDBOX_ID     - sandbox UUID (for logging/debugging)
#   VROOLI_SANDBOX_MERGED - absolute path to the overlay's merged/ directory
#   VROOLI_SANDBOX_SCOPE  - relative scope path within the project
#                           (e.g. "scenarios/git-control-tower")
#
# This allows agents to run "vrooli scenario restart foo" and have the lifecycle
# system build and run from the sandbox (where their changes live), without the
# agent needing to pass any special flags or even know it's sandboxed.
#
# IMPORTANT — Scope vs Acceptance:
#   The sandbox SCOPE should always cover the full scenario directory (e.g.,
#   "scenarios/my-app"), because the lifecycle system needs Makefile, service.json,
#   and all subdirectories to build and run. If the scope is too narrow (e.g.,
#   just "scenarios/my-app/ui"), the lifecycle can't restart the scenario from
#   the sandbox and falls back to the real repo — making changes invisible.
#
#   To restrict WHICH changes get approved (blast radius), use the acceptance
#   config (allow/deny patterns) on the sandbox, not a narrower scope. See:
#   scenarios/workspace-sandbox/docs/ARCHITECTURE.md ("Scope vs Acceptance")
#
# See also:
#   - scenarios/agent-manager/api/internal/orchestration/run_executor.go
#     (SandboxEnvVars method — where the env vars are injected)
#   - scenarios/agent-manager/api/internal/domain/types.go
#     (ScopePath and SandboxAcceptanceConfig — design rationale)
#   - scenarios/workspace-sandbox/docs/ARCHITECTURE.md
#     ("Scope vs Acceptance" section — full explanation with examples)
# =============================================================================

# sandbox::scenario_in_scope checks whether a scenario slug falls within the
# sandbox's scoped path. This determines whether the CLI should redirect path
# resolution to the sandbox's merged/ directory or use the real repo.
#
# An agent sandboxing scenario A should NOT affect scenario B's restarts —
# only in-scope scenarios are redirected, everything else uses the real repo.
#
# Arguments:
#   $1 - scenario slug (e.g. "git-control-tower")
#   $2 - sandbox scope path, relative to project root
#         (e.g. "scenarios/git-control-tower", "scenarios", "")
#
# Returns 0 (true) if the scenario is in scope, 1 (false) otherwise.
#
# Scope matching rules:
#   ""  or "." or "/"        → whole repo is scoped, ALL scenarios match
#   "scenarios"              → all scenarios are scoped
#   "scenarios/foo"          → only scenario "foo" matches
#   "scenarios/foo/api"      → still matches "foo" (scope is deeper but within it)
#   "scenarios/other"        → does NOT match "foo"
#   "packages/shared"        → no scenarios match (scope outside scenarios/)
sandbox::scenario_in_scope() {
    local scenario_name="$1"
    local scope="$2"

    # Empty scope or root scope means the overlay covers everything
    if [[ -z "$scope" || "$scope" == "/" || "$scope" == "." ]]; then
        return 0
    fi

    # Normalize: remove trailing slash
    scope="${scope%/}"

    # If scope IS "scenarios" or a parent of it, all scenarios are in scope.
    # e.g. scope="scenarios" covers everything under scenarios/.
    if [[ "$scope" == "scenarios" ]]; then
        return 0
    fi

    # If scope starts with "scenarios/", check if this specific scenario matches.
    # Extract the first path component after "scenarios/" and compare.
    # e.g. scope="scenarios/foo/api" → scoped_name="foo" → matches scenario "foo"
    if [[ "$scope" == scenarios/* ]]; then
        local scoped_name="${scope#scenarios/}"
        scoped_name="${scoped_name%%/*}"  # Take only the first path component
        if [[ "$scenario_name" == "$scoped_name" ]]; then
            return 0
        fi
    fi

    # Scope is outside scenarios/ or targets a different scenario
    return 1
}

# sandbox::resolve_merged_path computes the absolute path to a scenario within
# the sandbox's merged directory.
#
# The overlay's merged/ dir maps to the scoped directory, NOT the project root.
# This means the path to a scenario within merged/ depends on the scope:
#
#   scope="scenarios/agent-inbox"  → merged/ IS the scenario → return $merged
#   scope="scenarios"              → merged/ has scenario dirs → return $merged/agent-inbox
#   scope="" or "." or "/"         → merged/ is project root  → return $merged/scenarios/agent-inbox
#
# The algorithm: compute the scenario's full relative path ("scenarios/{name}"),
# then strip the scope prefix to get the remaining path within the merged dir.
#
# Arguments:
#   $1 - scenario slug (e.g. "agent-inbox")
#   $2 - sandbox scope path (e.g. "scenarios/agent-inbox")
#   $3 - merged directory absolute path
#
# Outputs the resolved absolute path to stdout.
sandbox::resolve_merged_path() {
    local scenario_name="$1"
    local scope="$2"
    local merged="$3"

    # Normalize: remove trailing slash
    scope="${scope%/}"

    # The full canonical path to the scenario relative to project root
    local scenario_rel="scenarios/${scenario_name}"

    # If scope is empty/root, merged/ is the project root — use full relative path
    if [[ -z "$scope" || "$scope" == "/" || "$scope" == "." ]]; then
        echo "${merged}/${scenario_rel}"
        return
    fi

    # Strip the scope prefix from the scenario's relative path.
    # If scenario_rel starts with scope, remove it to get the path within merged/.
    # If they're equal, the remaining path is empty — merged/ IS the scenario dir.
    if [[ "$scenario_rel" == "$scope" ]]; then
        # Scope exactly matches the scenario path — merged/ IS the scenario
        echo "${merged}"
    elif [[ "$scenario_rel" == "${scope}/"* ]]; then
        # Scope is a parent dir — strip it to get the relative path within merged/
        local relative="${scenario_rel#"${scope}/"}"
        echo "${merged}/${relative}"
    else
        # Fallback: shouldn't happen if scenario_in_scope passed, but be safe
        echo "${merged}/${scenario_rel}"
    fi
}

scenario::clean_stale_locks() {
    local state_dir="$HOME/.vrooli/state/scenarios"
    local cleaned=0

    printf '%s\n' "[INFO]    Cleaning stale port locks..."

    [[ -d "$state_dir" ]] || {
        printf '%s\n' "[INFO]    No scenario lock directory found."
        return 0
    }

    local found=false
    for lock_file in "$state_dir"/.port_*.lock; do
        [[ -e "$lock_file" ]] || continue
        found=true

        local lock_info lock_pid
        lock_info=$(cat "$lock_file" 2>/dev/null || echo "")
        lock_pid=""
        if [[ "$lock_info" == *:* ]]; then
            lock_pid=${lock_info#*:}
            lock_pid=${lock_pid%%:*}
        fi

        if [[ -n "$lock_pid" && "$lock_pid" =~ ^[0-9]+$ ]] && kill -0 "$lock_pid" 2>/dev/null; then
            continue
        fi

        if rm -f "$lock_file" 2>/dev/null; then
            cleaned=$((cleaned + 1))
            printf '[INFO]    Removed stale lock %s\n' "$(basename "$lock_file")"
        fi
    done

    if [[ "$found" == "false" ]]; then
        printf '%s\n' "[INFO]    No port lock files found."
    else
        printf '[SUCCESS] Cleaned %d stale port lock(s)\n' "$cleaned"
    fi
}

scenario::run() {
    local scenario_name="$1"
    shift

    # Check for optional flags (must happen before phase parsing)
    local custom_path=""
    local clean_stale=false
    local allow_skip_missing_runtime=false
    local manage_runtime=false
    local best_effort=false
    local had_prior_allow_var=false
    local prior_allow_value=""
    local had_prior_manage_var=false
    local prior_manage_value=""
    local had_prior_gowork_var=false
    local prior_gowork_value=""

    if [[ -n "${TEST_ALLOW_SKIP_MISSING_RUNTIME+x}" ]]; then
        had_prior_allow_var=true
        prior_allow_value="${TEST_ALLOW_SKIP_MISSING_RUNTIME}"
    fi

    if [[ -n "${TEST_MANAGE_RUNTIME+x}" ]]; then
        had_prior_manage_var=true
        prior_manage_value="${TEST_MANAGE_RUNTIME}"
    fi

    if [[ -n "${GOWORK+x}" ]]; then
        had_prior_gowork_var=true
        prior_gowork_value="${GOWORK}"
    fi

    # Get the phase (default to 'develop' if not specified)
    local phase="${1:-develop}"
    shift || true

    local selection=""
    local -a remaining_args=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --path)
                custom_path="$2"
                shift 2
                ;;
            --clean-stale)
                clean_stale=true
                shift
                ;;
            --allow-skip-missing-runtime)
                allow_skip_missing_runtime=true
                shift
                ;;
            --manage-runtime)
                manage_runtime=true
                shift
                ;;
            --best-effort)
                best_effort=true
                shift
                ;;
            *)
                if [[ "$phase" == "test" && -z "$selection" && "$1" != "-"* ]]; then
                    selection="$1"
                    shift
                else
                    remaining_args+=("$1")
                    shift
                fi
                ;;
        esac
    done

    # Resolve scenario path (support custom paths, sandbox redirection, and default)
    local scenario_path
    local sandbox_redirected=false
    if [[ -n "$custom_path" ]]; then
        # Explicit --path flag always takes priority over everything else
        if [[ "$custom_path" = /* ]]; then
            scenario_path="$custom_path"
        else
            scenario_path="$(cd "$(dirname "$custom_path")" 2>/dev/null && pwd)/$(basename "$custom_path")"
        fi
    elif [[ -n "${VROOLI_SANDBOX_MERGED:-}" && -n "${VROOLI_SANDBOX_SCOPE:-}" ]]; then
        # Sandbox-aware path redirection.
        #
        # WHY: Agents running in overlayfs sandboxes make file changes that are
        # invisible to the lifecycle system, because it reads from the real repo.
        # Without this redirection, "vrooli scenario restart" would rebuild from
        # unchanged source — the agent could never test its own changes.
        #
        # HOW: The agent-manager sets VROOLI_SANDBOX_MERGED (overlay merged path)
        # and VROOLI_SANDBOX_SCOPE (which part of the repo the sandbox covers).
        # When both are present and the requested scenario falls within scope,
        # we redirect to the merged directory so the lifecycle builds/runs the
        # agent's modified code.
        #
        # DESIGN CONSTRAINTS:
        #   1. --path flag always wins (checked above in the first branch)
        #   2. Only in-scope scenarios redirect — others use the real repo
        #   3. If the merged directory doesn't exist (sandbox torn down), fall back
        if sandbox::scenario_in_scope "$scenario_name" "${VROOLI_SANDBOX_SCOPE}"; then
            # Compute the scenario path within the merged directory.
            # The overlay's merged/ dir maps to the scope path, NOT the project root.
            # We must strip the scope prefix from "scenarios/{name}" to get the
            # correct relative path within the merged dir.
            #
            # Examples (scope → merged/ contains → path to scenario):
            #   "scenarios/agent-inbox"  → agent-inbox files at root → $MERGED
            #   "scenarios"              → all scenario dirs         → $MERGED/agent-inbox
            #   "" (whole repo)          → full repo tree            → $MERGED/scenarios/agent-inbox
            local sandbox_scenario_path
            sandbox_scenario_path="$(sandbox::resolve_merged_path "$scenario_name" "${VROOLI_SANDBOX_SCOPE}" "${VROOLI_SANDBOX_MERGED}")"
            if [[ -d "$sandbox_scenario_path" ]]; then
                scenario_path="$sandbox_scenario_path"
                sandbox_redirected=true
                log::info "Sandbox redirect: using ${scenario_path} (sandbox=${VROOLI_SANDBOX_ID:-unknown})"
            else
                # Scope says this scenario should be here but merged dir is missing
                # (sandbox may have been torn down). Fall back to real repo.
                scenario_path="${var_ROOT_DIR}/scenarios/${scenario_name}"
                log::debug "Sandbox: ${sandbox_scenario_path} not found in merged view, using real repo"
            fi
        else
            # Scenario is outside sandbox scope — use real repo unchanged
            scenario_path="${var_ROOT_DIR}/scenarios/${scenario_name}"
        fi
    else
        # Default: look in standard scenarios directory
        scenario_path="${var_ROOT_DIR}/scenarios/${scenario_name}"
    fi

    if [[ ! -d "$scenario_path" ]]; then
        log::error "Scenario not found: $scenario_name (path: $scenario_path)"
        return 1
    fi

    if [[ ${#SCENARIO_DEPENDENCY_STACK[@]} -eq 0 ]]; then
        scenario::dependencies::ready_reset
    fi

    if scenario::dependencies::phase_requires_bootstrap "$phase"; then
        scenario::dependencies::stack_push "$scenario_name"
        if ! scenario::dependencies::ensure_started "$scenario_name" "$phase" "$best_effort"; then
            scenario::dependencies::stack_pop "$scenario_name"
            return 1
        fi
        scenario::dependencies::stack_pop "$scenario_name"
    fi

    if [[ "$allow_skip_missing_runtime" == "true" && "$manage_runtime" == "true" ]]; then
        log::warning "⚠️  --manage-runtime overrides --allow-skip-missing-runtime"
        allow_skip_missing_runtime=false
    fi

    if [[ "$phase" == "test" && -n "$selection" ]]; then
        local -a valid_selections=(structure dependencies unit integration business performance all e2e)
        local is_valid=false
        for sel in "${valid_selections[@]}"; do
            if [[ "$selection" == "$sel" ]]; then
                is_valid=true
                break
            fi
        done

        if [[ "$is_valid" == "false" ]]; then
            log::error "Invalid test selector: $selection"
            log::info "Valid selections: ${valid_selections[*]}"
            return 1
        fi

        if [[ "$selection" == "e2e" ]]; then
            log::info "Note: 'e2e' runs the integration phase (current end-to-end coverage)."
            selection="integration"
        fi

        if [[ "$selection" == "all" ]]; then
            remaining_args+=("all")
        else
            remaining_args+=("$selection")
        fi
    fi
    
    # For develop phase, check if already running and healthy
    if [[ "$phase" == "develop" ]]; then
        # Source lifecycle.sh to get the idempotency functions
        source "${SCRIPT_DIR}/../utils/lifecycle.sh"
        
        # Check if scenario is already running and healthy
        if lifecycle::is_scenario_running "$scenario_name"; then
            if lifecycle::is_scenario_healthy "$scenario_name"; then
                # Even if running and healthy, check if setup is needed (stale code)
                # This ensures code changes trigger rebuild even when scenario is running
                cd "$scenario_path" || return 1

                # Source setup utilities for staleness detection
                source "${SCRIPT_DIR}/../utils/setup.sh" 2>/dev/null || true

                local force_setup="${FORCE_SETUP:-false}"
                local force_setup_target="${FORCE_SETUP_SCENARIO:-}"
                local force_setup_applies=false
                if [[ "$force_setup" == "true" ]]; then
                    if [[ -z "$force_setup_target" || "$force_setup_target" == "$scenario_name" ]]; then
                        # Only honor forced rebuilds for the scenario the user
                        # explicitly restarted so dependencies stay untouched.
                        force_setup_applies=true
                    fi
                fi
                if command -v setup::is_needed >/dev/null 2>&1; then
                    if setup::is_needed "$scenario_path"; then
                        log::warning "⚠️  Scenario running but code is stale (${SETUP_REASONS[*]:-binaries/bundles outdated}), restarting..."
                        lifecycle::stop_scenario_processes "$scenario_name"
                        sleep 2
                        # Continue to normal execution below
                    elif [[ "$force_setup_applies" == "true" ]]; then
                        log::info "🔄 Forced restart requested, stopping and rebuilding..."
                        lifecycle::stop_scenario_processes "$scenario_name"
                        sleep 2
                        # Continue to normal execution below
                    else
                        log::success "✓ Scenario '$scenario_name' is already running and healthy"

                        # Show ports for user convenience
                        local scenario_dir="$HOME/.vrooli/processes/scenarios/$scenario_name"
                        if [[ -f "$scenario_dir/start-api.json" ]]; then
                            local api_port=$(jq -r '.port // ""' "$scenario_dir/start-api.json" 2>/dev/null)
                            [[ -n "$api_port" && "$api_port" != "null" ]] && echo "  API: http://localhost:$api_port"
                        fi
                        if [[ -f "$scenario_dir/start-ui.json" ]]; then
                            local ui_port=$(jq -r '.port // ""' "$scenario_dir/start-ui.json" 2>/dev/null)
                            [[ -n "$ui_port" && "$ui_port" != "null" ]] && echo "  UI: http://localhost:$ui_port"
                        fi

                        return 0  # Already running, healthy, and code is current
                    fi
                else
                    # setup::is_needed not available, fall back to old behavior
                    log::success "✓ Scenario '$scenario_name' is already running and healthy"

                    # Show ports for user convenience
                    local scenario_dir="$HOME/.vrooli/processes/scenarios/$scenario_name"
                    if [[ -f "$scenario_dir/start-api.json" ]]; then
                        local api_port=$(jq -r '.port // ""' "$scenario_dir/start-api.json" 2>/dev/null)
                        [[ -n "$api_port" && "$api_port" != "null" ]] && echo "  API: http://localhost:$api_port"
                    fi
                    if [[ -f "$scenario_dir/start-ui.json" ]]; then
                        local ui_port=$(jq -r '.port // ""' "$scenario_dir/start-ui.json" 2>/dev/null)
                        [[ -n "$ui_port" && "$ui_port" != "null" ]] && echo "  UI: http://localhost:$ui_port"
                    fi

                    return 0  # Already running and healthy
                fi
            else
                log::warning "⚠ Scenario '$scenario_name' is running but unhealthy, restarting..."
                lifecycle::stop_scenario_processes "$scenario_name"
                sleep 2  # Give processes time to clean up
                # Continue to normal execution below
            fi
        fi
    fi
    
    # Run stale lock cleanup if requested
    if [[ "$clean_stale" == "true" ]]; then
        log::info "🧹 Cleaning stale locks before starting scenario..."
        scenario::clean_stale_locks || {
            log::warning "Lock cleanup encountered errors but continuing"
        }
        log::success "✅ Stale lock cleanup completed"
    fi
    
    # Set up logging for the scenario lifecycle execution
    local lifecycle_log="${HOME}/.vrooli/logs/${scenario_name}.log"
    mkdir -p "$(dirname "$lifecycle_log")" 2>/dev/null || true

    if ! ( : > "$lifecycle_log" ) 2>/dev/null; then
        lifecycle_log="${scenario_path}/logs/${scenario_name}.lifecycle.log"
        mkdir -p "$(dirname "$lifecycle_log")"
        : > "$lifecycle_log"
    fi
    
    # Call lifecycle.sh directly, capturing output to both console and log file
    log::info "Running scenario '$scenario_name' with direct lifecycle execution"
    
    # Optionally allow skipping runtime-dependent phases (tests only)
    if [[ "$allow_skip_missing_runtime" == "true" ]]; then
        export TEST_ALLOW_SKIP_MISSING_RUNTIME="true"
    fi

    if [[ "$manage_runtime" == "true" ]]; then
        export TEST_MANAGE_RUNTIME="true"
    fi

    # Default to disabling Go workspace mode for scenario lifecycle execution so a broken
    # repo-wide go.work can't block unrelated scenarios. Opt-in by setting
    # VROOLI_SCENARIO_GOWORK=on, or by setting GOWORK explicitly.
    local scenario_gowork_mode="${VROOLI_SCENARIO_GOWORK:-off}"
    local scenario_did_disable_gowork=false
    if [[ "$had_prior_gowork_var" == "false" && "$scenario_gowork_mode" != "on" && "$scenario_gowork_mode" != "auto" ]]; then
        export GOWORK=off
        scenario_did_disable_gowork=true
    fi

    # Export custom path so lifecycle.sh uses the correct directory.
    # SCENARIO_CUSTOM_PATH is the existing mechanism lifecycle.sh checks for
    # non-default scenario paths (see lifecycle.sh lines 618, 943, 986).
    # Sandbox redirection piggybacks on this same mechanism rather than
    # introducing a parallel path — both --path and sandbox redirect result
    # in the same downstream behavior.
    if [[ -n "$custom_path" || "$sandbox_redirected" == "true" ]]; then
        export SCENARIO_CUSTOM_PATH="$scenario_path"
    fi

    # Use tee to show output on console AND write to log file
    # This preserves real-time output while capturing for later review
    # IMPORTANT: Redirect stdin from /dev/null to prevent lifecycle commands from
    # consuming stdin that belongs to the parent's process substitution loop
    # (e.g., the dependency iteration loop in dependencies.sh)
    bash "${SCRIPT_DIR}/../utils/lifecycle.sh" "$scenario_name" "$phase" "${remaining_args[@]}" < /dev/null 2>&1 | tee -a "$lifecycle_log"
    local run_exit="${PIPESTATUS[0]}"

    # If best-effort mode produced failed dependencies, report it in logs only.
    if [[ "$run_exit" -eq 0 && ${#SCENARIO_FAILED_DEPS[@]} -gt 0 ]]; then
        log::warning "⚠️  Scenario '$scenario_name' started in DEGRADED mode. Failed dependencies: ${SCENARIO_FAILED_DEPS[*]}"
    fi

    # Clean up custom path export (whether from --path or sandbox redirection)
    if [[ -n "$custom_path" || "$sandbox_redirected" == "true" ]]; then
        unset SCENARIO_CUSTOM_PATH
    fi

    if [[ "$allow_skip_missing_runtime" == "true" ]]; then
        if [[ "$had_prior_allow_var" == "true" ]]; then
            export TEST_ALLOW_SKIP_MISSING_RUNTIME="${prior_allow_value}"
        else
            unset TEST_ALLOW_SKIP_MISSING_RUNTIME || true
        fi
    fi

    if [[ "$manage_runtime" == "true" ]]; then
        if [[ "$had_prior_manage_var" == "true" ]]; then
            export TEST_MANAGE_RUNTIME="${prior_manage_value}"
        else
            unset TEST_MANAGE_RUNTIME || true
        fi
    fi

    if [[ "$scenario_did_disable_gowork" == "true" ]]; then
        if [[ "$had_prior_gowork_var" == "true" ]]; then
            export GOWORK="${prior_gowork_value}"
        else
            unset GOWORK || true
        fi
    fi

    return "$run_exit"
}
scenario::list() {
    local json_output=false
    local include_ports=false
    local -a positional=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                ;;
            --include-ports)
                include_ports=true
                ;;
            *)
                positional+=("$1")
                ;;
        esac
        shift
    done

    # Collect scenario data
    local scenarios_json="[]"
    local text_output=""

    for scenario in "${var_ROOT_DIR}"/scenarios/*/; do
        if [[ -d "$scenario" ]]; then
            local name="${scenario%/}"
            name="${name##*/}"
            local service_json="${scenario}/.vrooli/service.json"
            local description=""
            local version=""
            local status="available"
            local tags=""
            
            if [[ -f "$service_json" ]]; then
                description=$(jq -r '.service.description // ""' "$service_json" 2>/dev/null || echo "")
                version=$(jq -r '.service.version // ""' "$service_json" 2>/dev/null || echo "")
                tags=$(jq -r '.service.tags // [] | join(",")' "$service_json" 2>/dev/null || echo "")
            fi
            
            # Check if scenario is running (reuse lifecycle check if available)
            if command -v lifecycle::is_scenario_running &>/dev/null; then
                source "${SCRIPT_DIR}/../utils/lifecycle.sh" 2>/dev/null || true
                if lifecycle::is_scenario_running "$name" 2>/dev/null; then
                    status="running"
                fi
            fi
            
            local ports_json="[]"
            if [[ "$include_ports" == "true" ]]; then
                local port_dir="$HOME/.vrooli/processes/scenarios/${name}"
                if compgen -G "${port_dir}"'/*.json' >/dev/null 2>&1; then
                    local ports_result
                    ports_result=$(scenario::ports::get_all "$name" true 2>/dev/null || true)
                    if [[ -n "$ports_result" ]]; then
                        ports_json=$(echo "$ports_result" | jq '.ports // []' 2>/dev/null || echo "[]")
                    fi
                fi
            fi

            if [[ "$json_output" == "true" ]]; then
                # Build JSON object for this scenario
                local scenario_obj=$(jq -n \
                    --arg name "$name" \
                    --arg description "$description" \
                    --arg version "$version" \
                    --arg status "$status" \
                    --arg tags "$tags" \
                    --arg path "${scenario}" \
                    --arg include_ports "$include_ports" \
                    --argjson ports "$ports_json" \
                    '{
                        name: $name,
                        description: $description,
                        version: $version,
                        status: $status,
                        tags: (if $tags == "" then [] else ($tags | split(",")) end),
                        path: $path,
                        ports: (if $include_ports == "true" then $ports else [] end)
                    }')
                scenarios_json=$(echo "$scenarios_json" | jq ". += [$scenario_obj]")
            else
                # Build text output
                local line
                if [[ -n "$description" ]]; then
                    line="  • $name - $description"
                else
                    line="  • $name"
                fi

                if [[ "$include_ports" == "true" && "$ports_json" != "[]" ]]; then
                    local ports_text
                    ports_text=$(echo "$ports_json" | jq -r 'map("\(.key)=\(.port)") | join(", ")' 2>/dev/null || echo "")
                    if [[ -n "$ports_text" ]]; then
                        line+=" (ports: $ports_text)"
                    fi
                fi

                text_output+="$line"$'\n'
            fi
        fi
    done
    
    # Output results
    if [[ "$json_output" == "true" ]]; then
        # Create final JSON response with metadata
        local total_count=$(echo "$scenarios_json" | jq 'length')
        local running_count=$(echo "$scenarios_json" | jq '[.[] | select(.status == "running")] | length')
        
        jq -n \
            --argjson scenarios "$scenarios_json" \
            --argjson total "$total_count" \
            --argjson running "$running_count" \
            '{
                success: true,
                summary: {
                    total_scenarios: $total,
                    running: $running,
                    available: ($total - $running)
                },
                scenarios: $scenarios
            }'
    else
        log::info "Available scenarios:"
        echo -n "$text_output"
    fi
}

scenario::test() {
    local scenario_name="$1"
    shift
    scenario::run "$scenario_name" test "$@"
}
