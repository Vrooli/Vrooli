#!/usr/bin/env bash
# System Commands Mocks
# Provides mocks for common system commands (jq, systemctl, etc.)

# Prevent duplicate loading
if [[ "${COMMANDS_MOCKS_LOADED:-}" == "true" ]]; then
    return 0
fi
export COMMANDS_MOCKS_LOADED="true"

# Command mock state storage
declare -A MOCK_COMMAND_OUTPUTS
declare -A MOCK_COMMAND_EXIT_CODES
declare -A MOCK_SERVICE_STATES

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
# jq command mock
#######################################
jq() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "jq $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Check for custom mock output
    if [[ -n "${MOCK_COMMAND_OUTPUTS[jq]:-}" ]]; then
        echo "${MOCK_COMMAND_OUTPUTS[jq]}"
        return "${MOCK_COMMAND_EXIT_CODES[jq]:-0}"
    fi
    
    # Handle common jq patterns
    local filter="$1"
    local input=""
    local raw_output=false
    
    # Check for -r flag
    if [[ "$filter" == "-r" ]]; then
        raw_output=true
        shift
        filter="$1"
    fi
    
    # Read from stdin if no file specified
    if [[ $# -eq 1 ]] || ([[ $# -eq 2 ]] && [[ "$raw_output" == true ]]); then
        input=$(cat)
    else
        # Handle file input
        if [[ "$raw_output" == true ]]; then
            shift
        fi
        shift
        input=$(cat "$@" 2>/dev/null || echo "{}")
    fi
    
    # Try to extract value from actual JSON input first
    local extracted_value=""
    if [[ -n "$input" && "$input" =~ ^\{.*\}$ ]]; then
        case "$filter" in
            ".status")
                extracted_value=$(echo "$input" | sed -n 's/.*"status"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
                ;;
            ".version")
                extracted_value=$(echo "$input" | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
                ;;
            ".enabled")
                extracted_value=$(echo "$input" | sed -n 's/.*"enabled"[[:space:]]*:[[:space:]]*\([^,}]*\).*/\1/p')
                ;;
            ".health")
                extracted_value=$(echo "$input" | sed -n 's/.*"health"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
                ;;
        esac
    fi
    
    # Output the result
    case "$filter" in
        ".status")
            if [[ -n "$extracted_value" ]]; then
                if [[ "$raw_output" == true ]]; then
                    echo "$extracted_value"
                else
                    echo "\"$extracted_value\""
                fi
            else
                if [[ "$raw_output" == true ]]; then
                    echo "ok"
                else
                    echo "\"ok\""
                fi
            fi
            ;;
        ".version")
            if [[ -n "$extracted_value" ]]; then
                if [[ "$raw_output" == true ]]; then
                    echo "$extracted_value"
                else
                    echo "\"$extracted_value\""
                fi
            else
                if [[ "$raw_output" == true ]]; then
                    echo "1.0.0"
                else
                    echo "\"1.0.0\""
                fi
            fi
            ;;
        ".enabled")
            if [[ -n "$extracted_value" ]]; then
                echo "$extracted_value"
            else
                echo "true"
            fi
            ;;
        ".health")
            if [[ -n "$extracted_value" ]]; then
                if [[ "$raw_output" == true ]]; then
                    echo "$extracted_value"
                else
                    echo "\"$extracted_value\""
                fi
            else
                if [[ "$raw_output" == true ]]; then
                    echo "healthy"
                else
                    echo "\"healthy\""
                fi
            fi
            ;;
        ".[]")
            echo "\"item1\""
            echo "\"item2\""
            ;;
        ".")
            # Identity filter - validate and pretty print JSON
            if echo "$input" | grep -q "^{"; then
                echo "$input"
            else
                echo "{}"
            fi
            ;;
        "keys")
            echo "[\"key1\", \"key2\", \"key3\"]"
            ;;
        "length")
            echo "3"
            ;;
        "empty")
            # Return nothing
            ;;
        "has(\"*\")")
            echo "true"
            ;;
        *)
            # For complex filters, return a reasonable default
            if [[ "$filter" =~ \[\] ]]; then
                echo "[]"
            elif [[ "$filter" =~ \{\} ]]; then
                echo "{}"
            else
                echo "\"mock-value\""
            fi
            ;;
    esac
}

#######################################
# systemctl command mock
#######################################
systemctl() {
    # Track command calls
    if [[ -n "${MOCK_RESPONSES_DIR:-}" ]]; then
        echo "systemctl $*" >> "${MOCK_RESPONSES_DIR}/command_calls.log"
    fi
    
    # Check for custom mock output
    if [[ -n "${MOCK_COMMAND_OUTPUTS[systemctl]:-}" ]]; then
        echo "${MOCK_COMMAND_OUTPUTS[systemctl]}"
        return "${MOCK_COMMAND_EXIT_CODES[systemctl]:-0}"
    fi
    
    local action="$1"
    local service="$2"
    
    case "$action" in
        "status")
            local state="${MOCK_SERVICE_STATES[$service]:-active}"
            case "$state" in
                "active")
                    echo "● $service.service - Mock Service"
                    echo "   Loaded: loaded (/etc/systemd/system/$service.service; enabled)"
                    echo "   Active: active (running) since $(date)"
                    echo "   Main PID: 1234 ($service)"
                    return 0
                    ;;
                "inactive")
                    echo "● $service.service - Mock Service"
                    echo "   Loaded: loaded (/etc/systemd/system/$service.service; disabled)"
                    echo "   Active: inactive (dead)"
                    return 3
                    ;;
                "failed")
                    echo "● $service.service - Mock Service"
                    echo "   Loaded: loaded (/etc/systemd/system/$service.service; enabled)"
                    echo "   Active: failed (Result: exit-code)"
                    return 3
                    ;;
            esac
            ;;
        "start"|"stop"|"restart"|"reload")
            local state="${MOCK_SERVICE_STATES[$service]:-active}"
            if [[ "$state" != "failed" ]]; then
                return 0
            else
                echo "Job for $service.service failed." >&2
                return 1
            fi
            ;;
        "enable"|"disable")
            return 0
            ;;
        "is-active")
            local state="${MOCK_SERVICE_STATES[$service]:-active}"
            case "$state" in
                "active") echo "active"; return 0 ;;
                *) echo "inactive"; return 1 ;;
            esac
            ;;
        "is-enabled")
            echo "enabled"
            return 0
            ;;
        "list-units"|"list-unit-files")
            echo "UNIT FILE                STATE"
            echo "docker.service           enabled"
            echo "ssh.service              enabled"
            echo "$service.service        enabled"
            ;;
        "daemon-reload")
            return 0
            ;;
        *)
            echo "Unknown systemctl command: $action" >&2
            return 1
            ;;
    esac
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