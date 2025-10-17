#!/usr/bin/env bash
################################################################################
# Execute Command Tool Executor
# 
# Safely executes shell commands with sandboxing and security controls
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Execute Command Tool Implementation
################################################################################

#######################################
# Execute execute_command tool
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context (sandbox, local)
# Returns:
#   Execution result (JSON)
#######################################
tool_execute_command::execute() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    log::debug "Executing command tool in $context context"
    
    # Extract arguments
    local command timeout working_dir
    command=$(echo "$arguments" | jq -r '.command // ""')
    timeout=$(echo "$arguments" | jq -r '.timeout // 30')
    working_dir=$(echo "$arguments" | jq -r '.working_dir // "."')
    
    # Validate arguments
    if [[ -z "$command" ]]; then
        echo '{"success": false, "error": "Missing required parameter: command"}'
        return 1
    fi
    
    # Security validation
    if ! tool_execute_command::validate_command "$command" "$context"; then
        echo '{"success": false, "error": "Command blocked by security policy"}'
        return 1
    fi
    
    # Get workspace directory based on context
    local workspace_dir
    case "$context" in
        sandbox)
            workspace_dir="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
            mkdir -p "$workspace_dir"
            ;;
        local)
            workspace_dir="${PWD}"
            ;;
        *)
            workspace_dir="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
            mkdir -p "$workspace_dir"
            ;;
    esac
    
    # Ensure working directory is within workspace for sandbox mode
    local final_working_dir
    if [[ "$context" == "sandbox" ]]; then
        if [[ "$working_dir" == "." ]]; then
            final_working_dir="$workspace_dir"
        elif [[ "$working_dir" =~ \.\./|^/ ]]; then
            # Prevent path traversal
            final_working_dir="$workspace_dir"
        elif [[ "$working_dir" =~ ^$workspace_dir/ ]]; then
            final_working_dir="$working_dir"
        else
            final_working_dir="$workspace_dir/$working_dir"
        fi
    else
        if [[ "$working_dir" =~ ^/ ]]; then
            final_working_dir="$working_dir"
        elif [[ "$working_dir" == "." ]]; then
            final_working_dir="$workspace_dir"
        else
            final_working_dir="$workspace_dir/$working_dir"
        fi
    fi
    
    # Create working directory if it doesn't exist
    if [[ ! -d "$final_working_dir" ]]; then
        if ! mkdir -p "$final_working_dir" 2>/dev/null; then
            echo '{"success": false, "error": "Failed to create working directory"}'
            return 1
        fi
    fi
    
    log::debug "Executing command: $command in $final_working_dir"
    
    # Set up environment for sandbox mode
    local env_vars=()
    if [[ "$context" == "sandbox" ]]; then
        # Restrict environment in sandbox mode
        env_vars=(
            "HOME=$workspace_dir"
            "PWD=$final_working_dir"
            "PATH=/usr/local/bin:/usr/bin:/bin"
            "SHELL=/bin/bash"
        )
    fi
    
    # Execute command with timeout and capture output
    local start_time=$(date +%s)
    local output exit_code
    
    # Change to working directory and execute
    cd "$final_working_dir" || {
        echo '{"success": false, "error": "Failed to change to working directory"}'
        return 1
    }
    
    # Execute with timeout and environment restrictions
    if [[ "$context" == "sandbox" ]]; then
        output=$(timeout "$timeout" env -i "${env_vars[@]}" bash -c "$command" 2>&1)
    else
        output=$(timeout "$timeout" bash -c "$command" 2>&1)
    fi
    exit_code=$?
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Handle timeout
    if [[ $exit_code -eq 124 ]]; then
        echo "{\"success\": false, \"error\": \"Command timed out after $timeout seconds\", \"exit_code\": $exit_code, \"duration\": $duration}"
        return 1
    fi
    
    # Escape output for JSON
    local escaped_output
    escaped_output=$(echo "$output" | jq -Rs .)
    
    # Determine success based on exit code
    local success
    if [[ $exit_code -eq 0 ]]; then
        success="true"
    else
        success="false"
    fi
    
    # Build response
    cat << EOF
{
  "success": $success,
  "exit_code": $exit_code,
  "output": $escaped_output,
  "duration": $duration,
  "command": $(echo "$command" | jq -Rs .),
  "working_dir": "$final_working_dir",
  "context": "$context"
}
EOF
    
    return 0
}

