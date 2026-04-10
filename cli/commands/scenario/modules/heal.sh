#!/usr/bin/env bash
# =============================================================================
# Scenario Healing for Sandbox Teardown (Two-Phase Architecture)
#
# When workspace-sandbox tears down a sandbox (stop, delete, approve, reject),
# it runs pre-teardown hooks before unmounting the overlayfs. This command is
# designed to be called by those hooks.
#
# The problem: scenarios may be actively running from the sandbox's merged/
# directory. When the overlay is unmounted, those processes lose access to
# their filesystem — they can't read config, spawn children, reload code, or
# write logs. Without intervention, they become orphaned: still alive but
# effectively broken, leading to crashes, hangs, or silent data loss.
#
# Two-Phase Architecture
# ----------------------
# This command uses a two-phase approach to work within the hook's timeout
# budget (~60s, see config.go) while ensuring scenarios are safely restarted:
#
#   Phase 1 (synchronous, time-critical): STOP all affected scenarios.
#     Every scenario MUST be stopped before this hook returns, because after
#     the hook the overlay is unmounted and processes lose their filesystem.
#     Budget: ~4s per scenario (SIGTERM → 2s grace → SIGKILL → 1s cleanup).
#
#   Phase 2 (background, NOT time-critical): RESTART scenarios from the
#     canonical repo. Restarts are launched in detached background subshells
#     so the hook can return promptly. Each restart may involve Go builds,
#     UI bundling, etc. — operations that can take minutes.
#
# Why restarting from the canonical repo is correct:
#   - On approval: changes are already committed to the real repo by the
#     approval process (git apply + commit) before teardown hooks fire
#   - On rejection: the real repo has the original (correct) code
#   - On stop/delete: the sandbox is going away, real repo is the only option
#
# The background restart subshells intentionally do NOT have VROOLI_SANDBOX_*
# env vars set, which ensures the restarted scenario uses the real repo path
# (via the sandbox-aware path resolution in scripts/lib/scenario/runner.sh).
#
# Called by: workspace-sandbox pre-teardown hook (see teardown.go)
# See also:
#   - scenarios/workspace-sandbox/api/internal/policy/teardown.go (hook caller)
#   - scenarios/workspace-sandbox/api/internal/sandbox/service.go (Stop, Delete)
#   - scenarios/workspace-sandbox/api/internal/config/config.go (timeout config)
#   - scripts/lib/scenario/runner.sh (sandbox-aware path resolution)
# =============================================================================

scenario::heal::from_sandbox() {
    local merged_path="${SANDBOX_MERGED_DIR:-}"
    local dry_run=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --merged-path)
                merged_path="$2"
                shift 2
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done

    if [[ -z "$merged_path" ]]; then
        log::error "heal-from-sandbox: no merged path provided (set SANDBOX_MERGED_DIR or use --merged-path)"
        return 1
    fi

    local process_base="$HOME/.vrooli/processes/scenarios"
    if [[ ! -d "$process_base" ]]; then
        # No process metadata at all — nothing to heal
        return 0
    fi

    # Source lifecycle utilities for stop/start functions
    source "${APP_ROOT}/scripts/lib/utils/lifecycle.sh" 2>/dev/null || true

    local -a affected_scenarios=()

    # Scan all scenario process metadata to find scenarios running from this sandbox
    for scenario_dir in "$process_base"/*/; do
        [[ -d "$scenario_dir" ]] || continue
        local scenario_name
        scenario_name="$(basename "$scenario_dir")"

        # Check each step's metadata for a working_dir inside the sandbox.
        # The working_dir is recorded by lifecycle.sh when the scenario starts;
        # it reflects the actual directory the process runs in (which is the
        # sandbox merged path when started via sandbox-aware path resolution).
        local is_affected=false
        for step_json in "$scenario_dir"/*.json; do
            [[ -f "$step_json" ]] || continue

            local working_dir
            working_dir="$(jq -r '.working_dir // ""' "$step_json" 2>/dev/null)" || continue

            # Check if working_dir is inside the sandbox's merged path
            if [[ -n "$working_dir" && "$working_dir" == "${merged_path}"* ]]; then
                is_affected=true
                break
            fi
        done

        if [[ "$is_affected" == "true" ]]; then
            affected_scenarios+=("$scenario_name")
        fi
    done

    if [[ ${#affected_scenarios[@]} -eq 0 ]]; then
        log::debug "heal-from-sandbox: no scenarios affected by sandbox at ${merged_path}"
        return 0
    fi

    log::info "heal-from-sandbox: ${#affected_scenarios[@]} scenario(s) affected: ${affected_scenarios[*]}"

    if [[ "$dry_run" == "true" ]]; then
        log::info "heal-from-sandbox: dry-run mode, would stop and restart: ${affected_scenarios[*]}"
        return 0
    fi

    # =========================================================================
    # PHASE 1: Stop all affected scenarios (synchronous, time-critical)
    #
    # This is the critical phase — every scenario MUST be stopped before this
    # hook returns. After the hook completes, workspace-sandbox unmounts the
    # overlay filesystem. Any process still reading from the merged directory
    # at that point loses its filesystem and becomes orphaned.
    #
    # Budget: ~4s per scenario (lifecycle::stop_scenario_processes sends
    # SIGTERM, waits 2s for graceful shutdown, sends SIGKILL to survivors,
    # then 1s for cleanup). With the 60s hook timeout, this supports ~15
    # scenarios stopping sequentially.
    # =========================================================================
    local -a stopped_scenarios=()

    for scenario_name in "${affected_scenarios[@]}"; do
        log::info "heal-from-sandbox: stopping ${scenario_name}..."
        if command -v lifecycle::stop_scenario_processes &>/dev/null; then
            lifecycle::stop_scenario_processes "$scenario_name" 2>/dev/null || true
        fi
        stopped_scenarios+=("$scenario_name")
    done

    # Brief pause to ensure all processes have fully exited and released
    # file handles before the overlay is unmounted
    sleep 1

    log::info "heal-from-sandbox: stopped ${#stopped_scenarios[@]} scenario(s)"

    # =========================================================================
    # PHASE 2: Restart scenarios from the canonical repo (background)
    #
    # Restarts happen in detached background subshells so this hook can return
    # within its timeout budget. The actual restart may be slow (Go builds,
    # UI bundling, dependency checks) but that's fine — the critical work
    # (stopping) is already done.
    #
    # Why restarting from the canonical repo is always correct:
    #   - Approval: changes were committed to the real repo before hooks fired
    #   - Rejection: real repo has the original (pre-sandbox) code
    #   - Stop/Delete: sandbox is going away, real repo is the only option
    #
    # These subshells intentionally do NOT inherit VROOLI_SANDBOX_* env vars
    # (they were never set in this hook's environment — workspace-sandbox
    # sets SANDBOX_* vars, not VROOLI_SANDBOX_* vars). This ensures the
    # restarted scenario uses normal path resolution → canonical repo.
    # =========================================================================
    for scenario_name in "${stopped_scenarios[@]}"; do
        # Double-fork pattern: the inner & backgrounds within the subshell,
        # and the ( ... ) subshell ensures the backgrounded process is
        # reparented to init (PID 1), surviving the hook process exit.
        # stdout/stderr are redirected to avoid polluting the hook output.
        ( scenario::lifecycle::start "$scenario_name" >/dev/null 2>&1 & )
        log::info "heal-from-sandbox: restart of ${scenario_name} launched in background"
    done

    log::info "heal-from-sandbox: all ${#stopped_scenarios[@]} scenario(s) stopped and restarts launched"
    return 0
}
