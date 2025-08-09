#!/usr/bin/env bats
# Tests for helm.sh - Helm package manager functions

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

# Load mocks
load "${BATS_TEST_DIRNAME}/../../__test/fixtures/mocks/helm"

setup() {
    vrooli_setup_unit_test
    
    # Source the helm utilities
    source "${BATS_TEST_DIRNAME}/helm.sh"
    
    # Reset mocks
    mock::helm::reset
}

teardown() {
    vrooli_cleanup_test
}

@test "helm::is_installed returns correct status" {
    # Test when helm is not installed
    mock::helm::set_installed false
    run helm::is_installed
    assert_failure
    
    # Test when helm is installed
    mock::helm::set_installed true
    run helm::is_installed
    assert_success
}

@test "helm::get_version returns version when installed" {
    mock::helm::set_installed true
    mock::helm::set_version "v3.12.0"
    
    run helm::get_version
    assert_success
    assert_output "v3.12.0"
}

@test "helm::get_version returns empty when not installed" {
    mock::helm::set_installed false
    
    run helm::get_version
    assert_success
    assert_output ""
}

@test "helm::install skips when already installed" {
    mock::helm::set_installed true
    mock::helm::set_version "v3.12.0"
    
    run helm::install
    assert_success
    assert_output --partial "already installed"
}

@test "helm::install downloads and installs helm" {
    mock::helm::set_installed false
    
    # Mock successful installation
    mock::helm::set_install_success true
    
    run helm::install
    assert_success
    assert_output --partial "Installing Helm"
}

@test "helm::add_repo adds helm repository" {
    mock::helm::set_installed true
    
    run helm::add_repo "bitnami" "https://charts.bitnami.com/bitnami"
    assert_success
    assert_output --partial "Adding Helm repository: bitnami"
}

@test "helm::add_repo fails when helm not installed" {
    mock::helm::set_installed false
    
    run helm::add_repo "bitnami" "https://charts.bitnami.com/bitnami"
    assert_failure
    assert_output --partial "not installed"
}

@test "helm::repo_exists checks repository existence" {
    mock::helm::set_installed true
    mock::helm::add_repo "bitnami" "https://charts.bitnami.com/bitnami"
    
    run helm::repo_exists "bitnami"
    assert_success
    
    run helm::repo_exists "nonexistent"
    assert_failure
}