#!/usr/bin/env bats

# Test suite for embedding-validator.sh
# Tests the embedding validation utilities

# Setup test directory and source var.sh first
COMMON_DIR="$BATS_TEST_DIRNAME"
_HERE="$COMMON_DIR"

# shellcheck disable=SC1091
source "${_HERE}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/test_helper.bash"

# Load mocks for testing
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/http.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/jq.sh"

# Setup and teardown
setup() {
    # Reset mock states
    if command -v mock::http::reset &>/dev/null; then
        mock::http::reset
    fi
    if command -v mock::jq::reset &>/dev/null; then
        mock::jq::reset
    fi
    
    # Export required test variables
    export var_LOG_FILE="/test/lib/utils/log.sh"
    
    # Create mock log file
    mkdir -p "$(dirname "$var_LOG_FILE")"
    cat > "$var_LOG_FILE" << 'EOF'
log::info() { echo "[INFO] $*"; }
log::error() { echo "[ERROR] $*" >&2; }
log::success() { echo "[SUCCESS] $*"; }
log::warn() { echo "[WARN] $*"; }
log::header() { echo "=== $* ==="; }
EOF
}

teardown() {
    # Clean up test artifacts
    rm -rf /test 2>/dev/null || true
}

# ============================================================================
# Function existence tests
# ============================================================================

@test "embedding-validator.sh can be sourced" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        echo 'SUCCESS'
    "
    
    assert_success
    [[ "$output" =~ "SUCCESS" ]]
}

@test "embedding_validator::validate_model function exists" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        type -t embedding_validator::validate_model
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "embedding_validator::get_model_dimensions function exists" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        type -t embedding_validator::get_model_dimensions
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "embedding_validator::get_collection_dimensions function exists" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        type -t embedding_validator::get_collection_dimensions
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "embedding_validator::validate_compatibility function exists" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        type -t embedding_validator::validate_compatibility
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "embedding_validator::generate_validated function exists" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        type -t embedding_validator::generate_validated
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "embedding_validator::insert_validated function exists" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        type -t embedding_validator::insert_validated
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "embedding_validator::pipeline_validated function exists" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        type -t embedding_validator::pipeline_validated
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "embedding_validator::diagnose function exists" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        type -t embedding_validator::diagnose
    "
    
    assert_success
    [ "$output" = "function" ]
}

# ============================================================================
# Validation tests with mocked responses
# ============================================================================

