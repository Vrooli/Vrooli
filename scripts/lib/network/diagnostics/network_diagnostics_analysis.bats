#!/usr/bin/env bats

# Tests for network diagnostics analysis functions
# Tests TLS analysis, IPv4/IPv6 comparisons, and detailed debugging

setup() {
    export BATS_TEST_DIRNAME="$BATS_TEST_DIRNAME"
    export TEST_TEMP_DIR="$(mktemp -d)"
    
    # Source trash module for safe test cleanup
    SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
    # shellcheck disable=SC1091
    source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
    
    # Source the script under test
    source "${BATS_TEST_DIRNAME}/network_diagnostics_analysis.sh"
    
    # Mock log functions
    log::header() { echo "[HEADER] $*"; }
    log::subheader() { echo "[SUBHEADER] $*"; }
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*"; }
    
    export -f log::header log::subheader log::info log::success log::warning log::error
}

teardown() {
    trash::safe_remove "$TEST_TEMP_DIR" --test-cleanup
}

# =============================================================================
# Tests for TLS handshake analysis
# =============================================================================

@test "analyze_tls_handshake - openssl not available" {
    command() { return 1; }
    export -f command
    
    run network_diagnostics_analysis::analyze_tls_handshake "test.com"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"OpenSSL not available for TLS analysis"* ]]
}

@test "analyze_tls_handshake - successful TLS analysis" {
    # Mock openssl to succeed
    openssl() {
        case "$*" in
            *"tls1"*)
                echo "Verify return code: 0 (ok)"
                return 0
                ;;
            *"tls1_2"*|*"tls1_3"*)
                echo "Verify return code: 0 (ok)"
                return 0
                ;;
            *"showcerts"*)
                echo "Certificate chain"
                echo " 0 s:/CN=test.com"
                echo "   i:/CN=Test CA"
                echo "-----BEGIN CERTIFICATE-----"
                echo "TESTCERT1"
                echo "-----END CERTIFICATE-----"
                echo " 1 s:/CN=Test CA"
                echo "   i:/CN=Root CA"
                echo "-----BEGIN CERTIFICATE-----"
                echo "TESTCERT2"
                echo "-----END CERTIFICATE-----"
                return 0
                ;;
            *"cipher"*)
                echo "Cipher    : ECDHE-RSA-AES256-GCM-SHA384"
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "openssl" ]] && return 0 || return 1; }
    
    export -f openssl timeout command
    
    run network_diagnostics_analysis::analyze_tls_handshake "test.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"✓ tls1: Supported"* ]]
    [[ "$output" == *"✓ tls1_2: Supported"* ]]
    [[ "$output" == *"✓ tls1_3: Supported"* ]]
    [[ "$output" == *"Working TLS versions:"* ]]
    [[ "$output" == *"Negotiated cipher: ECDHE-RSA-AES256-GCM-SHA384"* ]]
    [[ "$output" == *"Certificate chain length: 2"* ]]
}

@test "analyze_tls_handshake - no TLS versions work" {
    # Mock openssl to fail for all versions
    openssl() { return 1; }
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "openssl" ]] && return 0 || return 1; }
    
    export -f openssl timeout command
    
    run network_diagnostics_analysis::analyze_tls_handshake "test.com"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"✗ tls1: Not supported or failed"* ]]
    [[ "$output" == *"✗ No TLS versions work with test.com"* ]]
    [[ "$output" == *"Attempting detailed SSL diagnostics"* ]]
}

# =============================================================================
# Tests for IPv4 vs IPv6 connectivity
# =============================================================================

