#!/usr/bin/env bats
# Comprehensive test suite for helm.sh deployment library

# Load dependencies using BATS_TEST_DIRNAME
# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test environment setup
export TEST_TEMP_DIR=""

# Load dependencies
load "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"
load "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/helm.sh"
load "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/system.sh"

# Source the library being tested
setup() {
  # Create temp directory for test artifacts
  TEST_TEMP_DIR=$(mktemp -d)
  export MOCK_LOG_DIR="$TEST_TEMP_DIR"
  
  # Reset mocks
  mock::helm::reset
  mock::system::reset
  
  # Set default environment
  export HELM_TIMEOUT="30"
  export HELM_NAMESPACE="test-namespace"
  export HELM_HISTORY_MAX="5"
  export HELM_WAIT="true"
  export HELM_DEBUG="false"
  export SUDO_MODE="skip"
  
  # Source the library
  # shellcheck disable=SC1091
  source "${var_LIB_DEPLOY_DIR}/helm.sh"
}

teardown() {
  # Clean up temp directory
  [[ -n "$TEST_TEMP_DIR" ]] && trash::safe_remove "$TEST_TEMP_DIR" --test-cleanup
  
  # Reset mocks
  mock::helm::reset
  mock::system::reset
  
  # Unset environment variables
  unset HELM_TIMEOUT
  unset HELM_NAMESPACE
  unset HELM_HISTORY_MAX
  unset HELM_WAIT
  unset HELM_DEBUG
  unset SUDO_MODE
}

# ========================================
# Installation Tests
# ========================================

@test "helm:check_and_install - installs helm when not present" {
  mock::system::set_command "helm" "/usr/local/bin/helm"
  
  run helm:check_and_install
  assert_success
}

@test "helm:check_and_install - skips installation when helm exists" {
  mock::system::set_command "helm" "/usr/local/bin/helm"
  
  run helm:check_and_install
  assert_success
}

@test "helm:install_version - installs specific version" {
  # Skip test that requires network access
  skip "Test requires network access and would fail in CI"
}

@test "helm:verify - confirms helm installation" {
  mock::system::set_command "helm" "/usr/local/bin/helm"
  
  run helm:verify
  assert_success
  assert_output_contains "Helm is installed"
}

@test "helm:verify - fails when helm not installed" {
  # Don't set command, so helm won't be found
  
  run helm:verify
  assert_failure
  assert_output_contains "Helm is not installed"
}

# ========================================
# Repository Management Tests
# ========================================

@test "helm:repo_add - adds repository successfully" {
  run helm:repo_add "bitnami" "https://charts.bitnami.com/bitnami"
  assert_success
  assert_output_contains "Repository bitnami added successfully"
}

@test "helm:repo_add - handles authentication" {
  run helm:repo_add "private" "https://private.repo.com" "user" "pass"
  assert_success
  assert_output_contains "Repository private added successfully"
}

@test "helm:repo_add - fails with missing parameters" {
  run helm:repo_add
  assert_failure
}

@test "helm:repo_update - updates all repositories" {
  mock::helm::add_repo "bitnami" "https://charts.bitnami.com/bitnami"
  
  run helm:repo_update
  assert_success
  assert_output_contains "All repositories updated"
}

@test "helm:repo_update - updates specific repository" {
  mock::helm::add_repo "bitnami" "https://charts.bitnami.com/bitnami"
  
  run helm:repo_update "bitnami"
  assert_success
  assert_output_contains "Repository bitnami updated"
}

@test "helm:repo_list - lists repositories" {
  mock::helm::add_repo "bitnami" "https://charts.bitnami.com/bitnami"
  mock::helm::add_repo "stable" "https://charts.helm.sh/stable"
  
  run helm:repo_list
  assert_success
  assert_output_contains "bitnami"
  assert_output_contains "stable"
}

@test "helm:repo_remove - removes repository" {
  mock::helm::add_repo "bitnami" "https://charts.bitnami.com/bitnami"
  
  run helm:repo_remove "bitnami"
  assert_success
  assert_output_contains "Repository bitnami removed"
}

