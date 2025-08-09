#!/usr/bin/env bats
# Tests for audio-intelligence-platform validate.sh script
# Validates comprehensive deployment validation functionality

bats_require_minimum_version 1.5.0

# Load test setup
# shellcheck disable=SC1091
load "../../../../__test/fixtures/setup.bash"

# Load mocks
# shellcheck disable=SC1091
load "../../../../__test/fixtures/mocks/system.sh"
# shellcheck disable=SC1091
load "../../../../__test/fixtures/mocks/http.sh"

setup() {
    vrooli_setup_unit_test
    
    # Reset all mocks
    mock::system::reset
    mock::http::reset
    
    # Set script path
    VALIDATE_SCRIPT="${BATS_TEST_DIRNAME}/validate.sh"
    
    # Create mock service.json
    export SCENARIO_DIR="${BATS_TEST_TMPDIR}/scenario"
    mkdir -p "$SCENARIO_DIR/.vrooli"
    
    cat > "$SCENARIO_DIR/.vrooli/service.json" << 'EOF'
{
  "resources": {
    "postgres": { "postgres": { "required": true } },
    "qdrant": { "qdrant": { "required": true } },
    "minio": { "minio": { "required": true } },
    "ollama": { "ollama": { "required": true } },
    "n8n": { "n8n": { "required": true } },
    "windmill": { "windmill": { "required": true } },
    "whisper": { "whisper": { "required": true } }
  }
}
EOF

    # Create mock configuration files
    mkdir -p "$SCENARIO_DIR/initialization/configuration"
    echo '{"test": true}' > "$SCENARIO_DIR/initialization/configuration/app-config.json"
    echo '{"test": true}' > "$SCENARIO_DIR/initialization/configuration/transcription-config.json"
    echo '{"test": true}' > "$SCENARIO_DIR/initialization/configuration/search-config.json"
    echo '{"test": true}' > "$SCENARIO_DIR/initialization/configuration/ui-config.json"
    
    # Create mock workflow files
    mkdir -p "$SCENARIO_DIR/initialization/automation/n8n"
    echo '{"test": true}' > "$SCENARIO_DIR/initialization/automation/n8n/transcription-pipeline.json"
    echo '{"test": true}' > "$SCENARIO_DIR/initialization/automation/n8n/ai-analysis-workflow.json"
    echo '{"test": true}' > "$SCENARIO_DIR/initialization/automation/n8n/semantic-search.json"
}

teardown() {
    vrooli_cleanup_test
}

# Test script loading and basic functionality
@test "validate.sh loads without errors" {
    [[ -f "$VALIDATE_SCRIPT" ]]
    [[ -x "$VALIDATE_SCRIPT" ]]
}

@test "validate.sh help command works" {
    run bash "$VALIDATE_SCRIPT" help
    assert_success
    assert_output --partial "Usage:"
    assert_output --partial "Commands:"
    assert_output --partial "validate"
    assert_output --partial "quick"
}

@test "validate.sh unknown command shows error" {
    run bash "$VALIDATE_SCRIPT" unknown_command
    assert_failure
    assert_output --partial "Unknown command: unknown_command"
}

# Test configuration loading
@test "load_configuration fails with missing service.json" {
    # Remove service.json
    rm -f "$SCENARIO_DIR/.vrooli/service.json"
    
    run bash -c "source '$VALIDATE_SCRIPT'; SCENARIO_DIR='$SCENARIO_DIR'; load_configuration"
    assert_failure
    assert_output --partial "service.json not found"
}

@test "load_configuration succeeds with valid service.json" {
    run bash -c "source '$VALIDATE_SCRIPT'; SCENARIO_DIR='$SCENARIO_DIR'; load_configuration"
    assert_success
    assert_output --partial "Loading scenario configuration"
    assert_output --partial "Required resources:"
}

