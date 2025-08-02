#!/bin/bash
# ====================================================================
# Test Cleanup Utilities
# ====================================================================
#
# Cleanup functions to ensure tests don't leave artifacts or interfere
# with each other. Provides safe cleanup for various resource types.
#
# Functions:
#   - cleanup_test_artifacts()    - Clean all test artifacts
#   - cleanup_minio_objects()     - Clean MinIO test objects
#   - cleanup_qdrant_data()       - Clean Qdrant test collections
#   - cleanup_temp_files()        - Clean temporary files
#   - cleanup_test_containers()   - Clean test containers
#   - safe_cleanup()              - Safe cleanup with error handling
#   - register_cleanup_handler()  - Register cleanup on exit
#
# ====================================================================

# Global cleanup registry
declare -a CLEANUP_TASKS=()
declare -a CLEANUP_FILES=()
declare -a CLEANUP_DIRS=()

# Colors for cleanup output
CLEANUP_GREEN='\033[0;32m'
CLEANUP_YELLOW='\033[1;33m'
CLEANUP_RED='\033[0;31m'
CLEANUP_NC='\033[0m'

# Add a cleanup command to be executed on exit
add_cleanup_command() {
    local command="$1"
    CLEANUP_TASKS+=("$command")
}

# Add a file to be cleaned up on exit
add_cleanup_file() {
    local file="$1"
    CLEANUP_FILES+=("$file")
}

# Add a directory to be cleaned up on exit
add_cleanup_dir() {
    local dir="$1"
    CLEANUP_DIRS+=("$dir")
}

# Register cleanup handler for exit
register_cleanup_handler() {
    trap 'perform_cleanup' EXIT
}

# Perform all registered cleanup tasks
perform_cleanup() {
    if [[ "${#CLEANUP_TASKS[@]}" -gt 0 || "${#CLEANUP_FILES[@]}" -gt 0 || "${#CLEANUP_DIRS[@]}" -gt 0 ]]; then
        echo -e "${CLEANUP_YELLOW}🧹 Performing test cleanup...${CLEANUP_NC}"
        
        # Execute cleanup tasks
        for task in "${CLEANUP_TASKS[@]}"; do
            eval "$task" 2>/dev/null || true
        done
        
        # Remove cleanup files
        for file in "${CLEANUP_FILES[@]}"; do
            if [[ -f "$file" ]]; then
                rm -f "$file" 2>/dev/null || true
            fi
        done
        
        # Remove cleanup directories
        for dir in "${CLEANUP_DIRS[@]}"; do
            if [[ -d "$dir" ]]; then
                rm -rf "$dir" 2>/dev/null || true
            fi
        done
        
        echo -e "${CLEANUP_GREEN}✓ Cleanup complete${CLEANUP_NC}"
    fi
}

# Add cleanup task to registry
add_cleanup_task() {
    local task="$1"
    CLEANUP_TASKS+=("$task")
}

# Add file to cleanup registry
add_cleanup_file() {
    local file="$1"
    CLEANUP_FILES+=("$file")
}

# Add directory to cleanup registry
add_cleanup_dir() {
    local dir="$1"
    CLEANUP_DIRS+=("$dir")
}

# Clean up all test artifacts
cleanup_test_artifacts() {
    echo "🧹 Cleaning up test artifacts..."
    
    # Clean resource-specific artifacts
    cleanup_minio_objects
    cleanup_qdrant_data
    cleanup_temp_files
    cleanup_test_containers
    cleanup_log_files
    
    echo "✓ Test artifact cleanup complete"
}

# Clean MinIO test objects
cleanup_minio_objects() {
    local bucket="${1:-test-integration}"
    
    if ! command -v mc >/dev/null 2>&1; then
        return 0  # Skip if MinIO client not available
    fi
    
    # Check if MinIO is running
    if ! curl -s --max-time 5 "http://localhost:9000/minio/health/live" >/dev/null 2>&1; then
        return 0  # Skip if MinIO not running
    fi
    
    echo "🗑️  Cleaning MinIO test objects..."
    
    # Configure MinIO client (simplified - in real implementation you'd need proper credentials)
    mc alias set testminio http://localhost:9000 minioadmin minioadmin >/dev/null 2>&1 || true
    
    # Remove test objects
    mc rm --recursive --force "testminio/$bucket/test-*" >/dev/null 2>&1 || true
    mc rm --recursive --force "testminio/$bucket/temp-*" >/dev/null 2>&1 || true
    
    echo "✓ MinIO cleanup complete"
}

