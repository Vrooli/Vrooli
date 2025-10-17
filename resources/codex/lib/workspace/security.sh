#!/usr/bin/env bash
################################################################################
# Workspace Security - Advanced security controls and monitoring
# 
# Provides comprehensive security for workspace operations including:
# - Process isolation and control
# - Resource monitoring and limits
# - Security policy enforcement
# - Threat detection and response
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Security Policy Management
################################################################################

#######################################
# Apply security policy to workspace
# Arguments:
#   $1 - Workspace ID
#   $2 - Security level (strict, moderate, relaxed)
#   $3 - Custom policy (JSON, optional)
# Returns:
#   Policy application result JSON
#######################################
workspace_security::apply_policy() {
    local workspace_id="$1"
    local security_level="$2"
    local custom_policy="${3:-{}}"
    
    log::info "Applying security policy to workspace: $workspace_id (level: $security_level)"
    
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    if [[ ! -d "$workspace_dir" ]]; then
        echo '{"success": false, "error": "Workspace not found"}'
        return 1
    fi
    
    # Create security policy file
    local policy_file="$workspace_dir/config/security_policy.json"
    local policy
    policy=$(workspace_security::create_policy "$security_level" "$custom_policy")
    
    echo "$policy" > "$policy_file"
    
    # Apply filesystem restrictions
    workspace_security::apply_filesystem_policy "$workspace_dir" "$security_level"
    
    # Set up process controls
    workspace_security::setup_process_controls "$workspace_id" "$security_level"
    
    # Configure network restrictions
    workspace_security::setup_network_policy "$workspace_id" "$security_level"
    
    # Initialize monitoring
    workspace_security::setup_security_monitoring "$workspace_id"
    
    log::success "Security policy applied to workspace: $workspace_id"
    
    cat << EOF
{
  "success": true,
  "workspace_id": "$workspace_id",
  "security_level": "$security_level",
  "policy_applied_at": "$(date -Iseconds)",
  "policy_file": "$policy_file"
}
EOF
}

#######################################
# Create security policy configuration
# Arguments:
#   $1 - Security level
#   $2 - Custom policy overrides (JSON)
# Returns:
#   Security policy JSON
#######################################
workspace_security::create_policy() {
    local security_level="$1"
    local custom_policy="$2"
    
    # Base policy templates
    local base_policy
    case "$security_level" in
        strict)
            base_policy='{
                "filesystem": {
                    "read_only_system": true,
                    "allowed_paths": ["/tmp", "/var/tmp"],
                    "forbidden_paths": ["/etc", "/bin", "/sbin", "/usr", "/lib", "/boot", "/dev", "/proc", "/sys"],
                    "max_file_size": "10M",
                    "max_total_size": "50M"
                },
                "processes": {
                    "max_processes": 5,
                    "max_threads": 10,
                    "allowed_commands": ["cat", "echo", "grep", "sed", "awk", "sort", "head", "tail"],
                    "forbidden_commands": ["sudo", "su", "chmod", "chown", "mount", "umount", "reboot", "shutdown"],
                    "max_cpu_percent": 25,
                    "max_memory": "128M"
                },
                "network": {
                    "allowed": false,
                    "allowed_domains": [],
                    "allowed_ports": [],
                    "max_connections": 0
                },
                "execution": {
                    "timeout": 60,
                    "max_output_size": "1M",
                    "allow_shell_access": false,
                    "allow_script_execution": true
                }
            }'
            ;;
        moderate)
            base_policy='{
                "filesystem": {
                    "read_only_system": true,
                    "allowed_paths": ["/tmp", "/var/tmp", "/home"],
                    "forbidden_paths": ["/etc/passwd", "/etc/shadow", "/boot", "/dev", "/proc", "/sys"],
                    "max_file_size": "50M",
                    "max_total_size": "100M"
                },
                "processes": {
                    "max_processes": 10,
                    "max_threads": 20,
                    "allowed_commands": ["*"],
                    "forbidden_commands": ["sudo", "su", "reboot", "shutdown", "mkfs", "fdisk"],
                    "max_cpu_percent": 50,
                    "max_memory": "256M"
                },
                "network": {
                    "allowed": true,
                    "allowed_domains": ["*"],
                    "blocked_domains": ["malicious.example.com"],
                    "allowed_ports": [80, 443, 8080, 8443],
                    "max_connections": 5
                },
                "execution": {
                    "timeout": 300,
                    "max_output_size": "10M",
                    "allow_shell_access": true,
                    "allow_script_execution": true
                }
            }'
            ;;
        relaxed)
            base_policy='{
                "filesystem": {
                    "read_only_system": false,
                    "allowed_paths": ["*"],
                    "forbidden_paths": ["/etc/passwd", "/etc/shadow"],
                    "max_file_size": "500M",
                    "max_total_size": "1G"
                },
                "processes": {
                    "max_processes": 25,
                    "max_threads": 50,
                    "allowed_commands": ["*"],
                    "forbidden_commands": ["reboot", "shutdown"],
                    "max_cpu_percent": 80,
                    "max_memory": "512M"
                },
                "network": {
                    "allowed": true,
                    "allowed_domains": ["*"],
                    "blocked_domains": [],
                    "allowed_ports": ["*"],
                    "max_connections": 20
                },
                "execution": {
                    "timeout": 900,
                    "max_output_size": "100M",
                    "allow_shell_access": true,
                    "allow_script_execution": true
                }
            }'
            ;;
    esac
    
    # Merge with custom policy if provided
    if [[ "$custom_policy" != "{}" ]]; then
        echo "$base_policy" | jq --argjson custom "$custom_policy" '. * $custom'
    else
        echo "$base_policy"
    fi
}

