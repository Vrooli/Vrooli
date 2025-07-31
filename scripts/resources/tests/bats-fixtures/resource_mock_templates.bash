#!/usr/bin/env bash
# Lightweight Mock Templates for Resource Tests
# These templates provide fast, in-memory mocks for common resource testing patterns

#######################################
# Docker Mock Template - Ultra Fast
# Provides consistent Docker command simulation without file operations
#######################################
create_docker_mock() {
    local behavior="${1:-running}"  # running, stopped, not_installed, unhealthy
    
    docker() {
        case "$1" in
            "ps")
                case "$behavior" in
                    "running") echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES\nabc123        test:latest   \"./start\"   1 min     Up 1 min  0.0.0.0:8080->8080/tcp   test-container" ;;
                    "stopped"|"not_installed") echo "CONTAINER ID   IMAGE   COMMAND   CREATED   STATUS   PORTS   NAMES" ;;
                    *) echo "CONTAINER ID   IMAGE   COMMAND   CREATED   STATUS   PORTS   NAMES" ;;
                esac ;;
            "container")
                case "$2" in
                    "inspect")
                        case "$behavior" in
                            "running") echo "true" ;;
                            "stopped") echo "false" ;;
                            "not_installed") return 1 ;;
                            *) echo "unknown" ;;
                        esac ;;
                    *) return 0 ;;
                esac ;;
            "stats") echo "CONTAINER     CPU %     MEM USAGE / LIMIT     MEM %     NET I/O     BLOCK I/O\ntest-container     0.50%     100MiB / 2GiB        5.00%     1kB / 2kB   10MB / 20MB" ;;
            "logs") echo "$(date): [INFO] Service started successfully" ;;
            "exec") echo "Mock exec output" ;;
            "inspect") echo '[{"Config":{"Env":["PATH=/usr/bin"],"Image":"test:latest"},"State":{"Status":"running","Health":{"Status":"healthy"}}}]' ;;
            *) return 0 ;;
        esac
    }
    export -f docker
}

#######################################
# HTTP Service Mock Template
# Simulates curl responses for API health checks
#######################################
create_http_mock() {
    local status="${1:-healthy}"  # healthy, unhealthy, unavailable
    
    curl() {
        case "$status" in
            "healthy")
                if [[ "$*" =~ "health" ]] || [[ "$*" =~ "status" ]]; then
                    echo '{"status":"ok","health":"healthy","version":"1.0.0"}'
                    return 0
                else
                    echo '{"result":"success"}'
                    return 0
                fi ;;
            "unhealthy")
                if [[ "$*" =~ "health" ]] || [[ "$*" =~ "status" ]]; then
                    echo '{"status":"error","health":"unhealthy"}'
                    return 1
                else
                    return 1
                fi ;;
            "unavailable") return 1 ;;
            *) echo '{"status":"unknown"}'; return 0 ;;
        esac
    }
    export -f curl
}

#######################################
# System Command Mock Template
# Mocks common system commands for consistent test results
#######################################
create_system_mocks() {
    # Mock jq for JSON processing
    jq() {
        case "$1" in
            "-r") shift; echo "mock-string-value" ;;
            ".status") echo "ok" ;;
            ".health") echo "healthy" ;;
            ".version") echo "1.0.0" ;;
            ".[]") echo "item1 item2 item3" ;;
            *) echo "null" ;;
        esac
    }
    
    # Mock systemctl for service management
    systemctl() {
        case "$2" in
            "status") echo "● mock-service - Mock Service\n   Active: active (running)" ;;
            "start"|"stop"|"restart"|"enable"|"disable") return 0 ;;
            *) return 0 ;;
        esac
    }
    
    # Mock netstat/ss for port checking
    netstat() { echo ""; }  # No ports in use
    ss() { echo ""; }       # No ports in use
    
    # Mock wget for downloads
    wget() {
        case "$*" in
            *"-O"*) touch "${@: -1}"; return 0 ;;  # Create the output file
            *) return 0 ;;
        esac
    }
    
    # Mock tar for archive operations
    tar() { return 0; }
    
    # Mock find for file searching
    find() {
        case "$*" in
            *"*.json"*) echo "/mock/path/file1.json /mock/path/file2.json" ;;
            *"*.log"*) echo "/mock/path/app.log" ;;
            *) echo "/mock/path/file" ;;
        esac
    }
    
    # Export all mocks
    export -f jq systemctl netstat ss wget tar find
}