# Test resource validation with mocks
@test "validate_postgres with mock success" {
    # Mock PostgreSQL commands
    mock::system::set_command "pg_isready" "/usr/bin/pg_isready"
    mock::system::set_command "psql" "/usr/bin/psql"
    
    # Mock successful pg_isready
    pg_isready() {
        case "$*" in
            *"-h localhost -p 5433"*) return 0 ;;
            *) return 1 ;;
        esac
    }
    export -f pg_isready
    
    # Mock successful psql commands
    psql() {
        case "$*" in
            *"audio_intelligence_platform"*) return 0 ;;
            *"\dt transcriptions"*) return 0 ;;
            *"\dt ai_analyses"*) return 0 ;;
            *"\dt user_sessions"*) return 0 ;;
            *"\dt search_queries"*) return 0 ;;
            *"\dt app_config"*) return 0 ;;
            *"SELECT COUNT(*) FROM app_config"*) echo "5"; return 0 ;;
            *) return 0 ;;
        esac
    }
    export -f psql
    
    run bash -c "source '$VALIDATE_SCRIPT'; validate_postgres"
    assert_success
    assert_output --partial "Validating PostgreSQL"
    assert_output --partial "PostgreSQL validation completed"
}

@test "validate_postgres with mock failure" {
    # Mock failed pg_isready
    pg_isready() {
        return 1
    }
    export -f pg_isready
    
    run bash -c "source '$VALIDATE_SCRIPT'; validate_postgres"
    assert_failure
    assert_output --partial "PostgreSQL is not running"
}

@test "validate_minio with mock success" {
    # Mock successful curl for MinIO health
    curl() {
        case "$*" in
            *"localhost:9000/minio/health/live"*) return 0 ;;
            *) return 1 ;;
        esac
    }
    export -f curl
    
    # Mock MinIO client not available
    mc() {
        return 1
    }
    export -f mc
    
    run bash -c "source '$VALIDATE_SCRIPT'; validate_minio"
    assert_success
    assert_output --partial "Validating MinIO"
    assert_output --partial "MinIO is running"
    assert_output --partial "MinIO client (mc) not available"
}

@test "validate_qdrant with mock success" {
    # Mock successful curl for Qdrant health and collections
    curl() {
        case "$*" in
            *"localhost:6333/health"*) return 0 ;;
            *"localhost:6333/collections/transcription-embeddings"*) 
                echo '{"result": {"points_count": 100}}'
                return 0
                ;;
            *) return 1 ;;
        esac
    }
    export -f curl
    
    run bash -c "source '$VALIDATE_SCRIPT'; validate_qdrant"
    assert_success
    assert_output --partial "Validating Qdrant"
    assert_output --partial "Qdrant is running"
    assert_output --partial "Collection 'transcription-embeddings' exists"
}

@test "validate_ollama with mock success" {
    # Mock successful Ollama API responses
    curl() {
        case "$*" in
            *"localhost:11434/api/tags"*) 
                echo '{"models": [{"name": "llama3.1:8b"}, {"name": "nomic-embed-text"}]}'
                return 0
                ;;
            *"localhost:11434/api/generate"*)
                echo '{"response": "Test response"}'
                return 0
                ;;
            *) return 1 ;;
        esac
    }
    export -f curl
    
    run bash -c "source '$VALIDATE_SCRIPT'; validate_ollama"
    assert_success
    assert_output --partial "Validating Ollama"
    assert_output --partial "Ollama is running"
    assert_output --partial "Text generation model 'llama3.1:8b' is available"
    assert_output --partial "Embedding model 'nomic-embed-text' is available"
}

@test "validate_n8n with mock success" {
    # Mock successful n8n health check
    curl() {
        case "$*" in
            *"localhost:5678/healthz"*) return 0 ;;
            *"localhost:5678/webhook/"*) return 0 ;;
            *) return 1 ;;
        esac
    }
    export -f curl
    
    run bash -c "source '$VALIDATE_SCRIPT'; validate_n8n"
    assert_success
    assert_output --partial "Validating n8n"
    assert_output --partial "n8n is running"
}

@test "validate_windmill with mock success" {
    # Mock successful Windmill API responses
    curl() {
        case "$*" in
            *"localhost:5681/api/version"*) return 0 ;;
            *"localhost:5681/"*) return 0 ;;
            *) return 1 ;;
        esac
    }
    export -f curl
    
    run bash -c "source '$VALIDATE_SCRIPT'; validate_windmill"
    assert_success
    assert_output --partial "Validating Windmill"
    assert_output --partial "Windmill is running"
    assert_output --partial "Windmill UI is accessible"
}

