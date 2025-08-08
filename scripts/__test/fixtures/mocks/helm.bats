#!/usr/bin/env bats
# Test suite for Helm mock functions

# Load test helpers
load "${BATS_TEST_DIRNAME}/../setup.bash"
load "${BATS_TEST_DIRNAME}/helm.sh"

setup() {
  # Reset helm mocks before each test
  mock::helm::reset
}

teardown() {
  # Clean up after each test
  mock::helm::reset
}

# ----------------------------
# Version tests
# ----------------------------

@test "helm version returns correct version" {
  run helm version --short
  assert_success
  assert_output --partial "v3.12.0"
}

@test "helm version fails in error mode" {
  mock::helm::set_mode error
  run helm version
  assert_failure
  assert_output --partial "Error: helm not found"
}

# ----------------------------
# Repository tests
# ----------------------------

@test "helm repo add successfully adds repository" {
  run helm repo add bitnami https://charts.bitnami.com/bitnami
  assert_success
  assert_output --partial "has been added to your repositories"
  
  # Verify repo was added
  run helm repo list
  assert_success
  assert_output --partial "bitnami"
}

@test "helm repo update updates repositories" {
  mock::helm::add_repo bitnami https://charts.bitnami.com/bitnami
  mock::helm::add_repo stable https://charts.helm.sh/stable
  
  run helm repo update
  assert_success
  assert_output --partial "Successfully got an update"
  assert_output --partial "bitnami"
  assert_output --partial "stable"
}

@test "helm repo remove removes repository" {
  mock::helm::add_repo bitnami https://charts.bitnami.com/bitnami
  
  run helm repo remove bitnami
  assert_success
  assert_output --partial "has been removed"
  
  # Verify repo was removed
  run helm repo list
  assert_success
  refute_output --partial "bitnami"
}

# ----------------------------
# Search tests
# ----------------------------

@test "helm search repo returns results" {
  run helm search repo nginx
  assert_success
  assert_output --partial "nginx"
  assert_output --partial "CHART VERSION"
}

# ----------------------------
# Install tests
# ----------------------------

@test "helm install creates new release" {
  run helm install myapp bitnami/nginx --namespace production --create-namespace
  assert_success
  assert_output --partial "NAME: myapp"
  assert_output --partial "STATUS: deployed"
  assert_output --partial "NAMESPACE: production"
  
  # Verify release exists
  assert mock::helm::release_exists myapp
}

@test "helm install fails in error mode" {
  mock::helm::set_mode error
  run helm install myapp bitnami/nginx
  assert_failure
  assert_output --partial "Failed to install"
}

# ----------------------------
# Upgrade tests
# ----------------------------

@test "helm upgrade updates existing release" {
  mock::helm::add_release myapp production bitnami/nginx deployed 1
  
  run helm upgrade myapp bitnami/nginx --namespace production
  assert_success
  assert_output --partial "has been upgraded"
  assert_output --partial "REVISION: 2"
}

@test "helm upgrade --install creates release if not exists" {
  run helm upgrade --install myapp bitnami/nginx --namespace production
  assert_success
  assert_output --partial "does not exist. Installing it now"
  
  assert mock::helm::release_exists myapp
}

# ----------------------------
# Uninstall tests
# ----------------------------

@test "helm uninstall removes release" {
  mock::helm::add_release myapp production bitnami/nginx deployed 1
  
  run helm uninstall myapp --namespace production
  assert_success
  assert_output --partial "uninstalled"
  
  # Verify release was removed
  run mock::helm::release_exists myapp
  assert_failure
}

@test "helm uninstall fails for non-existent release" {
  run helm uninstall nonexistent
  assert_failure
  assert_output --partial "not found"
}

# ----------------------------
# List tests
# ----------------------------

@test "helm list shows all releases" {
  mock::helm::add_release app1 default nginx deployed 1
  mock::helm::add_release app2 production postgresql deployed 2
  
  run helm list --all-namespaces
  assert_success
  assert_output --partial "app1"
  assert_output --partial "app2"
}

@test "helm list filters by namespace" {
  mock::helm::add_release app1 default nginx deployed 1
  mock::helm::add_release app2 production postgresql deployed 2
  
  run helm list --namespace production
  assert_success
  assert_output --partial "app2"
  refute_output --partial "app1"
}

# ----------------------------
# Status tests
# ----------------------------

@test "helm status shows release status" {
  mock::helm::add_release myapp production nginx deployed 3
  
  run helm status myapp --namespace production
  assert_success
  assert_output --partial "NAME: myapp"
  assert_output --partial "STATUS: deployed"
  assert_output --partial "REVISION: 3"
}

@test "helm status supports json output" {
  mock::helm::add_release myapp production nginx deployed 3
  
  run helm status myapp --namespace production --output json
  assert_success
  assert_output --partial '"name":"myapp"'
  assert_output --partial '"status":"deployed"'
}

# ----------------------------
# Get values tests
# ----------------------------

@test "helm get values returns release values" {
  mock::helm::add_release myapp default nginx deployed 1
  mock::helm::set_values myapp "replicaCount=3,service.type=LoadBalancer"
  
  run helm get values myapp
  assert_success
  assert_output --partial "replicaCount=3"
  assert_output --partial "service.type=LoadBalancer"
}

# ----------------------------
# History tests
# ----------------------------

@test "helm history shows release revisions" {
  mock::helm::add_release myapp default nginx deployed 3
  mock::helm::set_values myapp "1|2|3"
  
  run helm history myapp
  assert_success
  assert_output --partial "REVISION"
  assert_output --partial "deployed"
}

# ----------------------------
# Test tests
# ----------------------------

@test "helm test runs release tests" {
  mock::helm::add_release myapp default nginx deployed 1
  
  run helm test myapp
  assert_success
  assert_output --partial "TEST SUITE"
  assert_output --partial "Phase:          Succeeded"
}

# ----------------------------
# Lint tests
# ----------------------------

@test "helm lint validates chart" {
  run helm lint ./mychart
  assert_success
  assert_output --partial "1 chart(s) linted, 0 chart(s) failed"
}

@test "helm lint fails in error mode" {
  mock::helm::set_mode error
  run helm lint ./mychart
  assert_failure
  assert_output --partial "1 chart(s) linted, 1 chart(s) failed"
}

# ----------------------------
# Template tests
# ----------------------------

@test "helm template renders chart" {
  run helm template myrelease ./mychart
  assert_success
  assert_output --partial "apiVersion: apps/v1"
  assert_output --partial "kind: Deployment"
}

# ----------------------------
# Package tests
# ----------------------------

@test "helm package creates chart archive" {
  run helm package ./mychart
  assert_success
  assert_output --partial "Successfully packaged"
  assert_output --partial ".tgz"
}

# ----------------------------
# Create tests
# ----------------------------

@test "helm create creates new chart" {
  run helm create mynewchart
  assert_success
  assert_output --partial "Creating mynewchart"
}

# ----------------------------
# Rollback tests
# ----------------------------

@test "helm rollback reverts release" {
  mock::helm::add_release myapp default nginx deployed 3
  
  run helm rollback myapp 2
  assert_success
  assert_output --partial "Rollback was a success"
}