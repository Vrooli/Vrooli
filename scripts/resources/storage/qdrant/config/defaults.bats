#!/usr/bin/env bats
# Tests for Qdrant defaults.sh configuration

# Setup for each test
setup() {
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    QDRANT_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Clear any existing configuration
    unset QDRANT_PORT QDRANT_GRPC_PORT QDRANT_BASE_URL QDRANT_GRPC_URL
    unset QDRANT_CONTAINER_NAME QDRANT_DATA_DIR QDRANT_CONFIG_DIR QDRANT_SNAPSHOTS_DIR
    unset QDRANT_IMAGE QDRANT_API_KEY QDRANT_DEFAULT_COLLECTIONS QDRANT_COLLECTION_CONFIGS
    unset QDRANT_NETWORK_NAME QDRANT_HEALTH_CHECK_INTERVAL QDRANT_HEALTH_CHECK_MAX_ATTEMPTS
    unset QDRANT_API_TIMEOUT QDRANT_STARTUP_MAX_WAIT QDRANT_STARTUP_WAIT_INTERVAL
    unset QDRANT_INITIALIZATION_WAIT QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE
    unset QDRANT_STORAGE_MEMMAP_THRESHOLD QDRANT_STORAGE_INDEXING_THRESHOLD
    unset QDRANT_MAX_REQUEST_SIZE_MB QDRANT_MAX_WORKERS QDRANT_MIN_DISK_SPACE_GB
    
    # Mock resources function
    resources::get_default_port() {
        case "$1" in
            "qdrant") echo "6333" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Load the configuration file to test
    source "${QDRANT_DIR}/config/defaults.sh"
}

# Test qdrant::export_config function exists
@test "qdrant::export_config function is defined" {
    declare -f qdrant::export_config >/dev/null
}

# Test basic configuration export
@test "qdrant::export_config sets all required variables" {
    qdrant::export_config
    
    [ -n "$QDRANT_PORT" ]
    [ -n "$QDRANT_GRPC_PORT" ]
    [ -n "$QDRANT_BASE_URL" ]
    [ -n "$QDRANT_GRPC_URL" ]
    [ -n "$QDRANT_CONTAINER_NAME" ]
    [ -n "$QDRANT_DATA_DIR" ]
    [ -n "$QDRANT_CONFIG_DIR" ]
    [ -n "$QDRANT_SNAPSHOTS_DIR" ]
    [ -n "$QDRANT_IMAGE" ]
}

# Test default port configuration
@test "qdrant::export_config sets correct default ports" {
    qdrant::export_config
    
    [ "$QDRANT_PORT" = "6333" ]
    [ "$QDRANT_GRPC_PORT" = "6334" ]
}

# Test custom port configuration
@test "qdrant::export_config respects custom ports" {
    export QDRANT_CUSTOM_PORT="9999"
    export QDRANT_CUSTOM_GRPC_PORT="9998"
    
    qdrant::export_config
    
    [ "$QDRANT_PORT" = "9999" ]
    [ "$QDRANT_GRPC_PORT" = "9998" ]
}

# Test URL generation
@test "qdrant::export_config generates correct URLs" {
    qdrant::export_config
    
    [[ "$QDRANT_BASE_URL" =~ "http://localhost:6333" ]]
    [[ "$QDRANT_GRPC_URL" =~ "grpc://localhost:6334" ]]
}

# Test custom URL generation
@test "qdrant::export_config generates URLs with custom ports" {
    export QDRANT_CUSTOM_PORT="9999"
    export QDRANT_CUSTOM_GRPC_PORT="9998"
    
    qdrant::export_config
    
    [[ "$QDRANT_BASE_URL" =~ "http://localhost:9999" ]]
    [[ "$QDRANT_GRPC_URL" =~ "grpc://localhost:9998" ]]
}

# Test container name configuration
@test "qdrant::export_config sets correct container name" {
    qdrant::export_config
    
    [ "$QDRANT_CONTAINER_NAME" = "qdrant" ]
}

