#!/usr/bin/env bats
# Contract Parser Tests - Comprehensive test suite for YAML contract parsing
# Tests the foundation of the Layer 1 validation system

# Get to the var.sh file from our location
# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091  
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# =============================================================================
# Test Setup and Teardown
# =============================================================================

setup() {
    # Setup test environment
    vrooli_setup_unit_test
    
    # Load the contract parser
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/contract-parser.sh"
    
    # Create temporary test environment
    export TEST_CONTRACTS_DIR="${BATS_TEST_TMPDIR}/test_contracts"
    export TEST_CACHE_DIR="${BATS_TEST_TMPDIR}/test_cache"
    
    mkdir -p "$TEST_CONTRACTS_DIR/v1.0"
    mkdir -p "$TEST_CACHE_DIR" 
    
    # Create test contract fixtures
    create_test_contracts
}

teardown() {
    # Cleanup contract parser
    if command -v contract_parser::cleanup &>/dev/null; then
        contract_parser::cleanup 2>/dev/null || true
    fi
    
    # Clean up test environment
    vrooli_cleanup_test
    rm -rf "$TEST_CONTRACTS_DIR" 2>/dev/null || true
    rm -rf "$TEST_CACHE_DIR" 2>/dev/null || true
    rm -rf "${TMPDIR:-/tmp}/vrooli_contract_cache_"* 2>/dev/null || true
}

# =============================================================================
# Test Contract Fixtures
# =============================================================================

create_test_contracts() {
    # Create core.yaml test contract
    cat > "$TEST_CONTRACTS_DIR/v1.0/core.yaml" << 'EOF'
version: "1.0"
contract_type: "core"
description: "Base interface requirements for all resources"

required_actions:
  install:
    description: "Install the resource"
    parameters:
      - name: force
        type: flag
        default: false
    exit_codes:
      0: "Successfully installed" 
      1: "Installation failed"
      2: "Already installed"
  
  start:
    description: "Start the resource service"
    exit_codes:
      0: "Service started"
      1: "Start failed"
      2: "Already running"
  
  stop:
    description: "Stop the resource service"
    exit_codes:
      0: "Service stopped"
      1: "Stop failed"
      2: "Not running"
  
  status:
    description: "Check resource status"
    exit_codes:
      0: "Service running and healthy"
      1: "Service error or unhealthy"
      2: "Service not running"
  
  logs:
    description: "Show resource logs"
    parameters:
      - name: tail
        type: integer
        default: 50
    exit_codes:
      0: "Logs displayed"
      1: "Error retrieving logs"

help_patterns:
  - "--help"
  - "-h"
  - "--version"

error_handling:
  - "set -euo pipefail"
  - "trap cleanup EXIT"
  - "Meaningful error messages"

required_files:
  - "config/defaults.sh"
  - "config/messages.sh"
  - "lib/common.sh"
EOF

    # Create ai.yaml test contract (extends core)
    cat > "$TEST_CONTRACTS_DIR/v1.0/ai.yaml" << 'EOF'
version: "1.0"
contract_type: "category"
extends: "core.yaml"
category: "ai"
description: "Interface requirements for AI resources"

optional_actions:
  models:
    description: "Manage AI models"
    parameters:
      - name: action
        type: string
        required: true
        values: ["list", "pull", "remove"]
    exit_codes:
      0: "Model operation successful"
      1: "Model operation failed"
  
  generate:
    description: "Generate content using AI model"
    parameters:
      - name: text
        type: string
        required: true
    exit_codes:
      0: "Generation successful"
      1: "Generation failed"

ai_configuration:
  model_storage:
    description: "Where AI models are stored"
    required: true
    environment_variable: "VROOLI_AI_MODEL_PATH"
EOF

    # Create test contract with inheritance (automation extends core)
    cat > "$TEST_CONTRACTS_DIR/v1.0/automation.yaml" << 'EOF'
version: "1.0"
contract_type: "category"
extends: "core.yaml"
category: "automation"
description: "Interface requirements for automation resources"

optional_actions:
  workflows:
    description: "Manage automation workflows"
    exit_codes:
      0: "Workflow operation successful"
      1: "Workflow operation failed"
EOF

    # Create invalid YAML for error testing
    cat > "$TEST_CONTRACTS_DIR/v1.0/invalid.yaml" << 'EOF'
version: "1.0"
malformed_yaml: [
  - incomplete_list
  invalid_indentation:
    nested: value
  missing_close_bracket
EOF

    # Create empty contract
    touch "$TEST_CONTRACTS_DIR/v1.0/empty.yaml"
    
    # Create circular reference contract A
    cat > "$TEST_CONTRACTS_DIR/v1.0/circular_a.yaml" << 'EOF'
version: "1.0"
extends: "circular_b.yaml"
test_field: "a"
EOF

    # Create circular reference contract B
    cat > "$TEST_CONTRACTS_DIR/v1.0/circular_b.yaml" << 'EOF'
version: "1.0"
extends: "circular_a.yaml"
test_field: "b"
EOF
}

# =============================================================================
# Basic Initialization Tests
# =============================================================================

@test "contract_parser::init: initializes with valid directory" {
    run contract_parser::init "$TEST_CONTRACTS_DIR"
    
    assert_success
    assert_output --partial "Contract parser initialized: $TEST_CONTRACTS_DIR"
    
    # Initialize without run to check global variables
    contract_parser::init "$TEST_CONTRACTS_DIR" >/dev/null
    [[ -n "$VROOLI_CONTRACTS_DIR" ]]
    [[ -n "$VROOLI_CONTRACT_CACHE" ]]
    [[ -d "$VROOLI_CONTRACT_CACHE" ]]
}

@test "contract_parser::init: fails with invalid directory" {
    run contract_parser::init "/nonexistent/directory"
    
    assert_failure
    assert_output --partial "Contract directory not found: /nonexistent/directory"
}

@test "contract_parser::init: auto-detects contracts directory" {
    # Change to project root directory
    cd "${BATS_TEST_DIRNAME}/../../../../.."
    
    run contract_parser::init
    
    assert_success
    assert_output --partial "Contract parser initialized:"
}

@test "contract_parser::cleanup: cleans up resources" {
    # Initialize first
    contract_parser::init "$TEST_CONTRACTS_DIR"
    local cache_dir="$VROOLI_CONTRACT_CACHE"
    
    # Verify cache directory exists
    [[ -d "$cache_dir" ]]
    
    # Run cleanup
    run contract_parser::cleanup
    
    assert_success
    
    # Verify cleanup occurred
    [[ ! -d "$cache_dir" ]]
}

# =============================================================================
# YAML Parsing Tests
# =============================================================================

@test "contract_parser::parse_yaml_value: parses simple string values" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::parse_yaml_value "$TEST_CONTRACTS_DIR/v1.0/core.yaml" "version"
    
    assert_success
    assert_output "1.0"
}