################################################################################
# Process Control and Monitoring
################################################################################

#######################################
# Setup process controls for workspace
# Arguments:
#   $1 - Workspace ID
#   $2 - Security level
#######################################
workspace_security::setup_process_controls() {
    local workspace_id="$1"
    local security_level="$2"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Create process control script
    cat > "$workspace_dir/config/process_control.sh" << 'EOF'
#!/bin/bash
# Process control wrapper for workspace execution

WORKSPACE_ID="$1"
SECURITY_LEVEL="$2"
COMMAND="$3"

# Load security policy
POLICY_FILE="$(dirname "$0")/security_policy.json"
if [[ -f "$POLICY_FILE" ]]; then
    MAX_PROCESSES=$(jq -r '.processes.max_processes' "$POLICY_FILE")
    MAX_MEMORY=$(jq -r '.processes.max_memory' "$POLICY_FILE")
    TIMEOUT=$(jq -r '.execution.timeout' "$POLICY_FILE")
    FORBIDDEN_COMMANDS=$(jq -r '.processes.forbidden_commands[]' "$POLICY_FILE")
else
    MAX_PROCESSES=10
    MAX_MEMORY="256M"
    TIMEOUT=300
    FORBIDDEN_COMMANDS=""
fi

# Check if command is forbidden
for forbidden in $FORBIDDEN_COMMANDS; do
    if [[ "$COMMAND" =~ $forbidden ]]; then
        echo "ERROR: Command '$forbidden' is forbidden in this workspace"
        exit 1
    fi
done

# Check current process count
CURRENT_PROCESSES=$(pgrep -f "workspace-$WORKSPACE_ID" | wc -l)
if [[ $CURRENT_PROCESSES -ge $MAX_PROCESSES ]]; then
    echo "ERROR: Maximum process limit ($MAX_PROCESSES) reached"
    exit 1
fi

# Execute with resource limits
if command -v ulimit >/dev/null 2>&1; then
    # Set memory limit (approximate)
    MEMORY_KB=$(echo "$MAX_MEMORY" | sed 's/M/*1024/g' | sed 's/G/*1024*1024/g' | bc -l 2>/dev/null || echo "262144")
    ulimit -v "$MEMORY_KB" 2>/dev/null
    
    # Set CPU time limit
    ulimit -t "$TIMEOUT" 2>/dev/null
fi

# Execute the command with timeout
timeout "$TIMEOUT" bash -c "$COMMAND"
EOF
    
    chmod +x "$workspace_dir/config/process_control.sh"
}

