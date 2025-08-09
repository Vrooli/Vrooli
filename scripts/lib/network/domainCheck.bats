#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Load test helpers
load "../../__test/helpers/bats-support/load"
load "../../__test/helpers/bats-assert/load"

# Load mocks
load "../../__test/fixtures/mocks/http.sh"
load "../../__test/fixtures/mocks/logs.sh"
load "../../__test/fixtures/mocks/dig.sh"

# Script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/domainCheck.sh"

# Source necessary dependencies that the script needs
# Note: These need to be sourced here so functions are available
source "${BATS_TEST_DIRNAME}/../utils/var.sh"
source "${var_LOG_FILE}"
source "${var_EXIT_CODES_FILE}"
source "${var_FLOW_FILE}"

setup() {
    # Initialize mocks
    mock::http::reset
    mock::dig::reset
    
    # Set up test environment
    export SITE_IP="192.168.1.100"
    export API_URL="http://example.com:8080/api"
    export LOCATION=""
    
    # Configure HTTP mock mode for normal operation
    export HTTP_MOCK_MODE="normal"
}

teardown() {
    # Clean up mocks
    mock::http::reset || true
    mock::dig::reset || true
    # Unset environment variables
    unset SITE_IP API_URL LOCATION HTTP_MOCK_MODE
}

# Tests for extract_domain function
@test "extract_domain removes protocol and path from URL" {
    source "$SCRIPT_PATH"
    
    run domainCheck::extract_domain "http://example.com/api/v1"
    assert_success
    assert_output "example.com"
}

@test "extract_domain handles HTTPS URLs" {
    source "$SCRIPT_PATH"
    
    run domainCheck::extract_domain "https://secure.example.com/path"
    assert_success
    assert_output "secure.example.com"
}

@test "extract_domain removes port numbers" {
    source "$SCRIPT_PATH"
    
    run domainCheck::extract_domain "http://example.com:8080/api"
    assert_success
    assert_output "example.com"
}

@test "extract_domain handles URLs without protocol" {
    source "$SCRIPT_PATH"
    
    run domainCheck::extract_domain "example.com/path"
    assert_success
    assert_output "example.com"
}

# Tests for validate_ip function
@test "validate_ip accepts valid IPv4 addresses" {
    source "$SCRIPT_PATH"
    
    run domainCheck::validate_ip "192.168.1.1"
    assert_success
}

@test "validate_ip accepts localhost" {
    source "$SCRIPT_PATH"
    
    run domainCheck::validate_ip "localhost"
    assert_success
}

@test "validate_ip rejects invalid IP formats" {
    source "$SCRIPT_PATH"
    
    run domainCheck::validate_ip "invalid.ip"
    assert_failure
    assert_output --partial "Invalid IP address format"
}

@test "validate_ip accepts basic IPv6 addresses" {
    source "$SCRIPT_PATH"
    
    run domainCheck::validate_ip "2001:db8:85a3:0:0:8a2e:370:7334"
    assert_success
}

# Tests for validate_url function  
@test "validate_url accepts HTTP URLs" {
    source "$SCRIPT_PATH"
    
    run domainCheck::validate_url "http://example.com"
    assert_success
}

@test "validate_url accepts HTTPS URLs" {
    source "$SCRIPT_PATH"
    
    run domainCheck::validate_url "https://example.com"
    assert_success
}

@test "validate_url rejects URLs without protocol" {
    source "$SCRIPT_PATH"
    
    run domainCheck::validate_url "example.com"
    assert_failure
    assert_output --partial "Invalid URL format"
}

# Tests for get_domain_ip function (mocked)
@test "get_domain_ip returns IP addresses for domain" {
    source "$SCRIPT_PATH"
    
    # Configure DNS records using the official mock
    mock::dig::set_record "example.com" "A" "192.168.1.100"
    mock::dig::set_record "example.com" "AAAA" "2001:db8::1"
    
    run domainCheck::get_domain_ip "example.com"
    assert_success
    assert_line "192.168.1.100"
    assert_line "2001:db8::1"
}