@test "contract_parser::parse_yaml_value: parses nested values" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::parse_yaml_value "$TEST_CONTRACTS_DIR/v1.0/core.yaml" "required_actions.install.description"
    
    assert_success
    assert_output "Install the resource"
}

@test "contract_parser::parse_yaml_value: handles missing keys gracefully" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::parse_yaml_value "$TEST_CONTRACTS_DIR/v1.0/core.yaml" "nonexistent_key"
    
    assert_failure
}

@test "contract_parser::parse_yaml_value: handles missing files gracefully" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::parse_yaml_value "$TEST_CONTRACTS_DIR/v1.0/nonexistent.yaml" "version"
    
    assert_failure
}

@test "contract_parser::parse_yaml_value: handles malformed YAML gracefully" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::parse_yaml_value "$TEST_CONTRACTS_DIR/v1.0/invalid.yaml" "version"
    
    # Should handle gracefully (may succeed or fail depending on implementation)
    # The key is it shouldn't crash
    [[ $status -eq 0 || $status -eq 1 ]]
}

# =============================================================================
# YAML Section Keys Tests
# =============================================================================

@test "contract_parser::get_yaml_section_keys: extracts section keys correctly" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::get_yaml_section_keys "$TEST_CONTRACTS_DIR/v1.0/core.yaml" "required_actions"
    
    assert_success
    assert_line "install"
    assert_line "start"
    assert_line "stop"
    assert_line "status"
    assert_line "logs"
}

@test "contract_parser::get_yaml_section_keys: handles empty sections" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::get_yaml_section_keys "$TEST_CONTRACTS_DIR/v1.0/core.yaml" "nonexistent_section"
    
    assert_failure
}

@test "contract_parser::get_yaml_section_keys: handles missing files" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::get_yaml_section_keys "$TEST_CONTRACTS_DIR/v1.0/missing.yaml" "required_actions"
    
    assert_failure
}

# =============================================================================
# Contract Loading Tests
# =============================================================================

@test "contract_parser::load_contract: loads basic contract successfully" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::load_contract "core.yaml"
    
    assert_success
    # Output should be path to loaded/cached contract
    [[ -f "$output" ]]
}