#######################################
# Monitor workspace processes
# Arguments:
#   $1 - Workspace ID
# Returns:
#   Process monitoring report JSON
#######################################
workspace_security::monitor_processes() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Get running processes for this workspace
    local processes=()
    local total_cpu=0
    local total_memory=0
    
    # Find processes associated with the workspace
    while IFS= read -r pid; do
        [[ -z "$pid" ]] && continue
        
        # Get process information
        if [[ -f "/proc/$pid/stat" ]]; then
            local proc_info
            proc_info=$(ps -p "$pid" -o pid,ppid,user,pcpu,pmem,args --no-headers 2>/dev/null)
            
            if [[ -n "$proc_info" ]]; then
                local cpu_percent memory_percent
                cpu_percent=$(echo "$proc_info" | awk '{print $4}')
                memory_percent=$(echo "$proc_info" | awk '{print $5}')
                
                total_cpu=$(echo "$total_cpu + $cpu_percent" | bc -l 2>/dev/null || echo "$total_cpu")
                total_memory=$(echo "$total_memory + $memory_percent" | bc -l 2>/dev/null || echo "$total_memory")
                
                processes+=("$proc_info")
            fi
        fi
    done < <(pgrep -f "workspace-$workspace_id")
    
    # Build process list JSON
    local processes_json="[]"
    for proc in "${processes[@]}"; do
        local pid ppid user cpu_percent mem_percent command
        read -r pid ppid user cpu_percent mem_percent command <<< "$proc"
        
        local proc_obj
        proc_obj=$(jq -n \
            --arg pid "$pid" \
            --arg ppid "$ppid" \
            --arg user "$user" \
            --arg cpu "$cpu_percent" \
            --arg memory "$mem_percent" \
            --arg command "$command" \
            '{
                pid: ($pid | tonumber),
                ppid: ($ppid | tonumber),
                user: $user,
                cpu_percent: ($cpu | tonumber),
                memory_percent: ($memory | tonumber),
                command: $command
            }')
        
        processes_json=$(echo "$processes_json" | jq ". + [$proc_obj]")
    done
    
    cat << EOF
{
  "workspace_id": "$workspace_id",
  "process_count": ${#processes[@]},
  "total_cpu_percent": $(echo "$total_cpu" | bc -l 2>/dev/null || echo "0"),
  "total_memory_percent": $(echo "$total_memory" | bc -l 2>/dev/null || echo "0"),
  "processes": $processes_json,
  "monitored_at": "$(date -Iseconds)"
}
EOF
}

################################################################################
# Network Security
################################################################################

#######################################
# Setup network policy for workspace
# Arguments:
#   $1 - Workspace ID
#   $2 - Security level
#######################################
workspace_security::setup_network_policy() {
    local workspace_id="$1"
    local security_level="$2"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Create network control script
    cat > "$workspace_dir/config/network_control.sh" << 'EOF'
#!/bin/bash
# Network control for workspace operations

WORKSPACE_ID="$1"
OPERATION="$2"  # allow, block, check
TARGET="$3"     # domain, IP, or port

# Load security policy
POLICY_FILE="$(dirname "$0")/security_policy.json"
if [[ -f "$POLICY_FILE" ]]; then
    NETWORK_ALLOWED=$(jq -r '.network.allowed' "$POLICY_FILE")
    BLOCKED_DOMAINS=$(jq -r '.network.blocked_domains[]?' "$POLICY_FILE" 2>/dev/null)
    ALLOWED_PORTS=$(jq -r '.network.allowed_ports[]?' "$POLICY_FILE" 2>/dev/null)
else
    NETWORK_ALLOWED="true"
    BLOCKED_DOMAINS=""
    ALLOWED_PORTS=""
fi

case "$OPERATION" in
    check)
        # Check if network access is allowed
        if [[ "$NETWORK_ALLOWED" == "false" ]]; then
            echo "BLOCKED: Network access disabled for this workspace"
            exit 1
        fi
        
        # Check domain blocklist
        for blocked in $BLOCKED_DOMAINS; do
            if [[ "$TARGET" == "$blocked" ]]; then
                echo "BLOCKED: Domain $TARGET is in blocklist"
                exit 1
            fi
        done
        
        echo "ALLOWED: Network access permitted for $TARGET"
        exit 0
        ;;
    monitor)
        # Monitor network connections
        netstat -tuln 2>/dev/null | grep -E "(LISTEN|ESTABLISHED)" || echo "No active connections"
        ;;
esac
EOF
    
    chmod +x "$workspace_dir/config/network_control.sh"
}

#######################################
# Monitor workspace network activity
# Arguments:
#   $1 - Workspace ID
# Returns:
#   Network monitoring report JSON
#######################################
workspace_security::monitor_network() {
    local workspace_id="$1"
    
    # Get network connections for workspace processes
    local connections=()
    local connection_count=0
    
    # Find processes associated with the workspace
    while IFS= read -r pid; do
        [[ -z "$pid" ]] && continue
        
        # Get network connections for this process
        local proc_connections
        proc_connections=$(lsof -p "$pid" -i 2>/dev/null | grep -E "(TCP|UDP)" || true)
        
        while IFS= read -r conn; do
            [[ -z "$conn" ]] && continue
            
            # Parse connection info
            local protocol local_addr foreign_addr status
            protocol=$(echo "$conn" | awk '{print $5}')
            local_addr=$(echo "$conn" | awk '{print $9}' | cut -d'-' -f1)
            foreign_addr=$(echo "$conn" | awk '{print $9}' | cut -d'-' -f2)
            status=$(echo "$conn" | awk '{print $8}')
            
            connections+=("{\"protocol\": \"$protocol\", \"local\": \"$local_addr\", \"foreign\": \"$foreign_addr\", \"status\": \"$status\", \"pid\": $pid}")
            ((connection_count++))
        done <<< "$proc_connections"
    done < <(pgrep -f "workspace-$workspace_id")
    
    # Build connections JSON
    local connections_json="[]"
    for conn in "${connections[@]}"; do
        connections_json=$(echo "$connections_json" | jq ". + [$conn]")
    done
    
    cat << EOF
{
  "workspace_id": "$workspace_id",
  "connection_count": $connection_count,
  "connections": $connections_json,
  "monitored_at": "$(date -Iseconds)"
}
EOF
}

