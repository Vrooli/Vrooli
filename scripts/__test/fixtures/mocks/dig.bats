#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Test the dig mock functionality

# Load test helpers
load "../../../__test/helpers/bats-support/load"
load "../../../__test/helpers/bats-assert/load"

# Load the dig mock
load "./dig.sh"

setup() {
    # Reset mock state before each test
    mock::dig::reset
}

teardown() {
    # Clean up after each test
    mock::dig::reset || true
}

# Basic functionality tests
@test "dig mock loads successfully" {
    [[ "${DIG_MOCK_LOADED}" == "true" ]]
}

@test "dig mock resets state properly" {
    # Set some state
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    mock::dig::set_failure "fail.com" "timeout"
    
    # Reset
    mock::dig::reset
    
    # Verify state is cleared
    run dig +short A example.com
    assert_success
    assert_output ""
    
    run dig +short A fail.com
    assert_success
    assert_output ""
}

# A record tests
@test "dig mock handles A record queries" {
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    
    run dig +short A example.com
    assert_success
    assert_output "192.0.2.1"
}

@test "dig mock handles multiple A records" {
    mock::dig::set_records "example.com" "A" "192.0.2.1" "192.0.2.2" "192.0.2.3"
    
    run dig +short A example.com
    assert_success
    assert_line "192.0.2.1"
    assert_line "192.0.2.2"
    assert_line "192.0.2.3"
}

# AAAA record tests
@test "dig mock handles AAAA record queries" {
    mock::dig::set_record "example.com" "AAAA" "2001:db8::1"
    
    run dig +short AAAA example.com
    assert_success
    assert_output "2001:db8::1"
}

@test "dig mock handles multiple AAAA records" {
    mock::dig::set_records "example.com" "AAAA" "2001:db8::1" "2001:db8::2"
    
    run dig +short AAAA example.com
    assert_success
    assert_line "2001:db8::1"
    assert_line "2001:db8::2"
}

# Option handling tests
@test "dig mock supports +short option" {
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    
    # With +short
    run dig +short A example.com
    assert_success
    assert_output "192.0.2.1"
    
    # Without +short (should show full output)
    run dig A example.com
    assert_success
    assert_output --partial "ANSWER SECTION"
    assert_output --partial "example.com"
    assert_output --partial "192.0.2.1"
}

@test "dig mock supports +time option" {
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    
    run dig +short +time=10 A example.com
    assert_success
    assert_output "192.0.2.1"
}

@test "dig mock supports +tries option" {
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    
    run dig +short +tries=3 A example.com
    assert_success
    assert_output "192.0.2.1"
}

@test "dig mock handles combined options" {
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    
    run dig +short +time=5 +tries=1 A example.com
    assert_success
    assert_output "192.0.2.1"
}

# Failure simulation tests
@test "dig mock simulates timeout failure" {
    mock::dig::set_failure "timeout.example.com" "timeout"
    
    run dig +short A timeout.example.com
    assert_failure
    assert_output --partial "connection timed out"
}

@test "dig mock simulates NXDOMAIN" {
    mock::dig::set_failure "nonexistent.example.com" "nxdomain"
    
    run dig A nonexistent.example.com
    assert_success
    assert_output --partial "NXDOMAIN"
}

@test "dig mock simulates SERVFAIL" {
    mock::dig::set_failure "servfail.example.com" "servfail"
    
    run dig A servfail.example.com
    assert_failure
    assert_output --partial "SERVFAIL"
}

# Global mode tests
@test "dig mock respects offline mode" {
    export DIG_MOCK_MODE="offline"
    
    run dig +short A example.com
    assert_failure
    assert_output --partial "connection timed out"
    
    # Reset mode
    export DIG_MOCK_MODE="normal"
}

@test "dig mock respects nxdomain mode" {
    export DIG_MOCK_MODE="nxdomain"
    
    run dig A example.com
    assert_success
    assert_output --partial "NXDOMAIN"
    
    # Reset mode
    export DIG_MOCK_MODE="normal"
}

