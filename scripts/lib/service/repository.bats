#!/usr/bin/env bats
# repository.bats - Tests for repository helper functions

setup() {
    # Set up test environment
    # BATS_TEST_DIRNAME is already provided by BATS
    export PROJECT_ROOT="${BATS_TEST_DIRNAME}/../../.."
    
    # Source the repository helper
    source "${BATS_TEST_DIRNAME}/repository.sh"
    
    # Create temporary test directory
    export TEST_DIR=$(mktemp -d)
    export TEST_SERVICE_JSON="${TEST_DIR}/service.json"
    
    # Create a test service.json
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "service": {
    "name": "test-service",
    "repository": {
      "type": "git",
      "url": "https://github.com/Vrooli/Vrooli",
      "sshUrl": "git@github.com:Vrooli/Vrooli.git",
      "directory": "/",
      "branch": "main",
      "private": false,
      "mirrors": [
        "https://gitlab.com/Vrooli/Vrooli",
        "https://codeberg.org/Vrooli/Vrooli"
      ],
      "submodules": false,
      "hooks": {
        "postClone": "echo 'Post clone hook'",
        "preBuild": "echo 'Pre build hook'",
        "postUpdate": "echo 'Post update hook'"
      }
    }
  }
}
EOF
    
    # Override the service.json path for testing
    export SERVICE_JSON_PATH="$TEST_SERVICE_JSON"
}

teardown() {
    # Clean up test directory
    if [[ -d "$TEST_DIR" ]]; then
        rm -rf "$TEST_DIR"
    fi
}

@test "repository::read_config - reads repository configuration" {
    run repository::read_config
    [ "$status" -eq 0 ]
    [[ "$output" == *'"type": "git"'* ]]
    [[ "$output" == *'"url": "https://github.com/Vrooli/Vrooli"'* ]]
}

@test "repository::read_config - fails with missing file" {
    export SERVICE_JSON_PATH="/nonexistent/path/service.json"
    run repository::read_config
    [ "$status" -eq 1 ]
    [[ "$output" == *"Service configuration not found"* ]]
}

@test "repository::get_url - returns primary repository URL" {
    run repository::get_url
    [ "$status" -eq 0 ]
    [ "$output" = "https://github.com/Vrooli/Vrooli" ]
}

@test "repository::get_ssh_url - returns SSH URL" {
    run repository::get_ssh_url
    [ "$status" -eq 0 ]
    [ "$output" = "git@github.com:Vrooli/Vrooli.git" ]
}

@test "repository::get_ssh_url - derives SSH URL from HTTPS" {
    # Create service.json without sshUrl
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "service": {
    "repository": {
      "type": "git",
      "url": "https://github.com/TestUser/TestRepo"
    }
  }
}
EOF
    
    run repository::get_ssh_url
    [ "$status" -eq 0 ]
    [ "$output" = "git@github.com:TestUser/TestRepo.git" ]
}

@test "repository::get_branch - returns configured branch" {
    run repository::get_branch
    [ "$status" -eq 0 ]
    [ "$output" = "main" ]
}

@test "repository::get_branch - returns default when not configured" {
    # Create minimal service.json
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "service": {
    "repository": {
      "type": "git",
      "url": "https://github.com/Vrooli/Vrooli"
    }
  }
}
EOF
    
    run repository::get_branch
    [ "$status" -eq 0 ]
    [ "$output" = "main" ]
}

@test "repository::get_mirrors - returns mirror URLs" {
    run repository::get_mirrors
    [ "$status" -eq 0 ]
    [[ "$output" == *"https://gitlab.com/Vrooli/Vrooli"* ]]
    [[ "$output" == *"https://codeberg.org/Vrooli/Vrooli"* ]]
}

@test "repository::is_private - returns false for public repo" {
    run repository::is_private
    [ "$status" -eq 1 ]  # Returns 1 for public (false)
}