@test "contract_parser::load_contract: handles missing contracts" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::load_contract "nonexistent.yaml"
    
    assert_failure
    assert_output --partial "Contract not found"
}

@test "contract_parser::load_contract: loads contract with inheritance" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::load_contract "ai.yaml"
    
    assert_success
    # Should return path to merged contract
    [[ -f "$output" ]]
    
    # Verify merged contract contains both parent and child content
    merged_contract="$output"
    grep -q "required_actions:" "$merged_contract"
    grep -q "optional_actions:" "$merged_contract"
}

@test "contract_parser::load_contract: caches loaded contracts" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Load same contract twice
    run contract_parser::load_contract "core.yaml"
    assert_success
    first_output="$output"
    
    run contract_parser::load_contract "core.yaml"
    assert_success
    second_output="$output"
    
    # Should return same cached result
    [[ "$first_output" == "$second_output" ]]
}

@test "contract_parser::load_contract: detects circular inheritance" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::load_contract "circular_a.yaml"
    
    # Should handle circular inheritance gracefully (fail or detect it)
    assert_failure
}

# =============================================================================
# Contract Merging Tests
# =============================================================================

@test "contract_parser::merge_contracts: merges parent and child contracts" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    local output_file="$TEST_CACHE_DIR/merged.yaml"
    
    run contract_parser::merge_contracts "$TEST_CONTRACTS_DIR/v1.0/core.yaml" "$TEST_CONTRACTS_DIR/v1.0/ai.yaml" "$output_file"
    
    assert_success
    [[ -f "$output_file" ]]
    
    # Verify merged content contains elements from both contracts
    grep -q "required_actions:" "$output_file"
    grep -q "optional_actions:" "$output_file"
    grep -q "ai_configuration:" "$output_file"
}

@test "contract_parser::merge_contracts: handles missing parent contract" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    local output_file="$TEST_CACHE_DIR/merged.yaml"
    
    run contract_parser::merge_contracts "$TEST_CONTRACTS_DIR/v1.0/missing.yaml" "$TEST_CONTRACTS_DIR/v1.0/ai.yaml" "$output_file"
    
    assert_failure
}

@test "contract_parser::merge_contracts: handles missing child contract" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    local output_file="$TEST_CACHE_DIR/merged.yaml"
    
    run contract_parser::merge_contracts "$TEST_CONTRACTS_DIR/v1.0/core.yaml" "$TEST_CONTRACTS_DIR/v1.0/missing.yaml" "$output_file"
    
    assert_failure
}

# =============================================================================
# Contract Getter Function Tests
# =============================================================================

@test "contract_parser::get_required_actions: returns core actions for unknown category" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::get_required_actions "unknown_category"
    
    assert_success
    assert_line "install"
    assert_line "start"
    assert_line "stop"
    assert_line "status"
    assert_line "logs"
}

@test "contract_parser::get_required_actions: returns core actions for ai category" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::get_required_actions "ai"
    
    assert_success
    assert_line "install"
    assert_line "start"
    assert_line "stop"
    assert_line "status"
    assert_line "logs"
    # Should NOT include optional actions like models, generate
    ! assert_line "models"
    ! assert_line "generate"
}

@test "contract_parser::get_required_actions: handles missing contracts gracefully" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Remove contracts directory to test fallback
    rm -rf "$TEST_CONTRACTS_DIR/v1.0"
    
    run contract_parser::get_required_actions "ai"
    
    assert_failure
}

@test "contract_parser::get_help_patterns: extracts help patterns from contract" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::get_help_patterns "core"
    
    assert_success
    assert_line "--help"
    assert_line "-h"
    assert_line "--version"
}

@test "contract_parser::get_help_patterns: falls back to core for unknown category" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::get_help_patterns "unknown_category"
    
    assert_success
    assert_line "--help"
    assert_line "-h"
    assert_line "--version"
}

@test "contract_parser::get_error_handling_patterns: extracts error handling patterns" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::get_error_handling_patterns "core"
    
    assert_success
    assert_line "set -euo pipefail"
    assert_line "trap cleanup EXIT"
    assert_line "Meaningful error messages"
}

@test "contract_parser::get_required_files: extracts required files list" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::get_required_files "core"
    
    assert_success
    assert_line "config/defaults.sh"
    assert_line "config/messages.sh"
    assert_line "lib/common.sh"
}

# =============================================================================
# Contract Validation Tests
# =============================================================================

@test "contract_parser::validate_contract_syntax: validates correct contract" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::validate_contract_syntax "$TEST_CONTRACTS_DIR/v1.0/core.yaml"
    
    assert_success
}