@test "validate_whisper with mock success" {
    # Mock successful Whisper health check
    curl() {
        case "$*" in
            *"localhost:8090/health"*) return 0 ;;
            *"localhost:8090/transcribe"*) return 1 ;; # Expected to fail with null file
            *) return 1 ;;
        esac
    }
    export -f curl
    
    run bash -c "source '$VALIDATE_SCRIPT'; validate_whisper"
    assert_success
    assert_output --partial "Validating Whisper"
    assert_output --partial "Whisper is running"
}

# Test workflow validation
@test "validate_workflows with valid JSON files" {
    run bash -c "source '$VALIDATE_SCRIPT'; SCENARIO_DIR='$SCENARIO_DIR'; validate_workflows"
    assert_success
    assert_output --partial "Validating workflow files"
    assert_output --partial "Workflow 'transcription-pipeline.json' is valid JSON"
    assert_output --partial "Workflow 'ai-analysis-workflow.json' is valid JSON"
    assert_output --partial "Workflow 'semantic-search.json' is valid JSON"
}

@test "validate_workflows with invalid JSON" {
    # Create invalid JSON file
    echo "{ invalid json" > "$SCENARIO_DIR/initialization/automation/n8n/transcription-pipeline.json"
    
    run bash -c "source '$VALIDATE_SCRIPT'; SCENARIO_DIR='$SCENARIO_DIR'; validate_workflows"
    assert_success  # Function doesn't fail, just reports errors
    assert_output --partial "Workflow 'transcription-pipeline.json' has invalid JSON"
}

# Test configuration validation
@test "validate_configuration with valid JSON files" {
    run bash -c "source '$VALIDATE_SCRIPT'; SCENARIO_DIR='$SCENARIO_DIR'; validate_configuration"
    assert_success
    assert_output --partial "Validating configuration files"
    assert_output --partial "Configuration 'app-config.json' is valid JSON"
    assert_output --partial "Configuration 'transcription-config.json' is valid JSON"
}

@test "validate_configuration with missing files" {
    # Remove a config file
    rm "$SCENARIO_DIR/initialization/configuration/app-config.json"
    
    run bash -c "source '$VALIDATE_SCRIPT'; SCENARIO_DIR='$SCENARIO_DIR'; validate_configuration"
    assert_success  # Function doesn't fail, just reports errors
    assert_output --partial "Configuration file 'app-config.json' missing"
}

# Test quick validation mode
@test "quick validation mode works" {
    # Mock jq for service.json parsing
    jq() {
        case "$*" in
            *"to_entries"*) echo "postgres qdrant minio" ;;
            *) echo "test" ;;
        esac
    }
    export -f jq
    
    # Mock resource validators to succeed quickly
    validate_postgres() { echo "PostgreSQL OK"; }
    validate_qdrant() { echo "Qdrant OK"; }
    validate_minio() { echo "MinIO OK"; }
    export -f validate_postgres validate_qdrant validate_minio
    
    run bash "$VALIDATE_SCRIPT" quick
    assert_success
    assert_output --partial "Quick validation"
    assert_output --partial "Quick validation completed"
}

# Test error counting and exit codes
@test "validation fails with errors" {
    # Remove service.json to force error
    rm "$SCENARIO_DIR/.vrooli/service.json"
    
    run bash "$VALIDATE_SCRIPT" validate
    assert_failure
}

@test "validation succeeds with no errors" {
    # Mock all validations to succeed
    jq() {
        case "$*" in
            *"to_entries"*) echo "" ;; # No required resources
            *) echo "test" ;;
        esac
    }
    export -f jq
    
    run bash "$VALIDATE_SCRIPT" validate  
    assert_success
    assert_output --partial "All validations passed successfully"
}

# Test individual validator error cases
@test "validators handle missing dependencies gracefully" {
    # Mock missing commands
    command() {
        case "$*" in
            *"pg_isready"*) return 1 ;;
            *"mc"*) return 1 ;;
            *"jq"*) return 1 ;;
            *) return 0 ;;
        esac
    }
    export -f command
    
    # Each validator should handle missing dependencies without crashing
    run bash -c "source '$VALIDATE_SCRIPT'; validate_minio"
    assert_success  # Should succeed but warn about missing mc
}