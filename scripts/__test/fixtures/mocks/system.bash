#!/usr/bin/env bash
# System Mocks - Simplified System Command Mocking
# Consolidates basic system commands (systemctl, ps, etc.)

# Prevent duplicate loading
if [[ "${SYSTEM_MOCKS_LOADED:-}" == "true" ]]; then
    return 0
fi
export SYSTEM_MOCKS_LOADED="true"

echo "[MOCK] Loading system command mocks"

# Mock state storage
declare -A MOCK_SYSTEM_SERVICES
declare -A MOCK_PROCESSES

#######################################
# Mock systemctl command
#######################################
systemctl() {
    local action="$1"
    local svc_name="$2"
    
    echo "[MOCK] systemctl $action $svc_name"
    
    case "$action" in
        "start")
            MOCK_SYSTEM_SERVICES["$svc_name"]="active"
            echo "Started $svc_name"
            return 0
            ;;
        "stop")
            MOCK_SYSTEM_SERVICES["$svc_name"]="inactive"
            echo "Stopped $svc_name"
            return 0
            ;;
        "status")
            local svc_status="${MOCK_SYSTEM_SERVICES[$svc_name]:-active}"
            echo "● $svc_name - Mock Service"
            echo "   Active: $svc_status (mocked)"
            [[ "$svc_status" == "active" ]]
            ;;
        "is-active")
            local svc_status="${MOCK_SYSTEM_SERVICES[$svc_name]:-active}"
            echo "$svc_status"
            [[ "$svc_status" == "active" ]]
            ;;
        *)
            echo "Mock systemctl: action '$action' on service '$svc_name'"
            return 0
            ;;
    esac
}

#######################################
# Mock ps command
#######################################
ps() {
    echo "[MOCK] ps $*"
    
    # Check if we're looking for a specific PID
    if [[ "$*" =~ -p[[:space:]]*([0-9]+) ]]; then
        local check_pid="${BASH_REMATCH[1]}"
        # For real PIDs (not mock ones), use builtin ps
        if [[ ! "$check_pid" =~ ^(12345|67890|99999)$ ]]; then
            builtin ps "$@" 2>/dev/null
            return $?
        fi
    elif [[ "$1" == "-p" ]]; then
        local check_pid="$2"
        # For real PIDs (not mock ones), use builtin ps
        if [[ ! "$check_pid" =~ ^(12345|67890|99999)$ ]]; then
            builtin ps "$@" 2>/dev/null
            return $?
        fi
    fi
    
    # Check for common ps patterns
    case "$*" in
        *"grep"*)
            # If ps | grep pattern, return mock process
            echo "12345 pts/0    00:00:01 mock_process"
            ;;
        *"aux"*|*"-ef"*)
            # Full process list
            cat <<EOF
USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root         1  0.0  0.1   1234   567 ?        Ss   08:00   0:01 /sbin/init
test     12345  0.0  0.2   2345   678 pts/0    S+   08:01   0:00 mock_process
EOF
            ;;
        *)
            echo "  PID TTY          TIME CMD"
            echo "12345 pts/0    00:00:01 bash"
            ;;
    esac
    
    return 0
}

#######################################
# Mock pgrep command
#######################################
pgrep() {
    local pattern="$1"
    echo "[MOCK] pgrep $pattern"
    
    # Return mock PID for any pattern
    echo "12345"
    return 0
}

#######################################
# Mock pkill command
#######################################
pkill() {
    local pattern="$1"
    echo "[MOCK] pkill $pattern"
    echo "Killed processes matching: $pattern"
    return 0
}