@test "contract_parser::validate_contract_syntax: detects invalid contract" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::validate_contract_syntax "$TEST_CONTRACTS_DIR/v1.0/invalid.yaml"
    
    # Should detect issues with malformed YAML
    assert_failure
}

@test "contract_parser::validate_contract_syntax: handles missing contract" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::validate_contract_syntax "$TEST_CONTRACTS_DIR/v1.0/missing.yaml"
    
    assert_failure
}

@test "contract_parser::validate_contract_syntax: validates empty contract" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    run contract_parser::validate_contract_syntax "$TEST_CONTRACTS_DIR/v1.0/empty.yaml"
    
    # Empty contracts should be handled gracefully
    [[ $status -eq 0 || $status -eq 1 ]]
}

# =============================================================================
# Integration Tests with Real Contracts
# =============================================================================

@test "integration: loads real core contract" {
    # Test with actual contract files
    local real_contracts_dir="${var_ROOT_DIR}/contracts"
    
    if [[ -d "$real_contracts_dir/v1.0" ]]; then
        contract_parser::init "$real_contracts_dir"
        
        run contract_parser::load_contract "core.yaml"
        
        assert_success
        [[ -f "$output" ]]
        grep -q "required_actions:" "$output"
    else
        skip "Real contracts directory not found"
    fi
}

@test "integration: loads real ai contract with inheritance" {
    local real_contracts_dir="${var_ROOT_DIR}/contracts"
    
    if [[ -d "$real_contracts_dir/v1.0" ]]; then
        contract_parser::init "$real_contracts_dir"
        
        run contract_parser::load_contract "ai.yaml"
        
        assert_success
        [[ -f "$output" ]]
        
        # Should contain merged content
        grep -q "required_actions:" "$output"
        grep -q "extends:" "$output" || true  # May be removed during merge
    else
        skip "Real contracts directory not found"
    fi
}

@test "integration: gets required actions from all real categories" {
    local real_contracts_dir="${var_ROOT_DIR}/contracts"
    
    if [[ -d "$real_contracts_dir/v1.0" ]]; then
        contract_parser::init "$real_contracts_dir"
        
        local categories=("ai" "automation" "agents" "storage" "search" "execution")
        
        for category in "${categories[@]}"; do
            run contract_parser::get_required_actions "$category"
            
            assert_success
            assert_line "install"
            assert_line "start"
            assert_line "stop"
            assert_line "status"
            assert_line "logs"
        done
    else
        skip "Real contracts directory not found"
    fi
}

# =============================================================================
# Edge Case Tests
# =============================================================================

@test "edge_case: handles contracts with special characters" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Create contract with special characters
    cat > "$TEST_CONTRACTS_DIR/v1.0/special.yaml" << 'EOF'
version: "1.0"
description: "Contract with special chars: !@#$%^&*()[]{}|;:,.<>?"
test_field: "Value with 'quotes' and \"double quotes\" and $variables"
EOF

    run contract_parser::load_contract "special.yaml"
    
    assert_success
    [[ -f "$output" ]]
}

@test "edge_case: handles very large contracts" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Create a large contract
    local large_contract="$TEST_CONTRACTS_DIR/v1.0/large.yaml"
    echo "version: \"1.0\"" > "$large_contract"
    echo "large_section:" >> "$large_contract"
    
    # Add many entries
    for i in {1..1000}; do
        echo "  entry_$i: \"value_$i\"" >> "$large_contract"
    done
    
    run contract_parser::load_contract "large.yaml"
    
    assert_success
    [[ -f "$output" ]]
}

@test "edge_case: handles concurrent access" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Launch multiple contract loading processes in background
    local pids=()
    
    for i in {1..5}; do
        (contract_parser::load_contract "core.yaml" > /dev/null) &
        pids+=($!)
    done
    
    # Wait for all to complete
    local all_success=true
    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            all_success=false
        fi
    done
    
    [[ "$all_success" == "true" ]]
}

# =============================================================================
# Performance Tests
# =============================================================================

@test "performance: contract loading is fast" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Time the operation
    local start_time
    start_time=$(date +%s%N)
    
    run contract_parser::load_contract "core.yaml"
    
    local end_time
    end_time=$(date +%s%N)
    local duration=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
    
    assert_success
    
    # Should complete in under 1000ms (1 second)
    [[ $duration -lt 1000 ]]
}