# Clean Qdrant test collections
cleanup_qdrant_data() {
    # Check if Qdrant is running
    if ! curl -s --max-time 5 "http://localhost:6333/" >/dev/null 2>&1; then
        return 0  # Skip if Qdrant not running
    fi
    
    echo "🗑️  Cleaning Qdrant test data..."
    
    # Delete test collections
    local test_collections=("test_collection" "integration_test" "temp_collection")
    
    for collection in "${test_collections[@]}"; do
        curl -s -X DELETE "http://localhost:6333/collections/$collection" >/dev/null 2>&1 || true
    done
    
    echo "✓ Qdrant cleanup complete"
}

# Clean temporary files
cleanup_temp_files() {
    echo "🗑️  Cleaning temporary files..."
    
    # Clean test-specific temp files
    rm -f /tmp/vrooli_test_* 2>/dev/null || true
    rm -f /tmp/integration_test_* 2>/dev/null || true
    rm -f /tmp/test_output_* 2>/dev/null || true
    
    # Clean specific test directories
    if [[ -n "${TEST_DIR:-}" && -d "$TEST_DIR" ]]; then
        rm -rf "$TEST_DIR" 2>/dev/null || true
    fi
    
    echo "✓ Temporary files cleanup complete"
}

# Clean test containers (if any were created)
cleanup_test_containers() {
    echo "🗑️  Cleaning test containers..."
    
    # Remove any containers with test labels
    local test_containers
    test_containers=$(docker ps -aq --filter "label=vrooli.test=true" 2>/dev/null) || true
    
    if [[ -n "$test_containers" ]]; then
        docker rm -f $test_containers >/dev/null 2>&1 || true
    fi
    
    # Clean test networks
    local test_networks
    test_networks=$(docker network ls --filter "name=vrooli-test-*" --format "{{.Name}}" 2>/dev/null) || true
    
    if [[ -n "$test_networks" ]]; then
        docker network rm $test_networks >/dev/null 2>&1 || true
    fi
    
    echo "✓ Test containers cleanup complete"
}

# Clean log files
cleanup_log_files() {
    echo "🗑️  Cleaning test log files..."
    
    # Remove test log files older than 1 hour
    find /tmp -name "vrooli_test_*.log" -mmin +60 -delete 2>/dev/null || true
    find /tmp -name "integration_test_*.log" -mmin +60 -delete 2>/dev/null || true
    
    echo "✓ Log files cleanup complete"
}

# Safe cleanup with error handling
safe_cleanup() {
    local cleanup_function="$1"
    local description="${2:-Cleanup operation}"
    
    echo "🧹 $description..."
    
    if eval "$cleanup_function" 2>/dev/null; then
        echo -e "${CLEANUP_GREEN}✓${CLEANUP_NC} $description complete"
        return 0
    else
        echo -e "${CLEANUP_YELLOW}⚠${CLEANUP_NC} $description failed (non-critical)"
        return 1
    fi
}

# Cleanup specific MinIO bucket objects
cleanup_minio_bucket() {
    local bucket="$1"
    local prefix="${2:-test-}"
    
    if ! command -v mc >/dev/null 2>&1; then
        return 0
    fi
    
    mc alias set testminio http://localhost:9000 minioadmin minioadmin >/dev/null 2>&1 || return 1
    mc rm --recursive --force "testminio/$bucket/$prefix*" >/dev/null 2>&1 || true
}

# Cleanup specific Qdrant collection
cleanup_qdrant_collection() {
    local collection_name="$1"
    
    curl -s -X DELETE "http://localhost:6333/collections/$collection_name" >/dev/null 2>&1 || true
}

