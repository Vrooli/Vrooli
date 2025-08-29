#!/usr/bin/env bats
################################################################################
# json.test.bats - Comprehensive test suite for json.sh utilities
# 
# Tests all functions in the unified JSON utility library to ensure
# robust, reliable service.json parsing across the Vrooli codebase.
#
# Run with: bats scripts/lib/utils/json.test.bats
################################################################################

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test setup and teardown
setup() {
    # Load the JSON utilities
    source "$(dirname "$BATS_TEST_FILENAME")/json.sh"
    
    # Create temporary directory for test files
    export TEST_DIR
    TEST_DIR="$(mktemp -d)"
    
    # Enable debug mode for detailed test output
    json::set_debug true
    
    # Create sample service.json files for testing
    create_test_configs
}

teardown() {
    # Clean up temporary files
    [[ -d "$TEST_DIR" ]] && trash::safe_remove "$TEST_DIR" --test-cleanup
    
    # Clear JSON cache
    json::clear_cache
}

################################################################################
# Test Configuration Creation Helpers
################################################################################

create_test_configs() {
    # Minimal valid config
    cat > "$TEST_DIR/minimal.json" << 'EOF'
{
  "service": {
    "name": "test-service",
    "version": "1.0.0"
  },
  "resources": {
    "postgres": {
      "enabled": true,
      "required": true,
      "host": "localhost",
      "port": 5432
    },
    "redis": {
      "enabled": false,
      "required": false,
      "host": "localhost",
      "port": 6379
    },
    "ollama": {
      "enabled": true,
      "required": false,
      "baseUrl": "http://localhost:11434"
    }
  },
  "lifecycle": {
    "setup": {
      "description": "Setup phase",
      "steps": []
    },
    "develop": {
      "description": "Development phase", 
      "steps": []
    }
  },
  "deployment": {
    "testing": {
      "timeout": "30m",
      "ui": {
        "required": true,
        "type": "windmill"
      }
    }
  }
}
EOF

    # Complex config with deep nesting
    cat > "$TEST_DIR/complex.json" << 'EOF'
{
  "service": {
    "name": "complex-service",
    "displayName": "Complex Test Service",
    "version": "2.0.0",
    "maintainers": [
      {
        "name": "Test User",
        "email": "test@example.com"
      }
    ]
  },
  "resources": {
    "postgres": {
      "enabled": true,
      "required": true,
      "version": "16",
      "host": "localhost",
      "port": 5432,
      "database": "testdb",
      "healthCheck": {
        "type": "tcp",
        "port": 5432,
        "intervalMs": 30000
      }
    },
    "redis": {
      "enabled": true,
      "required": true,
      "version": "7",
      "host": "localhost",
      "port": 6379,
      "healthCheck": {
        "type": "tcp", 
        "port": 6379
      }
    },
    "minio": {
      "enabled": false,
      "required": false,
      "endpoint": "localhost:9000"
    },
    "ollama": {
      "enabled": true,
      "required": false,
      "baseUrl": "http://localhost:11434",
      "expectedModels": ["llama3.1:8b"]
    },
    "openrouter": {
      "enabled": false,
      "required": false,
      "baseUrl": "https://openrouter.ai/api/v1"
    },
    "n8n": {
      "enabled": true,
      "required": true,
      "baseUrl": "http://localhost:5678",
      "capabilities": ["workflow", "webhook"]
    },
    "windmill": {
      "enabled": false,
      "required": false,
      "baseUrl": "http://localhost:5681"
    }
  },
  "lifecycle": {
    "setup": {
      "description": "Complex setup phase",
      "steps": [
        {
          "name": "install-deps",
          "run": "pnpm install"
        }
      ]
    },
    "develop": {
      "description": "Development with hot reload",
      "steps": [
        {
          "name": "start-dev",
          "run": "pnpm dev"
        }
      ]
    },
    "build": {
      "description": "Production build",
      "steps": []
    }
  },
  "deployment": {
    "target": "docker",
    "strategy": "rolling-update",
    "testing": {
      "enabled": true,
      "timeout": "45m",
      "ui": {
        "required": true,
        "type": "windmill",
        "url": "http://localhost:3000"
      },
      "performance": {
        "loadTest": {
          "users": 10,
          "duration": "5m"
        }
      }
    },
    "monitoring": {
      "enabled": true,
      "healthChecks": {
        "application": {
          "endpoint": "/health",
          "interval": "30s"
        }
      }
    }
  }
}
EOF

    # Invalid JSON for error testing
    cat > "$TEST_DIR/invalid.json" << 'EOF'
{
  "service": {
    "name": "invalid-service",
    "version": "1.0.0"
  },
  "resources": {
    "storage": {
      "postgres": {
        "enabled": true
        "required": true // Missing comma - invalid JSON
      }
    }
  }
}
EOF

    # Empty file for edge case testing
    touch "$TEST_DIR/empty.json"
    
    # Non-existent file path for testing
    export NON_EXISTENT_PATH="$TEST_DIR/does-not-exist.json"
}