################################################################################
# Filesystem Security
################################################################################

#######################################
# Apply filesystem security policy
# Arguments:
#   $1 - Workspace directory
#   $2 - Security level
#######################################
workspace_security::apply_filesystem_policy() {
    local workspace_dir="$1"
    local security_level="$2"
    
    # Create filesystem wrapper script
    cat > "$workspace_dir/config/filesystem_control.sh" << 'EOF'
#!/bin/bash
# Filesystem access control for workspace

OPERATION="$1"  # read, write, execute, delete
TARGET_PATH="$2"
WORKSPACE_DIR="$3"

# Load security policy
POLICY_FILE="$(dirname "$0")/security_policy.json"
if [[ -f "$POLICY_FILE" ]]; then
    ALLOWED_PATHS=$(jq -r '.filesystem.allowed_paths[]?' "$POLICY_FILE" 2>/dev/null)
    FORBIDDEN_PATHS=$(jq -r '.filesystem.forbidden_paths[]?' "$POLICY_FILE" 2>/dev/null)
    MAX_FILE_SIZE=$(jq -r '.filesystem.max_file_size' "$POLICY_FILE")
    READ_ONLY_SYSTEM=$(jq -r '.filesystem.read_only_system' "$POLICY_FILE")
else
    ALLOWED_PATHS="*"
    FORBIDDEN_PATHS=""
    MAX_FILE_SIZE="100M"
    READ_ONLY_SYSTEM="true"
fi

# Check if path is forbidden
for forbidden in $FORBIDDEN_PATHS; do
    if [[ "$TARGET_PATH" =~ ^$forbidden ]]; then
        echo "ERROR: Access to path $TARGET_PATH is forbidden"
        exit 1
    fi
done

# Check if path is allowed (if not wildcard)
if [[ "$ALLOWED_PATHS" != "*" ]]; then
    ALLOWED=false
    for allowed in $ALLOWED_PATHS; do
        if [[ "$TARGET_PATH" =~ ^$allowed ]] || [[ "$TARGET_PATH" =~ ^$WORKSPACE_DIR ]]; then
            ALLOWED=true
            break
        fi
    done
    
    if [[ "$ALLOWED" != "true" ]]; then
        echo "ERROR: Access to path $TARGET_PATH is not allowed"
        exit 1
    fi
fi

# Check read-only restrictions for system paths
if [[ "$READ_ONLY_SYSTEM" == "true" && "$OPERATION" =~ (write|delete) ]]; then
    if [[ ! "$TARGET_PATH" =~ ^($WORKSPACE_DIR|/tmp|/var/tmp) ]]; then
        echo "ERROR: Write/delete operations outside workspace are forbidden"
        exit 1
    fi
fi

echo "ALLOWED: $OPERATION on $TARGET_PATH"
exit 0
EOF
    
    chmod +x "$workspace_dir/config/filesystem_control.sh"
    
    # Set up directory permissions based on security level
    case "$security_level" in
        strict)
            chmod 700 "$workspace_dir"
            find "$workspace_dir" -type d -exec chmod 700 {} \;
            find "$workspace_dir" -type f -exec chmod 600 {} \;
            ;;
        moderate)
            chmod 750 "$workspace_dir"
            find "$workspace_dir" -type d -exec chmod 750 {} \;
            find "$workspace_dir" -type f -exec chmod 640 {} \;
            ;;
        relaxed)
            chmod 755 "$workspace_dir"
            find "$workspace_dir" -type d -exec chmod 755 {} \;
            find "$workspace_dir" -type f -exec chmod 644 {} \;
            ;;
    esac
}

################################################################################
# Security Monitoring and Alerts
################################################################################

#######################################
# Setup comprehensive security monitoring
# Arguments:
#   $1 - Workspace ID
#######################################
workspace_security::setup_security_monitoring() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Create security monitoring script
    cat > "$workspace_dir/config/security_monitor.sh" << 'EOF'
#!/bin/bash
# Comprehensive security monitoring for workspace

