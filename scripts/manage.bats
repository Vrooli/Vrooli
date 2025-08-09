#!/usr/bin/env bats

# Test suite for manage.sh - the universal lifecycle management script

setup() {
    # Set test directory
    export TEST_DIR="$(cd "$BATS_TEST_DIRNAME" && pwd)"
    export MANAGE_SCRIPT="$TEST_DIR/manage.sh"
    
    # Create a temporary test directory
    export TEST_TEMP_DIR="$(mktemp -d)"
    
    # Mock service.json for testing
    export TEST_SERVICE_JSON="$TEST_TEMP_DIR/.vrooli/service.json"
    mkdir -p "$(dirname "$TEST_SERVICE_JSON")"
    
    # Create a basic test service.json
    cat > "$TEST_SERVICE_JSON" << 'EOF'
{
  "version": "1.0.0",
  "lifecycle": {
    "setup": {
      "description": "Setup phase for testing",
      "steps": [
        {"name": "test-setup", "run": "echo 'Running setup'"}
      ]
    },
    "develop": {
      "description": "Development phase",
      "steps": [
        {"name": "test-develop", "run": "echo 'Running develop'"}
      ]
    },
    "test-phase": {
      "description": "Test phase for unit testing",
      "steps": [
        {"name": "test-step", "run": "echo 'Test phase executed'"}
      ]
    }
  }
}
EOF
    
    # Mock lifecycle engine
    export MOCK_ENGINE="$TEST_TEMP_DIR/scripts/lib/lifecycle/engine.sh"
    mkdir -p "$(dirname "$MOCK_ENGINE")"
    cat > "$MOCK_ENGINE" << 'EOF'
#!/usr/bin/env bash
echo "Mock engine executing phase: $1"
echo "Arguments: ${@:2}"
exit 0
EOF
    chmod +x "$MOCK_ENGINE"
    
    # Mock log.sh
    export MOCK_LOG="$TEST_TEMP_DIR/scripts/lib/utils/log.sh"
    mkdir -p "$(dirname "$MOCK_LOG")"
    cat > "$MOCK_LOG" << 'EOF'
#!/usr/bin/env bash
log::info() { echo "[INFO] $*"; }
log::error() { echo "[ERROR] $*" >&2; }
log::warning() { echo "[WARN] $*" >&2; }
log::success() { echo "[SUCCESS] $*"; }
EOF
    
    # Save original PWD and change to test directory
    export ORIGINAL_PWD="$PWD"
    cd "$TEST_TEMP_DIR"
}

teardown() {
    # Return to original directory
    cd "$ORIGINAL_PWD"
    
    # Clean up temporary directory
    if [[ -n "$TEST_TEMP_DIR" ]] && [[ -d "$TEST_TEMP_DIR" ]]; then
        rm -rf "$TEST_TEMP_DIR"
    fi
}

# Helper function to run manage.sh with test environment
run_manage() {
    run bash "$MANAGE_SCRIPT" "$@"
}

# Test: Help flag displays usage information
@test "manage.sh --help displays usage information" {
    run_manage --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Universal Application Management Interface" ]]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "PHASES:" ]]
    [[ "$output" =~ "OPTIONS:" ]]
}

# Test: Short help flag also works
@test "manage.sh -h displays usage information" {
    run_manage -h
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Universal Application Management Interface" ]]
}

# Test: No arguments shows help
@test "manage.sh with no arguments shows help" {
    run_manage
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Universal Application Management Interface" ]]
}

# Test: Version flag shows version information
@test "manage.sh --version shows version information" {
    run_manage --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Application version:" ]]
    [[ "$output" =~ "Manage script: 1.0.0" ]]
    [[ "$output" =~ "Project root:" ]]
}

# Test: Short version flag works
@test "manage.sh -v shows version information" {
    run_manage -v
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Manage script: 1.0.0" ]]
}

# Test: List phases shows available lifecycle phases
@test "manage.sh --list-phases shows available phases" {
    run_manage --list-phases
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Available lifecycle phases:" ]]
    # Check for phases defined in our test service.json
    # Note: The output format may vary with or without jq
}

