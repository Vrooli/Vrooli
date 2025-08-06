#!/usr/bin/env bash
# System Commands Mocks
# Provides mocks for common system commands (jq, systemctl, etc.)

# Prevent duplicate loading
if [[ "${COMMANDS_MOCKS_LOADED:-}" == "true" ]]; then
    return 0
fi
export COMMANDS_MOCKS_LOADED="true"

# Command mock state storage
declare -gA MOCK_COMMAND_OUTPUTS
declare -gA MOCK_COMMAND_EXIT_CODES
declare -gA MOCK_SERVICE_STATES

#######################################
# Set mock command output
# Arguments: $1 - command name, $2 - output, $3 - exit code
#######################################
mock::command::set_output() {
    local command="$1"
    local output="$2"
    local exit_code="${3:-0}"
    
    MOCK_COMMAND_OUTPUTS["$command"]="$output"
    MOCK_COMMAND_EXIT_CODES["$command"]="$exit_code"
}

#######################################
# Set mock service state for systemctl
# Arguments: $1 - service name, $2 - state (active/inactive/failed)
#######################################
mock::service::set_state() {
    local service="$1"
    local state="$2"
    
    MOCK_SERVICE_STATES["$service"]="$state"
}

#######################################
# command command mock
#######################################
command() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "command $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    case "$1" in
        "-v")
            local cmd="$2"
            case "$cmd" in
                "docker"|"curl"|"jq"|"systemctl"|"git"|"ssh"|"openssl"|"sudo")
                    echo "/usr/bin/$cmd"
                    return 0
                    ;;
                *)
                    return 1
                    ;;
            esac
            ;;
        *)
            return 0
            ;;
    esac
}

#######################################
# which command mock
#######################################
which() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "which $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    local cmd="$1"
    case "$cmd" in
        "docker"|"curl"|"jq"|"systemctl"|"git"|"ssh"|"openssl"|"sudo"|"bash"|"sh")
            echo "/usr/bin/$cmd"
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

#######################################
# type command mock
#######################################
type() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "type $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    case "$1" in
        "-t")
            local cmd="$2"
            case "$cmd" in
                "docker"|"curl"|"jq"|"systemctl"|"git"|"ssh"|"openssl"|"sudo")
                    echo "file"
                    return 0
                    ;;
                *)
                    return 1
                    ;;
            esac
            ;;
        *)
            local cmd="$1"
            case "$cmd" in
                "docker"|"curl"|"jq"|"systemctl"|"git"|"ssh"|"openssl"|"sudo")
                    echo "$cmd is /usr/bin/$cmd"
                    return 0
                    ;;
                *)
                    echo "bash: type: $cmd: not found" >&2
                    return 1
                    ;;
            esac
            ;;
    esac
}

#######################################
# sudo command mock
#######################################
sudo() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "sudo $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Just execute the command without sudo (in test environment)
    "$@"
}

#######################################
# id command mock
#######################################
id() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "id $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    case "$1" in
        "-u")
            echo "1000"  # Regular user
            ;;
        "-g")
            echo "1000"
            ;;
        "-un")
            echo "testuser"
            ;;
        "-gn")
            echo "testuser"
            ;;
        "root")
            return 1  # User doesn't exist
            ;;
        *)
            echo "uid=1000(testuser) gid=1000(testuser) groups=1000(testuser),4(adm),27(sudo),999(docker)"
            ;;
    esac
}

#######################################
# usermod command mock
#######################################
usermod() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "usermod $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Always succeed in test environment
    return 0
}

#######################################
# newgrp command mock
#######################################
newgrp() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "newgrp $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # In tests, just continue execution
    return 0
}

#######################################
# sleep command mock (for performance)
#######################################
sleep() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "sleep $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Don't actually sleep in tests unless in slow mode
    if [[ "${TEST_PERFORMANCE_MODE:-}" == "true" ]]; then
        return 0
    else
        # Very short sleep to simulate timing
        builtin sleep 0.01
    fi
}

#######################################
# timeout command mock
#######################################
timeout() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "timeout $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Skip the timeout and just run the command
    local duration="$1"
    shift
    "$@"
}

#######################################
# git command mock
#######################################
git() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "git $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    case "$1" in
        "clone")
            echo "Cloning into '$(basename "$2" .git)'..."
            echo "remote: Counting objects: 100, done."
            echo "Receiving objects: 100% (100/100), done."
            return 0
            ;;
        "pull")
            echo "Already up to date."
            return 0
            ;;
        "status")
            echo "On branch main"
            echo "nothing to commit, working tree clean"
            return 0
            ;;
        "rev-parse")
            echo "abc123def456"
            return 0
            ;;
        *)
            return 0
            ;;
    esac
}

#######################################
# openssl command mock
#######################################
openssl() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "openssl $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    case "$1" in
        "rand")
            echo "mock_random_string_123456"
            ;;
        "version")
            echo "OpenSSL 3.0.0 7 sep 2021"
            ;;
        *)
            echo "mock_openssl_output"
            ;;
    esac
}

#######################################
# ssh command mock
#######################################
ssh() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "ssh $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Mock SSH connection
    echo "Mock SSH connection to $1"
    return 0
}

#######################################
# lsof command mock (for port checking)
#######################################
lsof() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "lsof $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    case "$*" in
        *":8080"*)
            echo "COMMAND  PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME"
            echo "node    1234 user    3u  IPv4  12345      0t0  TCP *:8080 (LISTEN)"
            return 0
            ;;
        *":11434"*)
            echo "COMMAND  PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME"
            echo "ollama  5678 user    3u  IPv4  12346      0t0  TCP *:11434 (LISTEN)"
            return 0
            ;;
        *)
            # No ports in use by default
            return 1
            ;;
    esac
}

#######################################
# netstat command mock
#######################################
netstat() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "netstat $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    case "$*" in
        *"-ln"*)
            echo "Proto Recv-Q Send-Q Local Address           Foreign Address         State"
            echo "tcp        0      0 0.0.0.0:8080            0.0.0.0:*               LISTEN"
            echo "tcp        0      0 0.0.0.0:11434           0.0.0.0:*               LISTEN"
            ;;
        *)
            echo "Mock netstat output"
            ;;
    esac
}

#######################################
# ps command mock
#######################################
ps() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "ps $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    echo "  PID TTY          TIME CMD"
    echo " 1234 pts/0    00:00:01 bash"
    echo " 5678 pts/0    00:00:00 mock_process"
}

# Export all functions
export -f jq systemctl command which type sudo id usermod newgrp
export -f sleep timeout git openssl ssh lsof netstat ps
export -f mock::command::set_output mock::service::set_state

echo "[COMMANDS_MOCKS] System commands mocks loaded"