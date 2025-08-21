#!/usr/bin/env bash

# Health Command Module - System Health Checks
# Comprehensive health checks for the auto loop system

set -euo pipefail

#######################################
# Execute health check command
# Arguments: None
# Returns: 0 if all checks pass, 1 if any fail
#######################################
cmd_execute() {
    local ok=0
    
    echo "=== Basic Binary Checks ==="
    for c in timeout vrooli resource-claude-code; do 
        if ! command -v "$c" >/dev/null 2>&1; then 
            echo "❌ missing: $c"
            ok=1
        else
            echo "✅ found: $c"
        fi
    done
    
    echo "=== File System Checks ==="
    if [[ ! -w "$DATA_DIR" ]]; then 
        echo "❌ data dir not writable: $DATA_DIR"
        ok=1
    else
        echo "✅ data dir writable: $DATA_DIR"
    fi
    
    # Check disk space (require at least 100MB free)
    local free_space; free_space=$(df "$DATA_DIR" | awk 'NR==2 {print $4}' 2>/dev/null || echo "0")
    if [[ "$free_space" -lt 102400 ]]; then  # 100MB in KB
        echo "❌ low disk space: ${free_space}KB free"
        ok=1
    else
        echo "✅ disk space ok: ${free_space}KB free"
    fi
    
    echo "=== Prompt & Configuration ==="
    local prompt
    if ! prompt=$(select_prompt); then 
        echo "❌ no prompt found"
        ok=1
    else 
        echo "✅ prompt ok: $prompt"
        # Check prompt file size
        local prompt_size; prompt_size=$(wc -c < "$prompt" 2>/dev/null || echo "0")
        if [[ "$prompt_size" -lt 100 ]]; then
            echo "❌ prompt file too small: ${prompt_size} bytes"
            ok=1
        else
            echo "✅ prompt size ok: ${prompt_size} bytes"
        fi
    fi
    
    echo "=== Resource Availability ==="
    # Check if jq is available (needed for JSON processing)
    if ! command -v jq >/dev/null 2>&1; then
        echo "❌ missing: jq (required for JSON processing)"
        ok=1
    else
        echo "✅ found: jq"
    fi
    
    # Check Ollama availability if resource-ollama exists
    if command -v resource-ollama >/dev/null 2>&1; then
        if resource-ollama info >/dev/null 2>&1; then
            echo "✅ ollama available"
        else
            echo "⚠️  ollama installed but not responding"
        fi
    else
        echo "ℹ️  ollama not available (optional)"
    fi
    
    echo "=== Task-specific Checks ==="
    if declare -F task_check_worker_available >/dev/null 2>&1; then
        if task_check_worker_available; then
            echo "✅ task worker available"
        else
            echo "❌ task worker unavailable"
            ok=1
        fi
    else
        echo "ℹ️  no task-specific worker checks defined"
    fi
    
    echo "=== Overall Status ==="
    if [[ $ok -eq 0 ]]; then
        echo "✅ All health checks passed"
        return 0
    else
        echo "❌ Some health checks failed"
        return 1
    fi
}

#######################################
# Validate health command arguments
# Arguments: Command arguments
# Returns: 0 if valid, 1 if invalid
#######################################
cmd_validate() {
    # Health command takes no arguments
    if [[ $# -gt 0 ]]; then
        echo "ERROR: Health command does not accept arguments" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Show help for health command
#######################################
cmd_help() {
    cat << EOF
health - Comprehensive system health check

Usage: health

Description:
  Performs comprehensive health checks on the auto loop system.
  
  The command checks:
  - Required binaries (timeout, vrooli, resource-claude-code)
  - File system permissions and disk space
  - Prompt file availability and validity
  - Resource availability (jq, ollama)
  - Task-specific worker dependencies
  
  Each check is marked with:
  ✅ Pass - check succeeded
  ❌ Fail - check failed (affects exit code)
  ⚠️  Warning - potential issue but not fatal
  ℹ️  Info - informational status
  
  Exit Codes:
  0 - All health checks passed
  1 - One or more health checks failed

Examples:
  task-manager.sh --task resource-improvement health
  manage-resource-loop.sh health
  
  # Use in scripts
  if task-manager.sh health; then
    echo "System healthy, starting loop"
    task-manager.sh start
  fi

Dependencies:
  - Basic system commands (df, awk, wc)
  - Task-specific dependencies vary

See also: status, dry-run, once
EOF
}