# Test directory configuration
@test "qdrant::export_config sets correct directories" {
    qdrant::export_config
    
    [[ "$QDRANT_DATA_DIR" =~ "${HOME}/.qdrant/data" ]]
    [[ "$QDRANT_CONFIG_DIR" =~ "${HOME}/.qdrant/config" ]]
    [[ "$QDRANT_SNAPSHOTS_DIR" =~ "${HOME}/.qdrant/snapshots" ]]
}

# Test Docker image configuration
@test "qdrant::export_config sets correct Docker image" {
    qdrant::export_config
    
    [ "$QDRANT_IMAGE" = "qdrant/qdrant:latest" ]
}

# Test API key configuration
@test "qdrant::export_config handles API key configuration" {
    qdrant::export_config
    
    # API key should be empty by default (no authentication)
    [ -z "$QDRANT_API_KEY" ]
}

# Test custom API key configuration
@test "qdrant::export_config respects custom API key" {
    export QDRANT_CUSTOM_API_KEY="test_api_key_123"
    
    qdrant::export_config
    
    [ "$QDRANT_API_KEY" = "test_api_key_123" ]
}

# Test default collections configuration
@test "qdrant::export_config sets default collections" {
    qdrant::export_config
    
    [ -n "$QDRANT_DEFAULT_COLLECTIONS" ]
    [[ "${QDRANT_DEFAULT_COLLECTIONS[*]}" =~ "agent_memory" ]]
    [[ "${QDRANT_DEFAULT_COLLECTIONS[*]}" =~ "code_embeddings" ]]
    [[ "${QDRANT_DEFAULT_COLLECTIONS[*]}" =~ "document_chunks" ]]
    [[ "${QDRANT_DEFAULT_COLLECTIONS[*]}" =~ "conversation_history" ]]
}

# Test collection configurations
@test "qdrant::export_config sets collection configurations" {
    qdrant::export_config
    
    [ -n "$QDRANT_COLLECTION_CONFIGS" ]
    [[ "${QDRANT_COLLECTION_CONFIGS[*]}" =~ "agent_memory:1536:Cosine" ]]
    [[ "${QDRANT_COLLECTION_CONFIGS[*]}" =~ "code_embeddings:768:Dot" ]]
    [[ "${QDRANT_COLLECTION_CONFIGS[*]}" =~ "document_chunks:1536:Cosine" ]]
    [[ "${QDRANT_COLLECTION_CONFIGS[*]}" =~ "conversation_history:1536:Cosine" ]]
}

# Test network configuration
@test "qdrant::export_config sets network configuration" {
    qdrant::export_config
    
    [ "$QDRANT_NETWORK_NAME" = "qdrant-network" ]
}

# Test health check configuration
@test "qdrant::export_config sets health check configuration" {
    qdrant::export_config
    
    [ "$QDRANT_HEALTH_CHECK_INTERVAL" = "5" ]
    [ "$QDRANT_HEALTH_CHECK_MAX_ATTEMPTS" = "12" ]
    [ "$QDRANT_API_TIMEOUT" = "10" ]
}

# Test timeout configuration
@test "qdrant::export_config sets timeout configuration" {
    qdrant::export_config
    
    [ "$QDRANT_STARTUP_MAX_WAIT" = "60" ]
    [ "$QDRANT_STARTUP_WAIT_INTERVAL" = "2" ]
    [ "$QDRANT_INITIALIZATION_WAIT" = "10" ]
}

# Test storage configuration
@test "qdrant::export_config sets storage configuration" {
    qdrant::export_config
    
    [ "$QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE" = "20000" ]
    [ "$QDRANT_STORAGE_MEMMAP_THRESHOLD" = "100000" ]
    [ "$QDRANT_STORAGE_INDEXING_THRESHOLD" = "20000" ]
}

# Test performance configuration
@test "qdrant::export_config sets performance configuration" {
    qdrant::export_config
    
    [ "$QDRANT_MAX_REQUEST_SIZE_MB" = "32" ]
    [ "$QDRANT_MAX_WORKERS" = "0" ]  # 0 = auto-detect
}

# Test resource limits
@test "qdrant::export_config sets resource limits" {
    qdrant::export_config
    
    [ "$QDRANT_MIN_DISK_SPACE_GB" = "2" ]
}