################################################################################
# Configuration Loading Tests
################################################################################

@test "json::find_service_config finds config in current directory" {
    # Create .vrooli directory structure
    mkdir -p "$TEST_DIR/.vrooli"
    cp "$TEST_DIR/minimal.json" "$TEST_DIR/.vrooli/service.json"
    
    # Change to test directory
    pushd "$TEST_DIR" > /dev/null
    
    run json::find_service_config
    [ "$status" -eq 0 ]
    [[ "$output" == *".vrooli/service.json" ]]
    
    popd > /dev/null
}

@test "json::find_service_config respects SERVICE_JSON_PATH environment variable" {
    export SERVICE_JSON_PATH="$TEST_DIR/minimal.json"
    
    run json::find_service_config
    [ "$status" -eq 0 ]
    [[ "$output" == "$TEST_DIR/minimal.json" ]]
    
    unset SERVICE_JSON_PATH
}

@test "json::find_service_config returns error when no config found" {
    # Create a completely isolated temporary directory
    local isolated_dir
    isolated_dir="$(mktemp -d)"
    
    # Ensure we're in the isolated directory without any parent configs
    pushd "$isolated_dir" > /dev/null
    
    # Explicitly unset all environment variables that might point to configs
    unset SERVICE_JSON_PATH SCENARIO_DIR APP_ROOT var_SERVICE_JSON_FILE
    
    run json::find_service_config
    [ "$status" -eq 1 ]
    [[ "$output" == *"No service.json found"* ]]
    
    popd > /dev/null
    trash::safe_remove "$isolated_dir" --test-cleanup
}

@test "json::load_service_config loads valid configuration" {
    run json::load_service_config "$TEST_DIR/minimal.json"
    [ "$status" -eq 0 ]
    
    # Load again to verify caching works (this time not in run subshell)
    json::load_service_config "$TEST_DIR/minimal.json"
    
    # Verify configuration is cached
    cached_path=$(json::get_cached_path)
    [[ "$cached_path" == "$TEST_DIR/minimal.json" ]]
}

@test "json::load_service_config handles non-existent file" {
    run json::load_service_config "$NON_EXISTENT_PATH"
    [ "$status" -eq 1 ]
    [[ "$output" == *"not found"* ]]
}

@test "json::load_service_config handles invalid JSON" {
    run json::load_service_config "$TEST_DIR/invalid.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Invalid JSON syntax"* ]]
}

@test "json::load_service_config handles empty file" {
    run json::load_service_config "$TEST_DIR/empty.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"empty"* ]]
}

@test "json::load_service_config uses caching for repeated calls" {
    # First load
    json::load_service_config "$TEST_DIR/minimal.json"
    first_load_time=$(date +%s%N)
    
    # Second load should be faster (cached)
    json::load_service_config "$TEST_DIR/minimal.json"
    second_load_time=$(date +%s%N)
    
    # Cache should be used (same path)
    cached_path=$(json::get_cached_path)
    [[ "$cached_path" == "$TEST_DIR/minimal.json" ]]
}

################################################################################
# Value Extraction Tests
################################################################################

@test "json::get_value extracts simple values" {
    run json::get_value '.service.name' '' "$TEST_DIR/minimal.json"
    [ "$status" -eq 0 ]
    [[ "$output" == "test-service" ]]
}

@test "json::get_value extracts nested values" {
    run json::get_value '.resources.postgres.enabled' '' "$TEST_DIR/minimal.json"
    [ "$status" -eq 0 ]
    [[ "$output" == "true" ]]
}

@test "json::get_value returns default for non-existent path" {
    run json::get_value '.non.existent.path' 'default-value' "$TEST_DIR/minimal.json"
    [ "$status" -eq 0 ]
    [[ "$output" == "default-value" ]]
}

@test "json::get_value handles array indices" {
    run json::get_value '.service.maintainers[0].name' '' "$TEST_DIR/complex.json"
    [ "$status" -eq 0 ]
    [[ "$output" == "Test User" ]]
}

