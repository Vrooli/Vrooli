# Dig Mock Implementation Plan

## Overview
This document outlines the plan for implementing a proper `dig` mock for the Vrooli test infrastructure. The mock will provide consistent, controllable DNS query responses for testing network-related functionality.

## Current Usage Analysis
Based on codebase analysis, `dig` is currently used in:
- `scripts/lib/network/domainCheck.sh` - For DNS resolution of A and AAAA records
  - Uses flags: `+short`, `+time=5`, `+tries=1`
  - Queries: A records (IPv4) and AAAA records (IPv6)

## Design Goals
1. **Consistency**: Provide predictable responses for testing
2. **Flexibility**: Support various DNS record types and query options
3. **State Management**: Allow tests to configure expected responses
4. **Error Simulation**: Support timeout, NXDOMAIN, and other DNS failures
5. **Compatibility**: Match real dig command interface closely

## Proposed API Design

### Core Mock Functions
```bash
# Reset/initialize mock state
mock::dig::reset()

# Set DNS record responses
mock::dig::set_record(domain, record_type, value, [ttl])
mock::dig::set_records(domain, record_type, values_array)

# Set DNS query behavior
mock::dig::set_response_time(domain, milliseconds)
mock::dig::set_failure(domain, failure_type)  # timeout, nxdomain, servfail

# Query state management
mock::dig::enable_caching(true/false)
mock::dig::set_nameserver(ip_address)

# Verification helpers
mock::dig::get_query_count(domain, [record_type])
mock::dig::was_queried(domain, record_type)
```

### State Structure
```bash
declare -A MOCK_DIG_RECORDS=()        # domain:type -> "value1,value2,..."
declare -A MOCK_DIG_TTLS=()           # domain:type -> ttl
declare -A MOCK_DIG_FAILURES=()       # domain -> failure_type
declare -A MOCK_DIG_RESPONSE_TIMES=() # domain -> milliseconds
declare -A MOCK_DIG_QUERY_COUNT=()    # domain:type -> count
declare -A MOCK_DIG_OPTIONS=()        # option -> value
```

## Implementation Structure

### File: `scripts/__test/fixtures/mocks/dig.sh`
```bash
#!/usr/bin/env bash
# DNS (dig) Command Mock
# Provides comprehensive DNS query mocking for testing

# Prevent duplicate loading
if [[ "${DIG_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export DIG_MOCK_LOADED="true"

# State management
declare -gA MOCK_DIG_RECORDS=()
declare -gA MOCK_DIG_TTLS=()
declare -gA MOCK_DIG_FAILURES=()
declare -gA MOCK_DIG_RESPONSE_TIMES=()
declare -gA MOCK_DIG_QUERY_COUNT=()
declare -gA MOCK_DIG_OPTIONS=()

# Main dig function override
dig() {
    # Parse arguments
    # Handle various dig options
    # Return configured responses or defaults
}

# Public API functions
mock::dig::reset() { ... }
mock::dig::set_record() { ... }
# ... etc
```

### File: `scripts/__test/fixtures/mocks/dig.bats`
```bash
#!/usr/bin/env bats
# Tests for dig mock functionality

@test "dig mock handles A record queries" { ... }
@test "dig mock handles AAAA record queries" { ... }
@test "dig mock supports +short option" { ... }
@test "dig mock simulates timeouts" { ... }
@test "dig mock tracks query counts" { ... }
# ... comprehensive test coverage
```

## Integration Plan

### Phase 1: Basic Implementation
1. Create `dig.sh` with core functionality
2. Implement A and AAAA record support
3. Support +short, +time, +tries options
4. Add basic state management

### Phase 2: Testing
1. Create `dig.bats` with comprehensive tests
2. Test error conditions and edge cases
3. Verify state persistence across subshells

### Phase 3: Integration
1. Update `domainCheck.bats` to use new mock
2. Remove ad-hoc dig mocking from tests
3. Update any other tests using dig

### Phase 4: Documentation
1. Add usage examples to mock file
2. Update test documentation
3. Create migration guide for existing tests

## Usage Examples

### Basic Usage
```bash
# In test setup
load "../../__test/fixtures/mocks/dig.sh"

setup() {
    mock::dig::reset
    
    # Configure expected DNS responses
    mock::dig::set_record "example.com" "A" "192.0.2.1"
    mock::dig::set_record "example.com" "AAAA" "2001:db8::1"
}

@test "domain resolution works" {
    run dig +short A example.com
    assert_success
    assert_output "192.0.2.1"
}
```

### Error Simulation
```bash
@test "handle DNS timeout" {
    mock::dig::set_failure "slow.example.com" "timeout"
    
    run timeout 2 dig +time=1 A slow.example.com
    assert_failure
}
```

### Multiple Records
```bash
@test "handle multiple A records" {
    local ips=("192.0.2.1" "192.0.2.2" "192.0.2.3")
    mock::dig::set_records "multi.example.com" "A" ips
    
    run dig +short A multi.example.com
    assert_success
    assert_line "192.0.2.1"
    assert_line "192.0.2.2"
    assert_line "192.0.2.3"
}
```

## Benefits
1. **Consistency**: Eliminates flaky tests due to real DNS queries
2. **Speed**: No network calls means faster tests
3. **Coverage**: Can test error conditions easily
4. **Maintainability**: Centralized mock reduces duplication

## Migration Guide
For existing tests using ad-hoc dig mocking:

### Before:
```bash
dig() {
    case "$*" in
        *"A example.com"*)
            echo "192.0.2.1"
            ;;
    esac
}
export -f dig
```

### After:
```bash
load "../../__test/fixtures/mocks/dig.sh"

setup() {
    mock::dig::reset
    mock::dig::set_record "example.com" "A" "192.0.2.1"
}
```

## Timeline
- Phase 1: 2-3 hours (basic implementation)
- Phase 2: 1-2 hours (testing)
- Phase 3: 1 hour (integration)
- Phase 4: 1 hour (documentation)

Total estimated time: 5-7 hours

## Success Criteria
1. All existing dig-dependent tests pass with new mock
2. Mock handles all current dig usage patterns
3. Tests run faster and more reliably
4. Documentation is clear and comprehensive
5. Mock follows existing mock patterns in the project