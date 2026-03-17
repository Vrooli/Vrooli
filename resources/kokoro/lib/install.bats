#!/usr/bin/env bats
# Tests for Kokoro installation functions

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "kokoro"

    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    KOKORO_DIR="$(dirname "$SCRIPT_DIR")"

    # Source manage.sh to get install functions
    source "${KOKORO_DIR}/cli.sh"

    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_KOKORO_DIR="$KOKORO_DIR"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup

    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    KOKORO_DIR="${SETUP_FILE_KOKORO_DIR}"

    # Set test environment
    export KOKORO_CONTAINER_NAME="kokoro-test"
    export KOKORO_IMAGE="ghcr.io/remsky/kokoro-fastapi-gpu:latest"
    export KOKORO_CPU_IMAGE="ghcr.io/remsky/kokoro-fastapi-cpu:latest"
    export KOKORO_GPU_ENABLED="no"
    export FORCE="no"
    export YES="no"

    # Export config functions
    defaults::export_config
    messages::export_messages
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# === Installation Tests ===
@test "kokoro::install checks prerequisites" {
    # Mock Docker check to succeed
    common::check_docker() { return 0; }
    docker() {
        case "$1" in
            "info") return 0 ;;
            "pull") return 0 ;;
            *) return 0 ;;
        esac
    }

    run kokoro::install
    [ "$status" -eq 0 ]
}

@test "kokoro::install is idempotent" {
    # Mock container already exists
    common::container_exists() { return 0; }
    common::is_running() { return 0; }
    export FORCE="no"

    run kokoro::install
    [ "$status" -eq 0 ]
    [[ "$output" == *"already installed"* ]] || [[ "$output" == *"already exists"* ]]
}

@test "kokoro::install respects force flag" {
    # Mock container exists
    common::container_exists() { return 0; }
    kokoro::stop() { return 0; }
    kokoro::uninstall() { return 0; }
    docker() {
        case "$1" in
            "pull") echo "Pulling image..." ;;
            "run") echo "Creating container..." ;;
            *) return 0 ;;
        esac
    }
    export FORCE="yes"

    run kokoro::install
    [ "$status" -eq 0 ]
}

@test "kokoro::install handles failures gracefully" {
    # Mock Docker pull failure
    docker() {
        case "$1" in
            "pull") return 1 ;;
            *) return 0 ;;
        esac
    }

    run kokoro::install
    [ "$status" -eq 1 ]
}

@test "kokoro::uninstall removes cleanly" {
    # Mock container exists and running
    common::container_exists() { return 0; }
    common::is_running() { return 0; }
    kokoro::stop() { return 0; }
    docker() {
        case "$1" in
            "rm") echo "Container removed" ;;
            *) return 0 ;;
        esac
    }
    export REMOVE_DATA="no"

    run kokoro::uninstall
    [ "$status" -eq 0 ]
    [[ "$output" == *"removed"* ]]
}

@test "kokoro::install selects CPU image when GPU not available" {
    # Mock no GPU available
    kokoro::is_gpu_available() { return 1; }
    export KOKORO_GPU_ENABLED="yes"

    docker() {
        if [[ "$1" == "pull" ]]; then
            # Verify CPU image is being pulled
            [[ "$2" == *"cpu"* ]] && echo "Pulling CPU image"
            return 0
        fi
        return 0
    }

    run kokoro::install
    [ "$status" -eq 0 ]
    [[ "$output" == *"CPU"* ]] || [[ "$output" == *"cpu"* ]]
}

@test "kokoro::install creates necessary directories" {
    # Mock directory creation
    kokoro::create_directories() {
        echo "Creating directories"
        return 0
    }
    docker() { return 0; }

    run kokoro::install
    [ "$status" -eq 0 ]
    [[ "$output" == *"directories"* ]]
}