# Test: --list also works for listing phases
@test "manage.sh --list shows available phases" {
    run_manage --list
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Available lifecycle phases:" ]]
}

# Test: Error when service.json is missing
@test "manage.sh errors when service.json is missing" {
    rm -f "$TEST_SERVICE_JSON"
    run_manage setup
    [ "$status" -eq 1 ]
    [[ "$output" =~ "No service.json found" ]]
    [[ "$output" =~ "does not appear to be a properly configured application" ]]
}

# Test: Error when lifecycle engine is missing
@test "manage.sh errors when lifecycle engine is missing" {
    rm -f "$MOCK_ENGINE"
    run_manage setup
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Lifecycle engine not found" ]]
}

# Test: Detects monorepo context
@test "manage.sh detects monorepo context when packages directory exists" {
    # Create monorepo structure
    mkdir -p "$TEST_TEMP_DIR/packages/server"
    cat > "$TEST_TEMP_DIR/packages/server/package.json" << 'EOF'
{"name": "@vrooli/server"}
EOF
    
    run_manage test-phase
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Detected Vrooli monorepo context" ]]
}

# Test: Detects standalone context
@test "manage.sh detects standalone context when packages directory is missing" {
    # Ensure no packages directory exists
    rm -rf "$TEST_TEMP_DIR/packages"
    
    run_manage test-phase
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Detected standalone application context" ]]
}

# Test: Passes phase to lifecycle engine
@test "manage.sh passes phase to lifecycle engine" {
    run_manage setup
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Executing phase 'setup' via lifecycle engine" ]]
    [[ "$output" =~ "Mock engine executing phase: setup" ]]
}

# Test: Passes arguments to lifecycle engine
@test "manage.sh passes arguments to lifecycle engine" {
    run_manage develop --target docker --detached yes
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Mock engine executing phase: develop" ]]
    [[ "$output" =~ "--target docker --detached yes" ]]
}

# Test: Warns when phase doesn't exist in service.json (with jq)
@test "manage.sh warns about non-existent phase when jq is available" {
    # Only run if jq is available
    if ! command -v jq &> /dev/null; then
        skip "jq not installed"
    fi
    
    run_manage nonexistent-phase
    [ "$status" -eq 0 ]  # Should still try to run
    [[ "$output" =~ "Phase 'nonexistent-phase' not found in service.json" ]]
    [[ "$output" =~ "Attempting to run anyway" ]]
}

# Test: Handles custom phases
@test "manage.sh handles custom phases from service.json" {
    run_manage test-phase
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Executing phase 'test-phase' via lifecycle engine" ]]
    [[ "$output" =~ "Mock engine executing phase: test-phase" ]]
}

# Test: PROJECT_ROOT is exported correctly
@test "manage.sh exports PROJECT_ROOT environment variable" {
    # Create a mock engine that prints PROJECT_ROOT
    cat > "$MOCK_ENGINE" << 'EOF'
#!/usr/bin/env bash
echo "PROJECT_ROOT is: $PROJECT_ROOT"
exit 0
EOF
    chmod +x "$MOCK_ENGINE"
    
    run_manage setup
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PROJECT_ROOT is: $TEST_TEMP_DIR" ]]
}

# Test: VROOLI_CONTEXT is exported
@test "manage.sh exports VROOLI_CONTEXT environment variable" {
    # Create a mock engine that prints VROOLI_CONTEXT
    cat > "$MOCK_ENGINE" << 'EOF'
#!/usr/bin/env bash
echo "VROOLI_CONTEXT is: $VROOLI_CONTEXT"
exit 0
EOF
    chmod +x "$MOCK_ENGINE"
    
    run_manage setup
    [ "$status" -eq 0 ]
    [[ "$output" =~ "VROOLI_CONTEXT is: standalone" ]]
}

# Test: Respects existing VROOLI_CONTEXT
@test "manage.sh respects pre-set VROOLI_CONTEXT" {
    export VROOLI_CONTEXT="custom-context"
    
    # Create a mock engine that prints VROOLI_CONTEXT
    cat > "$MOCK_ENGINE" << 'EOF'
#!/usr/bin/env bash
echo "VROOLI_CONTEXT is: $VROOLI_CONTEXT"
exit 0
EOF
    chmod +x "$MOCK_ENGINE"
    
    run_manage setup
    [ "$status" -eq 0 ]
    [[ "$output" =~ "VROOLI_CONTEXT is: custom-context" ]]
    [[ ! "$output" =~ "Detected" ]]  # Should not detect when already set
}

