#!/usr/bin/env bats
# Tests for Qdrant common.sh functions

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Setup for each test
setup() {
    # Load Vrooli test infrastructure
    source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"
    
    # Setup Qdrant test environment
    vrooli_setup_service_test "qdrant"
    
    # Set test environment
    export QDRANT_PORT="6333"
    export QDRANT_GRPC_PORT="6334"
    export QDRANT_CONTAINER_NAME="qdrant-test"
    export QDRANT_BASE_URL="http://localhost:6333"
    export QDRANT_GRPC_URL="grpc://localhost:6334"
    export QDRANT_DATA_DIR="/tmp/qdrant-test/data"
    export QDRANT_CONFIG_DIR="/tmp/qdrant-test/config"
    export QDRANT_SNAPSHOTS_DIR="/tmp/qdrant-test/snapshots"
    export QDRANT_IMAGE="qdrant/qdrant:latest"
    export QDRANT_API_KEY="test_qdrant_api_key_123"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    QDRANT_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories
    mkdir -p "$QDRANT_DATA_DIR"
    mkdir -p "$QDRANT_CONFIG_DIR"
    mkdir -p "$QDRANT_SNAPSHOTS_DIR"
    
    # Mock system functions
    
    # Mock docker commands
    
    # Mock curl for API calls
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".status"*) echo "ok" ;;
            *".version"*) echo "1.7.4" ;;
            *".result.status"*) echo "enabled" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock openssl for key generation
    openssl() {
        echo "qdrant_api_key_abc123def456"
    }
    
    # Mock tar and gzip for backups
    tar() {
        echo "TAR: $*"
        return 0
    }
    
    gzip() {
        echo "GZIP: $*"
        return 0
    }
    
    # Mock log functions
    
    # Load configuration and messages
    source "${QDRANT_DIR}/config/defaults.sh"
    source "${QDRANT_DIR}/config/messages.sh"
    qdrant::export_config
    qdrant::messages::init
    
    # Load the functions to test
    source "${QDRANT_DIR}/lib/common.sh"
}

# Cleanup after each test
teardown() {
    trash::safe_remove "/tmp/qdrant-test" --test-cleanup
}

# Test directory creation
@test "qdrant::create_directories creates required directories" {
    # Remove test directories first
    trash::safe_remove "/tmp/qdrant-test" --test-cleanup
    
    result=$(qdrant::create_directories)
    
    [[ "$result" =~ "directories" ]] || [[ "$result" =~ "created" ]]
    [ -d "$QDRANT_DATA_DIR" ]
    [ -d "$QDRANT_CONFIG_DIR" ]
    [ -d "$QDRANT_SNAPSHOTS_DIR" ]
}


# Test directory creation with existing directories
@test "qdrant::create_directories handles existing directories" {
    # Directories already exist from setup
    result=$(qdrant::create_directories)
    
    [[ "$result" =~ "directories" ]] || [[ "$result" =~ "already" ]] || [[ "$result" =~ "OK" ]]
}


