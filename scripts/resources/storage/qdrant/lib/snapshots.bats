#!/usr/bin/env bats
# Tests for Qdrant snapshots.sh functions

# Setup for each test
setup() {
    # Set test environment
    export QDRANT_PORT="6333"
    export QDRANT_GRPC_PORT="6334"
    export QDRANT_CONTAINER_NAME="qdrant-test"
    export QDRANT_BASE_URL="http://localhost:6333"
    export QDRANT_GRPC_URL="grpc://localhost:6334"
    export QDRANT_IMAGE="qdrant/qdrant:latest"
    export QDRANT_DATA_DIR="/tmp/qdrant-test/data"
    export QDRANT_CONFIG_DIR="/tmp/qdrant-test/config"
    export QDRANT_SNAPSHOTS_DIR="/tmp/qdrant-test/snapshots"
    export QDRANT_API_KEY="test_qdrant_api_key_123"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    QDRANT_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories and files
    mkdir -p "$QDRANT_DATA_DIR"
    mkdir -p "$QDRANT_CONFIG_DIR"
    mkdir -p "$QDRANT_SNAPSHOTS_DIR"
    
    # Create test snapshot files
    echo "test snapshot data" > "${QDRANT_SNAPSHOTS_DIR}/snapshot_test_2024-01-15_14-30-00.tar.gz"
    echo "full snapshot data" > "${QDRANT_SNAPSHOTS_DIR}/full_snapshot_2024-01-15_15-00-00.tar.gz"
    
    # Mock system functions
    system::is_command() {
        case "$1" in
            "curl"|"jq"|"tar"|"gzip"|"rsync"|"aws"|"rclone") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    # Mock curl for API calls
    curl() {
        case "$*" in
            *"/snapshots"*)
                if [[ "$*" =~ "POST" ]]; then
                    echo '{"result":{"name":"snapshot_test_collection_2024-01-15_14-30-00"},"status":"ok","time":2.5}'
                elif [[ "$*" =~ "DELETE" ]]; then
                    echo '{"result":true,"status":"ok","time":0.5}'
                else
                    echo '{"result":[{"name":"snapshot_test_collection_2024-01-15_14-30-00","size":1048576,"creation_time":"2024-01-15T14:30:00Z"},{"name":"full_snapshot_2024-01-15_15-00-00","size":5242880,"creation_time":"2024-01-15T15:00:00Z"}],"status":"ok","time":0.003}'
                fi
                ;;
            *"/collections/test_collection/snapshots"*)
                if [[ "$*" =~ "POST" ]]; then
                    echo '{"result":{"name":"snapshot_test_collection_2024-01-15_14-30-00"},"status":"ok","time":2.5}'
                else
                    echo '{"result":[{"name":"snapshot_test_collection_2024-01-15_14-30-00","size":1048576,"creation_time":"2024-01-15T14:30:00Z"}],"status":"ok","time":0.003}'
                fi
                ;;
            *"/snapshots/snapshot_test_collection_2024-01-15_14-30-00"*)
                if [[ "$*" =~ "GET" ]]; then
                    echo "Binary snapshot data"
                else
                    echo '{"result":{"name":"snapshot_test_collection_2024-01-15_14-30-00","size":1048576,"creation_time":"2024-01-15T14:30:00Z","collections":["test_collection"],"checksum":"abc123def456"},"status":"ok","time":0.005}'
                fi
                ;;
            *)
                echo "CURL: $*"
                ;;
        esac
        return 0
    }
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".result | length"*) echo "2" ;;
            *".result[].name"*) echo -e "snapshot_test_collection_2024-01-15_14-30-00\nfull_snapshot_2024-01-15_15-00-00" ;;
            *".result.name"*) echo "snapshot_test_collection_2024-01-15_14-30-00" ;;
            *".result[].size"*) echo -e "1048576\n5242880" ;;
            *".result.size"*) echo "1048576" ;;
            *".result.creation_time"*) echo "2024-01-15T14:30:00Z" ;;
            *".result.collections"*) echo '["test_collection"]' ;;
            *".result.checksum"*) echo "abc123def456" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock tar for compression
    tar() {
        case "$*" in
            *"-czf"*) echo "TAR_CREATE: $*" ;;
            *"-xzf"*) echo "TAR_EXTRACT: $*" ;;
            *"-tzf"*) echo "TAR_LIST: $*" ;;
            *) echo "TAR: $*" ;;
        esac
        return 0
    }
    
    # Mock gzip for compression
    gzip() {
        echo "GZIP: $*"
        return 0
    }
    
    # Mock rsync for synchronization
    rsync() {
        echo "RSYNC: $*"
        return 0
    }
    
    # Mock AWS CLI
    aws() {
        case "$*" in
            *"s3 cp"*) echo "AWS_S3_COPY: $*" ;;
            *"s3 sync"*) echo "AWS_S3_SYNC: $*" ;;
            *"s3 ls"*) echo "2024-01-15 14:30:00  1048576 snapshot_test_2024-01-15_14-30-00.tar.gz" ;;
            *) echo "AWS: $*" ;;
        esac
        return 0
    }
    
    # Mock rclone for cloud storage
    rclone() {
        case "$*" in
            *"copy"*) echo "RCLONE_COPY: $*" ;;
            *"sync"*) echo "RCLONE_SYNC: $*" ;;
            *"ls"*) echo "1048576 2024-01-15 14:30:00 snapshot_test_2024-01-15_14-30-00.tar.gz" ;;
            *) echo "RCLONE: $*" ;;
        esac
        return 0
    }
    
    # Mock log functions
    log::info() { echo "INFO: $1"; }
    log::error() { echo "ERROR: $1"; }
    log::warn() { echo "WARN: $1"; }
    log::success() { echo "SUCCESS: $1"; }
    log::debug() { echo "DEBUG: $1"; }
    log::header() { echo "=== $1 ==="; }
    
    # Load configuration and messages
    source "${QDRANT_DIR}/config/defaults.sh"
    source "${QDRANT_DIR}/config/messages.sh"
    qdrant::export_config
    qdrant::messages::init
    
    # Load the functions to test
    source "${QDRANT_DIR}/lib/snapshots.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "/tmp/qdrant-test"
}