@test "get_domain_ip fails when domain cannot be resolved" {
    source "$SCRIPT_PATH"
    
    # Don't configure any records for this domain - mock will return empty
    
    run domainCheck::get_domain_ip "nonexistent.domain"
    assert_failure
}

# Tests for get_current_ip function (mocked)
@test "get_current_ip returns current IP addresses" {
    source "$SCRIPT_PATH"
    
    # Mock curl to return test IPs for both IPv4 and IPv6 requests
    # Note: The http mock will handle the curl calls
    curl() {
        case "$*" in
            *"-4"*)
                echo "203.0.113.1"
                ;;
            *"-6"*)
                echo "2001:db8::1"
                ;;
        esac
    }
    export -f curl
    
    run domainCheck::get_current_ip
    assert_success
    assert_line "203.0.113.1"
    assert_line "2001:db8::1"
}

@test "get_current_ip fails when no connection available" {
    source "$SCRIPT_PATH"
    
    # Mock curl to return empty results
    curl() {
        return 0
    }
    export -f curl
    
    run domainCheck::get_current_ip
    assert_failure
    assert_output --partial "Failed to retrieve current IP"
}

# Tests for is_location_valid function
@test "is_location_valid accepts 'local'" {
    source "$SCRIPT_PATH"
    
    run domainCheck::is_location_valid "local"
    assert_success
    assert_output "Location is valid: local"
}

@test "is_location_valid accepts 'remote'" {
    source "$SCRIPT_PATH"
    
    run domainCheck::is_location_valid "remote"
    assert_success
    assert_output "Location is valid: remote"
}

@test "is_location_valid rejects invalid values" {
    source "$SCRIPT_PATH"
    
    run domainCheck::is_location_valid "invalid"
    assert_failure
}

# Tests for check_location function (integration tests with mocks)
@test "check_location validates required parameters" {
    source "$SCRIPT_PATH"
    
    # Test missing SITE_IP - unset env variable and pass empty string
    unset SITE_IP
    run domainCheck::check_location "" "http://example.com"
    assert_failure
    assert_output --partial "Required parameter: SITE_IP"
    
    # Test missing SERVER_URL
    export SITE_IP="192.168.1.100"
    unset API_URL
    run domainCheck::check_location "192.168.1.100" ""
    assert_failure
    assert_output --partial "Required parameter: SERVER_URL"
}

@test "check_location validates URL format" {
    source "$SCRIPT_PATH"
    
    run domainCheck::check_location "192.168.1.100" "invalid-url"
    assert_failure
    assert_output --partial "Invalid URL format"
}

@test "check_location detects remote server when IPs match" {
    source "$SCRIPT_PATH"
    
    # Configure DNS records using the official mock
    mock::dig::set_record "example.com" "A" "192.168.1.100"
    
    # Mock current IP to be same as SITE_IP
    curl() {
        case "$*" in
            *"-4"*)
                echo "192.168.1.100"
                ;;
            *"-6"*)
                echo ""
                ;;
        esac
    }
    export -f curl
    
    run domainCheck::check_location "192.168.1.100" "http://example.com"
    assert_success
    # The output should be "remote" since current IP matches site IP (we ARE the remote server)
    assert_output --partial "remote"
}

@test "check_location detects local server when current IP differs from site IP" {
    source "$SCRIPT_PATH"
    
    # Configure DNS records using the official mock
    mock::dig::set_record "example.com" "A" "192.168.1.100"
    
    # Mock current IP to be different from SITE_IP
    curl() {
        case "$*" in
            *"-4"*)
                echo "203.0.113.1"
                ;;
            *"-6"*)
                echo ""
                ;;
        esac
    }
    export -f curl
    
    run domainCheck::check_location "192.168.1.100" "http://example.com"
    assert_success
    # The output should be "local" since current IP differs from site IP (we're running locally)
    assert_output --partial "local"
}