#######################################
# Validate command for security
# Arguments:
#   $1 - Command string
#   $2 - Execution context
# Returns:
#   0 if command is safe, 1 if blocked
#######################################
tool_execute_command::validate_command() {
    local command="$1"
    local context="$2"
    
    # Dangerous commands - always blocked
    local dangerous_patterns=(
        'rm.*-rf'
        'sudo'
        'su'
        'chmod.*777'
        'chown'
        'passwd'
        'mkfs'
        'fdisk'
        'dd.*if=/dev'
        'wget.*\|.*sh'
        'curl.*\|.*sh'
        'eval.*\$'
        '.*>.*\/dev\/'
        'reboot'
        'shutdown'
        'halt'
        'poweroff'
        'init.*0'
        'kill.*-9.*1'
        'pkill.*-9'
        'killall'
        'format'
        'mount'
        'umount'
        'crontab'
        'at[[:space:]]'
        'nohup.*&'
    )
    
    for pattern in "${dangerous_patterns[@]}"; do
        if [[ "$command" =~ $pattern ]]; then
            log::debug "Command blocked by dangerous pattern: $pattern"
            return 1
        fi
    done
    
    # Network commands - allowed in local context only
    if [[ "$context" == "sandbox" ]]; then
        local network_patterns=(
            'wget'
            'curl'
            'nc'
            'netcat'
            'ssh'
            'scp'
            'rsync.*::'
            'ftp'
            'telnet'
        )
        
        for pattern in "${network_patterns[@]}"; do
            if [[ "$command" =~ $pattern ]]; then
                log::debug "Network command blocked in sandbox: $pattern"
                return 1
            fi
        done
    fi
    
    # File system modification limits in sandbox
    if [[ "$context" == "sandbox" ]]; then
        # Block access to system directories
        if [[ "$command" =~ (^|[[:space:]]).*[[:space:]]*/etc/|/bin/|/sbin/|/usr/|/lib/|/proc/|/sys/|/dev/ ]]; then
            log::debug "System directory access blocked in sandbox"
            return 1
        fi
    fi
    
    # Check for command injection attempts
    local injection_patterns=(
        '.*;.*'
        '.*\|\|.*'
        '.*&&.*'
        '`.*`'
        '\$\(.*\)'
        '.*\|.*sh'
        '.*\|.*bash'
    )
    
    for pattern in "${injection_patterns[@]}"; do
        if [[ "$command" =~ $pattern ]]; then
            # Allow some safe chaining
            if [[ "$command" =~ ^[a-zA-Z0-9_\-\./[:space:]]*(\|\||\&\&)[a-zA-Z0-9_\-\./[:space:]]*$ ]]; then
                continue
            fi
            log::debug "Potential command injection blocked: $pattern"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Execute command with output streaming (for long-running commands)
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Execution context
# Returns:
#   Streamed execution result
#######################################
tool_execute_command::execute_streaming() {
    local arguments="$1"
    local context="${2:-sandbox}"
    
    # Extract arguments
    local command timeout working_dir
    command=$(echo "$arguments" | jq -r '.command // ""')
    timeout=$(echo "$arguments" | jq -r '.timeout // 30')
    working_dir=$(echo "$arguments" | jq -r '.working_dir // "."')
    
    # Validate command
    if ! tool_execute_command::validate_command "$command" "$context"; then
        echo '{"success": false, "error": "Command blocked by security policy"}'
        return 1
    fi
    
    # Set up working directory (same logic as regular execute)
    local workspace_dir
    case "$context" in
        sandbox)
            workspace_dir="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
            mkdir -p "$workspace_dir"
            ;;
        local)
            workspace_dir="${PWD}"
            ;;
        *)
            workspace_dir="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
            mkdir -p "$workspace_dir"
            ;;
    esac
    
    local final_working_dir
    if [[ "$context" == "sandbox" ]]; then
        if [[ "$working_dir" == "." ]]; then
            final_working_dir="$workspace_dir"
        else
            final_working_dir="$workspace_dir/$working_dir"
        fi
    else
        if [[ "$working_dir" =~ ^/ ]]; then
            final_working_dir="$working_dir"
        else
            final_working_dir="$workspace_dir/$working_dir"
        fi
    fi
    
    mkdir -p "$final_working_dir"
    cd "$final_working_dir" || return 1
    
    # Execute with streaming output
    echo '{"status": "started", "command": "'$command'", "working_dir": "'$final_working_dir'"}'
    
    local exit_code
    if [[ "$context" == "sandbox" ]]; then
        timeout "$timeout" env -i "HOME=$workspace_dir" "PWD=$final_working_dir" "PATH=/usr/local/bin:/usr/bin:/bin" bash -c "$command" 2>&1 | while IFS= read -r line; do
            echo '{"status": "output", "line": "'$(echo "$line" | sed 's/"/\\"/g')'"}' 
        done
        exit_code=${PIPESTATUS[0]}
    else
        timeout "$timeout" bash -c "$command" 2>&1 | while IFS= read -r line; do
            echo '{"status": "output", "line": "'$(echo "$line" | sed 's/"/\\"/g')'"}'
        done
        exit_code=${PIPESTATUS[0]}
    fi
    
    echo '{"status": "completed", "exit_code": '$exit_code'}'
}

