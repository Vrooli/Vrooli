#!/usr/bin/env bats
# Tests for target_matcher.sh - Target string matching utilities

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

setup() {
    vrooli_setup_unit_test
    
    # Source the target matcher utility
    source "${BATS_TEST_DIRNAME}/target_matcher.sh"
}

teardown() {
    vrooli_cleanup_test
}

# Test target_matcher::match_target function
@test "target_matcher::match_target returns native-linux for linux variations" {
    run target_matcher::match_target "linux"
    assert_success
    assert_output "native-linux"
    
    run target_matcher::match_target "l"
    assert_success
    assert_output "native-linux"
    
    run target_matcher::match_target "nl"
    assert_success
    assert_output "native-linux"
    
    run target_matcher::match_target "unix"
    assert_success
    assert_output "native-linux"
    
    run target_matcher::match_target "ubuntu"
    assert_success
    assert_output "native-linux"
    
    run target_matcher::match_target "native-linux"
    assert_success
    assert_output "native-linux"
}

@test "target_matcher::match_target returns native-mac for mac variations" {
    run target_matcher::match_target "mac"
    assert_success
    assert_output "native-mac"
    
    run target_matcher::match_target "m"
    assert_success
    assert_output "native-mac"
    
    run target_matcher::match_target "nm"
    assert_success
    assert_output "native-mac"
    
    run target_matcher::match_target "macos"
    assert_success
    assert_output "native-mac"
    
    run target_matcher::match_target "mac-os"
    assert_success
    assert_output "native-mac"
    
    run target_matcher::match_target "native-mac"
    assert_success
    assert_output "native-mac"
    
    run target_matcher::match_target "native-macos"
    assert_success
    assert_output "native-mac"
}

@test "target_matcher::match_target returns native-win for windows variations" {
    run target_matcher::match_target "windows"
    assert_success
    assert_output "native-win"
    
    run target_matcher::match_target "w"
    assert_success
    assert_output "native-win"
    
    run target_matcher::match_target "nw"
    assert_success
    assert_output "native-win"
    
    run target_matcher::match_target "win"
    assert_success
    assert_output "native-win"
    
    run target_matcher::match_target "windows-os"
    assert_success
    assert_output "native-win"
    
    run target_matcher::match_target "native-win"
    assert_success
    assert_output "native-win"
}

@test "target_matcher::match_target returns docker-only for docker variations" {
    run target_matcher::match_target "docker"
    assert_success
    assert_output "docker-only"
    
    run target_matcher::match_target "d"
    assert_success
    assert_output "docker-only"
    
    run target_matcher::match_target "dc"
    assert_success
    assert_output "docker-only"
    
    run target_matcher::match_target "docker-only"
    assert_success
    assert_output "docker-only"
    
    run target_matcher::match_target "docker-compose"
    assert_success
    assert_output "docker-only"
}

@test "target_matcher::match_target returns k8s-cluster for kubernetes variations" {
    run target_matcher::match_target "kubernetes"
    assert_success
    assert_output "k8s-cluster"
    
    run target_matcher::match_target "k"
    assert_success
    assert_output "k8s-cluster"
    
    run target_matcher::match_target "kc"
    assert_success
    assert_output "k8s-cluster"
    
    run target_matcher::match_target "k8s"
    assert_success
    assert_output "k8s-cluster"
    
    run target_matcher::match_target "cluster"
    assert_success
    assert_output "k8s-cluster"
    
    run target_matcher::match_target "k8s-cluster"
    assert_success
    assert_output "k8s-cluster"
}

@test "target_matcher::match_target is case-insensitive" {
    # Test uppercase variations
    run target_matcher::match_target "LINUX"
    assert_success
    assert_output "native-linux"
    
    run target_matcher::match_target "MAC"
    assert_success
    assert_output "native-mac"
    
    run target_matcher::match_target "WINDOWS"
    assert_success
    assert_output "native-win"
    
    run target_matcher::match_target "DOCKER"
    assert_success
    assert_output "docker-only"
    
    run target_matcher::match_target "K8S"
    assert_success
    assert_output "k8s-cluster"
    
    # Test mixed case variations
    run target_matcher::match_target "Linux"
    assert_success
    assert_output "native-linux"
    
    run target_matcher::match_target "MacOS"
    assert_success
    assert_output "native-mac"
    
    run target_matcher::match_target "Windows"
    assert_success
    assert_output "native-win"
    
    run target_matcher::match_target "Docker"
    assert_success
    assert_output "docker-only"
    
    run target_matcher::match_target "Kubernetes"
    assert_success
    assert_output "k8s-cluster"
}

@test "target_matcher::match_target returns error for invalid targets" {
    run target_matcher::match_target "invalid"
    assert_failure
    assert_output --partial "[ERROR]   Bad --target: invalid"
    
    run target_matcher::match_target "unknown"
    assert_failure
    assert_output --partial "[ERROR]   Bad --target: unknown"
    
    run target_matcher::match_target "xyz"
    assert_failure
    assert_output --partial "[ERROR]   Bad --target: xyz"
    
    run target_matcher::match_target ""
    assert_failure
    assert_output --partial "[ERROR]   Bad --target:"
}

@test "target_matcher::match_target handles special characters safely" {
    run target_matcher::match_target "linux*"
    assert_failure
    assert_output --partial "[ERROR]   Bad --target: linux*"
    
    run target_matcher::match_target "linux!"
    assert_failure
    assert_output --partial "[ERROR]   Bad --target: linux!"
    
    run target_matcher::match_target "linux?"
    assert_failure
    assert_output --partial "[ERROR]   Bad --target: linux?"
}

@test "target_matcher::match_target function is available and exported" {
    # Test that the function is available
    declare -f target_matcher::match_target >/dev/null || { echo "target_matcher::match_target function not available"; return 1; }
}

# Integration test
@test "target_matcher::match_target integration test with all valid targets" {
    local -A expected_targets=(
        ["l"]="native-linux"
        ["linux"]="native-linux"
        ["m"]="native-mac"
        ["mac"]="native-mac"
        ["w"]="native-win"
        ["windows"]="native-win"
        ["d"]="docker-only"
        ["docker"]="docker-only"
        ["k"]="k8s-cluster"
        ["kubernetes"]="k8s-cluster"
    )
    
    for input in "${!expected_targets[@]}"; do
        local expected="${expected_targets[$input]}"
        run target_matcher::match_target "$input"
        assert_success
        assert_output "$expected"
    done
}

# Edge case tests
@test "target_matcher::match_target handles whitespace in input" {
    # Should fail with whitespace (not a valid target format)
    run target_matcher::match_target " linux "
    assert_failure
    assert_output --partial "[ERROR]   Bad --target:  linux "
}

@test "target_matcher::match_target handles numbers in input" {
    run target_matcher::match_target "linux2"
    assert_failure
    assert_output --partial "[ERROR]   Bad --target: linux2"
}