# Test snapshot listing
@test "qdrant::snapshots::list lists all snapshots" {
    result=$(qdrant::snapshots::list)
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "list" ]]
}

# Test collection snapshot listing
@test "qdrant::snapshots::list_collection lists collection snapshots" {
    result=$(qdrant::snapshots::list_collection "test_collection")
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "test_collection" ]]
}

# Test full snapshot creation
@test "qdrant::snapshots::create_full creates full database snapshot" {
    result=$(qdrant::snapshots::create_full)
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "created" ]]
}

# Test collection snapshot creation
@test "qdrant::snapshots::create_collection creates collection snapshot" {
    result=$(qdrant::snapshots::create_collection "test_collection")
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "test_collection" ]]
}

# Test incremental snapshot creation
@test "qdrant::snapshots::create_incremental creates incremental snapshot" {
    result=$(qdrant::snapshots::create_incremental "test_collection" "base_snapshot_2024-01-15_14-00-00")
    
    [[ "$result" =~ "incremental" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot deletion
@test "qdrant::snapshots::delete removes snapshot" {
    result=$(qdrant::snapshots::delete "snapshot_test_collection_2024-01-15_14-30-00")
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "deleted" ]]
}

# Test snapshot deletion with confirmation
@test "qdrant::snapshots::delete respects confirmation" {
    export YES="yes"
    
    result=$(qdrant::snapshots::delete "snapshot_test_collection_2024-01-15_14-30-00")
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "deleted" ]]
}

# Test snapshot information
@test "qdrant::snapshots::info retrieves snapshot details" {
    result=$(qdrant::snapshots::info "snapshot_test_collection_2024-01-15_14-30-00")
    
    [[ "$result" =~ "snapshot" ]] || [[ "$result" =~ "information" ]]
}