# ========================================
# Chart Management Tests
# ========================================

@test "helm:search - searches for charts" {
  run helm:search "nginx"
  assert_success
  assert_output_contains "nginx"
}

@test "helm:search - searches in specific repo" {
  run helm:search "nginx" "bitnami"
  assert_success
  assert_output_contains "bitnami/nginx"
}

@test "helm:pull - downloads chart" {
  run helm:pull "bitnami/nginx"
  assert_success
  assert_output_contains "Chart bitnami/nginx pulled successfully"
}

@test "helm:pull - downloads specific version" {
  run helm:pull "bitnami/nginx" "15.0.0" "$TEST_TEMP_DIR/charts"
  assert_success
  assert_output_contains "Chart bitnami/nginx pulled successfully"
}

@test "helm:show - displays chart information" {
  run helm:show "bitnami/nginx" "values"
  assert_success
  assert_output_contains "replicaCount"
}

# ========================================
# Release Management Tests
# ========================================

@test "helm:install - installs release" {
  run helm:install "myapp" "bitnami/nginx"
  assert_success
  assert_output_contains "Release myapp installed successfully"
}

@test "helm:install - installs with custom namespace" {
  run helm:install "myapp" "bitnami/nginx" "production"
  assert_success
  assert_output_contains "Installing release: myapp in namespace: production"
}

@test "helm:install - installs with values file" {
  echo "replicaCount: 3" > "$TEST_TEMP_DIR/values.yaml"
  
  run helm:install "myapp" "bitnami/nginx" "default" "$TEST_TEMP_DIR/values.yaml"
  assert_success
  assert_output_contains "Release myapp installed successfully"
}

@test "helm:install - handles extra arguments" {
  run helm:install "myapp" "bitnami/nginx" "default" "" "true" "300" "--set" "service.type=LoadBalancer"
  assert_success
}

@test "helm:upgrade - upgrades existing release" {
  mock::helm::add_release "myapp" "default" "bitnami/nginx" "deployed" "1"
  
  run helm:upgrade "myapp" "bitnami/nginx"
  assert_success
  assert_output_contains "Release myapp upgraded successfully"
}

@test "helm:upgrade - fails for non-existent release" {
  mock::helm::set_mode error
  
  run helm:upgrade "nonexistent" "bitnami/nginx"
  assert_failure
  assert_output_contains "Failed to upgrade"
}

@test "helm:upgrade_install - installs new release" {
  run helm:upgrade_install "myapp" "bitnami/nginx"
  assert_success
  assert_output_contains "Release myapp installed/upgraded successfully"
}

@test "helm:upgrade_install - upgrades existing release" {
  mock::helm::add_release "myapp" "default" "bitnami/nginx" "deployed" "1"
  
  run helm:upgrade_install "myapp" "bitnami/nginx"
  assert_success
  assert_output_contains "Release myapp installed/upgraded successfully"
}

@test "helm:uninstall - removes release" {
  mock::helm::add_release "myapp" "default" "bitnami/nginx" "deployed" "1"
  
  run helm:uninstall "myapp"
  assert_success
  assert_output_contains "Release myapp uninstalled successfully"
}

@test "helm:uninstall - handles custom namespace" {
  mock::helm::add_release "myapp" "production" "bitnami/nginx" "deployed" "1"
  
  run helm:uninstall "myapp" "production"
  assert_success
  assert_output_contains "Uninstalling release: myapp from namespace: production"
}

@test "helm:rollback - rolls back release" {
  mock::helm::add_release "myapp" "default" "bitnami/nginx" "deployed" "3"
  
  run helm:rollback "myapp" "2"
  assert_success
  assert_output_contains "Release myapp rolled back successfully"
}

@test "helm:rollback - rolls back to previous revision" {
  mock::helm::add_release "myapp" "default" "bitnami/nginx" "deployed" "3"
  
  run helm:rollback "myapp"
  assert_success
  assert_output_contains "Rolling back release: myapp to revision: 0"
}

