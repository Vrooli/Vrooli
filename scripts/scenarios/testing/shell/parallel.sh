#!/usr/bin/env bash
# Parallel execution utilities for test phases
set -euo pipefail

# Global parallel execution state
declare -A TESTING_PARALLEL_GROUPS=()
declare -A TESTING_PARALLEL_DEPENDENCIES=()
declare -A TESTING_PARALLEL_STATUS=()
declare -A TESTING_PARALLEL_PIDS=()
TESTING_PARALLEL_MAX_WORKERS=4
TESTING_PARALLEL_ENABLED=true

# Configure parallel execution settings
# Usage: testing::parallel::configure [options]
# Options:
#   --enabled BOOL          Enable parallel execution (default: true)
#   --max-workers COUNT     Maximum parallel workers (default: 4, 0 = CPU count)
testing::parallel::configure() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --enabled)
                TESTING_PARALLEL_ENABLED="$2"
                shift 2
                ;;
            --max-workers)
                TESTING_PARALLEL_MAX_WORKERS="$2"
                shift 2
                ;;
            *)
                echo "Unknown option to testing::parallel::configure: $1" >&2
                return 1
                ;;
        esac
    done
    
    # Auto-detect CPU count if max-workers is 0
    if [ "$TESTING_PARALLEL_MAX_WORKERS" -eq 0 ]; then
        if command -v nproc >/dev/null 2>&1; then
            TESTING_PARALLEL_MAX_WORKERS=$(nproc)
        elif command -v sysctl >/dev/null 2>&1; then
            TESTING_PARALLEL_MAX_WORKERS=$(sysctl -n hw.ncpu 2>/dev/null || echo 4)
        else
            TESTING_PARALLEL_MAX_WORKERS=4
        fi
    fi
}

# Define a group of phases that can run in parallel
# Usage: testing::parallel::define_group GROUP_NAME PHASE1 [PHASE2 ...]
testing::parallel::define_group() {
    local group_name="$1"
    shift
    
    TESTING_PARALLEL_GROUPS["$group_name"]="$*"
    
    # Mark each phase as part of this group
    for phase in "$@"; do
        local current_groups="${TESTING_PARALLEL_DEPENDENCIES[$phase]:-}"
        if [ -n "$current_groups" ]; then
            TESTING_PARALLEL_DEPENDENCIES["$phase"]="$current_groups,$group_name"
        else
            TESTING_PARALLEL_DEPENDENCIES["$phase"]="$group_name"
        fi
    done
}

# Define dependencies between phases
# Usage: testing::parallel::add_dependency PHASE DEPENDS_ON
testing::parallel::add_dependency() {
    local phase="$1"
    local depends_on="$2"
    
    local current="${TESTING_PARALLEL_DEPENDENCIES[$phase]:-}"
    if [ -n "$current" ]; then
        TESTING_PARALLEL_DEPENDENCIES["$phase"]="$current:depends:$depends_on"
    else
        TESTING_PARALLEL_DEPENDENCIES["$phase"]="depends:$depends_on"
    fi
}

# Check if phases can run in parallel
# Usage: testing::parallel::can_run_parallel PHASE1 PHASE2
testing::parallel::can_run_parallel() {
    local phase1="$1"
    local phase2="$2"
    
    if [ "$TESTING_PARALLEL_ENABLED" != "true" ]; then
        return 1
    fi
    
    # Check if they're in the same parallel group
    for group in "${!TESTING_PARALLEL_GROUPS[@]}"; do
        local members="${TESTING_PARALLEL_GROUPS[$group]}"
        if [[ " $members " =~ " $phase1 " ]] && [[ " $members " =~ " $phase2 " ]]; then
            return 0
        fi
    done
    
    # Check if there are explicit dependencies
    local deps1="${TESTING_PARALLEL_DEPENDENCIES[$phase1]:-}"
    local deps2="${TESTING_PARALLEL_DEPENDENCIES[$phase2]:-}"
    
    if [[ "$deps1" =~ "depends:$phase2" ]] || [[ "$deps2" =~ "depends:$phase1" ]]; then
        return 1
    fi
    
    return 1
}

