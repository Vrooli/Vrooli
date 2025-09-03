#!/bin/bash

# Health-Check Dispatcher for Swarm Manager
# Verifies core health before dispatching tasks and handles core issues with priority

set -euo pipefail

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SCENARIO_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_DIR="$SCENARIO_ROOT/config"
TASKS_DIR="$SCENARIO_ROOT/tasks"

# Load configuration
source "$CONFIG_DIR/settings.yaml" 2>/dev/null || true

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Log function
log() {
    echo "[$(date -Iseconds)] $1" | tee -a "$SCENARIO_ROOT/logs/dispatcher.log"
}

# Check core health
check_core_health() {
    log "Checking core infrastructure health..."
    
    # Run core-debugger status check
    local health_json=$(core-debugger status --json 2>/dev/null || echo '{"status":"unknown"}')
    local status=$(echo "$health_json" | jq -r '.status')
    
    case "$status" in
        "healthy")
            log "✅ Core infrastructure is healthy"
            return 0
            ;;
        "degraded")
            log "⚠️  Core infrastructure is degraded"
            # Check for workarounds
            local issues=$(core-debugger list-issues --json 2>/dev/null || echo '[]')
            local issue_count=$(echo "$issues" | jq '. | length')
            
            if [ "$issue_count" -gt 0 ]; then
                log "Found $issue_count issues with potential workarounds"
                apply_workarounds
            fi
            
            # Allow execution with warning
            return 0
            ;;
        "critical")
            log "❌ Core infrastructure is CRITICAL"
            handle_critical_failure
            return 1
            ;;
        *)
            log "⚠️  Unable to determine core health status"
            return 0
            ;;
    esac
}

# Apply available workarounds
apply_workarounds() {
    log "Attempting to apply workarounds..."
    
    # Get list of issues
    local issues=$(core-debugger list-issues --json)
    
    # For each issue, check if workaround exists
    echo "$issues" | jq -c '.[]' | while read -r issue; do
        local issue_id=$(echo "$issue" | jq -r '.id')
        local description=$(echo "$issue" | jq -r '.description')
        
        log "Checking workaround for: $description"
        
        # Get workaround
        local workaround=$(core-debugger get-workaround "$description" 2>/dev/null)
        
        if [ -n "$workaround" ]; then
            log "Found workaround, applying..."
            # Note: In production, you'd execute the workaround commands
            echo "$workaround"
        fi
    done
}

# Handle critical infrastructure failure
handle_critical_failure() {
    log "CRITICAL: Core infrastructure failure detected"
    
    # List critical issues
    local critical_issues=$(core-debugger list-issues --severity critical)
    echo -e "${RED}Critical Issues:${NC}"
    echo "$critical_issues"
    
    # Create emergency task to fix core
    create_core_fix_task
    
    # Block normal task execution
    echo -e "${RED}Blocking normal task execution until core is fixed${NC}"
    
    # Notify (would integrate with notification system)
    log "ALERT: Core infrastructure critical - manual intervention may be required"
}

# Create high-priority task to fix core issues
create_core_fix_task() {
    local timestamp=$(date +%Y%m%d-%H%M%S)
    local task_file="$TASKS_DIR/backlog/manual/core-fix-$timestamp.yaml"
    
    cat > "$task_file" << EOF
id: core-fix-$timestamp
title: "CRITICAL: Fix core infrastructure issues"
type: core-infrastructure
target: core-debugger
priority: 10000
created_by: health-dispatcher
created_at: $(date -Iseconds)
severity: critical
description: |
  Core infrastructure has critical failures that must be resolved immediately.
  All other work is blocked until this is fixed.
  
  Run: core-debugger list-issues --severity critical
  
scenario: core-debugger
commands:
  - core-debugger list-issues --severity critical
  - core-debugger analyze-issue <issue-id>
  - core-debugger get-workaround <error-pattern>
EOF
    
    log "Created emergency core-fix task: $task_file"
}

# Dispatch task with health check
dispatch_task() {
    local task_file="$1"
    
    if [ ! -f "$task_file" ]; then
        log "ERROR: Task file not found: $task_file"
        return 1
    fi
    
    # Parse task
    local task_type=$(yq eval '.type' "$task_file" 2>/dev/null || echo "unknown")
    local task_scenario=$(yq eval '.scenario' "$task_file" 2>/dev/null || echo "")
    local task_title=$(yq eval '.title' "$task_file" 2>/dev/null || echo "Unknown task")
    
    log "Preparing to dispatch: $task_title"
    
    # Special handling for core-infrastructure tasks
    if [ "$task_type" == "core-infrastructure" ] || [ "$task_scenario" == "core-debugger" ]; then
        log "Core infrastructure task - bypassing health check"
    else
        # Check core health before proceeding
        if ! check_core_health; then
            log "Task dispatch blocked due to critical core issues"
            return 1
        fi
    fi
    
    # Check component-specific health if needed
    case "$task_scenario" in
        "scenario-generator-v1")
            check_component_health "orchestrator"
            ;;
        "resource-experimenter")
            check_component_health "resource-manager"
            ;;
        *)
            # Default: just check CLI
            check_component_health "cli"
            ;;
    esac
    
    # Dispatch the task
    log "Dispatching task to $task_scenario..."
    
    # Move task to active
    local active_task="$TASKS_DIR/active/$(basename "$task_file")"
    mv "$task_file" "$active_task"
    
    # Execute task (would call actual scenario CLI here)
    log "Task moved to active: $active_task"
    
    return 0
}

# Check specific component health
check_component_health() {
    local component="$1"
    
    log "Checking $component health..."
    
    if core-debugger check-health --component "$component" | grep -q "unhealthy"; then
        log "WARNING: $component is unhealthy"
        
        # Try to get workaround
        local workaround=$(core-debugger get-workaround "$component unhealthy")
        if [ -n "$workaround" ]; then
            log "Workaround available for $component"
        else
            log "No workaround available for $component issue"
        fi
    fi
}

# Main execution
main() {
    log "=== Health Check Dispatcher Started ==="
    
    # Create log directory if needed
    mkdir -p "$SCENARIO_ROOT/logs"
    
    # Check if core-debugger is available
    if ! command -v core-debugger &> /dev/null; then
        log "WARNING: core-debugger not found - health checks disabled"
    fi
    
    # Process tasks in backlog
    for priority_dir in "$TASKS_DIR/backlog/manual" "$TASKS_DIR/backlog/generated"; do
        if [ ! -d "$priority_dir" ]; then
            continue
        fi
        
        for task_file in "$priority_dir"/*.yaml; do
            if [ ! -f "$task_file" ]; then
                continue
            fi
            
            # Check if we have capacity (max 5 active tasks)
            active_count=$(find "$TASKS_DIR/active" -name "*.yaml" 2>/dev/null | wc -l)
            if [ "$active_count" -ge 5 ]; then
                log "Max active tasks reached (5), waiting..."
                break 2
            fi
            
            # Dispatch task with health check
            if dispatch_task "$task_file"; then
                log "Task dispatched successfully"
            else
                log "Task dispatch failed"
            fi
            
            # Small delay between dispatches
            sleep 2
        done
    done
    
    log "=== Dispatcher Run Complete ==="
}

# Run if executed directly
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi