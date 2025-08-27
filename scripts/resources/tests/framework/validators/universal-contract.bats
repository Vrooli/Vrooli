#!/usr/bin/env bats
# ============================================================================
# Universal Contract BATS Tests
# ============================================================================
# 
# Unit tests for the universal contract validation framework.
# Tests the test framework itself to ensure it works correctly.
# ============================================================================

setup() {
    # Get directories
    APP_ROOT="${APP_ROOT:-$(builtin cd "${BATS_TEST_DIRNAME}/../../../../../.." && builtin pwd)}"
    TEST_FRAMEWORK_DIR="${APP_ROOT}/scripts/resources/tests/framework/validators"
    
    # Source the framework
    source "${TEST_FRAMEWORK_DIR}/universal-contract.sh"
    
    # Create a temporary test resource for testing
    TEST_TMP_DIR="$(mktemp -d)"
    TEST_RESOURCE_NAME="test-resource"
    TEST_RESOURCE_DIR="$TEST_TMP_DIR/$TEST_RESOURCE_NAME"
    
    # Create basic structure
    mkdir -p "$TEST_RESOURCE_DIR"/{lib,config,test}
}

teardown() {
    # Clean up temporary directory
    [[ -n "$TEST_TMP_DIR" ]] && rm -rf "$TEST_TMP_DIR"
}

# Test initialization
@test "test::init sets resource directory and name" {
    test::init "$TEST_RESOURCE_DIR"
    
    [[ "$TEST_RESOURCE_DIR" == "$TEST_TMP_DIR/$TEST_RESOURCE_NAME" ]]
    [[ "$TEST_RESOURCE_NAME" == "test-resource" ]]
}

# Test smoke tests
@test "test::smoke fails when cli.sh is missing" {
    test::init "$TEST_RESOURCE_DIR"
    
    run test::smoke
    [ "$status" -eq 1 ]
}

@test "test::smoke passes when cli.sh exists and works" {
    test::init "$TEST_RESOURCE_DIR"
    
    # Create a working cli.sh
    cat > "$TEST_RESOURCE_DIR/cli.sh" << 'EOF'
#!/usr/bin/env bash
case "$1" in
    help) echo "Help text"; exit 0 ;;
    status) exit 2 ;;  # Not running
    *) exit 1 ;;
esac
EOF
    chmod +x "$TEST_RESOURCE_DIR/cli.sh"
    
    run test::smoke
    [ "$status" -eq 0 ]
}