# Test: List phases without jq (fallback mode)
@test "manage.sh lists phases without jq using fallback" {
    # Create a mock jq that doesn't exist
    export PATH="/tmp/fake-path:$PATH"
    
    run_manage --list-phases
    [ "$status" -eq 0 ]
    [[ "$output" =~ "jq is not installed - showing raw phase names only" ]]
    # Should still show phase names
    [[ "$output" =~ "setup" ]]
    [[ "$output" =~ "develop" ]]
}

# Test: Version shows correct app version from service.json
@test "manage.sh shows app version from service.json" {
    # Update service.json with a specific version
    cat > "$TEST_SERVICE_JSON" << 'EOF'
{
  "version": "2.5.3",
  "lifecycle": {
    "setup": {
      "description": "Setup phase"
    }
  }
}
EOF
    
    run_manage --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Application version: 2.5.3" ]]
}

# Test: Helpful error for missing lifecycle engine in monorepo
@test "manage.sh provides monorepo-specific help when engine missing" {
    # Create monorepo structure
    mkdir -p "$TEST_TEMP_DIR/packages"
    rm -f "$MOCK_ENGINE"
    
    run_manage setup
    [ "$status" -eq 1 ]
    [[ "$output" =~ "This appears to be the Vrooli monorepo" ]]
    [[ "$output" =~ "git restore scripts/lib" ]]
}

# Test: Helpful error for missing lifecycle engine in standalone
@test "manage.sh provides standalone-specific help when engine missing" {
    # Ensure no packages directory
    rm -rf "$TEST_TEMP_DIR/packages"
    rm -f "$MOCK_ENGINE"
    
    run_manage setup
    [ "$status" -eq 1 ]
    [[ "$output" =~ "This appears to be a standalone app" ]]
    [[ "$output" =~ "Ensure scripts/lib was properly copied" ]]
}

# Test: Uses exec to replace process when calling engine
@test "manage.sh uses exec to replace itself with lifecycle engine" {
    # This is hard to test directly, but we can verify the mock engine runs
    # and that manage.sh doesn't continue after exec
    cat > "$MOCK_ENGINE" << 'EOF'
#!/usr/bin/env bash
echo "Engine started"
exit 42  # Unique exit code
EOF
    chmod +x "$MOCK_ENGINE"
    
    run_manage setup
    [ "$status" -eq 42 ]  # Should get engine's exit code
    [[ "$output" =~ "Engine started" ]]
}

# Test: Complex phase with special characters
@test "manage.sh handles phases with hyphens and underscores" {
    # Add a phase with special characters
    cat > "$TEST_SERVICE_JSON" << 'EOF'
{
  "version": "1.0.0",
  "lifecycle": {
    "pre-build_test": {
      "description": "Complex phase name"
    }
  }
}
EOF
    
    run_manage pre-build_test --some-flag value
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Executing phase 'pre-build_test' via lifecycle engine" ]]
}

# Test: Empty lifecycle doesn't crash
@test "manage.sh handles empty lifecycle gracefully" {
    cat > "$TEST_SERVICE_JSON" << 'EOF'
{
  "version": "1.0.0",
  "lifecycle": {}
}
EOF
    
    run_manage --list-phases
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Available lifecycle phases:" ]]
    # Should complete without crashing
}

# Test: Malformed JSON is handled
@test "manage.sh handles malformed service.json gracefully" {
    cat > "$TEST_SERVICE_JSON" << 'EOF'
{
  "version": "1.0.0",
  "lifecycle": {
    invalid json here
  }
}
EOF
    
    if command -v jq &> /dev/null; then
        run_manage --list-phases
        # Should show warning about failed parsing
        [[ "$output" =~ "Failed to parse service.json" ]] || [[ "$output" =~ "Unable to list phases" ]]
    fi
}