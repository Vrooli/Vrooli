#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/network_diagnostics.sh"

@test "sourcing network_diagnostics.sh defines network_diagnostics::run function" {
    run bash -c "source '$SCRIPT_PATH' && declare -f network_diagnostics::run"
    [ "$status" -eq 0 ]
    [[ "$output" =~ network_diagnostics::run ]]
}

@test "network_diagnostics::run prints header when starting" {
    # Mock critical commands to succeed quickly
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock basic networking commands
        ping() { return 0; }
        getent() { echo 'google.com has address 8.8.8.8'; }
        nc() { return 0; }
        curl() { return 0; }
        # Mock array for test results
        declare -A TEST_RESULTS
        network_diagnostics::run
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ \[HEADER\].*Running\ Comprehensive\ Network\ Diagnostics ]]
}

@test "network_diagnostics::run fails with critical network issues" {
    # Mock ping to fail (critical test)
    run bash -c "
        export ERROR_NO_INTERNET=5
        source '$SCRIPT_PATH'
        # Mock ping to fail
        ping() { return 1; }
        # Mock other commands to prevent further testing
        getent() { return 0; }
        nc() { return 0; }
        curl() { return 0; }
        network_diagnostics::run
    "
    [ "$status" -eq 5 ]
    [[ "$output" =~ \[ERROR\].*Critical\ network\ issues\ detected ]]
}

@test "network_diagnostics::run includes TLS handshake timing test" {
    # Mock curl to simulate timeout
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock basic commands to succeed
        ping() { return 0; }
        getent() { echo 'google.com has address 8.8.8.8'; }
        nc() { return 0; }
        # Mock curl to simulate TLS timeout
        curl() { 
            if [[ \"\$*\" =~ https ]]; then
                sleep 1  # Simulate delay
                echo 'SSL connection timeout'
                return 1
            fi
            return 0
        }
        timeout() { shift; \"\$@\"; }
        date() { echo '1234567890'; }
        # Mock array for test results
        declare -A TEST_RESULTS
        network_diagnostics::run
    "
    [[ "$output" =~ TLS\ handshake ]]
}

@test "network_diagnostics::run checks local services" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock basic commands to succeed
        ping() { return 0; }
        getent() { echo 'google.com has address 8.8.8.8'; }
        curl() { return 0; }
        # Mock nc to simulate SearXNG port open but HTTP failing
        nc() { 
            if [[ \"\$*\" =~ 9200 ]]; then
                return 0  # Port is open
            fi
            return 0
        }
        # Mock array for test results
        declare -A TEST_RESULTS
        network_diagnostics::run
    "
    [[ "$output" =~ "Localhost port 9200" ]]
}

@test "network_diagnostics::run provides diagnosis for TLS issues" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock basic networking to work
        ping() { return 0; }
        getent() { echo 'google.com has address 8.8.8.8'; }
        nc() { return 0; }
        # Mock HTTP to work but HTTPS to fail, TLS specific versions to work
        curl() {
            if [[ \"\$*\" =~ https ]] && [[ ! \"\$*\" =~ tlsv1 ]]; then
                return 1  # HTTPS without explicit TLS version fails
            fi
            return 0  # HTTP and explicit TLS versions work
        }
        timeout() { shift; \"\$@\"; }
        date() { echo '1234567890'; }
        dig() { return 0; }
        sudo() { return 1; }  # No sudo access
        ethtool() { echo 'tcp-segmentation-offload: on'; }
        ip() { 
            if [[ \"\$*\" =~ route ]]; then
                echo 'default via 192.168.1.1 dev eth0'
            elif [[ \"\$*\" =~ link ]]; then
                echo 'eth0: mtu 1500'
            fi
        }
        systemctl() { return 1; }  # systemd-resolved not active
        command() { 
            case \"\$2\" in
                ethtool|dig|curl|nc|ping|getent) return 0 ;;
                *) return 1 ;;
            esac
        }
        # Mock array for test results
        declare -A TEST_RESULTS
        network_diagnostics::run
    " <<< "y"  # Answer yes to continue despite failures
    [[ "$output" =~ "PARTIAL TLS ISSUE" ]]
    [[ "$output" =~ "TCP segmentation offload" ]]
}

@test "network_diagnostics::run includes automatic network optimizations" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock basic networking to work
        ping() { return 0; }
        getent() { echo 'google.com has address 8.8.8.8'; }
        nc() { return 0; }
        # Mock HTTPS to fail initially
        curl() {
            if [[ \"\$*\" =~ https ]]; then
                return 1
            fi
            return 0
        }
        timeout() { shift; \"\$@\"; }
        dig() { return 0; }
        sudo() { return 1; }
        ethtool() { echo 'tcp-segmentation-offload: on'; }
        ip() { echo 'default via 192.168.1.1 dev eth0'; }
        command() { return 0; }
        # Mock array for test results
        declare -A TEST_RESULTS
        network_diagnostics::run
    " <<< "y"
    [[ "$output" =~ "Automatic Network Optimizations" ]]
    [[ "$output" =~ "Network issues detected" ]]
    [[ "$output" =~ "Testing alternative DNS servers" ]]
}