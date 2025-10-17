#!/usr/bin/env bats
# Tests for remote session protection functionality

setup() {
    export TEST_MODE=1
    SCRIPT_DIR="$(cd "$BATS_TEST_DIRNAME" && pwd)"
    REMOTE_SESSION_SCRIPT="${SCRIPT_DIR}/remote_session_protect.sh"
    
    # Source the script for function access
    source "${SCRIPT_DIR}/../utils/log.sh"
    source "${SCRIPT_DIR}/../utils/flow.sh"
    source "${SCRIPT_DIR}/../utils/system_commands.sh"
    source "$REMOTE_SESSION_SCRIPT"
}

@test "remote_session::calculate_memory_allocation - calculates correct values for 16GB system" {
    # Mock /proc/meminfo with 16GB RAM
    function awk() {
        if [[ "$1" == "/MemTotal:/" ]]; then
            echo "16777216"  # 16GB in KB
        fi
    }
    export -f awk
    
    remote_session::calculate_memory_allocation
    
    # Check calculations
    [ "$SYSTEM_MEM_GB" -eq 16 ]
    [ "$DESKTOP_MIN_CALC_MB" -ge 2400 ]  # 15% of 16GB
    [ "$DESKTOP_LOW_CALC_MB" -eq $((DESKTOP_MIN_CALC_MB + 2048)) ]
    [ "$TARGET_SWAP_GB" -eq 16 ]
}

@test "remote_session::calculate_memory_allocation - enforces minimum desktop memory" {
    # Mock /proc/meminfo with 8GB RAM
    function awk() {
        if [[ "$1" == "/MemTotal:/" ]]; then
            echo "8388608"  # 8GB in KB
        fi
    }
    export -f awk
    
    remote_session::calculate_memory_allocation
    
    # 15% of 8GB is ~1200MB, but minimum should be 4096MB
    [ "$DESKTOP_MIN_CALC_MB" -eq 4096 ]
}

@test "remote_session::calculate_memory_allocation - caps swap at 64GB" {
    # Mock /proc/meminfo with 128GB RAM
    function awk() {
        if [[ "$1" == "/MemTotal:/" ]]; then
            echo "134217728"  # 128GB in KB
        fi
    }
    export -f awk
    
    remote_session::calculate_memory_allocation
    
    # Swap should be capped at 64GB
    [ "$TARGET_SWAP_GB" -eq 64 ]
}

@test "remote_session::is_remote_desktop_installed - detects xrdp service" {
    # Mock systemctl to simulate xrdp installed
    function systemctl() {
        if [[ "$1" == "list-unit-files" ]]; then
            echo "xrdp.service enabled"
        fi
    }
    export -f systemctl
    
    run remote_session::is_remote_desktop_installed
    [ "$status" -eq 0 ]
}

@test "remote_session::is_remote_desktop_installed - detects RDP port in firewall" {
    # Mock ufw to simulate RDP port open
    function command() {
        if [[ "$2" == "ufw" ]]; then
            return 0
        fi
    }
    function sudo() {
        if [[ "$1" == "ufw" && "$2" == "status" ]]; then
            echo "3389/tcp ALLOW Anywhere"
        fi
    }
    export -f command sudo
    
    run remote_session::is_remote_desktop_installed
    [ "$status" -eq 0 ]
}

@test "remote_session::is_remote_desktop_installed - returns false when no remote desktop" {
    # Mock systemctl to show no remote services
    function systemctl() {
        if [[ "$1" == "list-unit-files" ]]; then
            echo "other.service enabled"
        fi
    }
    function command() {
        return 1  # ufw not found
    }
    export -f systemctl command
    
    run remote_session::is_remote_desktop_installed
    [ "$status" -eq 1 ]
}

@test "remote_session::main check - succeeds when no remote desktop installed" {
    # Mock no remote desktop
    function remote_session::is_remote_desktop_installed() {
        return 1
    }
    export -f remote_session::is_remote_desktop_installed
    
    run bash "$REMOTE_SESSION_SCRIPT" check
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No remote desktop installed" ]]
}

@test "remote_session::main configure - skips when no remote desktop" {
    # Mock no remote desktop
    function remote_session::is_remote_desktop_installed() {
        return 1
    }
    export -f remote_session::is_remote_desktop_installed
    
    run bash "$REMOTE_SESSION_SCRIPT" configure
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No remote desktop detected" ]]
}

@test "remote_session::main - handles unknown operation" {
    run bash "$REMOTE_SESSION_SCRIPT" invalid_operation
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown operation" ]]
}