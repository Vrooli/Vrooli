#!/usr/bin/env bash
# Judge0 Resource Configuration Defaults
# This file defines all configuration constants for the Judge0 code execution service

# ============================================================================
# SERVICE CONFIGURATION
# ============================================================================
export JUDGE0_SERVICE_NAME="judge0"
export JUDGE0_DISPLAY_NAME="Judge0 Code Execution"
export JUDGE0_DESCRIPTION="Secure sandboxed code execution supporting 60+ programming languages"
export JUDGE0_CATEGORY="execution"

# ============================================================================
# DOCKER CONFIGURATION
# ============================================================================
export JUDGE0_IMAGE="judge0/judge0"
export JUDGE0_VERSION="1.13.1"
export JUDGE0_CONTAINER_NAME="vrooli-judge0-server"
export JUDGE0_WORKERS_NAME="vrooli-judge0-workers"
export JUDGE0_NETWORK_NAME="vrooli-judge0-network"
export JUDGE0_VOLUME_NAME="vrooli-judge0-data"

# ============================================================================
# PORT CONFIGURATION
# ============================================================================
export JUDGE0_PORT="2358"
export JUDGE0_BASE_URL="http://localhost:${JUDGE0_PORT}"

# ============================================================================
# SECURITY CONFIGURATION
# ============================================================================
# Resource limits per submission
export JUDGE0_CPU_TIME_LIMIT="5"         # seconds
export JUDGE0_WALL_TIME_LIMIT="10"       # seconds
export JUDGE0_MEMORY_LIMIT="262144"      # KB (256MB)
export JUDGE0_MAX_PROCESSES="30"
export JUDGE0_MAX_FILE_SIZE="5120"       # KB (5MB)
export JUDGE0_STACK_LIMIT="262144"       # KB (256MB)

# Security features
export JUDGE0_ENABLE_NETWORK="false"     # Disable network in execution containers
export JUDGE0_ENABLE_CALLBACKS="false"   # Disable external callbacks
export JUDGE0_ENABLE_SUBMISSION_DELETE="true"
export JUDGE0_ENABLE_BATCHED_SUBMISSIONS="true"
export JUDGE0_ENABLE_WAIT_RESULT="true"

# ============================================================================
# PERFORMANCE CONFIGURATION
# ============================================================================
export JUDGE0_WORKERS_COUNT="2"
export JUDGE0_MAX_QUEUE_SIZE="100"
export JUDGE0_CPU_LIMIT="2"              # CPUs for server
export JUDGE0_MEMORY_LIMIT_DOCKER="2G"   # Memory for server
export JUDGE0_WORKER_CPU_LIMIT="1"       # CPUs per worker
export JUDGE0_WORKER_MEMORY_LIMIT="1G"   # Memory per worker

# ============================================================================
# HEALTH CHECK CONFIGURATION
# ============================================================================
export JUDGE0_HEALTH_ENDPOINT="/system_info"
export JUDGE0_HEALTH_INTERVAL="60000"    # milliseconds
export JUDGE0_HEALTH_TIMEOUT="5000"      # milliseconds
export JUDGE0_STARTUP_WAIT="30"          # seconds to wait for startup

# ============================================================================
# API CONFIGURATION
# ============================================================================
export JUDGE0_API_PREFIX="/api/v1"
export JUDGE0_ENABLE_AUTHENTICATION="true"
export JUDGE0_API_KEY_LENGTH="32"

# ============================================================================
# PATHS AND DIRECTORIES
# ============================================================================
export JUDGE0_DATA_DIR="${HOME}/.vrooli/resources/judge0"
export JUDGE0_CONFIG_DIR="${JUDGE0_DATA_DIR}/config"
export JUDGE0_LOGS_DIR="${JUDGE0_DATA_DIR}/logs"
export JUDGE0_SUBMISSIONS_DIR="${JUDGE0_DATA_DIR}/submissions"

# ============================================================================
# LOGGING CONFIGURATION
# ============================================================================
export JUDGE0_LOG_LEVEL="info"
export JUDGE0_LOG_MAX_SIZE="100M"
export JUDGE0_LOG_MAX_FILES="5"

