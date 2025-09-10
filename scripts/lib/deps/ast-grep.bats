#!/usr/bin/env bats
# Test ast-grep installation and basic functionality

setup() {
    # Load required libraries
    load "../../../__test/test_helper.bash"
    
    # Ensure ast-grep script is sourced
    source "${MAIN_SCRIPT_DIR}/lib/deps/ast-grep.sh"
}

@test "ast-grep installation function exists" {
    type ast_grep::install
}

@test "ast-grep verification function exists" {
    type ast_grep::verify  
}

@test "ast-grep test function exists" {
    type ast_grep::test
}

@test "ast-grep can be installed" {
    # Skip if already installed
    if command -v ast-grep >/dev/null 2>&1; then
        skip "ast-grep already installed"
    fi
    
    # Run installation
    run ast_grep::install
    [ "$status" -eq 0 ]
}

@test "ast-grep is available after installation" {
    run command -v ast-grep
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ast-grep" ]]
}

@test "ast-grep responds to --version" {
    run ast-grep --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ast-grep" ]]
}

@test "ast-grep verification passes" {
    run ast_grep::verify
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ast-grep is working" ]]
}

@test "ast-grep can match simple JavaScript patterns" {
    # Create a temporary test file
    test_file=$(mktemp --suffix=.js)
    cat > "$test_file" << 'EOF'
function testFunction() {
    console.log("Hello, world!");
    return true;
}

const arrowFunction = () => {
    return "test";
};
EOF
    
    # Test pattern matching
    run ast-grep --pattern 'function $NAME() { $$$ }' "$test_file"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "testFunction" ]]
    
    # Clean up
    rm -f "$test_file"
}

@test "ast-grep can match Go error patterns" {
    # Create a temporary test file
    test_file=$(mktemp --suffix=.go)
    cat > "$test_file" << 'EOF'
package main

func example() error {
    err := doSomething()
    if err != nil {
        return err
    }
    return nil
}
EOF
    
    # Test Go error pattern matching
    run ast-grep --lang go --pattern 'if err != nil { $$$ }' "$test_file"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "return err" ]]
    
    # Clean up
    rm -f "$test_file"
}

@test "ast-grep test function passes" {
    run ast_grep::test
    [ "$status" -eq 0 ]
    [[ "$output" =~ "pattern matching test passed" ]]
}