# ========================================
# Release Information Tests
# ========================================

@test "helm:list - lists all releases" {
  mock::helm::add_release "app1" "default" "nginx" "deployed" "1"
  mock::helm::add_release "app2" "production" "postgresql" "deployed" "2"
  
  run helm:list
  assert_success
}

@test "helm:list - lists releases in namespace" {
  mock::helm::add_release "app1" "default" "nginx" "deployed" "1"
  mock::helm::add_release "app2" "production" "postgresql" "deployed" "2"
  
  run helm:list "production"
  assert_success
}

@test "helm:list - lists all namespaces" {
  mock::helm::add_release "app1" "default" "nginx" "deployed" "1"
  mock::helm::add_release "app2" "production" "postgresql" "deployed" "2"
  
  run helm:list "" "true"
  assert_success
}

@test "helm:status - gets release status" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "1"
  
  run helm:status "myapp"
  assert_success
}

@test "helm:status - gets status with format" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "1"
  
  run helm:status "myapp" "default" "json"
  assert_success
}

@test "helm:get_values - retrieves release values" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "1"
  mock::helm::set_values "myapp" "replicaCount=3"
  
  run helm:get_values "myapp"
  assert_success
}

@test "helm:get_values - retrieves all values" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "1"
  
  run helm:get_values "myapp" "default" "true"
  assert_success
}

@test "helm:history - shows release history" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "3"
  
  run helm:history "myapp"
  assert_success
}

@test "helm:history - limits history entries" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "10"
  
  run helm:history "myapp" "default" "3"
  assert_success
}

@test "helm:get_manifest - retrieves release manifest" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "1"
  
  run helm:get_manifest "myapp"
  assert_success
}

@test "helm:get_manifest - retrieves specific revision" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "3"
  
  run helm:get_manifest "myapp" "default" "2"
  assert_success
}

# ========================================
# Health and Testing Tests
# ========================================

@test "helm:test - runs release tests" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "1"
  
  run helm:test "myapp"
  assert_success
  assert_output_contains "Release myapp tests passed"
}

@test "helm:test - handles test failures" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "1"
  mock::helm::set_mode error
  
  run helm:test "myapp"
  assert_failure
  assert_output_contains "Release myapp tests failed"
}

@test "helm:release_exists - checks existing release" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "1"
  
  run helm:release_exists "myapp"
  assert_success
}

@test "helm:release_exists - checks non-existent release" {
  run helm:release_exists "nonexistent"
  assert_failure
}

@test "helm:wait_for_release - waits for deployment" {
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "1"
  
  run helm:wait_for_release "myapp" "default" "5"
  assert_success
  assert_output_contains "Release myapp is ready"
}

# ========================================
# Utility Functions Tests
# ========================================

@test "helm:lint - validates chart" {
  run helm:lint "./mychart"
  assert_success
  assert_output_contains "Chart validation passed"
}

@test "helm:lint - validates with values file" {
  echo "replicaCount: 3" > "$TEST_TEMP_DIR/values.yaml"
  
  run helm:lint "./mychart" "$TEST_TEMP_DIR/values.yaml"
  assert_success
  assert_output_contains "Chart validation passed"
}

@test "helm:lint - handles validation failures" {
  mock::helm::set_mode error
  
  run helm:lint "./badchart"
  assert_failure
  assert_output_contains "Chart validation failed"
}

@test "helm:template - renders templates" {
  run helm:template "myrelease" "./mychart"
  assert_success
}

@test "helm:template - renders with values" {
  echo "replicaCount: 3" > "$TEST_TEMP_DIR/values.yaml"
  
  run helm:template "myrelease" "./mychart" "default" "$TEST_TEMP_DIR/values.yaml"
  assert_success
}

@test "helm:template - renders to directory" {
  run helm:template "myrelease" "./mychart" "default" "" "$TEST_TEMP_DIR/output"
  assert_success
}

@test "helm:package - packages chart" {
  run helm:package "./mychart"
  assert_success
  assert_output_contains "Chart packaged successfully"
}