@test "validate_model handles Ollama not running" {
    # Mock curl to fail (service not running)
    mock::http::set_response "http://localhost:11434/" "" 7
    mock::http::set_exit_code 7
    
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        cat > /test/lib/utils/log.sh << 'EOF'
log::info() { echo \"[INFO] \$*\"; }
log::error() { echo \"[ERROR] \$*\" >&2; }
log::success() { echo \"[SUCCESS] \$*\"; }
EOF
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        embedding_validator::validate_model 'test-model' 2>&1
    "
    
    # Should fail with appropriate error
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Ollama service not running" ]]
}

@test "validate_model handles model not found" {
    # Mock Ollama service running
    mock::http::set_response "http://localhost:11434/" "OK" 0
    
    # Mock model validation response with error
    mock::jq::set_response '{"error": "model not found"}' ".error"
    
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        cat > /test/lib/utils/log.sh << 'EOF'
log::info() { echo \"[INFO] \$*\"; }
log::error() { echo \"[ERROR] \$*\" >&2; }
log::success() { echo \"[SUCCESS] \$*\"; }
EOF
        # Mock curl command for testing
        curl() {
            if [[ \"\$*\" =~ \"api/embed\" ]]; then
                echo '{\"error\": \"model not found\"}'
            else
                echo 'OK'
            fi
            return 0
        }
        export -f curl
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        embedding_validator::validate_model 'nonexistent-model' 2>&1
    "
    
    # Should report model not found
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Model validation failed" ]] || [[ "$output" =~ "not found" ]]
}

@test "get_model_dimensions returns correct dimensions" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        echo 'log::info() { echo \"\$*\"; }' > /test/lib/utils/log.sh
        
        # Mock curl to return embedding response
        curl() {
            echo '{\"embeddings\":[[1,2,3,4,5,6,7,8]]}'
            return 0
        }
        export -f curl
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        embedding_validator::get_model_dimensions 'test-model'
    "
    
    assert_success
    [ "$output" = "8" ]
}

@test "get_collection_dimensions handles Qdrant not running" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        cat > /test/lib/utils/log.sh << 'EOF'
log::info() { echo \"[INFO] \$*\"; }
log::error() { echo \"[ERROR] \$*\" >&2; }
EOF
        # Mock curl to fail
        curl() {
            return 7
        }
        export -f curl
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        embedding_validator::get_collection_dimensions 'test-collection' 2>&1
    "
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Qdrant service not running" ]]
}

@test "validate_compatibility detects dimension mismatch" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        cat > /test/lib/utils/log.sh << 'EOF'
log::info() { echo \"[INFO] \$*\"; }
log::error() { echo \"[ERROR] \$*\" >&2; }
log::success() { echo \"[SUCCESS] \$*\"; }
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        
        # Override functions to return different dimensions
        embedding_validator::get_model_dimensions() { echo 768; }
        embedding_validator::get_collection_dimensions() { echo 1536; }
        
        embedding_validator::validate_compatibility 'model' 'collection' 2>&1
    "
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Dimension mismatch" ]]
}

@test "validate_compatibility confirms compatible dimensions" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        cat > /test/lib/utils/log.sh << 'EOF'
log::info() { echo \"[INFO] \$*\"; }
log::error() { echo \"[ERROR] \$*\" >&2; }
log::success() { echo \"[SUCCESS] \$*\"; }
EOF
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        
        # Override functions to return same dimensions
        embedding_validator::get_model_dimensions() { echo 768; }
        embedding_validator::get_collection_dimensions() { echo 768; }
        
        embedding_validator::validate_compatibility 'model' 'collection' 2>&1
    "
    
    assert_success
    [[ "$output" =~ "Compatible" ]]
}

# ============================================================================
# Diagnostic tests
# ============================================================================

@test "diagnose checks Ollama service" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        cat > /test/lib/utils/log.sh << 'EOF'
log::info() { echo \"[INFO] \$*\"; }
log::error() { echo \"[ERROR] \$*\" >&2; }
log::success() { echo \"[SUCCESS] \$*\"; }
log::warn() { echo \"[WARN] \$*\"; }
log::header() { echo \"=== \$* ===\"; }
EOF
        
        # Mock curl to simulate services not running
        curl() {
            return 7
        }
        export -f curl
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        embedding_validator::diagnose 2>&1
    "
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Ollama is not running" ]]
}

@test "diagnose reports all systems healthy when services are up" {
    run bash -c "
        export var_LOG_FILE='/test/lib/utils/log.sh'
        mkdir -p /test/lib/utils
        cat > /test/lib/utils/log.sh << 'EOF'
log::info() { echo \"[INFO] \$*\"; }
log::error() { echo \"[ERROR] \$*\" >&2; }
log::success() { echo \"[SUCCESS] \$*\"; }
log::warn() { echo \"[WARN] \$*\"; }
log::header() { echo \"=== \$* ===\"; }
EOF
        
        # Mock curl to simulate services running
        curl() {
            echo '{\"result\":{\"collections\":[]}}'
            return 0
        }
        export -f curl
        
        # Mock model validation to succeed
        embedding_validator::validate_model() { return 0; }
        export -f embedding_validator::validate_model
        
        source '$COMMON_DIR/../../../lib/utils/var.sh'
        source '$COMMON_DIR/embedding-validator.sh'
        embedding_validator::diagnose 2>&1
    "
    
    assert_success
    [[ "$output" =~ "All systems healthy" ]]
}