#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/network_diagnostics_core.sh"

@test "sourcing network_diagnostics_core.sh defines network_diagnostics_core::run function" {
    run bash -c "source '$SCRIPT_PATH' && declare -f network_diagnostics_core::run"
    [ "$status" -eq 0 ]
    [[ "$output" =~ network_diagnostics_core::run ]]
}

@test "network_diagnostics_core::run prints header when starting" {
    # Mock critical commands to succeed quickly
    ping() { return 0; }
    getent() { echo 'google.com has address 8.8.8.8'; }
    nc() { return 0; }
    curl() { return 0; }
    timeout() { shift; "$@"; }
    command() { return 0; }
    date() { echo '1234567890'; }
    ip() { echo 'default via 192.168.1.1 dev eth0'; }
    export -f ping getent nc curl timeout command date ip
    
    run bash -c "
        source '$SCRIPT_PATH'
        network_diagnostics_core::run
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Running Core Network Diagnostics" ]]
}

@test "network_diagnostics_core::run fails with critical network issues" {
    # Mock ping to fail (critical test)
    ping() { 
        if [[ "$*" =~ 8\.8\.8\.8 ]]; then
            return 1  # Critical test fails
        fi
        return 0
    }
    getent() { return 0; }
    nc() { return 0; }
    curl() { return 0; }
    timeout() { shift; "$@"; }
    command() { return 0; }
    export -f ping getent nc curl timeout command
    
    run bash -c "
        export ERROR_NO_INTERNET=5
        source '$SCRIPT_PATH'
        network_diagnostics_core::run
    "
    [ "$status" -eq 5 ]
    [[ "$output" =~ "Critical network issues detected" ]]
}

@test "network_diagnostics_core::run includes TLS handshake timing test" {
    # Mock basic commands to succeed
    ping() { return 0; }
    getent() { echo 'google.com has address 8.8.8.8'; }
    nc() { return 0; }
    curl() { return 0; }
    timeout() { shift; "$@"; }
    command() { return 0; }
    date() { echo '1234567890'; }
    ip() { echo 'default via 192.168.1.1 dev eth0'; }
    export -f ping getent nc curl timeout command date ip
    
    run bash -c "
        source '$SCRIPT_PATH'
        network_diagnostics_core::run
    "
    [[ "$output" =~ "TLS handshake" ]]
}

@test "network_diagnostics_core::run checks local services" {
    # Mock basic commands to succeed
    ping() { return 0; }
    getent() { echo 'google.com has address 8.8.8.8'; }
    curl() { return 0; }
    timeout() { shift; "$@"; }
    command() { return 0; }
    date() { echo '1234567890'; }
    ip() { echo 'default via 192.168.1.1 dev eth0'; }
    # Mock nc to simulate SearXNG port open
    nc() { 
        if [[ "$*" =~ 9200 ]]; then
            return 0  # Port is open
        fi
        return 0
    }
    export -f ping getent nc curl timeout command date ip
    
    run bash -c "
        source '$SCRIPT_PATH'
        network_diagnostics_core::run
    "
    [[ "$output" =~ "Localhost port 9200" ]]
}

@test "network_diagnostics_core::run provides diagnosis for TLS issues" {
    # Mock basic networking to work
    ping() { return 0; }
    getent() { echo 'google.com has address 8.8.8.8'; }
    nc() { return 0; }
    # Mock HTTPS to fail but basic connectivity to work
    curl() {
        if [[ "$*" =~ https ]]; then
            return 1  # HTTPS fails
        fi
        return 0  # Other commands work
    }
    timeout() { shift; "$@"; }
    command() { return 0; }
    date() { echo '1234567890'; }
    ip() { echo 'default via 192.168.1.1 dev eth0'; }
    ethtool() { echo 'tcp-segmentation-offload: on'; }
    export -f ping getent nc curl timeout command date ip ethtool
    
    run bash -c "
        source '$SCRIPT_PATH'
        network_diagnostics_core::run
    "
    [[ "$output" =~ "PARTIAL TLS ISSUE" ]]
    [[ "$output" =~ "TCP segmentation offload" ]]
}

@test "network_diagnostics_core::run includes automatic network optimizations" {
    # Mock basic networking to work but HTTPS to fail
    ping() { return 0; }
    getent() { echo 'google.com has address 8.8.8.8'; }
    nc() { return 0; }
    # Mock HTTPS to fail
    curl() {
        if [[ "$*" =~ https ]]; then
            return 1
        fi
        return 0
    }
    timeout() { shift; "$@"; }
    command() { return 0; }
    date() { echo '1234567890'; }
    ip() { echo 'default via 192.168.1.1 dev eth0'; }
    export -f ping getent nc curl timeout command date ip
    
    run bash -c "
        source '$SCRIPT_PATH'
        network_diagnostics_core::run
    "
    [[ "$output" =~ "Automatic Network Optimizations" ]]
    [[ "$output" =~ "Network issues detected" ]]
    [[ "$output" =~ "Testing alternative DNS servers" ]]
}