# Execute phases in parallel where possible
# Usage: testing::parallel::execute_phases PHASE_ARRAY LOG_DIR TIMEOUT_MULTIPLIER
testing::parallel::execute_phases() {
    local -n phases=$1
    local log_dir="$2"
    local timeout_multiplier="${3:-1}"
    
    if [ "$TESTING_PARALLEL_ENABLED" != "true" ] || [ ${#phases[@]} -lt 2 ]; then
        # Fall back to sequential execution
        return 1
    fi
    
    local remaining=("${phases[@]}")
    local running=()
    local completed=()
    local failed=()
    
    echo "🚀 Executing ${#phases[@]} phases with parallel optimization (max workers: $TESTING_PARALLEL_MAX_WORKERS)"
    echo ""
    
    while [ ${#remaining[@]} -gt 0 ] || [ ${#running[@]} -gt 0 ]; do
        # Start new phases if we have capacity
        while [ ${#running[@]} -lt $TESTING_PARALLEL_MAX_WORKERS ] && [ ${#remaining[@]} -gt 0 ]; do
            local next_phase=""
            local next_index=-1
            
            # Find next phase that can run
            for i in "${!remaining[@]}"; do
                local phase="${remaining[$i]}"
                if testing::parallel::can_start_phase "$phase" "${completed[@]}" "${running[@]}"; then
                    next_phase="$phase"
                    next_index=$i
                    break
                fi
            done
            
            if [ -n "$next_phase" ]; then
                # Start the phase in background
                testing::parallel::start_phase "$next_phase" "$log_dir" "$timeout_multiplier" &
                local pid=$!
                
                TESTING_PARALLEL_PIDS["$next_phase"]=$pid
                TESTING_PARALLEL_STATUS["$next_phase"]="running"
                
                running+=("$next_phase")
                unset 'remaining[$next_index]'
                remaining=("${remaining[@]}")  # Reindex array
                
                echo "▶️  Started phase: $next_phase (PID: $pid)"
            else
                # No phases can start right now, wait
                break
            fi
        done
        
        # Check for completed phases
        if [ ${#running[@]} -gt 0 ]; then
            local still_running=()
            
            for phase in "${running[@]}"; do
                local pid="${TESTING_PARALLEL_PIDS[$phase]}"
                
                if ! kill -0 "$pid" 2>/dev/null; then
                    # Phase completed
                    wait "$pid"
                    local exit_code=$?
                    
                    if [ $exit_code -eq 0 ]; then
                        TESTING_PARALLEL_STATUS["$phase"]="completed"
                        completed+=("$phase")
                        echo "✅ Phase completed: $phase"
                    else
                        TESTING_PARALLEL_STATUS["$phase"]="failed"
                        failed+=("$phase")
                        echo "❌ Phase failed: $phase (exit code: $exit_code)"
                    fi
                else
                    still_running+=("$phase")
                fi
            done
            
            running=("${still_running[@]}")
            
            # Brief sleep to avoid busy-waiting
            if [ ${#running[@]} -gt 0 ]; then
                sleep 0.5
            fi
        fi
    done
    
    echo ""
    echo "📊 Parallel Execution Summary:"
    echo "   Completed: ${#completed[@]} phases"
    echo "   Failed: ${#failed[@]} phases"
    
    if [ ${#failed[@]} -gt 0 ]; then
        return 1
    else
        return 0
    fi
}

# Check if a phase can start based on dependencies
testing::parallel::can_start_phase() {
    local phase="$1"
    shift
    local completed=("$@")
    
    local deps="${TESTING_PARALLEL_DEPENDENCIES[$phase]:-}"
    
    if [[ "$deps" =~ "depends:" ]]; then
        # Extract dependency requirements
        local required_phases=$(echo "$deps" | grep -o 'depends:[^,]*' | cut -d: -f2)
        
        for required in $required_phases; do
            local found=false
            for complete in "${completed[@]}"; do
                if [ "$complete" = "$required" ]; then
                    found=true
                    break
                fi
            done
            
            if [ "$found" = false ]; then
                return 1  # Dependency not satisfied
            fi
        done
    fi
    
    return 0
}

# Start a phase execution (to be implemented by caller)
# This is a placeholder that should be overridden
testing::parallel::start_phase() {
    local phase="$1"
    local log_dir="$2"
    local timeout_multiplier="$3"
    
    echo "ERROR: testing::parallel::start_phase must be implemented by the runner" >&2
    return 1
}

# Check if we can optimize parallel execution for given items
testing::parallel::can_optimize() {
    local items=("$@")
    
    if [ "$TESTING_PARALLEL_ENABLED" != "true" ]; then
        return 1
    fi
    
    if [ ${#items[@]} -lt 2 ]; then
        return 1  # Not worth parallelizing single items
    fi
    
    # Check if any items are in parallel groups
    for item in "${items[@]}"; do
        if [ -n "${TESTING_PARALLEL_DEPENDENCIES[$item]:-}" ]; then
            return 0  # Found at least one item with parallel configuration
        fi
    done
    
    return 1  # No parallel configuration found
}