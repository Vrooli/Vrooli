#!/usr/bin/env bats
# Tests for Whisper installation functions

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "whisper"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    WHISPER_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Source manage.sh to get install functions
    source "${WHISPER_DIR}/manage.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_WHISPER_DIR="$WHISPER_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    WHISPER_DIR="${SETUP_FILE_WHISPER_DIR}"
    
    # Set test environment
    export WHISPER_CONTAINER_NAME="whisper-test"
    export WHISPER_IMAGE="openai/whisper:latest"
    export WHISPER_CPU_IMAGE="openai/whisper-cpu:latest"
    export WHISPER_GPU_ENABLED="no"
    export FORCE="no"
    export YES="no"
    
    # Export config functions
    whisper::export_config
    whisper::export_messages
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# === Installation Tests ===
@test "whisper::install checks prerequisites" {
    # Mock Docker check to succeed
    whisper::check_docker() { return 0; }
    docker() {
        case "$1" in
            "info") return 0 ;;
            "pull") return 0 ;;
            *) return 0 ;;
        esac
    }
    
    run whisper::install
    [ "$status" -eq 0 ]
}

@test "whisper::install is idempotent" {
    # Mock container already exists
    whisper::container_exists() { return 0; }
    whisper::is_running() { return 0; }
    export FORCE="no"
    
    run whisper::install
    [ "$status" -eq 0 ]
    [[ "$output" == *"already installed"* ]] || [[ "$output" == *"already exists"* ]]
}

@test "whisper::install respects force flag" {
    # Mock container exists
    whisper::container_exists() { return 0; }
    whisper::stop() { return 0; }
    whisper::uninstall() { return 0; }
    docker() {
        case "$1" in
            "pull") echo "Pulling image..." ;;
            "run") echo "Creating container..." ;;
            *) return 0 ;;
        esac
    }
    export FORCE="yes"
    
    run whisper::install
    [ "$status" -eq 0 ]
}

@test "whisper::install handles failures gracefully" {
    # Mock Docker pull failure
    docker() {
        case "$1" in
            "pull") return 1 ;;
            *) return 0 ;;
        esac
    }
    
    run whisper::install
    [ "$status" -eq 1 ]
}

@test "whisper::uninstall removes cleanly" {
    # Mock container exists and running
    whisper::container_exists() { return 0; }
    whisper::is_running() { return 0; }
    whisper::stop() { return 0; }
    docker() {
        case "$1" in
            "rm") echo "Container removed" ;;
            *) return 0 ;;
        esac
    }
    export REMOVE_DATA="no"
    
    run whisper::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" == *"removed"* ]]
}

@test "whisper::install selects CPU image when GPU not available" {
    # Mock no GPU available
    whisper::is_gpu_available() { return 1; }
    export WHISPER_GPU_ENABLED="yes"
    
    docker() {
        if [[ "$1" == "pull" ]]; then
            # Verify CPU image is being pulled
            [[ "$2" == *"cpu"* ]] && echo "Pulling CPU image"
            return 0
        fi
        return 0
    }
    
    run whisper::install
    [ "$status" -eq 0 ]
    [[ "$output" == *"CPU"* ]] || [[ "$output" == *"cpu"* ]]
}

@test "whisper::install creates necessary directories" {
    # Mock directory creation
    whisper::create_directories() {
        echo "Creating directories"
        return 0
    }
    docker() { return 0; }
    
    run whisper::install
    [ "$status" -eq 0 ]
    [[ "$output" == *"directories"* ]]
}

@test "whisper::install validates configuration before proceeding" {
    # Mock invalid port
    system::is_port_in_use() { return 0; }  # Port in use
    
    run whisper::install
    [ "$status" -eq 1 ]
    [[ "$output" == *"port"* ]] || [[ "$output" == *"Port"* ]]
}