# Test variable export
@test "qdrant::export_config exports all variables" {
    qdrant::export_config
    
    # Check that variables are exported (available in subshells)
    result=$(bash -c 'echo $QDRANT_PORT')
    [ -n "$result" ]
    
    result=$(bash -c 'echo $QDRANT_BASE_URL')
    [ -n "$result" ]
    
    result=$(bash -c 'echo $QDRANT_CONTAINER_NAME')
    [ -n "$result" ]
}

# Test idempotent execution
@test "qdrant::export_config is idempotent" {
    # Run export twice
    qdrant::export_config
    local first_port="$QDRANT_PORT"
    local first_url="$QDRANT_BASE_URL"
    
    qdrant::export_config
    local second_port="$QDRANT_PORT"
    local second_url="$QDRANT_BASE_URL"
    
    [ "$first_port" = "$second_port" ]
    [ "$first_url" = "$second_url" ]
}

# Test readonly variable protection
@test "qdrant::export_config handles readonly variables" {
    # Set a variable as readonly
    export QDRANT_PORT="7777"
    readonly QDRANT_PORT
    
    # Should not fail when trying to set readonly variable
    qdrant::export_config
    
    # Value should remain unchanged
    [ "$QDRANT_PORT" = "7777" ]
}

# Test array variable handling
@test "qdrant::export_config handles array variables correctly" {
    qdrant::export_config
    
    # Check that arrays are properly set
    [ "${#QDRANT_DEFAULT_COLLECTIONS[@]}" -eq 4 ]
    [ "${#QDRANT_COLLECTION_CONFIGS[@]}" -eq 4 ]
}

# Test numeric validation
@test "qdrant::export_config sets valid numeric values" {
    qdrant::export_config
    
    # Check that numeric values are actually numeric
    [[ "$QDRANT_PORT" =~ ^[0-9]+$ ]]
    [[ "$QDRANT_GRPC_PORT" =~ ^[0-9]+$ ]]
    [[ "$QDRANT_HEALTH_CHECK_INTERVAL" =~ ^[0-9]+$ ]]
    [[ "$QDRANT_HEALTH_CHECK_MAX_ATTEMPTS" =~ ^[0-9]+$ ]]
    [[ "$QDRANT_API_TIMEOUT" =~ ^[0-9]+$ ]]
    [[ "$QDRANT_STARTUP_MAX_WAIT" =~ ^[0-9]+$ ]]
    [[ "$QDRANT_STARTUP_WAIT_INTERVAL" =~ ^[0-9]+$ ]]
    [[ "$QDRANT_INITIALIZATION_WAIT" =~ ^[0-9]+$ ]]
}

# Test port range validation
@test "qdrant::export_config sets ports in valid range" {
    qdrant::export_config
    
    # Ports should be in valid range (1-65535)
    [ "$QDRANT_PORT" -ge 1 ] && [ "$QDRANT_PORT" -le 65535 ]
    [ "$QDRANT_GRPC_PORT" -ge 1 ] && [ "$QDRANT_GRPC_PORT" -le 65535 ]
}