@test "check_location handles localhost normalization" {
    source "$SCRIPT_PATH"
    
    # Configure DNS records using the official mock
    mock::dig::set_record "localhost" "A" "127.0.0.1"
    
    # Mock current IP
    curl() {
        case "$*" in
            *"-4"*)
                echo "203.0.113.1"
                ;;
            *"-6"*)
                echo ""
                ;;
        esac
    }
    export -f curl
    
    run domainCheck::check_location "localhost" "http://localhost:8080"
    assert_success
    assert_line --partial "127.0.0.1"
}

# Tests for check_location_if_not_set function
@test "check_location_if_not_set uses provided location when valid" {
    source "$SCRIPT_PATH"
    
    run domainCheck::check_location_if_not_set "local"
    assert_success
}

@test "check_location_if_not_set detects location when not provided" {
    source "$SCRIPT_PATH"
    
    # Set up environment for location detection
    export SITE_IP="192.168.1.100"
    export API_URL="http://example.com"
    
    # Mock the check_location function to return "remote"
    domainCheck::check_location() {
        echo "Detecting..."
        echo "remote"
    }
    export -f domainCheck::check_location
    
    run domainCheck::check_location_if_not_set ""
    assert_success
    assert_output --partial "Detecting server location"
}

@test "check_location_if_not_set uses LOCATION env var when set" {
    source "$SCRIPT_PATH"
    
    # Set LOCATION environment variable
    export LOCATION="local"
    
    run domainCheck::check_location_if_not_set ""
    assert_success
}

@test "check_location fails when domain cannot be resolved" {
    source "$SCRIPT_PATH"
    
    # Don't configure any DNS records - mock will return empty
    
    run check_location "192.168.1.100" "http://nonexistent.example.com"
    assert_failure
    assert_output --partial "Failed to resolve domain"
}

@test "check_location validates IP address format" {
    source "$SCRIPT_PATH"
    
    run domainCheck::check_location "invalid.ip.format" "http://example.com"
    assert_failure
    assert_output --partial "Invalid IP address format"
}

@test "validate_ip accepts compressed IPv6 addresses" {
    source "$SCRIPT_PATH"
    
    run domainCheck::validate_ip "::1"
    assert_success
    
    run domainCheck::validate_ip "fe80::"
    assert_success
    
    run domainCheck::validate_ip "2001:db8::8a2e:370:7334"
    assert_success
}

@test "extract_domain handles complex URLs with query strings" {
    source "$SCRIPT_PATH"
    
    run domainCheck::extract_domain "https://api.example.com:8443/v1/endpoint?param=value&foo=bar#section"
    assert_success
    assert_output "api.example.com"
}

@test "extract_domain handles URLs with authentication" {
    source "$SCRIPT_PATH"
    
    run domainCheck::extract_domain "http://user:pass@example.com/path"
    assert_success
    assert_output "example.com"
}

@test "get_domain_ip handles IPv4-only domains" {
    source "$SCRIPT_PATH"
    
    # Configure only IPv4 record using the official mock
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    
    run domainCheck::get_domain_ip "example.com"
    assert_success
    assert_output "192.0.2.1"
}

@test "get_domain_ip handles IPv6-only domains" {
    source "$SCRIPT_PATH"
    
    # Configure only IPv6 record using the official mock
    mock::dig::set_record "example.com" "AAAA" "2001:db8::1"
    
    run domainCheck::get_domain_ip "example.com"
    assert_success
    assert_output "2001:db8::1"
}

@test "check_location continues with error when SITE_IP doesn't match domain" {
    source "$SCRIPT_PATH"
    
    # Configure DNS record that doesn't match SITE_IP using the official mock
    mock::dig::set_record "example.com" "A" "10.0.0.1"
    
    # Mock current IP
    curl() {
        case "$*" in
            *"-4"*)
                echo "203.0.113.1"
                ;;
            *"-6"*)
                echo ""
                ;;
        esac
    }
    export -f curl
    
    run domainCheck::check_location "192.168.1.100" "http://example.com"
    assert_failure
    assert_output --partial "SITE_IP does not point to the server"
}