# Test API key generation
@test "qdrant::generate_api_key generates valid API key" {
    result=$(qdrant::generate_api_key)
    
    [ -n "$result" ]
    [[ ${#result} -ge 16 ]]
    [[ "$result" =~ ^[a-zA-Z0-9_-]+$ ]]
}


# Test API key generation with custom length
@test "qdrant::generate_api_key supports custom length" {
    result=$(qdrant::generate_api_key 32)
    
    [ -n "$result" ]
    [[ ${#result} -eq 32 ]]
}


# Test API key storage
@test "qdrant::save_api_key saves API key securely" {
    local test_key="test_qdrant_key_987654321"
    
    result=$(qdrant::save_api_key "$test_key")
    
    [[ "$result" =~ "saved" ]] || [[ "$result" =~ "stored" ]]
    [ -f "${QDRANT_CONFIG_DIR}/api_key" ]
    
    # Check file permissions
    local perms=$(stat -c %a "${QDRANT_CONFIG_DIR}/api_key" 2>/dev/null || stat -f %A "${QDRANT_CONFIG_DIR}/api_key")
    [[ "$perms" =~ ^6[0-9][0-9]$ ]]  # Should be 600 or similar
}


# Test API key retrieval
@test "qdrant::get_api_key retrieves stored API key" {
    local test_key="test_qdrant_key_123456789"
    
    # Save key first
    echo "$test_key" > "${QDRANT_CONFIG_DIR}/api_key"
    chmod 600 "${QDRANT_CONFIG_DIR}/api_key"
    
    result=$(qdrant::get_api_key)
    
    [[ "$result" == "$test_key" ]]
}


# Test API key retrieval with missing file
@test "qdrant::get_api_key handles missing API key file" {
    trash::safe_remove "${QDRANT_CONFIG_DIR}/api_key" --test-cleanup
    
    result=$(qdrant::get_api_key)
    
    [ -z "$result" ]
}


# Test collection name validation
@test "qdrant::validate_collection_name validates collection names" {
    result=$(qdrant::validate_collection_name "valid_collection_name" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}


# Test collection name validation with invalid names
@test "qdrant::validate_collection_name rejects invalid names" {
    result=$(qdrant::validate_collection_name "invalid-name!" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}


# Test vector size validation
@test "qdrant::validate_vector_size validates vector dimensions" {
    result=$(qdrant::validate_vector_size "1536" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}


# Test vector size validation with invalid sizes
@test "qdrant::validate_vector_size rejects invalid sizes" {
    result=$(qdrant::validate_vector_size "0" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
    
    result=$(qdrant::validate_vector_size "abc" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}


# Test distance metric validation
@test "qdrant::validate_distance_metric validates distance metrics" {
    result=$(qdrant::validate_distance_metric "Cosine" && echo "valid" || echo "invalid")
    [[ "$result" == "valid" ]]
    
    result=$(qdrant::validate_distance_metric "Dot" && echo "valid" || echo "invalid")
    [[ "$result" == "valid" ]]
    
    result=$(qdrant::validate_distance_metric "Euclid" && echo "valid" || echo "invalid")
    [[ "$result" == "valid" ]]
}


# Test distance metric validation with invalid metrics
@test "qdrant::validate_distance_metric rejects invalid metrics" {
    result=$(qdrant::validate_distance_metric "Manhattan" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}


# Test configuration validation
@test "qdrant::validate_config validates configuration" {
    result=$(qdrant::validate_config)
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "configuration" ]]
}


# Test port validation
@test "qdrant::validate_port validates port numbers" {
    result=$(qdrant::validate_port "6333" && echo "valid" || echo "invalid")
    [[ "$result" == "valid" ]]
    
    result=$(qdrant::validate_port "65536" && echo "valid" || echo "invalid")
    [[ "$result" == "invalid" ]]
    
    result=$(qdrant::validate_port "abc" && echo "valid" || echo "invalid")
    [[ "$result" == "invalid" ]]
}


# Test network connectivity check
@test "qdrant::check_network_connectivity tests network access" {
    result=$(qdrant::check_network_connectivity)
    
    [[ "$result" =~ "network" ]] || [[ "$result" =~ "connectivity" ]]
}


# Test Docker network management
@test "qdrant::create_network creates Docker network" {
    result=$(qdrant::create_network)
    
    [[ "$result" =~ "network" ]]
    [[ "$result" =~ "qdrant-network" ]]
}


# Test Docker network cleanup
@test "qdrant::remove_network removes Docker network" {
    result=$(qdrant::remove_network)
    
    [[ "$result" =~ "network" ]]
    [[ "$result" =~ "qdrant-network" ]]
}


# Test data cleanup
@test "qdrant::cleanup_data removes data directory" {
    # Create test data
    echo "test vector data" > "${QDRANT_DATA_DIR}/test.dat"
    echo "test snapshot" > "${QDRANT_SNAPSHOTS_DIR}/test.snapshot"
    
    export YES="yes"
    result=$(qdrant::cleanup_data "yes")
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "removed" ]]
    [ ! -d "$QDRANT_DATA_DIR" ]
}


# Test data cleanup with confirmation
@test "qdrant::cleanup_data respects user confirmation" {
    export YES="no"
    result=$(qdrant::cleanup_data "no")
    
    [[ "$result" =~ "cancelled" ]] || [[ "$result" =~ "aborted" ]]
    [ -d "$QDRANT_DATA_DIR" ]  # Should still exist
}


# Test permission management
@test "qdrant::set_permissions sets correct file permissions" {
    echo "test" > "${QDRANT_CONFIG_DIR}/test_file"
    
    result=$(qdrant::set_permissions "${QDRANT_CONFIG_DIR}/test_file" "600")
    
    [[ "$result" =~ "permission" ]] || [[ "$result" =~ "set" ]]
    
    local perms=$(stat -c %a "${QDRANT_CONFIG_DIR}/test_file" 2>/dev/null || stat -f %A "${QDRANT_CONFIG_DIR}/test_file")
    [[ "$perms" == "600" ]]
}


# Test configuration generation
@test "qdrant::generate_config creates Qdrant configuration file" {
    result=$(qdrant::generate_config)
    
    [[ "$result" =~ "config" ]] || [[ "$result" =~ "generated" ]]
    [ -f "${QDRANT_CONFIG_DIR}/production.yaml" ]
    
    # Verify config contains expected settings
    grep -q "service:" "${QDRANT_CONFIG_DIR}/production.yaml"
    grep -q "storage:" "${QDRANT_CONFIG_DIR}/production.yaml"
}


# Test health monitoring
@test "qdrant::check_health performs basic health checks" {
    result=$(qdrant::check_health)
    
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "check" ]]
}


# Test backup creation
@test "qdrant::create_backup creates data backup" {
    result=$(qdrant::create_backup)
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "created" ]]
}


# Test backup restoration
@test "qdrant::restore_backup restores data from backup" {
    result=$(qdrant::restore_backup "/tmp/backup.tar.gz")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "backup" ]]
}


# Test environment preparation
@test "qdrant::prepare_environment prepares runtime environment" {
    result=$(qdrant::prepare_environment)
    
    [[ "$result" =~ "environment" ]] || [[ "$result" =~ "prepared" ]]
}


# Test metrics collection
@test "qdrant::collect_metrics gathers system metrics" {
    result=$(qdrant::collect_metrics)
    
    [[ "$result" =~ "metrics" ]] || [[ "$result" =~ "collected" ]]
}


# Test utility functions
@test "qdrant::format_size formats byte sizes" {
    result=$(qdrant::format_size "1048576")
    
    [[ "$result" =~ "MB" ]] || [[ "$result" =~ "1024" ]]
}


@test "qdrant::format_time formats durations" {
    result=$(qdrant::format_time "3661")
    
    [[ "$result" =~ "1h" ]] && [[ "$result" =~ "1m" ]]
}


# Test error handling
@test "qdrant::handle_error processes error conditions gracefully" {
    result=$(qdrant::handle_error "Test error message" 500)
    
    [[ "$result" =~ "error" ]] || [[ "$result" =~ "Test error message" ]]
}


# Test dependency verification
@test "qdrant::verify_dependencies checks required dependencies" {
    result=$(qdrant::verify_dependencies)
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "verified" ]]
}