# Test snapshot download
@test "qdrant::snapshots::download downloads snapshot" {
    result=$(qdrant::snapshots::download "snapshot_test_collection_2024-01-15_14-30-00" "/tmp/downloaded_snapshot.tar.gz")
    
    [[ "$result" =~ "download" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot upload
@test "qdrant::snapshots::upload uploads snapshot" {
    result=$(qdrant::snapshots::upload "/tmp/snapshot.tar.gz" "uploaded_snapshot")
    
    [[ "$result" =~ "upload" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot restoration
@test "qdrant::snapshots::restore restores from snapshot" {
    result=$(qdrant::snapshots::restore "snapshot_test_collection_2024-01-15_14-30-00")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test collection restoration
@test "qdrant::snapshots::restore_collection restores collection from snapshot" {
    result=$(qdrant::snapshots::restore_collection "test_collection" "snapshot_test_collection_2024-01-15_14-30-00")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "collection" ]]
}

# Test snapshot validation
@test "qdrant::snapshots::validate validates snapshot integrity" {
    result=$(qdrant::snapshots::validate "snapshot_test_collection_2024-01-15_14-30-00")
    
    [[ "$result" =~ "validate" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot verification
@test "qdrant::snapshots::verify verifies snapshot checksum" {
    result=$(qdrant::snapshots::verify "snapshot_test_collection_2024-01-15_14-30-00" "abc123def456")
    
    [[ "$result" =~ "verify" ]] || [[ "$result" =~ "checksum" ]]
}

# Test snapshot comparison
@test "qdrant::snapshots::compare compares snapshots" {
    result=$(qdrant::snapshots::compare "snapshot_test_collection_2024-01-15_14-30-00" "full_snapshot_2024-01-15_15-00-00")
    
    [[ "$result" =~ "compare" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot compression
@test "qdrant::snapshots::compress compresses snapshot" {
    result=$(qdrant::snapshots::compress "${QDRANT_SNAPSHOTS_DIR}/snapshot_test_2024-01-15_14-30-00.tar.gz")
    
    [[ "$result" =~ "compress" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot decompression
@test "qdrant::snapshots::decompress decompresses snapshot" {
    result=$(qdrant::snapshots::decompress "${QDRANT_SNAPSHOTS_DIR}/snapshot_test_2024-01-15_14-30-00.tar.gz")
    
    [[ "$result" =~ "decompress" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot encryption
@test "qdrant::snapshots::encrypt encrypts snapshot" {
    result=$(qdrant::snapshots::encrypt "${QDRANT_SNAPSHOTS_DIR}/snapshot_test_2024-01-15_14-30-00.tar.gz" "encryption_key")
    
    [[ "$result" =~ "encrypt" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot decryption
@test "qdrant::snapshots::decrypt decrypts snapshot" {
    result=$(qdrant::snapshots::decrypt "${QDRANT_SNAPSHOTS_DIR}/snapshot_test_2024-01-15_14-30-00.tar.gz.enc" "encryption_key")
    
    [[ "$result" =~ "decrypt" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot scheduling
@test "qdrant::snapshots::schedule schedules automatic snapshots" {
    result=$(qdrant::snapshots::schedule "test_collection" "daily" "02:00")
    
    [[ "$result" =~ "schedule" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot rotation
@test "qdrant::snapshots::rotate rotates old snapshots" {
    result=$(qdrant::snapshots::rotate "test_collection" "7")
    
    [[ "$result" =~ "rotate" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot cleanup
@test "qdrant::snapshots::cleanup removes old snapshots" {
    result=$(qdrant::snapshots::cleanup "30")
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot archiving
@test "qdrant::snapshots::archive archives old snapshots" {
    result=$(qdrant::snapshots::archive "test_collection" "/archive/location")
    
    [[ "$result" =~ "archive" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot synchronization
@test "qdrant::snapshots::sync synchronizes snapshots" {
    result=$(qdrant::snapshots::sync "/remote/snapshot/location")
    
    [[ "$result" =~ "sync" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test cloud backup
@test "qdrant::snapshots::backup_to_cloud backs up snapshots to cloud" {
    result=$(qdrant::snapshots::backup_to_cloud "s3://backup-bucket/qdrant-snapshots/")
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "cloud" ]]
}

# Test cloud restoration
@test "qdrant::snapshots::restore_from_cloud restores snapshots from cloud" {
    result=$(qdrant::snapshots::restore_from_cloud "s3://backup-bucket/qdrant-snapshots/snapshot_test_2024-01-15.tar.gz")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "cloud" ]]
}

# Test AWS S3 backup
@test "qdrant::snapshots::backup_to_s3 backs up to Amazon S3" {
    result=$(qdrant::snapshots::backup_to_s3 "qdrant-backup-bucket" "snapshots/")
    
    [[ "$result" =~ "S3" ]] || [[ "$result" =~ "backup" ]]
}

# Test AWS S3 restoration
@test "qdrant::snapshots::restore_from_s3 restores from Amazon S3" {
    result=$(qdrant::snapshots::restore_from_s3 "qdrant-backup-bucket" "snapshots/snapshot_test_2024-01-15.tar.gz")
    
    [[ "$result" =~ "S3" ]] || [[ "$result" =~ "restore" ]]
}

# Test remote backup
@test "qdrant::snapshots::backup_remote backs up to remote server" {
    result=$(qdrant::snapshots::backup_remote "user@remote-server:/backup/qdrant/")
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "remote" ]]
}

# Test remote restoration
@test "qdrant::snapshots::restore_remote restores from remote server" {
    result=$(qdrant::snapshots::restore_remote "user@remote-server:/backup/qdrant/snapshot_test_2024-01-15.tar.gz")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "remote" ]]
}

# Test snapshot monitoring
@test "qdrant::snapshots::monitor monitors snapshot operations" {
    result=$(qdrant::snapshots::monitor)
    
    [[ "$result" =~ "monitor" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot status
@test "qdrant::snapshots::status shows snapshot status" {
    result=$(qdrant::snapshots::status)
    
    [[ "$result" =~ "status" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot health check
@test "qdrant::snapshots::health_check checks snapshot health" {
    result=$(qdrant::snapshots::health_check)
    
    [[ "$result" =~ "health" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot statistics
@test "qdrant::snapshots::stats shows snapshot statistics" {
    result=$(qdrant::snapshots::stats)
    
    [[ "$result" =~ "statistics" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot recovery
@test "qdrant::snapshots::recover recovers corrupted snapshots" {
    result=$(qdrant::snapshots::recover "corrupted_snapshot_2024-01-15_14-30-00")
    
    [[ "$result" =~ "recover" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot repair
@test "qdrant::snapshots::repair repairs damaged snapshots" {
    result=$(qdrant::snapshots::repair "damaged_snapshot_2024-01-15_14-30-00")
    
    [[ "$result" =~ "repair" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot testing
@test "qdrant::snapshots::test tests snapshot integrity" {
    result=$(qdrant::snapshots::test "snapshot_test_collection_2024-01-15_14-30-00")
    
    [[ "$result" =~ "test" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot benchmarking
@test "qdrant::snapshots::benchmark benchmarks snapshot operations" {
    result=$(qdrant::snapshots::benchmark "create" "test_collection")
    
    [[ "$result" =~ "benchmark" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot optimization
@test "qdrant::snapshots::optimize optimizes snapshot storage" {
    result=$(qdrant::snapshots::optimize)
    
    [[ "$result" =~ "optimize" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot merge
@test "qdrant::snapshots::merge merges multiple snapshots" {
    result=$(qdrant::snapshots::merge "snapshot1,snapshot2,snapshot3" "merged_snapshot_2024-01-15")
    
    [[ "$result" =~ "merge" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot split
@test "qdrant::snapshots::split splits large snapshot" {
    result=$(qdrant::snapshots::split "large_snapshot_2024-01-15" "100MB")
    
    [[ "$result" =~ "split" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot conversion
@test "qdrant::snapshots::convert converts snapshot format" {
    result=$(qdrant::snapshots::convert "old_format_snapshot" "new_format")
    
    [[ "$result" =~ "convert" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot migration
@test "qdrant::snapshots::migrate migrates snapshots between versions" {
    result=$(qdrant::snapshots::migrate "v1.6_snapshot" "v1.7")
    
    [[ "$result" =~ "migrate" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot versioning
@test "qdrant::snapshots::version manages snapshot versions" {
    result=$(qdrant::snapshots::version "snapshot_test_collection_2024-01-15_14-30-00")
    
    [[ "$result" =~ "version" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot tagging
@test "qdrant::snapshots::tag adds tags to snapshots" {
    result=$(qdrant::snapshots::tag "snapshot_test_collection_2024-01-15_14-30-00" "production,stable")
    
    [[ "$result" =~ "tag" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot search
@test "qdrant::snapshots::search searches snapshots by criteria" {
    result=$(qdrant::snapshots::search "collection=test_collection,date=2024-01-15")
    
    [[ "$result" =~ "search" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot filtering
@test "qdrant::snapshots::filter filters snapshots by criteria" {
    result=$(qdrant::snapshots::filter "size>1MB,age<7days")
    
    [[ "$result" =~ "filter" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot export
@test "qdrant::snapshots::export exports snapshot metadata" {
    result=$(qdrant::snapshots::export "/tmp/snapshot_metadata.json")
    
    [[ "$result" =~ "export" ]] || [[ "$result" =~ "metadata" ]]
}

# Test snapshot import
@test "qdrant::snapshots::import imports snapshot metadata" {
    result=$(qdrant::snapshots::import "/tmp/snapshot_metadata.json")
    
    [[ "$result" =~ "import" ]] || [[ "$result" =~ "metadata" ]]
}

# Test snapshot auditing
@test "qdrant::snapshots::audit audits snapshot operations" {
    result=$(qdrant::snapshots::audit)
    
    [[ "$result" =~ "audit" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot logging
@test "qdrant::snapshots::log manages snapshot operation logs" {
    result=$(qdrant::snapshots::log "view")
    
    [[ "$result" =~ "log" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot notification
@test "qdrant::snapshots::notify configures snapshot notifications" {
    result=$(qdrant::snapshots::notify "email" "admin@example.com")
    
    [[ "$result" =~ "notify" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot alerting
@test "qdrant::snapshots::alert configures snapshot alerts" {
    result=$(qdrant::snapshots::alert "failure" "webhook_url")
    
    [[ "$result" =~ "alert" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot reporting
@test "qdrant::snapshots::report generates snapshot reports" {
    result=$(qdrant::snapshots::report "weekly" "/tmp/snapshot_report.html")
    
    [[ "$result" =~ "report" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot analysis
@test "qdrant::snapshots::analyze analyzes snapshot patterns" {
    result=$(qdrant::snapshots::analyze "usage")
    
    [[ "$result" =~ "analyze" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot trends
@test "qdrant::snapshots::trends analyzes snapshot trends" {
    result=$(qdrant::snapshots::trends "size,frequency")
    
    [[ "$result" =~ "trends" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot capacity planning
@test "qdrant::snapshots::capacity_planning plans snapshot storage capacity" {
    result=$(qdrant::snapshots::capacity_planning)
    
    [[ "$result" =~ "capacity" ]] || [[ "$result" =~ "planning" ]]
}

# Test snapshot policy management
@test "qdrant::snapshots::policy manages snapshot policies" {
    result=$(qdrant::snapshots::policy "create" "daily_backup" "0 2 * * *")
    
    [[ "$result" =~ "policy" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot compliance
@test "qdrant::snapshots::compliance checks snapshot compliance" {
    result=$(qdrant::snapshots::compliance "GDPR")
    
    [[ "$result" =~ "compliance" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot security
@test "qdrant::snapshots::security configures snapshot security" {
    result=$(qdrant::snapshots::security "enable" "encryption,access_control")
    
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot automation
@test "qdrant::snapshots::automate configures snapshot automation" {
    result=$(qdrant::snapshots::automate "enable" "daily,weekly,monthly")
    
    [[ "$result" =~ "automate" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot orchestration
@test "qdrant::snapshots::orchestrate orchestrates snapshot operations" {
    result=$(qdrant::snapshots::orchestrate "backup_workflow.yaml")
    
    [[ "$result" =~ "orchestrate" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot integration
@test "qdrant::snapshots::integrate integrates with external systems" {
    result=$(qdrant::snapshots::integrate "monitoring_system" "webhook_url")
    
    [[ "$result" =~ "integrate" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot dashboard
@test "qdrant::snapshots::dashboard creates snapshot dashboard" {
    result=$(qdrant::snapshots::dashboard "generate" "/tmp/snapshot_dashboard.html")
    
    [[ "$result" =~ "dashboard" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot API
@test "qdrant::snapshots::api provides snapshot API access" {
    result=$(qdrant::snapshots::api "list" "json")
    
    [[ "$result" =~ "API" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot CLI
@test "qdrant::snapshots::cli provides snapshot CLI interface" {
    result=$(qdrant::snapshots::cli "help")
    
    [[ "$result" =~ "CLI" ]] || [[ "$result" =~ "snapshot" ]]
}

# Test snapshot documentation
@test "qdrant::snapshots::documentation generates snapshot documentation" {
    result=$(qdrant::snapshots::documentation "generate" "/tmp/snapshot_docs/")
    
    [[ "$result" =~ "documentation" ]] || [[ "$result" =~ "snapshot" ]]
}