# Test URL format validation
@test "qdrant::export_config generates valid URLs" {
    qdrant::export_config
    
    # URLs should have correct format
    [[ "$QDRANT_BASE_URL" =~ ^http://[^:]+:[0-9]+$ ]]
    [[ "$QDRANT_GRPC_URL" =~ ^grpc://[^:]+:[0-9]+$ ]]
}

# Test directory path validation
@test "qdrant::export_config sets valid directory paths" {
    qdrant::export_config
    
    # Paths should be absolute
    [[ "$QDRANT_DATA_DIR" =~ ^/ ]]
    [[ "$QDRANT_CONFIG_DIR" =~ ^/ ]]
    [[ "$QDRANT_SNAPSHOTS_DIR" =~ ^/ ]]
}

# Test Docker image format validation
@test "qdrant::export_config sets valid Docker image" {
    qdrant::export_config
    
    # Image should have correct format
    [[ "$QDRANT_IMAGE" =~ ^[a-z0-9._/-]+:[a-z0-9._-]+$ ]]
}

# Test collection name validation
@test "qdrant::export_config sets valid collection names" {
    qdrant::export_config
    
    # Collection names should be valid
    for collection in "${QDRANT_DEFAULT_COLLECTIONS[@]}"; do
        [[ "$collection" =~ ^[a-z0-9_]+$ ]]
    done
}

# Test collection config format validation
@test "qdrant::export_config sets valid collection configs" {
    qdrant::export_config
    
    # Collection configs should have correct format
    for config in "${QDRANT_COLLECTION_CONFIGS[@]}"; do
        [[ "$config" =~ ^[a-z0-9_]+:[0-9]+:(Cosine|Dot|Euclid)$ ]]
    done
}

# Test network name validation
@test "qdrant::export_config sets valid network name" {
    qdrant::export_config
    
    # Network name should be valid Docker network name
    [[ "$QDRANT_NETWORK_NAME" =~ ^[a-z0-9_-]+$ ]]
}

# Test boolean-like values
@test "qdrant::export_config sets valid boolean-like values" {
    qdrant::export_config
    
    # MAX_WORKERS can be 0 (auto-detect) or positive integer
    [[ "$QDRANT_MAX_WORKERS" =~ ^[0-9]+$ ]]
}

# Test environment variable precedence
@test "qdrant::export_config respects environment variable precedence" {
    # Set custom values before export
    export QDRANT_CUSTOM_PORT="8888"
    export QDRANT_CUSTOM_GRPC_PORT="8887"
    export QDRANT_CUSTOM_API_KEY="custom_key"
    
    qdrant::export_config
    
    [ "$QDRANT_PORT" = "8888" ]
    [ "$QDRANT_GRPC_PORT" = "8887" ]
    [ "$QDRANT_API_KEY" = "custom_key" ]
}

# Test configuration completeness
@test "qdrant::export_config sets all documented variables" {
    qdrant::export_config
    
    # Check that all major configuration categories are covered
    # Service configuration
    [ -n "$QDRANT_PORT" ]
    [ -n "$QDRANT_GRPC_PORT" ]
    [ -n "$QDRANT_BASE_URL" ]
    [ -n "$QDRANT_GRPC_URL" ]
    [ -n "$QDRANT_CONTAINER_NAME" ]
    [ -n "$QDRANT_IMAGE" ]
    
    # Storage configuration
    [ -n "$QDRANT_DATA_DIR" ]
    [ -n "$QDRANT_CONFIG_DIR" ]
    [ -n "$QDRANT_SNAPSHOTS_DIR" ]
    
    # Collections configuration
    [ -n "$QDRANT_DEFAULT_COLLECTIONS" ]
    [ -n "$QDRANT_COLLECTION_CONFIGS" ]
    
    # Network configuration
    [ -n "$QDRANT_NETWORK_NAME" ]
    
    # Health check configuration
    [ -n "$QDRANT_HEALTH_CHECK_INTERVAL" ]
    [ -n "$QDRANT_HEALTH_CHECK_MAX_ATTEMPTS" ]
    [ -n "$QDRANT_API_TIMEOUT" ]
    
    # Timing configuration
    [ -n "$QDRANT_STARTUP_MAX_WAIT" ]
    [ -n "$QDRANT_STARTUP_WAIT_INTERVAL" ]
    [ -n "$QDRANT_INITIALIZATION_WAIT" ]
    
    # Performance configuration
    [ -n "$QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE" ]
    [ -n "$QDRANT_STORAGE_MEMMAP_THRESHOLD" ]
    [ -n "$QDRANT_STORAGE_INDEXING_THRESHOLD" ]
    [ -n "$QDRANT_MAX_REQUEST_SIZE_MB" ]
    [ -n "$QDRANT_MAX_WORKERS" ]
    
    # Resource limits
    [ -n "$QDRANT_MIN_DISK_SPACE_GB" ]
}

# Test configuration consistency
@test "qdrant::export_config maintains configuration consistency" {
    qdrant::export_config
    
    # Port and URL consistency
    [[ "$QDRANT_BASE_URL" =~ ":${QDRANT_PORT}" ]]
    [[ "$QDRANT_GRPC_URL" =~ ":${QDRANT_GRPC_PORT}" ]]
    
    # Directory structure consistency
    [[ "$QDRANT_DATA_DIR" =~ ".qdrant/data" ]]
    [[ "$QDRANT_CONFIG_DIR" =~ ".qdrant/config" ]]
    [[ "$QDRANT_SNAPSHOTS_DIR" =~ ".qdrant/snapshots" ]]
}

# Test default collection and config alignment
@test "qdrant::export_config aligns collections with configs" {
    qdrant::export_config
    
    # Number of collections should match number of configs
    [ "${#QDRANT_DEFAULT_COLLECTIONS[@]}" -eq "${#QDRANT_COLLECTION_CONFIGS[@]}" ]
    
    # Each default collection should have a corresponding config
    for collection in "${QDRANT_DEFAULT_COLLECTIONS[@]}"; do
        local found=false
        for config in "${QDRANT_COLLECTION_CONFIGS[@]}"; do
            if [[ "$config" =~ ^${collection}: ]]; then
                found=true
                break
            fi
        done
        [ "$found" = true ]
    done
}

# Test performance value ranges
@test "qdrant::export_config sets reasonable performance values" {
    qdrant::export_config
    
    # Storage values should be reasonable
    [ "$QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE" -ge 1000 ]
    [ "$QDRANT_STORAGE_MEMMAP_THRESHOLD" -ge 10000 ]
    [ "$QDRANT_STORAGE_INDEXING_THRESHOLD" -ge 1000 ]
    
    # Request size should be reasonable (in MB)
    [ "$QDRANT_MAX_REQUEST_SIZE_MB" -ge 1 ]
    [ "$QDRANT_MAX_REQUEST_SIZE_MB" -le 1024 ]
    
    # Disk space requirement should be reasonable (in GB)
    [ "$QDRANT_MIN_DISK_SPACE_GB" -ge 1 ]
}

# Test timeout value ranges
@test "qdrant::export_config sets reasonable timeout values" {
    qdrant::export_config
    
    # Health check values should be reasonable
    [ "$QDRANT_HEALTH_CHECK_INTERVAL" -ge 1 ]
    [ "$QDRANT_HEALTH_CHECK_INTERVAL" -le 60 ]
    
    [ "$QDRANT_HEALTH_CHECK_MAX_ATTEMPTS" -ge 1 ]
    [ "$QDRANT_HEALTH_CHECK_MAX_ATTEMPTS" -le 100 ]
    
    [ "$QDRANT_API_TIMEOUT" -ge 1 ]
    [ "$QDRANT_API_TIMEOUT" -le 300 ]
    
    # Startup values should be reasonable
    [ "$QDRANT_STARTUP_MAX_WAIT" -ge 10 ]
    [ "$QDRANT_STARTUP_MAX_WAIT" -le 600 ]
    
    [ "$QDRANT_STARTUP_WAIT_INTERVAL" -ge 1 ]
    [ "$QDRANT_STARTUP_WAIT_INTERVAL" -le 60 ]
    
    [ "$QDRANT_INITIALIZATION_WAIT" -ge 1 ]
    [ "$QDRANT_INITIALIZATION_WAIT" -le 120 ]
}

# Test distance metric validation
@test "qdrant::export_config uses valid distance metrics" {
    qdrant::export_config
    
    # All distance metrics should be valid Qdrant metrics
    for config in "${QDRANT_COLLECTION_CONFIGS[@]}"; do
        local metric=$(echo "$config" | cut -d: -f3)
        [[ "$metric" =~ ^(Cosine|Dot|Euclid)$ ]]
    done
}

# Test vector size validation
@test "qdrant::export_config uses valid vector sizes" {
    qdrant::export_config
    
    # All vector sizes should be positive integers
    for config in "${QDRANT_COLLECTION_CONFIGS[@]}"; do
        local size=$(echo "$config" | cut -d: -f2)
        [[ "$size" =~ ^[1-9][0-9]*$ ]]
        [ "$size" -ge 1 ]
        [ "$size" -le 65536 ]  # Reasonable upper limit
    done
}