#######################################
# Mock kill command
#######################################
kill() {
    local signal=""
    local pid=""
    
    # Parse arguments
    if [[ "$1" =~ ^-[0-9]+$ ]] || [[ "$1" =~ ^-[A-Z]+$ ]]; then
        signal="$1"
        pid="$2"
    else
        pid="$1"
    fi
    
    echo "[MOCK] kill ${signal:+$signal }$pid"
    
    # Special handling for kill -0 (test if process exists)
    if [[ "$signal" == "-0" ]]; then
        # Return success for mock PIDs, failure for others
        if [[ "$pid" =~ ^(12345|67890|99999)$ ]]; then
            return 0
        else
            return 1
        fi
    fi
    
    # For real PIDs (not mock ones), actually kill them
    if [[ ! "$pid" =~ ^(12345|67890|99999)$ ]]; then
        # Use builtin kill for real processes
        builtin kill ${signal:+$signal} "$pid" 2>/dev/null
        local result=$?
        if [[ $result -eq 0 ]]; then
            echo "Mock killed real process: $pid"
        fi
        return $result
    fi
    
    echo "Mock killed process: $pid"
    return 0
}

#######################################
# Mock which command
#######################################
which() {
    local command="$1"
    echo "[MOCK] which $command"
    
    # Return mock paths for common commands
    case "$command" in
        "docker"|"curl"|"jq"|"systemctl"|"ps")
            echo "/usr/bin/$command"
            ;;
        *)
            echo "/usr/local/bin/$command"
            ;;
    esac
    
    return 0
}

#######################################
# Mock id command
#######################################
id() {
    echo "[MOCK] id $*"
    
    case "$*" in
        *"-u"*)
            echo "1000"
            ;;
        *"-g"*)
            echo "1000"
            ;;
        *)
            echo "uid=1000(testuser) gid=1000(testgroup) groups=1000(testgroup)"
            ;;
    esac
    
    return 0
}

#######################################
# Mock whoami command
#######################################
whoami() {
    echo "[MOCK] whoami"
    echo "testuser"
    return 0
}

#######################################
# Mock uname command
#######################################
uname() {
    echo "[MOCK] uname $*"
    
    case "$*" in
        *"-a"*)
            echo "Linux testhost 5.4.0-test #1 SMP Mon Jan 1 12:00:00 UTC 2024 x86_64 x86_64 x86_64 GNU/Linux"
            ;;
        *"-s"*)
            echo "Linux"
            ;;
        *"-r"*)
            echo "5.4.0-test"
            ;;
        *"-m"*)
            echo "x86_64"
            ;;
        *)
            echo "Linux"
            ;;
    esac
    
    return 0
}

#######################################
# Mock date command (basic version)
#######################################
if ! command -v date >/dev/null 2>&1; then
    date() {
        echo "[MOCK] date $*"
        
        case "$*" in
            *"+%s"*)
                echo "1704067200"  # 2024-01-01 00:00:00 UTC
                ;;
            *"+%Y-%m-%d"*)
                echo "2024-01-01"
                ;;
            *)
                echo "Mon Jan  1 12:00:00 UTC 2024"
                ;;
        esac
        
        return 0
    }
fi

#######################################
# Set service status for systemctl mock
# Arguments: $1 - service name, $2 - status (active/inactive)
#######################################
mock_set_service_status() {
    local service="$1"
    local status="$2"
    
    MOCK_SYSTEM_SERVICES["$service"]="$status"
    echo "[MOCK] Set service $service status to: $status"
}

#######################################
# Add mock process for ps/pgrep
# Arguments: $1 - process name, $2 - PID
#######################################
mock_add_process() {
    local process="$1"
    local pid="$2"
    
    MOCK_PROCESSES["$process"]="$pid"
    echo "[MOCK] Added mock process: $process (PID: $pid)"
}

#######################################
# Cleanup system mocks
#######################################
system_mocks_cleanup() {
    unset MOCK_SYSTEM_SERVICES
    unset MOCK_PROCESSES
    echo "[MOCK] System mocks cleaned up"
}

# Export functions
export -f systemctl ps pgrep pkill kill which id whoami uname
export -f mock_set_service_status mock_add_process system_mocks_cleanup

echo "[MOCK] System command mocks loaded successfully"