# Create isolated test environment
create_test_env() {
    local test_id="${1:-$(date +%s)_$$}"
    
    # Create test directory
    local test_dir="/tmp/vrooli_integration_test_$test_id"
    mkdir -p "$test_dir"/{input,output,logs,temp}
    
    # Register for cleanup
    add_cleanup_dir "$test_dir"
    
    # Set environment variables
    export TEST_DIR="$test_dir"
    export TEST_INPUT_DIR="$test_dir/input"
    export TEST_OUTPUT_DIR="$test_dir/output"
    export TEST_LOGS_DIR="$test_dir/logs"
    export TEST_TEMP_DIR="$test_dir/temp"
    export TEST_ID="$test_id"
    
    echo "📁 Test environment created: $test_dir"
    return 0
}

# Cleanup test environment
cleanup_test_env() {
    if [[ -n "${TEST_DIR:-}" && -d "$TEST_DIR" ]]; then
        rm -rf "$TEST_DIR" 2>/dev/null || true
        unset TEST_DIR TEST_INPUT_DIR TEST_OUTPUT_DIR TEST_LOGS_DIR TEST_TEMP_DIR TEST_ID
        echo "✓ Test environment cleaned"
    fi
}

# Cleanup specific resource test artifacts
cleanup_resource_artifacts() {
    local resource="$1"
    
    case "$resource" in
        "ollama")
            # No specific cleanup needed for Ollama
            ;;
        "whisper")
            # Clean temporary audio files
            rm -f /tmp/whisper_test_*.wav 2>/dev/null || true
            ;;
        "minio")
            cleanup_minio_bucket "test-integration"
            ;;
        "qdrant")
            cleanup_qdrant_collection "test_collection"
            ;;
        "n8n")
            # N8N cleanup would require API calls to remove test workflows
            ;;
        "browserless")
            # Clean screenshot files
            rm -f /tmp/browserless_test_*.png 2>/dev/null || true
            ;;
        *)
            echo "No specific cleanup for resource: $resource"
            ;;
    esac
}

# Validate cleanup was successful
validate_cleanup() {
    local errors=0
    
    # Check for remaining test files
    local remaining_files
    remaining_files=$(find /tmp -name "vrooli_test_*" -o -name "integration_test_*" 2>/dev/null | wc -l)
    
    if [[ $remaining_files -gt 0 ]]; then
        echo -e "${CLEANUP_YELLOW}⚠${CLEANUP_NC} $remaining_files test files still present"
        errors=$((errors + 1))
    fi
    
    # Check for test containers
    local test_containers
    test_containers=$(docker ps -aq --filter "label=vrooli.test=true" 2>/dev/null | wc -l)
    
    if [[ $test_containers -gt 0 ]]; then
        echo -e "${CLEANUP_YELLOW}⚠${CLEANUP_NC} $test_containers test containers still running"
        errors=$((errors + 1))
    fi
    
    if [[ $errors -eq 0 ]]; then
        echo -e "${CLEANUP_GREEN}✓${CLEANUP_NC} Cleanup validation passed"
        return 0
    else
        echo -e "${CLEANUP_YELLOW}⚠${CLEANUP_NC} Cleanup validation found $errors issues"
        return 1
    fi
}

# Emergency cleanup - force remove everything
emergency_cleanup() {
    echo -e "${CLEANUP_RED}🚨 Emergency cleanup initiated${CLEANUP_NC}"
    
    # Force remove all test files
    rm -rf /tmp/vrooli_test_* /tmp/integration_test_* 2>/dev/null || true
    
    # Force remove test containers
    docker ps -aq --filter "label=vrooli.test=true" 2>/dev/null | xargs -r docker rm -f >/dev/null 2>&1 || true
    
    # Force remove test networks
    docker network ls --filter "name=vrooli-test-*" --format "{{.Name}}" 2>/dev/null | xargs -r docker network rm >/dev/null 2>&1 || true
    
    # Clean MinIO test data
    cleanup_minio_objects >/dev/null 2>&1 || true
    
    # Clean Qdrant test data
    cleanup_qdrant_data >/dev/null 2>&1 || true
    
    echo -e "${CLEANUP_GREEN}✓${CLEANUP_NC} Emergency cleanup complete"
}