# Test system compatibility
@test "qdrant::check_compatibility verifies system compatibility" {
    result=$(qdrant::check_compatibility)
    
    [[ "$result" =~ "compatible" ]] || [[ "$result" =~ "system" ]]
}


# Test vector format validation
@test "qdrant::validate_vector_format validates vector data format" {
    local test_vector='[0.1, 0.2, 0.3, 0.4]'
    
    result=$(qdrant::validate_vector_format "$test_vector" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}


# Test vector format validation with invalid data
@test "qdrant::validate_vector_format rejects invalid vector data" {
    local invalid_vector='invalid vector data'
    
    result=$(qdrant::validate_vector_format "$invalid_vector" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}


# Test index optimization
@test "qdrant::optimize_index optimizes vector indices" {
    result=$(qdrant::optimize_index "test_collection")
    
    [[ "$result" =~ "optimize" ]] || [[ "$result" =~ "index" ]]
    [[ "$result" =~ "test_collection" ]]
}


# Test memory usage monitoring
@test "qdrant::check_memory_usage monitors memory consumption" {
    result=$(qdrant::check_memory_usage)
    
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "usage" ]]
}


# Test disk space monitoring
@test "qdrant::check_disk_space monitors storage usage" {
    result=$(qdrant::check_disk_space)
    
    [[ "$result" =~ "disk" ]] || [[ "$result" =~ "space" ]]
}


# Test performance analysis
@test "qdrant::analyze_performance analyzes query performance" {
    result=$(qdrant::analyze_performance)
    
    [[ "$result" =~ "performance" ]] || [[ "$result" =~ "analysis" ]]
}


# Test query optimization
@test "qdrant::optimize_queries optimizes search queries" {
    result=$(qdrant::optimize_queries)
    
    [[ "$result" =~ "optimize" ]] || [[ "$result" =~ "queries" ]]
}


# Test cluster management
@test "qdrant::manage_cluster manages cluster operations" {
    result=$(qdrant::manage_cluster "status")
    
    [[ "$result" =~ "cluster" ]]
    [[ "$result" =~ "status" ]]
}


# Test shard management
@test "qdrant::manage_shards manages collection shards" {
    result=$(qdrant::manage_shards "test_collection" "redistribute")
    
    [[ "$result" =~ "shard" ]]
    [[ "$result" =~ "test_collection" ]]
}


# Test replica management
@test "qdrant::manage_replicas manages data replication" {
    result=$(qdrant::manage_replicas "test_collection" "2")
    
    [[ "$result" =~ "replica" ]]
    [[ "$result" =~ "test_collection" ]]
}


# Test consistency checks
@test "qdrant::check_consistency verifies data consistency" {
    result=$(qdrant::check_consistency)
    
    [[ "$result" =~ "consistency" ]] || [[ "$result" =~ "check" ]]
}


# Test data validation
@test "qdrant::validate_data validates stored data integrity" {
    result=$(qdrant::validate_data "test_collection")
    
    [[ "$result" =~ "validate" ]] || [[ "$result" =~ "data" ]]
    [[ "$result" =~ "test_collection" ]]
}


# Test migration utilities
@test "qdrant::migrate_data migrates data between versions" {
    result=$(qdrant::migrate_data "1.6.0" "1.7.0")
    
    [[ "$result" =~ "migrate" ]] || [[ "$result" =~ "data" ]]
}


# Test export utilities
@test "qdrant::export_collection exports collection data" {
    result=$(qdrant::export_collection "test_collection" "/tmp/export.json")
    
    [[ "$result" =~ "export" ]]
    [[ "$result" =~ "test_collection" ]]
}


# Test import utilities
@test "qdrant::import_collection imports collection data" {
    result=$(qdrant::import_collection "test_collection" "/tmp/import.json")
    
    [[ "$result" =~ "import" ]]
    [[ "$result" =~ "test_collection" ]]

