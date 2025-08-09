#!/usr/bin/env bats
# Tests for Runtime Resource Data Injection Engine (runtime-engine.sh)

# Load test setup
# shellcheck disable=SC1091
source "$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)/../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Set up test environment for runtime injection engine
    export TEST_MANIFEST_FILE="${VROOLI_TEST_TMPDIR}/injection-manifest.json"
    export TEST_INVALID_MANIFEST="${VROOLI_TEST_TMPDIR}/invalid-manifest.json"
    
    # Create test injection manifest
    cat > "${TEST_MANIFEST_FILE}" << 'EOF'
{
  "resources": [
    {
      "name": "n8n",
      "inject_script": "scripts/resources/automation/n8n/inject.sh",
      "initialization": {
        "workflows": [
          {
            "name": "test-workflow",
            "file": "/test/workflows/test.json",
            "enabled": true
          }
        ]
      }
    },
    {
      "name": "postgres",
      "initialization": {
        "data": [
          {
            "type": "schema",
            "file": "/test/sql/schema.sql"
          }
        ]
      }
    },
    {
      "name": "empty-resource",
      "initialization": {}
    }
  ]
}
EOF

    # Create invalid JSON manifest for error testing
    cat > "${TEST_INVALID_MANIFEST}" << 'EOF'
{
  "resources": [
    {
      "name": "broken"
      // Invalid JSON comment
    }
  ]
}
EOF

    # Create mock inject script for testing
    export TEST_INJECT_SCRIPT="${VROOLI_TEST_TMPDIR}/mock-inject.sh"
    cat > "${TEST_INJECT_SCRIPT}" << 'EOF'
#!/usr/bin/env bash
# Mock injection script for testing
case "$1" in
    --inject)
        echo "Mock injection executed for: $2"
        exit 0
        ;;
    *)
        echo "Unknown action: $1" >&2
        exit 1
        ;;
esac
EOF
    chmod +x "${TEST_INJECT_SCRIPT}"
}

teardown() {
    vrooli_cleanup_test
}

@test "runtime_engine::inject_from_manifest processes valid manifest" {
    # Source the runtime engine script
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    run runtime_injection::inject_from_manifest "${TEST_MANIFEST_FILE}"
    assert_success
    assert_output --partial "Loading injection manifest"
    assert_output --partial "Found 3 resources to process"
}

@test "runtime_engine::inject_from_manifest fails for non-existent manifest" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    run runtime_injection::inject_from_manifest "/non/existent/manifest.json"
    assert_failure
    assert_output --partial "Injection manifest not found"
}

@test "runtime_engine::inject_from_manifest fails for invalid JSON" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    run runtime_injection::inject_from_manifest "${TEST_INVALID_MANIFEST}"
    assert_failure
    assert_output --partial "Invalid JSON in manifest file"
}

@test "runtime_engine::inject_from_manifest handles empty resources" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    # Create manifest with empty resources array
    local empty_manifest="${VROOLI_TEST_TMPDIR}/empty-manifest.json"
    echo '{"resources": []}' > "${empty_manifest}"
    
    run runtime_injection::inject_from_manifest "${empty_manifest}"
    assert_success
    assert_output --partial "No resources to inject"
}

@test "runtime_engine::inject_resource processes resource with inject script" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    # Create resource info JSON with our mock inject script
    local resource_info=$(cat << EOF
{
  "name": "test-resource",
  "inject_script": "${TEST_INJECT_SCRIPT}",
  "initialization": {
    "workflows": [
      {
        "name": "test-workflow",
        "file": "/test/workflows/test.json"
      }
    ]
  }
}
EOF
)
    
    run runtime_injection::inject_resource "$resource_info"
    assert_success
    assert_output --partial "Processing resource: test-resource"
    assert_output --partial "Injecting data using: ${TEST_INJECT_SCRIPT}"
    assert_output --partial "Successfully injected data for: test-resource"
}

@test "runtime_engine::inject_resource skips resource with no initialization data" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    local resource_info='{"name": "empty-resource", "initialization": {}}'
    
    run runtime_injection::inject_resource "$resource_info"
    assert_success
    assert_output --partial "Processing resource: empty-resource"
    assert_output --partial "No initialization data for resource: empty-resource"
}

@test "runtime_engine::inject_resource fails for non-executable inject script" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    # Create non-executable script
    local non_executable="${VROOLI_TEST_TMPDIR}/non-executable.sh"
    echo "#!/bin/bash" > "${non_executable}"
    # Don't make it executable
    
    local resource_info=$(cat << EOF
{
  "name": "test-resource",
  "inject_script": "${non_executable}",
  "initialization": {"test": "data"}
}
EOF
)
    
    run runtime_injection::inject_resource "$resource_info"
    assert_failure
    assert_output --partial "Injection script not found or not executable"
}

@test "runtime_engine::inject_resource finds inject script dynamically when path not specified" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    # Mock the find command to return our test script
    find() {
        if [[ "$*" == *"inject.sh"* ]]; then
            echo "${TEST_INJECT_SCRIPT}"
        else
            command find "$@"
        fi
    }
    export -f find
    
    local resource_info='{"name": "test-resource", "initialization": {"test": "data"}}'
    
    run runtime_injection::inject_resource "$resource_info"
    assert_success
    assert_output --partial "Successfully injected data for: test-resource"
}

@test "runtime_engine::inject_resource handles relative paths correctly" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    # Create a relative path that should be resolved
    local relative_path="mock-inject.sh"
    
    local resource_info=$(cat << EOF
{
  "name": "test-resource",
  "inject_script": "${relative_path}",
  "initialization": {"test": "data"}
}
EOF
)
    
    # The script should try to resolve the relative path
    run runtime_injection::inject_resource "$resource_info"
    # This will fail because the relative path resolution won't find our script,
    # but we're testing that the path resolution logic is executed
    assert_failure
    assert_output --partial "Injection script not found or not executable"
}

@test "runtime_engine::resource_supports_injection detects supported resources" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    # Mock the find command to simulate finding an inject script
    find() {
        if [[ "$*" == *"test-resource"* && "$*" == *"inject.sh"* ]]; then
            echo "${TEST_INJECT_SCRIPT}"
        else
            command find "$@" 2>/dev/null
        fi
    }
    export -f find
    
    run runtime_injection::resource_supports_injection "test-resource"
    assert_success
}

@test "runtime_engine::resource_supports_injection returns false for unsupported resources" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    # Mock find to return nothing
    find() {
        if [[ "$*" == *"inject.sh"* ]]; then
            return 1
        else
            command find "$@"
        fi
    }
    export -f find
    
    run runtime_injection::resource_supports_injection "unsupported-resource"
    assert_failure
}

@test "runtime_engine::main function requires manifest argument" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    run main
    assert_failure
}

@test "runtime_engine::main function processes valid manifest successfully" {
    source "${BATS_TEST_DIRNAME}/runtime-engine.sh"
    
    # Create a minimal manifest for successful processing
    local simple_manifest="${VROOLI_TEST_TMPDIR}/simple-manifest.json"
    echo '{"resources": []}' > "${simple_manifest}"
    
    run main "${simple_manifest}"
    assert_success
    assert_output --partial "No resources to inject"
}