@test "check_ipv4_vs_ipv6 - IPv4 only works" {
    # Mock ping
    ping() {
        case "$*" in
            *"-4"*)
                return 0
                ;;
            *"-6"*)
                return 1
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    # Mock getent
    getent() {
        case "$*" in
            *"ahostsv4"*)
                echo "192.168.1.1 test.com"
                return 0
                ;;
            *"ahostsv6"*)
                return 1
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    # Mock curl
    curl() {
        if [[ "$*" == *"-4"* ]]; then
            echo "200"
            return 0
        else
            return 1
        fi
    }
    
    timeout() { shift; "$@"; }
    command() { [[ "$2" == "curl" ]] && return 0 || return 1; }
    
    export -f ping getent curl timeout command
    
    run network_diagnostics_analysis::check_ipv4_vs_ipv6 "test.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"✓ IPv4 connectivity works"* ]]
    [[ "$output" == *"✗ IPv6 connectivity failed"* ]]
    [[ "$output" == *"IPv4 address: 192.168.1.1"* ]]
    [[ "$output" == *"✓ HTTPS over IPv4 works"* ]]
    [[ "$output" == *"Recommendation: Use IPv4-only mode"* ]]
}

# =============================================================================
# Tests for time synchronization
# =============================================================================

@test "check_time_sync - time is correct" {
    # Mock date
    date() {
        case "$*" in
            "+%Y")
                echo "2024"
                return 0
                ;;
            *)
                echo "Wed Aug  9 17:30:00 UTC 2024"
                return 0
                ;;
        esac
    }
    
    # Mock timedatectl
    timedatectl() {
        echo "NTP synchronized: yes"
        return 0
    }
    
    command() { [[ "$2" == "timedatectl" ]] && return 0 || return 1; }
    
    export -f date timedatectl command
    
    run network_diagnostics_analysis::check_time_sync
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"✓ System time appears reasonable"* ]]
    [[ "$output" == *"NTP status: NTP synchronized: yes"* ]]
}

@test "check_time_sync - incorrect year" {
    # Mock date to return invalid year
    date() {
        case "$*" in
            "+%Y")
                echo "2010"
                return 0
                ;;
            *)
                echo "Wed Jan  1 00:00:00 UTC 2010"
                return 0
                ;;
        esac
    }
    
    command() { return 1; }
    
    export -f date command
    
    run network_diagnostics_analysis::check_time_sync
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"✗ System time appears incorrect"* ]]
    [[ "$output" == *"This can cause TLS certificate validation failures"* ]]
}

# =============================================================================
# Tests for verbose HTTPS debugging
# =============================================================================

@test "verbose_https_debug - diagnoses connection issues" {
    # Create temp file for test
    local mock_temp_file="$TEST_TEMP_DIR/curl_output"
    
    # Mock mktemp
    mktemp() {
        echo "$mock_temp_file"
    }
    
    # Mock timeout and bash
    timeout() {
        shift  # Remove timeout value
        # Simulate curl failure with SSL error
        echo "* SSL_ERROR_SYSCALL in connection to github.com:443" > "$mock_temp_file"
        echo "curl: (35) SSL connection error" >> "$mock_temp_file"
        return 1
    }
    
    # Mock grep
    grep() {
        case "$*" in
            *"SSL_ERROR"*)
                command grep "$@" && return 0 || return 1
                ;;
            *"curl:"*)
                echo "curl: (35) SSL connection error"
                return 0
                ;;
            *)
                command grep "$@"
                ;;
        esac
    }
    
    bash() {
        timeout 10 "$@"
    }
    
    export -f mktemp timeout grep bash
    
    run network_diagnostics_analysis::verbose_https_debug "curl -s https://github.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Command failed"* ]]
    [[ "$output" == *"SSL/TLS certificate issue detected"* ]]
    [[ "$output" == *"curl: (35) SSL connection error"* ]]
}

@test "verbose_https_debug - successful connection" {
    # Mock successful scenario
    local mock_temp_file="$TEST_TEMP_DIR/curl_output"
    
    mktemp() { echo "$mock_temp_file"; }
    
    timeout() {
        shift
        echo "HTTP/1.1 200 OK" > "$mock_temp_file"
        return 0
    }
    
    bash() {
        timeout 10 "$@"
    }
    
    export -f mktemp timeout bash
    
    run network_diagnostics_analysis::verbose_https_debug "curl -s https://github.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Command succeeded with verbose output"* ]]
}