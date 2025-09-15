#!/usr/bin/env bash
################################################################################
# Agent Log Monitoring
# 
# Multi-source log detection and retrieval for all agent-capable resources
# Handles standalone agents, piped agents, and various log sources
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Get logs for an agent using multi-source detection
# Arguments:
#   $1 - Registry file path
#   $2 - Resource name
#   $3 - Search keywords
#   $4 - Agent ID
#   $5 - Follow mode (true/false)
#   $6 - Number of lines to show
#   $7 - JSON output (true/false)
# Returns:
#   0 on success, 1 on error
#######################################
agents::logs::get() {
    local registry_file="$1"
    local resource_name="$2"
    local search_keywords="$3"
    local agent_id="$4"
    local follow="$5"
    local lines="$6"
    local json_output="$7"
    
    if [[ -z "$agent_id" ]]; then
        log::error "agents::logs::get: Agent ID required"
        return 1
    fi
    
    [[ -f "$registry_file" ]] || {
        if [[ "$json_output" == "true" ]]; then
            echo '{"error": "No agents registry found"}'
        else
            log::error "No $resource_name agents registry found"
        fi
        return 1
    }
    
    # Get agent info including PID
    local agent_data
    agent_data=$(jq --arg id "$agent_id" '.agents[$id] // empty' "$registry_file" 2>/dev/null)
    
    if [[ -z "$agent_data" || "$agent_data" == "null" ]]; then
        if [[ "$json_output" == "true" ]]; then
            echo '{"error": "Agent not found"}'
        else
            log::error "$resource_name agent not found: $agent_id"
        fi
        return 1
    fi
    
    local pid
    pid=$(echo "$agent_data" | jq -r '.pid')
    
    if [[ -z "$pid" || "$pid" == "null" ]]; then
        if [[ "$json_output" == "true" ]]; then
            echo '{"error": "No PID found for agent"}'
        else
            log::error "No PID found for $resource_name agent: $agent_id"
        fi
        return 1
    fi
    
    # Try different log sources in order of preference
    local log_found="false"
    local log_source=""
    local log_content=""
    
    # Method 1: Check if process has redirected stdout/stderr to files
    if agents::logs::check_stdout_redirect "$pid" "$follow" "$lines" "$json_output"; then
        return 0
    fi
    
    # Method 2: Check standard log location
    if agents::logs::check_standard_location "$registry_file" "$resource_name" "$agent_id" "$pid" "$follow" "$lines" "$json_output"; then
        return 0
    fi
    
    # Method 3: Check parent process for captured output (our sophisticated detection)
    if agents::logs::check_parent_process "$pid" "$agent_id" "$search_keywords" "$follow" "$lines" "$json_output"; then
        return 0
    fi
    
    # Method 4: Try systemd journal if available
    if agents::logs::check_journal "$pid" "$follow" "$lines" "$json_output"; then
        return 0
    fi
    
    # Method 5: Check process command line for log redirection hints
    if agents::logs::check_cmdline_hints "$pid" "$follow" "$lines" "$json_output"; then
        return 0
    fi
    
    # No logs found
    if [[ "$json_output" == "true" ]]; then
        echo "{\"error\": \"No logs found for agent\", \"pid\": $pid, \"suggestions\": [\"Process may not be logging to file\", \"Try checking process output directly\"]}"
    else
        log::warn "No logs found for $resource_name agent: $agent_id (PID: $pid)"
        log::info "Suggestions:"
        log::info "  - Process may not be logging to file"
        log::info "  - Try running agent with explicit log redirection"
        log::info "  - Check if process is still running: kill -0 $pid"
    fi
    return 1
}