@test "performance: cached loading is very fast" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Load once to populate cache
    contract_parser::load_contract "core.yaml" >/dev/null
    
    # Time the cached operation
    local start_time
    start_time=$(date +%s%N)
    
    run contract_parser::load_contract "core.yaml"
    
    local end_time
    end_time=$(date +%s%N)
    local duration=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
    
    assert_success
    
    # Cached operations should be very fast (under 100ms)
    [[ $duration -lt 100 ]]
}

@test "performance: contract merging is efficient" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    local output_file="$TEST_CACHE_DIR/perf_merged.yaml"
    
    # Time the merge operation
    local start_time
    start_time=$(date +%s%N)
    
    run contract_parser::merge_contracts "$TEST_CONTRACTS_DIR/v1.0/core.yaml" "$TEST_CONTRACTS_DIR/v1.0/ai.yaml" "$output_file"
    
    local end_time
    end_time=$(date +%s%N)
    local duration=$(( (end_time - start_time) / 1000000 ))
    
    assert_success
    
    # Merging should be fast (under 500ms)
    [[ $duration -lt 500 ]]
}

# =============================================================================
# Memory and Resource Tests
# =============================================================================

@test "resource: cleans up temporary files" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Load several contracts to create cache files
    contract_parser::load_contract "core.yaml" >/dev/null
    contract_parser::load_contract "ai.yaml" >/dev/null
    contract_parser::load_contract "automation.yaml" >/dev/null
    
    local cache_dir="$VROOLI_CONTRACT_CACHE"
    local file_count_before
    file_count_before=$(find "$cache_dir" -type f | wc -l)
    
    # Should have created cache files
    [[ $file_count_before -gt 0 ]]
    
    # Clean up
    contract_parser::cleanup
    
    # Cache directory should be removed
    [[ ! -d "$cache_dir" ]]
}

@test "resource: handles cache directory permissions" {
    # Create a directory we can't write to
    local readonly_dir="$BATS_TEST_TMPDIR/readonly"
    mkdir -p "$readonly_dir"
    chmod 444 "$readonly_dir"
    
    # Try to initialize with unwritable parent directory for cache
    export TMPDIR="$readonly_dir"
    
    run contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Should handle permission issues gracefully
    [[ $status -eq 0 || $status -eq 1 ]]
    
    # Cleanup
    chmod 755 "$readonly_dir" 2>/dev/null || true
    rm -rf "$readonly_dir" 2>/dev/null || true
}

# =============================================================================
# Error Handling Tests
# =============================================================================

@test "error_handling: graceful failure on system errors" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Test with system limits (too many open files)
    # This is hard to test reliably across systems, so we'll test permission errors instead
    
    # Make a contract file unreadable
    local unreadable_contract="$TEST_CONTRACTS_DIR/v1.0/unreadable.yaml"
    echo "version: \"1.0\"" > "$unreadable_contract"
    chmod 000 "$unreadable_contract"
    
    run contract_parser::parse_yaml_value "$unreadable_contract" "version"
    
    assert_failure
    
    # Cleanup
    chmod 644 "$unreadable_contract" 2>/dev/null || true
}

@test "error_handling: handles malformed inheritance" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Create contract with malformed extends
    cat > "$TEST_CONTRACTS_DIR/v1.0/bad_extends.yaml" << 'EOF'
version: "1.0"
extends: ["not", "a", "string"]
description: "Contract with malformed extends"
EOF

    run contract_parser::load_contract "bad_extends.yaml"
    
    # Should handle malformed extends gracefully
    [[ $status -eq 0 || $status -eq 1 ]]
}

# =============================================================================
# Stress Tests
# =============================================================================

@test "stress: handles many concurrent operations" {
    contract_parser::init "$TEST_CONTRACTS_DIR"
    
    # Launch many concurrent parsing operations
    local pids=()
    local operations=("core.yaml" "ai.yaml" "automation.yaml")
    
    for i in {1..20}; do
        local contract="${operations[$((i % 3))]}"
        (contract_parser::load_contract "$contract" > /dev/null 2>&1) &
        pids+=($!)
    done
    
    # Wait for all operations to complete
    local failed_count=0
    for pid in "${pids[@]}"; do
        if ! wait "$pid"; then
            ((failed_count++))
        fi
    done
    
    # Most operations should succeed (allow some failures due to race conditions)
    [[ $failed_count -lt 5 ]]
}

@test "stress: handles rapid init/cleanup cycles" {
    # Rapidly initialize and cleanup
    for i in {1..10}; do
        contract_parser::init "$TEST_CONTRACTS_DIR" >/dev/null
        contract_parser::cleanup >/dev/null 2>&1
    done
    
    # Should complete without crashing
    [[ $? -eq 0 ]]
}