# Query counting tests
@test "dig mock tracks query count" {
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    
    # Initial count should be 0
    run mock::dig::get_query_count "example.com" "A"
    assert_success
    assert_output "0"
    
    # Make a query
    dig +short A example.com >/dev/null
    
    # Count should be 1
    run mock::dig::get_query_count "example.com" "A"
    assert_success
    assert_output "1"
    
    # Make another query
    dig +short A example.com >/dev/null
    
    # Count should be 2
    run mock::dig::get_query_count "example.com" "A"
    assert_success
    assert_output "2"
}

@test "dig mock tracks different record types separately" {
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    mock::dig::set_record "example.com" "AAAA" "2001:db8::1"
    
    # Query A record
    dig +short A example.com >/dev/null
    dig +short A example.com >/dev/null
    
    # Query AAAA record
    dig +short AAAA example.com >/dev/null
    
    # Check counts
    run mock::dig::get_query_count "example.com" "A"
    assert_success
    assert_output "2"
    
    run mock::dig::get_query_count "example.com" "AAAA"
    assert_success
    assert_output "1"
}

@test "dig mock was_queried helper works" {
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    
    # Should not be queried initially
    run mock::dig::was_queried "example.com" "A"
    assert_failure
    
    # Make a query
    dig +short A example.com >/dev/null
    
    # Should now be queried
    run mock::dig::was_queried "example.com" "A"
    assert_success
}

# TTL tests
@test "dig mock handles custom TTL" {
    mock::dig::set_record "example.com" "A" "192.0.2.1" "3600"
    
    run dig A example.com
    assert_success
    assert_output --partial "3600"
}

# Empty response tests
@test "dig mock returns empty for unconfigured domains" {
    run dig +short A unconfigured.example.com
    assert_success
    assert_output ""
}

@test "dig mock returns empty for unconfigured record types" {
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    
    run dig +short AAAA example.com
    assert_success
    assert_output ""
}

# Other record types
@test "dig mock handles MX records" {
    mock::dig::set_record "example.com" "MX" "10 mail.example.com"
    
    run dig +short MX example.com
    assert_success
    assert_output "10 mail.example.com"
}

@test "dig mock handles TXT records" {
    mock::dig::set_record "example.com" "TXT" "v=spf1 include:_spf.example.com ~all"
    
    run dig +short TXT example.com
    assert_success
    assert_output "v=spf1 include:_spf.example.com ~all"
}

@test "dig mock handles NS records" {
    mock::dig::set_records "example.com" "NS" "ns1.example.com" "ns2.example.com"
    
    run dig +short NS example.com
    assert_success
    assert_line "ns1.example.com"
    assert_line "ns2.example.com"
}

# Nameserver tests
@test "dig mock accepts nameserver specification" {
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    
    run dig @8.8.8.8 +short A example.com
    assert_success
    assert_output "192.0.2.1"
}

# Complex scenario tests
@test "dig mock handles domain with both A and AAAA records" {
    mock::dig::set_record "dual.example.com" "A" "192.0.2.1"
    mock::dig::set_record "dual.example.com" "AAAA" "2001:db8::1"
    
    # Query A record
    run dig +short A dual.example.com
    assert_success
    assert_output "192.0.2.1"
    
    # Query AAAA record
    run dig +short AAAA dual.example.com
    assert_success
    assert_output "2001:db8::1"
}

@test "dig mock preserves state across function calls" {
    # Set record in one function
    set_test_record() {
        mock::dig::set_record "test.example.com" "A" "192.0.2.100"
    }
    
    # Query in another function
    query_test_record() {
        dig +short A test.example.com
    }
    
    set_test_record
    run query_test_record
    assert_success
    assert_output "192.0.2.100"
}

@test "dig mock works in subshells" {
    mock::dig::set_record "subshell.example.com" "A" "192.0.2.200"
    
    # Query in a subshell
    run bash -c 'source ./dig.sh && dig +short A subshell.example.com'
    assert_success
    assert_output "192.0.2.200"
}