@test "json::get_value handles complex nested paths" {
    run json::get_value '.deployment.testing.ui.type' '' "$TEST_DIR/complex.json"
    [ "$status" -eq 0 ]
    [[ "$output" == "windmill" ]]
}

@test "json::path_exists returns true for existing paths" {
    run json::path_exists '.service.name' "$TEST_DIR/minimal.json"
    [ "$status" -eq 0 ]
}

@test "json::path_exists returns false for non-existent paths" {
    run json::path_exists '.non.existent.path' "$TEST_DIR/minimal.json"
    [ "$status" -eq 1 ]
}

################################################################################
# Resource Management Tests
################################################################################

@test "json::get_enabled_resources returns all enabled resources" {
    json::load_service_config "$TEST_DIR/complex.json"
    
    run json::get_enabled_resources
    [ "$status" -eq 0 ]
    [[ "$output" == *"postgres"* ]]
    [[ "$output" == *"redis"* ]]
    [[ "$output" == *"ollama"* ]]
    [[ "$output" == *"n8n"* ]]
}

@test "json::get_enabled_resources filters by pattern" {
    json::load_service_config "$TEST_DIR/complex.json"
    
    run json::get_enabled_resources 'postgres'
    [ "$status" -eq 0 ]
    [[ "$output" == *"postgres"* ]]
    [[ "$output" != *"redis"* ]]
    [[ "$output" != *"ollama"* ]]
}

@test "json::get_required_resources returns only required resources" {
    json::load_service_config "$TEST_DIR/complex.json"
    
    run json::get_required_resources
    [ "$status" -eq 0 ]
    [[ "$output" == *"postgres"* ]]
    [[ "$output" == *"redis"* ]]
    [[ "$output" == *"n8n"* ]]
    [[ "$output" != *"minio"* ]]  # minio is not required
}

@test "json::get_resource_config returns resource configuration" {
    run json::get_resource_config 'postgres' "$TEST_DIR/complex.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"localhost"* ]]
    [[ "$output" == *"5432"* ]]
    [[ "$output" == *"enabled"* ]]
}

@test "json::get_resource_config handles non-existent resources" {
    run json::get_resource_config 'nonexistent' "$TEST_DIR/minimal.json"
    [ "$status" -eq 0 ]
    [[ "$output" == "{}" ]]
}

################################################################################
# Lifecycle Configuration Tests
################################################################################

@test "json::get_lifecycle_phase returns phase configuration" {
    run json::get_lifecycle_phase 'setup' "$TEST_DIR/complex.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Complex setup phase"* ]]
    [[ "$output" == *"steps"* ]]
}

@test "json::get_lifecycle_phase handles non-existent phase" {
    run json::get_lifecycle_phase 'nonexistent' "$TEST_DIR/minimal.json"
    [ "$status" -eq 0 ]
    [[ "$output" == "{}" ]]
}

@test "json::list_lifecycle_phases returns all available phases" {
    run json::list_lifecycle_phases "$TEST_DIR/complex.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"setup"* ]]
    [[ "$output" == *"develop"* ]]
    [[ "$output" == *"build"* ]]
}

################################################################################
# Deployment Configuration Tests
################################################################################

@test "json::get_deployment_config extracts deployment values" {
    run json::get_deployment_config 'testing.timeout' '' "$TEST_DIR/complex.json"
    [ "$status" -eq 0 ]
    [[ "$output" == "45m" ]]
}

@test "json::get_deployment_config uses default for missing values" {
    run json::get_deployment_config 'testing.nonexistent' 'default-timeout' "$TEST_DIR/minimal.json"
    [ "$status" -eq 0 ]
    [[ "$output" == "default-timeout" ]]
}

@test "json::get_testing_config returns testing configuration" {
    run json::get_testing_config "$TEST_DIR/complex.json"
    [ "$status" -eq 0 ]
    [[ "$output" == *"enabled"* ]]
    [[ "$output" == *"45m"* ]]
    [[ "$output" == *"windmill"* ]]
}

################################################################################
# Validation Tests
################################################################################

@test "json::validate_config passes for valid configuration" {
    run json::validate_config "$TEST_DIR/minimal.json"
    [ "$status" -eq 0 ]
}

@test "json::validate_config fails for invalid JSON" {
    run json::validate_config "$TEST_DIR/invalid.json"
    [ "$status" -eq 1 ]
}