WORKSPACE_ID="$1"
WORKSPACE_DIR="$2"

# Monitoring loop
while true; do
    TIMESTAMP=$(date -Iseconds)
    
    # Monitor processes
    PROCESS_COUNT=$(pgrep -f "workspace-$WORKSPACE_ID" | wc -l)
    
    # Monitor disk usage
    DISK_USAGE=$(du -sb "$WORKSPACE_DIR" 2>/dev/null | cut -f1 || echo "0")
    
    # Monitor file count
    FILE_COUNT=$(find "$WORKSPACE_DIR" -type f 2>/dev/null | wc -l || echo "0")
    
    # Check for suspicious activities
    SUSPICIOUS=""
    
    # Check for excessive process creation
    if [[ $PROCESS_COUNT -gt 20 ]]; then
        SUSPICIOUS="$SUSPICIOUS;EXCESSIVE_PROCESSES"
    fi
    
    # Check for rapid file creation
    if [[ $FILE_COUNT -gt 5000 ]]; then
        SUSPICIOUS="$SUSPICIOUS;EXCESSIVE_FILES"
    fi
    
    # Check for large files
    LARGE_FILES=$(find "$WORKSPACE_DIR" -type f -size +100M 2>/dev/null | wc -l)
    if [[ $LARGE_FILES -gt 0 ]]; then
        SUSPICIOUS="$SUSPICIOUS;LARGE_FILES"
    fi
    
    # Log monitoring data
    echo "$TIMESTAMP MONITOR workspace=$WORKSPACE_ID processes=$PROCESS_COUNT disk=$DISK_USAGE files=$FILE_COUNT suspicious=$SUSPICIOUS" >> "$WORKSPACE_DIR/logs/security.log"
    
    # Alert on suspicious activity
    if [[ -n "$SUSPICIOUS" ]]; then
        echo "$TIMESTAMP ALERT workspace=$WORKSPACE_ID activities=$SUSPICIOUS" >> "$WORKSPACE_DIR/logs/security.log"
    fi
    
    sleep 60  # Monitor every minute
done
EOF
    
    chmod +x "$workspace_dir/config/security_monitor.sh"
    
    # Start monitoring in background
    if command -v nohup >/dev/null 2>&1; then
        nohup "$workspace_dir/config/security_monitor.sh" "$workspace_id" "$workspace_dir" &
        echo $! > "$workspace_dir/.security_monitor_pid"
    fi
}

#######################################
# Get security status for workspace
# Arguments:
#   $1 - Workspace ID
# Returns:
#   Security status JSON
#######################################
workspace_security::get_status() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    if [[ ! -d "$workspace_dir" ]]; then
        echo '{"success": false, "error": "Workspace not found"}'
        return 1
    fi
    
    # Get process monitoring data
    local process_status
    process_status=$(workspace_security::monitor_processes "$workspace_id")
    
    # Get network monitoring data
    local network_status
    network_status=$(workspace_security::monitor_network "$workspace_id")
    
    # Check for security alerts
    local alerts="[]"
    local security_log="$workspace_dir/logs/security.log"
    if [[ -f "$security_log" ]]; then
        alerts=$(grep "ALERT" "$security_log" | tail -10 | jq -R . | jq -s .)
    fi
    
    # Get policy information
    local policy="{}"
    local policy_file="$workspace_dir/config/security_policy.json"
    if [[ -f "$policy_file" ]]; then
        policy=$(cat "$policy_file")
    fi
    
    cat << EOF
{
  "success": true,
  "workspace_id": "$workspace_id",
  "security_level": $(echo "$policy" | jq -r '.security_level // "unknown"'),
  "processes": $process_status,
  "network": $network_status,
  "alerts": $alerts,
  "policy": $policy,
  "monitored_at": "$(date -Iseconds)"
}
EOF
}

#######################################
# Terminate security monitoring
# Arguments:
#   $1 - Workspace ID
#######################################
workspace_security::cleanup_monitoring() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Stop security monitoring process
    if [[ -f "$workspace_dir/.security_monitor_pid" ]]; then
        local monitor_pid
        monitor_pid=$(cat "$workspace_dir/.security_monitor_pid")
        
        if [[ -n "$monitor_pid" ]] && kill -0 "$monitor_pid" 2>/dev/null; then
            kill "$monitor_pid" 2>/dev/null
        fi
        
        rm -f "$workspace_dir/.security_monitor_pid"
    fi
}

# Export functions
export -f workspace_security::apply_policy
export -f workspace_security::monitor_processes
export -f workspace_security::monitor_network
export -f workspace_security::get_status
export -f workspace_security::cleanup_monitoring