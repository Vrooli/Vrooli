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
        # Enable test mode to avoid real network calls
        export VROOLI_TEST_MODE=true
        
        # Define mock functions first
        ping() { return 0; }
        getent() { echo 'google.com has address 8.8.8.8'; }
        nc() { return 0; }
        curl() { return 0; }
        
        # Export functions to make them available in subshells
        export -f ping getent nc curl
        
        # Source script and run test
        source '$SCRIPT_PATH'
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
        export VROOLI_TEST_MODE=true
        
        # Define mock functions first
        ping() { return 1; }
        getent() { return 0; }
        nc() { return 0; }
        curl() { return 0; }
        
        # Export functions to make them available in subshells
        export -f ping getent nc curl
        
        # Source script and run test
        source '$SCRIPT_PATH'
        network_diagnostics::run
    "
    [ "$status" -eq 5 ]
    [[ "$output" =~ \[ERROR\].*Critical\ network\ issues\ detected ]]
}

@test "network_diagnostics::run includes TLS handshake timing test" {
    # Mock curl to simulate timeout
    run bash -c "
        export VROOLI_TEST_MODE=true
        
        # Define mock functions first
        ping() { return 0; }
        getent() { echo 'google.com has address 8.8.8.8'; }
        nc() { return 0; }
        # Mock curl to simulate TLS timeout without real delays
        curl() { 
            if [[ \"\$*\" =~ https ]]; then
                echo 'SSL connection timeout'
                return 1
            fi
            return 0
        }
        timeout() { shift; \"\$@\"; }
        date() { echo '1234567890'; }
        
        # Export functions to make them available in subshells
        export -f ping getent nc curl timeout date
        
        # Source script and run test
        source '$SCRIPT_PATH'
        # Mock array for test results
        declare -A TEST_RESULTS
        network_diagnostics::run
    "
    [[ "$output" =~ TLS\ handshake ]]
}

@test "network_diagnostics::run checks local services" {
    run bash -c "
        export VROOLI_TEST_MODE=true
        
        # Define mock functions first
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
        
        # Export functions to make them available in subshells
        export -f ping getent curl nc
        
        # Source script and run test
        source '$SCRIPT_PATH'
        # Mock array for test results
        declare -A TEST_RESULTS
        network_diagnostics::run
    "
    [[ "$output" =~ "Localhost port 9200" ]]
}

@test "network_diagnostics::run provides diagnosis for TLS issues" {
    run bash -c "
        export VROOLI_TEST_MODE=true
        
        # Define mock functions first
        ping() { return 0; }
        getent() { echo 'google.com has address 8.8.8.8'; }
        nc() { return 0; }
        # Mock HTTPS to fail for PARTIAL TLS ISSUE test - with debug
        curl() {
            echo \"[DEBUG CURL] Called with args: \$*\" >&2
            if [[ \"\$*\" =~ https ]]; then
                echo \"[DEBUG CURL] HTTPS detected - returning failure\" >&2
                return 1  # All HTTPS calls fail
            fi
            echo \"[DEBUG CURL] Non-HTTPS - returning success\" >&2
            return 0  # Non-HTTPS calls work
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
        
        # Export functions to make them available in subshells
        export -f ping getent nc curl timeout date dig sudo ethtool ip systemctl command
        
        # Source script 
        source '$SCRIPT_PATH'
        
        # Override run_test function to force specific results for PARTIAL TLS ISSUE test
        run_test() {
            local test_name=\"\$1\"
            local test_command=\"\$2\"
            local is_critical=\"\${3:-false}\"
            
            log::info \"Testing: \$test_name\"
            
            # Force specific results to trigger PARTIAL TLS ISSUE condition
            case \"\$test_name\" in
                \"IPv4 ping to 8.8.8.8\"|\"IPv4 ping to google.com\"|\"DNS lookup (getent)\"|\"TCP port 443 (HTTPS)\")
                    TEST_RESULTS[\"\$test_name\"]=\"PASS\"
                    log::success \"  ✓ \$test_name\"
                    ;;
                \"HTTPS to google.com\"|\"HTTPS to github.com\")
                    TEST_RESULTS[\"\$test_name\"]=\"FAIL\"
                    log::error \"  ✗ \$test_name\"
                    ;;
                *)
                    TEST_RESULTS[\"\$test_name\"]=\"PASS\"
                    log::success \"  ✓ \$test_name\"
                    ;;
            esac
            return 0
        }
        export -f run_test
        
        # Mock array for test results
        declare -A TEST_RESULTS
        network_diagnostics::run
    " <<< "y"  # Answer yes to continue despite failures
    [[ "$output" =~ "PARTIAL TLS ISSUE" ]]
    [[ "$output" =~ "TCP segmentation offload" ]]
}

@test "network_diagnostics::run includes automatic network optimizations" {
    run bash -c "
        export VROOLI_TEST_MODE=true
        
        # Define mock functions first
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
        
        # Export functions to make them available in subshells
        export -f ping getent nc curl timeout dig sudo ethtool ip command
        
        # Source script
        source '$SCRIPT_PATH'
        
        # Override run_test function to force failures for network optimization test
        run_test() {
            local test_name=\"\$1\"
            local test_command=\"\$2\"
            local is_critical=\"\${3:-false}\"
            
            log::info \"Testing: \$test_name\"
            
            # Force some failures to trigger network optimization messages
            case \"\$test_name\" in
                \"HTTPS to google.com\"|\"HTTPS to github.com\")
                    TEST_RESULTS[\"\$test_name\"]=\"FAIL\"
                    log::error \"  ✗ \$test_name\"
                    ;;
                *)
                    TEST_RESULTS[\"\$test_name\"]=\"PASS\"
                    log::success \"  ✓ \$test_name\"
                    ;;
            esac
            return 0
        }
        export -f run_test
        
        # Mock array for test results
        declare -A TEST_RESULTS
        network_diagnostics::run
    " <<< "y"
    [[ "$output" =~ "Automatic Network Optimizations" ]]
    [[ "$output" =~ "Network issues detected" ]]
}