@test "json::validate_config fails for missing required sections" {
    # Create config missing required sections
    cat > "$TEST_DIR/missing-sections.json" << 'EOF'
{
  "service": {
    "name": "incomplete"
  }
}
EOF
    
    run json::validate_config "$TEST_DIR/missing-sections.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"Missing required section"* ]]
}

################################################################################
# Cache Management Tests
################################################################################

@test "json::clear_cache clears cached configuration" {
    json::load_service_config "$TEST_DIR/minimal.json"
    
    # Verify cache is populated
    cached_path=$(json::get_cached_path)
    [[ -n "$cached_path" ]]
    
    # Clear cache
    json::clear_cache
    
    # Verify cache is empty
    cached_path=$(json::get_cached_path)
    [[ -z "$cached_path" ]]
}

@test "json::get_cached_path returns current cached path" {
    json::load_service_config "$TEST_DIR/minimal.json"
    
    cached_path=$(json::get_cached_path)
    [[ "$cached_path" == "$TEST_DIR/minimal.json" ]]
}

################################################################################
# Debug Mode Tests
################################################################################

@test "json::set_debug enables debug logging" {
    json::set_debug true
    
    # Debug messages should appear in stderr
    run json::load_service_config "$TEST_DIR/minimal.json"
    [ "$status" -eq 0 ]
}

@test "json::set_debug disables debug logging" {
    json::set_debug false
    
    # No debug messages should appear
    run json::load_service_config "$TEST_DIR/minimal.json"
    [ "$status" -eq 0 ]
}

################################################################################
# Error Handling Tests
################################################################################

@test "json::get_value handles missing jq command" {
    # Mock jq command to simulate missing jq
    export PATH="/nonexistent:$PATH"
    
    skip "TODO: Implement jq availability testing"
    # This test would need to mock the system::is_command function
}

@test "json::load_service_config handles permission denied" {
    # Create file and remove read permission
    cp "$TEST_DIR/minimal.json" "$TEST_DIR/no-read.json"
    chmod 000 "$TEST_DIR/no-read.json"
    
    run json::load_service_config "$TEST_DIR/no-read.json"
    [ "$status" -eq 1 ]
    [[ "$output" == *"not readable"* ]]
    
    # Restore permissions for cleanup
    chmod 644 "$TEST_DIR/no-read.json"
}

################################################################################
# Integration Tests (Common Usage Patterns)
################################################################################

@test "Integration: Extract required resources pattern (common in scenarios)" {
    # This tests the exact pattern used throughout scenarios
    json::load_service_config "$TEST_DIR/complex.json"
    
    # Get required resources (mimics scenario startup scripts)
    required_resources=$(json::get_required_resources)
    
    [[ "$required_resources" == *"postgres"* ]]
    [[ "$required_resources" == *"redis"* ]]
    [[ "$required_resources" == *"n8n"* ]]
}

@test "Integration: Extract UI testing config (scenario pattern)" {
    json::load_service_config "$TEST_DIR/complex.json"
    
    # Extract testing configuration (common pattern)
    requires_ui=$(json::get_deployment_config 'testing.ui.required' 'false')
    ui_type=$(json::get_deployment_config 'testing.ui.type' 'none')
    timeout=$(json::get_deployment_config 'testing.timeout' '30m')
    
    [[ "$requires_ui" == "true" ]]
    [[ "$ui_type" == "windmill" ]]
    [[ "$timeout" == "45m" ]]
}

@test "Integration: Resource health check extraction" {
    json::load_service_config "$TEST_DIR/complex.json"
    
    # Extract health check configuration
    postgres_health=$(json::get_value '.resources.postgres.healthCheck.type')
    health_port=$(json::get_value '.resources.postgres.healthCheck.port')
    
    [[ "$postgres_health" == "tcp" ]]
    [[ "$health_port" == "5432" ]]
}

################################################################################
# Performance Tests
################################################################################

@test "Performance: Large configuration handling" {
    skip "TODO: Implement performance tests with large configs"
    # This would test with the full 1000+ line Vrooli service.json
}

@test "Performance: Repeated value extraction" {
    json::load_service_config "$TEST_DIR/complex.json"
    
    # Extract same value multiple times (should use cache)
    for i in {1..10}; do
        value=$(json::get_value '.service.name')
        [[ "$value" == "complex-service" ]]
    done
}

################################################################################
# Self-Test Integration
################################################################################

@test "json::self_test passes with valid utilities" {
    run json::self_test
    [ "$status" -eq 0 ]
    [[ "$output" == *"All JSON utility self-tests passed"* ]]
}