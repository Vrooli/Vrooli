#!/usr/bin/env bash
# Test setup helpers for embedding system tests

# Paths to important directories
EMBEDDING_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FIXTURE_ROOT="$EMBEDDING_ROOT/test/fixtures"
TEST_ROOT="$EMBEDDING_ROOT/test"

# Source required utilities before tests run
setup_embedding_environment() {
    # Source basic utilities
    source "$EMBEDDING_ROOT/../../../lib/utils/var.sh" 2>/dev/null || true
    source "$EMBEDDING_ROOT/../../../lib/utils/log.sh" 2>/dev/null || true
    
    # Mock log functions if not available
    if ! command -v log::info >/dev/null; then
        log::info() { echo "[INFO] $*" >&2; }
        log::warn() { echo "[WARN] $*" >&2; }
        log::error() { echo "[ERROR] $*" >&2; }
        log::debug() { echo "[DEBUG] $*" >&2; }
    fi
    
    # Set test environment variables
    export EMBEDDING_TEST_MODE="true"
    export DEFAULT_MODEL="mxbai-embed-large"
    export DEFAULT_DIMENSIONS=1024
    export BATCH_SIZE=5  # Smaller for tests
}

# Mock functions for testing
mock_qdrant_collections() {
    # Mock Qdrant collection functions for testing
    qdrant::collections::create() {
        echo "Mock: Created collection $1"
        return 0
    }
    
    qdrant::collections::delete() {
        echo "Mock: Deleted collection $1"
        return 0
    }
    
    qdrant::collections::upsert_point() {
        echo "Mock: Upserted point $2 to collection $1"
        return 0
    }
    
    export -f qdrant::collections::create
    export -f qdrant::collections::delete
    export -f qdrant::collections::upsert_point
}

# Mock embedding generation for testing
mock_embedding_generation() {
    qdrant::embeddings::generate() {
        local content="$1"
        local model="${2:-mxbai-embed-large}"
        
        # Generate fake embedding (1024 dimensions of random values)
        local embedding="["
        for i in {1..1024}; do
            embedding="${embedding}0.$(( RANDOM % 1000 ))"
            [ $i -lt 1024 ] && embedding="${embedding},"
        done
        embedding="${embedding}]"
        
        echo "$embedding"
        return 0
    }
    
    export -f qdrant::embeddings::generate
}

# Check if required test fixtures exist
verify_test_fixtures() {
    local missing_fixtures=()
    
    # Check for workflow fixtures
    [ ! -f "$FIXTURE_ROOT/workflows/email-notification.json" ] && missing_fixtures+=("workflows/email-notification.json")
    [ ! -f "$FIXTURE_ROOT/workflows/data-processing-pipeline.json" ] && missing_fixtures+=("workflows/data-processing-pipeline.json")
    
    # Check for scenario fixtures  
    [ ! -f "$FIXTURE_ROOT/scenarios/test-scenario-app/PRD.md" ] && missing_fixtures+=("scenarios/test-scenario-app/PRD.md")
    
    # Check for docs fixtures
    [ ! -f "$FIXTURE_ROOT/docs/ARCHITECTURE.md" ] && missing_fixtures+=("docs/ARCHITECTURE.md")
    
    # Check for code fixtures
    [ ! -f "$FIXTURE_ROOT/code/email-service.sh" ] && missing_fixtures+=("code/email-service.sh")
    
    if [ ${#missing_fixtures[@]} -gt 0 ]; then
        echo "Missing test fixtures:" >&2
        printf "  - %s\n" "${missing_fixtures[@]}" >&2
        return 1
    fi
    
    return 0
}

# Create temporary test identity file
create_test_identity() {
    local test_dir="$1"
    local app_id="${2:-test-app}"
    
    mkdir -p "$test_dir/.vrooli"
    
    cat > "$test_dir/.vrooli/app-identity.json" << EOF
{
  "app_id": "$app_id",
  "type": "test",
  "source_scenario": "test-scenario",
  "last_indexed": "2025-01-22T10:00:00Z",
  "index_commit": "test123456789",
  "embedding_config": {
    "model": "mxbai-embed-large",
    "dimensions": 1024,
    "collections": {
      "workflows": "$app_id-workflows",
      "scenarios": "$app_id-scenarios",
      "knowledge": "$app_id-knowledge",
      "code": "$app_id-code",
      "resources": "$app_id-resources"
    }
  },
  "stats": {
    "total_embeddings": 0,
    "last_refresh_duration_seconds": 0
  }
}
EOF
}

# Validate extractor output format
validate_extractor_output() {
    local output_file="$1"
    local extractor_type="$2"
    
    if [ ! -f "$output_file" ]; then
        echo "Output file does not exist: $output_file" >&2
        return 1
    fi
    
    # Check for separator usage (if multi-item)
    local line_count=$(wc -l < "$output_file")
    if [ "$line_count" -gt 10 ]; then
        if ! grep -q "---SEPARATOR---" "$output_file"; then
            echo "Multi-item output missing separators" >&2
            return 1
        fi
    fi
    
    # Check content is not empty
    if [ ! -s "$output_file" ]; then
        echo "Output file is empty" >&2
        return 1
    fi
    
    return 0
}

# Count expected items for validation
count_expected_items() {
    local directory="$1"
    local item_type="$2"
    
    case "$item_type" in
        "workflows")
            find "$directory" -name "*.json" -path "*/initialization/*" 2>/dev/null | wc -l
            ;;
        "scenarios") 
            find "$directory" -name "PRD.md" -path "*/scenarios/*" 2>/dev/null | wc -l
            ;;
        "docs")
            find "$directory" -name "*.md" -path "*/docs/*" 2>/dev/null | wc -l
            ;;
        "code")
            find "$directory" \( -name "*.sh" -o -name "*.ts" -o -name "*.js" -o -name "*.sql" \) ! -path "*/node_modules/*" ! -path "*/.git/*" 2>/dev/null | wc -l
            ;;
        "resources")
            find "$directory" -name "*.json" -path "*/resources/*" 2>/dev/null | wc -l
            ;;
        *)
            echo "0"
            ;;
    esac
}

# Setup test environment on import
setup_embedding_environment

# Verify fixtures exist
if ! verify_test_fixtures; then
    echo "Test setup failed: Missing fixtures" >&2
    exit 1
fi