#######################################
# Resource-Specific Mock Templates
#######################################

# Node-RED specific mocks
create_node_red_mocks() {
    create_docker_mock "running"
    create_http_mock "healthy"
    
    # Mock Node-RED specific API endpoints
    local original_curl=$(declare -f curl)
    curl() {
        if [[ "$*" =~ "settings" ]]; then
            echo '{"version":"3.0.2","userDir":"/data","httpRoot":"/","httpAdminRoot":"/admin"}'
        elif [[ "$*" =~ "flows" ]]; then
            echo '[{"id":"node1","type":"inject","name":"Test"}]'
        else
            eval "$original_curl"
        fi
    }
    export -f curl
}

# Ollama specific mocks
create_ollama_mocks() {
    create_http_mock "healthy"
    
    # Mock Ollama API endpoints
    curl() {
        if [[ "$*" =~ "api/tags" ]]; then
            echo '{"models":[{"name":"llama3.1:8b","size":4900000000},{"name":"deepseek-r1:8b","size":4700000000}]}'
        elif [[ "$*" =~ "api/generate" ]]; then
            echo '{"response":"Hello! How can I help you today?","done":true}'
        else
            echo '{"status":"ok"}'
        fi
        return 0
    }
    export -f curl
}

# Whisper specific mocks
create_whisper_mocks() {
    create_docker_mock "running"
    create_http_mock "healthy"
    
    # Mock file upload responses
    local original_curl=$(declare -f curl)
    curl() {
        if [[ "$*" =~ "transcribe" ]] || [[ "$*" =~ "translate" ]]; then
            echo '{"text":"This is a mock transcription result."}'
        else
            eval "$original_curl"
        fi
    }
    export -f curl
}

#######################################
# Fast Environment Setup
# Creates test environment in memory instead of filesystem
#######################################
setup_fast_test_env() {
    local resource_name="$1"
    
    # Set environment variables instead of reading config files
    export RESOURCE_NAME="$resource_name"
    export CONTAINER_NAME="${resource_name}-test"
    export RESOURCE_PORT="${RESOURCE_PORT:-8080}"
    export RESOURCE_HOST="${RESOURCE_HOST:-localhost}"
    export RESOURCE_URL="http://${RESOURCE_HOST}:${RESOURCE_PORT}"
    
    # Create logging functions
    log::info() { echo "[INFO] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::debug() { echo "[DEBUG] $*"; }
    log::header() { echo "=== $* ==="; }
    
    # Create flow control functions
    flow::confirm() { return 0; }  # Always confirm in tests
    flow::is_yes() { return 0; }
    
    # Export functions
    export -f log::info log::success log::warning log::error log::debug log::header
    export -f flow::confirm flow::is_yes
    
    # Set up system mocks
    create_system_mocks
    
    # Set up resource-specific mocks
    case "$resource_name" in
        "node-red") create_node_red_mocks ;;
        "ollama") create_ollama_mocks ;;
        "whisper") create_whisper_mocks ;;
        *) 
            create_docker_mock "running"
            create_http_mock "healthy"
            ;;
    esac
    
    echo "[FAST_ENV] Set up test environment for $resource_name"
}

#######################################
# Performance Test Helper
# Use this for tests that don't need full resource setup
#######################################
quick_test_setup() {
    local resource_name="$1"
    local mock_behavior="${2:-running}"
    
    # Ultra-minimal setup - just environment variables and mocks
    export RESOURCE_NAME="$resource_name"
    export CONTAINER_NAME="${resource_name}-test"
    
    # Basic logging
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    export -f log::info log::error
    
    # Basic mocks
    create_docker_mock "$mock_behavior"
    create_http_mock "healthy"
}

echo "[MOCK_TEMPLATES] Loaded lightweight mock templates"