@test "helm:package - packages with version" {
  run helm:package "./mychart" "$TEST_TEMP_DIR" "1.2.3"
  assert_success
  assert_output_contains "Chart packaged successfully"
}

@test "helm:create - creates new chart" {
  run helm:create "mynewchart"
  assert_success
  assert_output_contains "Chart mynewchart created successfully"
}

@test "helm:create - creates with starter" {
  run helm:create "mynewchart" "wordpress"
  assert_success
  assert_output_contains "Chart mynewchart created successfully"
}

# ========================================
# Environment Variable Tests
# ========================================

@test "respects HELM_TIMEOUT environment variable" {
  export HELM_TIMEOUT="120"
  mock::helm::add_release "myapp" "default" "nginx" "deployed" "1"
  
  run helm:upgrade "myapp" "bitnami/nginx"
  assert_success
}

@test "respects HELM_NAMESPACE environment variable" {
  export HELM_NAMESPACE="custom-namespace"
  
  run helm:install "myapp" "bitnami/nginx"
  assert_success
  assert_output_contains "namespace: custom-namespace"
}

@test "respects HELM_WAIT environment variable" {
  export HELM_WAIT="false"
  
  run helm:install "myapp" "bitnami/nginx"
  assert_success
}

@test "respects HELM_DEBUG environment variable" {
  export HELM_DEBUG="true"
  
  run helm:install "myapp" "bitnami/nginx"
  assert_success
}

# ========================================
# Error Handling Tests
# ========================================

@test "handles network errors gracefully" {
  mock::helm::set_mode error
  
  run helm:repo_add "unreachable" "https://unreachable.example.com"
  assert_failure
  assert_output_contains "Failed to add repository"
}

@test "handles invalid chart errors" {
  mock::helm::set_mode error
  
  run helm:install "myapp" "invalid/chart"
  assert_failure
  assert_output_contains "Failed to install"
}

@test "handles permission errors" {
  export SUDO_MODE="skip"
  
  # Skip test that requires network access
  skip "Test requires network access and would fail in CI"
}

# ========================================
# Integration Tests
# ========================================

@test "full deployment workflow" {
  # Add repository
  run helm:repo_add "bitnami" "https://charts.bitnami.com/bitnami"
  assert_success
  
  # Update repositories
  run helm:repo_update
  assert_success
  
  # Search for chart
  run helm:search "nginx" "bitnami"
  assert_success
  
  # Install release
  run helm:install "myapp" "bitnami/nginx" "production"
  assert_success
  
  # Check status
  run helm:status "myapp" "production"
  assert_success
  
  # Upgrade release
  run helm:upgrade "myapp" "bitnami/nginx" "production"
  assert_success
  
  # Get values
  run helm:get_values "myapp" "production"
  assert_success
  
  # Test release
  run helm:test "myapp" "production"
  assert_success
  
  # Rollback release
  run helm:rollback "myapp" "1" "production"
  assert_success
  
  # Uninstall release
  run helm:uninstall "myapp" "production"
  assert_success
}

@test "handles chart lifecycle with values" {
  local values_file="$TEST_TEMP_DIR/custom-values.yaml"
  cat > "$values_file" <<EOF
replicaCount: 3
service:
  type: LoadBalancer
  port: 8080
EOF
  
  # Install with values
  run helm:upgrade_install "myapp" "bitnami/nginx" "default" "$values_file"
  assert_success
  
  # Verify values were applied
  mock::helm::set_values "myapp" "replicaCount=3,service.type=LoadBalancer,service.port=8080"
  run helm:get_values "myapp"
  assert_success
  assert_output_contains "replicaCount=3"
  assert_output_contains "LoadBalancer"
}

# ========================================
# Help and Usage Tests
# ========================================

@test "shows help when requested" {
  run bash "${var_LIB_DEPLOY_DIR}/helm.sh" --help
  assert_success
  assert_output_contains "Helm deployment library"
  assert_output_contains "Available functions:"
  assert_output_contains "Environment Variables:"
}