# ============================================================================
# SUPPORTED LANGUAGES (subset of 60+ available)
# ============================================================================
# These are enabled by default - full list available via API
export JUDGE0_DEFAULT_LANGUAGES=(
    "javascript:93"      # JavaScript (Node.js 18.15.0)
    "typescript:94"      # TypeScript (5.0.3)
    "python:92"          # Python (3.11.2)
    "python2:70"         # Python (2.7.18)
    "go:95"              # Go (1.20.2)
    "rust:73"            # Rust (1.68.2)
    "java:91"            # Java (JDK 19.0.2)
    "cpp:105"            # C++ (GCC 12.2.0)
    "c:104"              # C (GCC 12.2.0)
    "csharp:51"          # C# (.NET 7.0)
    "ruby:72"            # Ruby (3.2.1)
    "php:68"             # PHP (8.2.3)
    "swift:83"           # Swift (5.8.0)
    "kotlin:78"          # Kotlin (1.8.20)
    "r:80"               # R (4.2.3)
    "bash:46"            # Bash (5.2.15)
    "sql:82"             # SQL (SQLite 3.40.1)
)

# ============================================================================
# HELPER FUNCTIONS
# ============================================================================

#######################################
# Export all Judge0 configuration variables
#######################################
judge0::export_config() {
    # Service info
    export JUDGE0_SERVICE_NAME
    export JUDGE0_DISPLAY_NAME
    export JUDGE0_DESCRIPTION
    export JUDGE0_CATEGORY
    
    # Docker config
    export JUDGE0_IMAGE
    export JUDGE0_VERSION
    export JUDGE0_CONTAINER_NAME
    export JUDGE0_WORKERS_NAME
    export JUDGE0_NETWORK_NAME
    export JUDGE0_VOLUME_NAME
    
    # Port config
    export JUDGE0_PORT
    export JUDGE0_BASE_URL
    
    # Security config
    export JUDGE0_CPU_TIME_LIMIT
    export JUDGE0_WALL_TIME_LIMIT
    export JUDGE0_MEMORY_LIMIT
    export JUDGE0_MAX_PROCESSES
    export JUDGE0_MAX_FILE_SIZE
    export JUDGE0_STACK_LIMIT
    export JUDGE0_ENABLE_NETWORK
    export JUDGE0_ENABLE_CALLBACKS
    export JUDGE0_ENABLE_SUBMISSION_DELETE
    export JUDGE0_ENABLE_BATCHED_SUBMISSIONS
    export JUDGE0_ENABLE_WAIT_RESULT
    
    # Performance config
    export JUDGE0_WORKERS_COUNT
    export JUDGE0_MAX_QUEUE_SIZE
    export JUDGE0_CPU_LIMIT
    export JUDGE0_MEMORY_LIMIT_DOCKER
    export JUDGE0_WORKER_CPU_LIMIT
    export JUDGE0_WORKER_MEMORY_LIMIT
    
    # Health check config
    export JUDGE0_HEALTH_ENDPOINT
    export JUDGE0_HEALTH_INTERVAL
    export JUDGE0_HEALTH_TIMEOUT
    export JUDGE0_STARTUP_WAIT
    
    # API config
    export JUDGE0_API_PREFIX
    export JUDGE0_ENABLE_AUTHENTICATION
    export JUDGE0_API_KEY_LENGTH
    
    # Paths
    export JUDGE0_DATA_DIR
    export JUDGE0_CONFIG_DIR
    export JUDGE0_LOGS_DIR
    export JUDGE0_SUBMISSIONS_DIR
    
    # Logging
    export JUDGE0_LOG_LEVEL
    export JUDGE0_LOG_MAX_SIZE
    export JUDGE0_LOG_MAX_FILES
}

#######################################
# Get language ID from name
# Arguments:
#   $1 - Language name
# Returns:
#   Language ID or empty string
#######################################
judge0::get_language_id() {
    local lang_name="$1"
    
    for lang_entry in "${JUDGE0_DEFAULT_LANGUAGES[@]}"; do
        local name="${lang_entry%%:*}"
        local id="${lang_entry##*:}"
        
        if [[ "$name" == "$lang_name" ]]; then
            echo "$id"
            return 0
        fi
    done
    
    return 1
}