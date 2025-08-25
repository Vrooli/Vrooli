#!/usr/bin/env bats

# Test file for Lychee link checker installation and functionality

setup() {
    # Load test helpers
    load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
    load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"
    
    # Set up APP_ROOT
    export APP_ROOT
    APP_ROOT="${BATS_TEST_DIRNAME}/../../.."
    
    # Source the lychee installation script
    source "${APP_ROOT}/scripts/lib/deps/lychee.sh"
}

@test "lychee::install function exists" {
    run type lychee::install
    assert_success
}

@test "lychee::verify function exists" {
    run type lychee::verify
    assert_success
}

@test "lychee installation creates executable" {
    # Skip if lychee is already installed system-wide
    if command -v lychee >/dev/null 2>&1; then
        skip "Lychee already installed system-wide"
    fi
    
    # Run installation
    run lychee::install
    
    # Should succeed (or skip if already installed)
    assert_success
    
    # Verify lychee command exists
    run command -v lychee
    assert_success
}

@test "lychee verification works" {
    # Install if not present
    if ! command -v lychee >/dev/null 2>&1; then
        run lychee::install
        assert_success
    fi
    
    # Test verification
    run lychee::verify
    assert_success
    assert_output --partial "Lychee is working"
}

@test "lychee can check version" {
    # Install if not present
    if ! command -v lychee >/dev/null 2>&1; then
        run lychee::install
        assert_success
    fi
    
    # Test version command
    run lychee --version
    assert_success
    assert_output --partial "lychee"
}

@test "lychee can process basic markdown" {
    # Install if not present
    if ! command -v lychee >/dev/null 2>&1; then
        run lychee::install
        assert_success
    fi
    
    # Create temporary markdown file with local link
    temp_file=$(mktemp --suffix=.md)
    echo "# Test" > "$temp_file"
    echo "[Local link](${temp_file})" >> "$temp_file"
    
    # Test lychee on the file (should pass)
    run lychee --offline "$temp_file"
    assert_success
    
    # Clean up
    rm -f "$temp_file"
}

@test "lychee detects broken local links" {
    # Install if not present
    if ! command -v lychee >/dev/null 2>&1; then
        run lychee::install
        assert_success
    fi
    
    # Create temporary markdown file with broken local link
    temp_file=$(mktemp --suffix=.md)
    echo "# Test" > "$temp_file"
    echo "[Broken link](./nonexistent-file.md)" >> "$temp_file"
    
    # Test lychee on the file (should fail due to broken link)
    run lychee --offline "$temp_file"
    assert_failure
    
    # Clean up
    rm -f "$temp_file"
}

@test "lychee handles include-verbatim flag" {
    # Install if not present
    if ! command -v lychee >/dev/null 2>&1; then
        run lychee::install
        assert_success
    fi
    
    # Test that the --include-verbatim flag is recognized
    run lychee --help
    assert_success
    assert_output --partial "include-verbatim"
}