@test "repository::is_private - returns true for private repo" {
    # Update service.json to have private: true
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "service": {
    "repository": {
      "type": "git",
      "url": "https://github.com/Vrooli/Vrooli",
      "private": true
    }
  }
}
EOF
    
    run repository::is_private
    [ "$status" -eq 0 ]  # Returns 0 for private (true)
}

@test "repository::get_submodules_config - returns false when disabled" {
    run repository::get_submodules_config
    [ "$status" -eq 0 ]
    [ "$output" = "false" ]
}

@test "repository::get_submodules_config - returns true when enabled" {
    # Update service.json with submodules: true
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "service": {
    "repository": {
      "type": "git",
      "url": "https://github.com/Vrooli/Vrooli",
      "submodules": true
    }
  }
}
EOF
    
    run repository::get_submodules_config
    [ "$status" -eq 0 ]
    [ "$output" = "true" ]
}

@test "repository::get_submodules_config - returns object configuration" {
    # Update service.json with complex submodules config
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "service": {
    "repository": {
      "type": "git",
      "url": "https://github.com/Vrooli/Vrooli",
      "submodules": {
        "enabled": true,
        "recursive": true,
        "shallow": false
      }
    }
  }
}
EOF
    
    run repository::get_submodules_config
    [ "$status" -eq 0 ]
    [[ "$output" == *'"enabled": true'* ]] || [[ "$output" == *'enabled: true'* ]]
    [[ "$output" == *'"recursive": true'* ]] || [[ "$output" == *'recursive: true'* ]]
}

@test "repository::run_hook - executes postClone hook" {
    run repository::run_hook "postClone"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Post clone hook"* ]]
    [[ "$output" == *"hook completed successfully"* ]]
}

@test "repository::run_hook - executes preBuild hook" {
    run repository::run_hook "preBuild"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Pre build hook"* ]]
}

@test "repository::run_hook - executes postUpdate hook" {
    run repository::run_hook "postUpdate"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Post update hook"* ]]
}

@test "repository::run_hook - returns success for undefined hook" {
    run repository::run_hook "nonExistentHook"
    [ "$status" -eq 0 ]
    # log::debug output is only shown when DEBUG=true, so expect empty output
    [[ -z "$output" ]]
}

@test "repository::run_hook - requires hook name" {
    run repository::run_hook
    [ "$status" -eq 1 ]
    [[ "$output" == *"Hook name required"* ]]
}

@test "repository::run_hook - handles failing hook command" {
    # Update service.json with a failing hook
    cat > "$TEST_SERVICE_JSON" <<'EOF'
{
  "service": {
    "repository": {
      "type": "git",
      "url": "https://github.com/Vrooli/Vrooli",
      "hooks": {
        "postClone": "exit 1"
      }
    }
  }
}
EOF
    
    run repository::run_hook "postClone"
    [ "$status" -eq 1 ]
    # Test should pass if hook fails with correct exit code
    # The error message may go to stderr which isn't always captured by BATS run
}

@test "repository::check_access - validates repository accessibility" {
    skip "Requires network access and git"
    run repository::check_access
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]  # May fail without network
}

@test "repository::info - displays repository information" {
    run repository::info
    [ "$status" -eq 0 ]
    [[ "$output" == *"Repository Information"* ]]
    [[ "$output" == *"URL: https://github.com/Vrooli/Vrooli"* ]]
    [[ "$output" == *"SSH URL: git@github.com:Vrooli/Vrooli.git"* ]]
    [[ "$output" == *"Branch: main"* ]]
    [[ "$output" == *"Private: false"* ]]
    [[ "$output" == *"Mirrors:"* ]]
    [[ "$output" == *"Hooks:"* ]]
}

@test "repository::clone - builds correct clone command" {
    skip "Requires network access and git"
    # This would test actual cloning, which requires network
}

@test "repository::update - updates repository" {
    skip "Requires git repository"
    # This would test actual repository update
}

@test "repository::get_mirror - returns accessible mirror" {
    skip "Requires network access"
    # This would test mirror accessibility
}