# Test file structure validation
@test "test::file_structure detects missing required files" {
    test::init "$TEST_RESOURCE_DIR"
    
    run test::file_structure
    [ "$status" -eq 1 ]
    
    # Should have failures for missing files
    [[ ${#TEST_FAILURES[@]} -gt 0 ]]
}

@test "test::file_structure passes with all required files" {
    test::init "$TEST_RESOURCE_DIR"
    
    # Create all required files
    touch "$TEST_RESOURCE_DIR/cli.sh"
    touch "$TEST_RESOURCE_DIR/lib/core.sh"
    touch "$TEST_RESOURCE_DIR/lib/test.sh"
    touch "$TEST_RESOURCE_DIR/config/defaults.sh"
    
    run test::file_structure
    [ "$status" -eq 0 ]
}

@test "test::file_structure warns about deprecated files" {
    test::init "$TEST_RESOURCE_DIR"
    
    # Create required files
    touch "$TEST_RESOURCE_DIR/cli.sh"
    touch "$TEST_RESOURCE_DIR/lib/core.sh"
    touch "$TEST_RESOURCE_DIR/lib/test.sh"
    touch "$TEST_RESOURCE_DIR/config/defaults.sh"
    
    # Add deprecated file
    touch "$TEST_RESOURCE_DIR/manage.sh"
    
    run test::file_structure
    [ "$status" -eq 0 ]  # Still passes but with warning
    
    # Check for deprecation warning
    found_warning=false
    for failure in "${TEST_FAILURES[@]}"; do
        if [[ "$failure" == *"WARN"* ]] && [[ "$failure" == *"manage.sh"* ]]; then
            found_warning=true
            break
        fi
    done
    [[ "$found_warning" == "true" ]]
}

# Test command structure validation
@test "test::command_structure detects missing commands" {
    test::init "$TEST_RESOURCE_DIR"
    
    # Create cli.sh without required commands
    cat > "$TEST_RESOURCE_DIR/cli.sh" << 'EOF'
#!/usr/bin/env bash
# Empty CLI
EOF
    
    run test::command_structure
    [ "$status" -eq 1 ]
    
    # Should have failures for missing commands
    [[ ${#TEST_FAILURES[@]} -eq 6 ]]  # All 6 required commands missing
}

@test "test::command_structure passes with all required commands" {
    test::init "$TEST_RESOURCE_DIR"
    
    # Create cli.sh with all required commands
    cat > "$TEST_RESOURCE_DIR/cli.sh" << 'EOF'
#!/usr/bin/env bash
source some_framework.sh
cli::register_command "help" "Show help" "show_help"
cli::register_command "status" "Show status" "show_status"
cli::register_command "logs" "Show logs" "show_logs"
cli::register_command "manage" "Manage resource" "manage_resource"
cli::register_command "test" "Run tests" "run_tests"
cli::register_command "content" "Manage content" "manage_content"
EOF
    
    run test::command_structure
    [ "$status" -eq 0 ]
}

# Test exit code validation
@test "test::check_exit_code validates exit codes correctly" {
    test::init "$TEST_RESOURCE_DIR"
    
    # Create cli.sh that returns specific codes
    cat > "$TEST_RESOURCE_DIR/cli.sh" << 'EOF'
#!/usr/bin/env bash
case "$1" in
    success) exit 0 ;;
    error) exit 1 ;;
    skip) exit 2 ;;
    *) exit 99 ;;
esac
EOF
    chmod +x "$TEST_RESOURCE_DIR/cli.sh"
    
    # Test exact match
    run test::check_exit_code "success" "0"
    [ "$status" -eq 0 ]
    
    # Test regex match
    run test::check_exit_code "error" "[01]"
    [ "$status" -eq 0 ]
    
    # Test mismatch
    run test::check_exit_code "skip" "0"
    [ "$status" -eq 1 ]
}

# Test command exists helper
@test "test::command_exists detects registered commands" {
    test::init "$TEST_RESOURCE_DIR"
    
    cat > "$TEST_RESOURCE_DIR/cli.sh" << 'EOF'
#!/usr/bin/env bash
cli::register_command "help" "Show help" "show_help"
cli::register_command "status" "Show status" "show_status"
EOF
    
    run test::command_exists "help"
    [ "$status" -eq 0 ]
    
    run test::command_exists "status"
    [ "$status" -eq 0 ]
    
    run test::command_exists "nonexistent"
    [ "$status" -eq 1 ]
}

# Test unit test detection
@test "test::unit returns 2 when no unit tests exist" {
    test::init "$TEST_RESOURCE_DIR"
    
    run test::unit
    [ "$status" -eq 2 ]
}

@test "test::unit runs test functions from lib/test.sh" {
    test::init "$TEST_RESOURCE_DIR"
    
    # Create lib/test.sh with test functions
    cat > "$TEST_RESOURCE_DIR/lib/test.sh" << 'EOF'
#!/usr/bin/env bash
test_example_pass() {
    return 0
}
test_example_fail() {
    return 1
}
EOF
    
    run test::unit
    [ "$status" -eq 1 ]  # One test fails
    
    # Check that both tests were run
    found_pass=false
    found_fail=false
    for result in "${TEST_RESULTS[@]}"; do
        [[ "$result" == *"test_example_pass"* ]] && found_pass=true
    done
    for failure in "${TEST_FAILURES[@]}"; do
        [[ "$failure" == *"test_example_fail"* ]] && found_fail=true
    done
    
    [[ "$found_pass" == "true" ]]
    [[ "$found_fail" == "true" ]]
}

# Test integration test detection
@test "test::integration returns 2 when no integration tests exist" {
    test::init "$TEST_RESOURCE_DIR"
    
    run test::integration
    [ "$status" -eq 2 ]
}

@test "test::integration runs integration test script" {
    test::init "$TEST_RESOURCE_DIR"
    
    # Create passing integration test
    cat > "$TEST_RESOURCE_DIR/test/integration-test.sh" << 'EOF'
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "$TEST_RESOURCE_DIR/test/integration-test.sh"
    
    run test::integration
    [ "$status" -eq 0 ]
    
    # Create failing integration test
    cat > "$TEST_RESOURCE_DIR/test/integration-test.sh" << 'EOF'
#!/usr/bin/env bash
exit 1
EOF
    
    run test::integration
    [ "$status" -eq 1 ]
}

# Test manage commands validation
@test "test::manage_commands validates manage subcommands" {
    test::init "$TEST_RESOURCE_DIR"
    
    # Create cli.sh with manage subcommands
    cat > "$TEST_RESOURCE_DIR/cli.sh" << 'EOF'
#!/usr/bin/env bash
case "$1" in
    manage)
        case "$2" in
            install|uninstall|start|stop|restart)
                [[ "$3" == "--help" ]] && echo "Help"
                exit 0
                ;;
        esac
        ;;
esac
exit 1
EOF
    chmod +x "$TEST_RESOURCE_DIR/cli.sh"
    
    run test::manage_commands
    [ "$status" -eq 0 ]
}

# Test content commands validation
@test "test::content_commands validates content subcommands" {
    test::init "$TEST_RESOURCE_DIR"
    
    # Create cli.sh with content subcommands
    cat > "$TEST_RESOURCE_DIR/cli.sh" << 'EOF'
#!/usr/bin/env bash
# Mock content commands
cli::register_command "content add" "Add content" "content_add"
cli::register_command "content list" "List content" "content_list"
cli::register_command "content get" "Get content" "content_get"  
cli::register_command "content remove" "Remove content" "content_remove"
EOF
    chmod +x "$TEST_RESOURCE_DIR/cli.sh"
    
    run test::content_commands
    [ "$status" -eq 0 ]
}

# Test the all suite runner
@test "test::all runs all test suites" {
    test::init "$TEST_RESOURCE_DIR"
    
    # Create minimal valid resource
    cat > "$TEST_RESOURCE_DIR/cli.sh" << 'EOF'
#!/usr/bin/env bash
case "$1" in
    help) exit 0 ;;
    status) exit 2 ;;
    *) exit 1 ;;
esac
EOF
    chmod +x "$TEST_RESOURCE_DIR/cli.sh"
    
    # Note: This will fail because we don't have all required files
    # but it should still run all suites
    run test::all
    [ "$status" -eq 1 ]
    
    # Verify that results were collected
    [[ ${#TEST_RESULTS[@]} -gt 0 || ${#TEST_FAILURES[@]} -gt 0 ]]
}