#######################################
# Validate execute_command arguments
# Arguments:
#   $1 - Tool arguments (JSON)
# Returns:
#   0 if valid, 1 if invalid
#######################################
tool_execute_command::validate() {
    local arguments="$1"
    
    # Check required fields
    if ! echo "$arguments" | jq -e '.command' &>/dev/null; then
        log::debug "execute_command: missing command parameter"
        return 1
    fi
    
    # Validate command is not empty
    local command
    command=$(echo "$arguments" | jq -r '.command // ""')
    if [[ -z "$command" ]]; then
        log::debug "execute_command: command parameter is empty"
        return 1
    fi
    
    # Validate timeout if provided
    local timeout
    timeout=$(echo "$arguments" | jq -r '.timeout // 30')
    if [[ ! "$timeout" =~ ^[0-9]+$ ]] || [[ $timeout -gt 300 ]]; then
        log::debug "execute_command: invalid timeout (must be number ≤ 300 seconds)"
        return 1
    fi
    
    return 0
}

#######################################
# Get tool information
# Returns:
#   Tool info JSON
#######################################
tool_execute_command::info() {
    cat << 'EOF'
{
  "name": "execute_command",
  "category": "command",
  "description": "Execute shell commands with security controls and sandboxing",
  "security_level": "high",
  "supports_contexts": ["sandbox", "local"],
  "features": [
    "Command timeout protection",
    "Security pattern filtering",
    "Environment isolation (sandbox)",
    "Output capture and streaming",
    "Working directory control"
  ],
  "restrictions": [
    "Dangerous commands blocked (rm -rf, sudo, etc.)",
    "Network commands blocked in sandbox",
    "System directory access restricted",
    "Command injection prevention",
    "Maximum timeout: 300 seconds"
  ],
  "blocked_patterns": [
    "rm -rf", "sudo", "chmod 777", "reboot", "shutdown",
    "wget|sh", "curl|sh", "eval $", "command injection"
  ]
}
EOF
}

# Export functions
export -f tool_execute_command::execute
export -f tool_execute_command::execute_streaming
export -f tool_execute_command::validate