#######################################
# Method 1: Check if stdout/stderr is redirected to files
# Arguments:
#   $1 - Process ID
#   $2 - Follow mode
#   $3 - Lines to show
#   $4 - JSON output
# Returns:
#   0 if logs found and displayed, 1 if not found
#######################################
agents::logs::check_stdout_redirect() {
    local pid="$1"
    local follow="$2"
    local lines="$3"
    local json_output="$4"
    
    if [[ -d "/proc/$pid/fd" ]]; then
        local stdout_file stderr_file
        stdout_file=$(readlink "/proc/$pid/fd/1" 2>/dev/null || echo "")
        stderr_file=$(readlink "/proc/$pid/fd/2" 2>/dev/null || echo "")
        
        # Check if stdout is redirected to a regular file
        if [[ -n "$stdout_file" && -f "$stdout_file" && "$stdout_file" != "/dev/null" && "$stdout_file" != /dev/pts/* ]]; then
            if [[ "$follow" == "true" ]]; then
                if [[ "$json_output" == "true" ]]; then
                    echo "{\"source\": \"stdout\", \"file\": \"$stdout_file\", \"following\": true}"
                fi
                tail -f -n "$lines" "$stdout_file"
            else
                if [[ "$json_output" == "true" ]]; then
                    local log_content
                    log_content=$(tail -n "$lines" "$stdout_file" | jq -Rs '.')
                    echo "{\"source\": \"stdout\", \"file\": \"$stdout_file\", \"content\": $log_content}"
                else
                    tail -n "$lines" "$stdout_file"
                fi
            fi
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Method 2: Check standard log location
# Arguments:
#   $1 - Registry file
#   $2 - Resource name
#   $3 - Agent ID
#   $4 - PID
#   $5 - Follow mode
#   $6 - Lines to show
#   $7 - JSON output
# Returns:
#   0 if logs found and displayed, 1 if not found
#######################################
agents::logs::check_standard_location() {
    local registry_file="$1"
    local resource_name="$2"
    local agent_id="$3"
    local pid="$4"
    local follow="$5"
    local lines="$6"
    local json_output="$7"
    
    # Derive log directory from registry file path
    local base_dir
    base_dir=$(dirname "$registry_file")
    local log_directory="${base_dir}/logs/resources/${resource_name}"
    
    local log_file="${log_directory}/${agent_id}.log"
    if [[ -f "$log_file" ]]; then
        if [[ "$follow" == "true" ]]; then
            if [[ "$json_output" == "true" ]]; then
                echo "{\"source\": \"logfile\", \"file\": \"$log_file\", \"following\": true}"
            fi
            tail -f -n "$lines" "$log_file"
        else
            if [[ "$json_output" == "true" ]]; then
                local log_content
                log_content=$(tail -n "$lines" "$log_file" | jq -Rs '.')
                echo "{\"source\": \"logfile\", \"file\": \"$log_file\", \"content\": $log_content}"
            else
                tail -n "$lines" "$log_file"
            fi
        fi
        return 0
    fi
    
    return 1
}

#######################################
# Method 3: Check parent process for captured output (SOPHISTICATED DETECTION)
# This is our crown jewel - generic detection for piped agents from any orchestrator
# Arguments:
#   $1 - Process ID
#   $2 - Agent ID
#   $3 - Search keywords (e.g., "ollama|agent|task")
#   $4 - Follow mode
#   $5 - Lines to show
#   $6 - JSON output
# Returns:
#   0 if logs found and displayed, 1 if not found
#######################################
agents::logs::check_parent_process() {
    local pid="$1"
    local agent_id="$2"
    local search_keywords="$3"
    local follow="$4"
    local lines="$5"
    local json_output="$6"
    
    if [[ -f "/proc/$pid/stat" ]]; then
        local ppid
        ppid=$(awk '{print $4}' "/proc/$pid/stat" 2>/dev/null)
        if [[ -n "$ppid" && "$ppid" != "1" ]]; then
            # Check parent process working directory for logs
            local parent_cwd
            parent_cwd=$(readlink "/proc/$ppid/cwd" 2>/dev/null || echo "")
            
            if [[ -n "$parent_cwd" ]]; then
                # Look for any log files that might contain our agent's output
                # Check common log file patterns in parent's working directory
                local logfile_found=""
                for pattern in "*.log" "logs/*.log" "api.log" "app.log" "server.log"; do
                    for logfile in $parent_cwd/$pattern; do
                        if [[ -f "$logfile" ]]; then
                            # Search for recent entries related to our agent
                            local recent_logs
                            recent_logs=$(grep -i "$agent_id\\|$search_keywords" "$logfile" 2>/dev/null | tail -n "$lines" || echo "")
                            if [[ -n "$recent_logs" ]]; then
                                logfile_found="$logfile"
                                if [[ "$follow" == "true" ]]; then
                                    if [[ "$json_output" == "true" ]]; then
                                        echo "{\"source\": \"parent_log\", \"file\": \"$logfile\", \"following\": true}"
                                    fi
                                    tail -f -n "$lines" "$logfile" | grep --line-buffered -i "$agent_id\\|$search_keywords" || true
                                else
                                    if [[ "$json_output" == "true" ]]; then
                                        local log_content
                                        log_content=$(echo "$recent_logs" | jq -Rs '.')
                                        echo "{\"source\": \"parent_log\", \"file\": \"$logfile\", \"content\": $log_content}"
                                    else
                                        echo "$recent_logs"
                                    fi
                                fi
                                return 0
                            fi
                        fi
                    done
                    # If we found logs in this pattern, stop looking
                    [[ -n "$logfile_found" ]] && break
                done
            fi
        fi
    fi
    
    return 1
}

#######################################
# Method 4: Check systemd journal
# Arguments:
#   $1 - Process ID
#   $2 - Follow mode
#   $3 - Lines to show
#   $4 - JSON output
# Returns:
#   0 if logs found and displayed, 1 if not found
#######################################
agents::logs::check_journal() {
    local pid="$1"
    local follow="$2"
    local lines="$3"
    local json_output="$4"
    
    if command -v journalctl &>/dev/null; then
        local journal_output
        if [[ "$follow" == "true" ]]; then
            if [[ "$json_output" == "true" ]]; then
                echo "{\"source\": \"journal\", \"pid\": $pid, \"following\": true}"
            fi
            journalctl _PID="$pid" -f -n "$lines" --no-pager 2>/dev/null || true
        else
            journal_output=$(journalctl _PID="$pid" -n "$lines" --no-pager 2>/dev/null || echo "")
            if [[ -n "$journal_output" ]]; then
                if [[ "$json_output" == "true" ]]; then
                    local log_content
                    log_content=$(echo "$journal_output" | jq -Rs '.')
                    echo "{\"source\": \"journal\", \"pid\": $pid, \"content\": $log_content}"
                else
                    echo "$journal_output"
                fi
                return 0
            fi
        fi
    fi
    
    return 1
}

#######################################
# Method 5: Check process command line for log redirection hints
# Arguments:
#   $1 - Process ID
#   $2 - Follow mode
#   $3 - Lines to show
#   $4 - JSON output
# Returns:
#   0 if logs found and displayed, 1 if not found
#######################################
agents::logs::check_cmdline_hints() {
    local pid="$1"
    local follow="$2"
    local lines="$3"
    local json_output="$4"
    
    if [[ -f "/proc/$pid/cmdline" ]]; then
        local cmdline
        cmdline=$(tr '\0' ' ' < "/proc/$pid/cmdline" 2>/dev/null || echo "")
        
        # Look for common log file patterns in command
        local potential_log
        potential_log=$(echo "$cmdline" | grep -oE '(--log-file|--log|--output)[= ][^ ]+' | awk '{print $NF}' | head -1)
        
        if [[ -n "$potential_log" && -f "$potential_log" ]]; then
            if [[ "$follow" == "true" ]]; then
                if [[ "$json_output" == "true" ]]; then
                    echo "{\"source\": \"cmdline\", \"file\": \"$potential_log\", \"following\": true}"
                fi
                tail -f -n "$lines" "$potential_log"
            else
                if [[ "$json_output" == "true" ]]; then
                    local log_content
                    log_content=$(tail -n "$lines" "$potential_log" | jq -Rs '.')
                    echo "{\"source\": \"cmdline\", \"file\": \"$potential_log\", \"content\": $log_content}"
                else
                    tail -n "$lines" "$potential_log"
                fi
            fi
            return 0
        fi
    fi
    
    return 1
}

# Export functions for use by resource-specific implementations
export -f agents::logs::get
export -f agents::logs::check_stdout_redirect
export -f agents::logs::check_standard_location
export -f agents::logs::check_parent_process
export -f agents::logs::check_journal
export -f agents::logs::check_cmdline_hints