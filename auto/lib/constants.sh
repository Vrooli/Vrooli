#!/usr/bin/env bash
# Constants used throughout the auto/ loop system
# Provides consistent configuration values across all modules

# Source var.sh with relative path first
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../scripts/lib/utils/var.sh"

# Prevent multiple sourcing
if [[ -n "${_AUTO_CONSTANTS_SOURCED:-}" ]]; then
    return 0
fi
readonly _AUTO_CONSTANTS_SOURCED=1

# Default configuration values
readonly DEFAULT_INTERVAL_SECONDS=300
readonly DEFAULT_MAX_TURNS=80
readonly DEFAULT_TIMEOUT=2700
readonly DEFAULT_MAX_CONCURRENT_WORKERS=3
readonly DEFAULT_MAX_TCP_CONNECTIONS=30
readonly DEFAULT_ROTATE_KEEP=5

# Default behavior configuration
readonly DEFAULT_LOOP_TCP_FILTER="claude|anthropic|resource-claude-code"
readonly DEFAULT_OLLAMA_SUMMARY_MODEL="llama3.2:3b"
readonly DEFAULT_ULTRA_THINK_PREFIX="Ultra think. "
readonly DEFAULT_ALLOWED_TOOLS="Read,Write,Edit,Bash,LS,Glob,Grep"
readonly DEFAULT_SKIP_PERMISSIONS="yes"

# File size and line limits
readonly LOG_MAX_BYTES=52428800              # 50MB in bytes
readonly EVENTS_MAX_LINES=200000             # Maximum lines in events.ndjson
readonly ITERATION_LOG_MAX_LINES=10000       # Maximum lines per iteration log
readonly ITERATION_LOG_TAIL_LINES=200       # Lines to include in main log from iteration

# Event processing limits
readonly EVENTS_TAIL_SIZE=2000               # Lines to process from events file
readonly DEFAULT_RECENT_EVENTS=10            # Default number of recent events to show

# Process management timeouts
readonly PID_FILE_WAIT_ITERATIONS=30         # Wait iterations for PID file creation (x0.1s)
readonly PROCESS_TERMINATION_WAIT=10         # Wait iterations for process termination (x1s)
readonly WORKER_KILL_AFTER_SECONDS=30        # Seconds before SIGKILL after SIGTERM
readonly SLEEP_CHUNK_SIZE=10                 # Maximum seconds to sleep in one chunk (for interruptibility)

# Exit codes specific to auto/ system (extending scripts/lib/utils/exit_codes.sh)
readonly EXIT_CONFIGURATION_ERROR=150        # Auto-specific configuration error
readonly EXIT_WORKER_UNAVAILABLE=151         # Worker/dependencies unavailable

# File descriptor numbers for file locking
readonly FD_EVENTS_LOCK=200                  # File descriptor for events file lock
readonly FD_PIDS_LOCK=201                     # File descriptor for PIDs file lock

# Export all constants
export DEFAULT_INTERVAL_SECONDS DEFAULT_MAX_TURNS DEFAULT_TIMEOUT
export DEFAULT_MAX_CONCURRENT_WORKERS DEFAULT_MAX_TCP_CONNECTIONS DEFAULT_ROTATE_KEEP
export DEFAULT_LOOP_TCP_FILTER DEFAULT_OLLAMA_SUMMARY_MODEL DEFAULT_ULTRA_THINK_PREFIX
export DEFAULT_ALLOWED_TOOLS DEFAULT_SKIP_PERMISSIONS
export LOG_MAX_BYTES EVENTS_MAX_LINES ITERATION_LOG_MAX_LINES ITERATION_LOG_TAIL_LINES
export EVENTS_TAIL_SIZE DEFAULT_RECENT_EVENTS
export PID_FILE_WAIT_ITERATIONS PROCESS_TERMINATION_WAIT WORKER_KILL_AFTER_SECONDS SLEEP_CHUNK_SIZE
export EXIT_CONFIGURATION_ERROR EXIT_WORKER_UNAVAILABLE
export FD_EVENTS_LOCK FD_PIDS_LOCK

# Helper function to get constant value with fallback
constants::get() {
    local constant_name="$1"
    local default_value="${2:-}"
    
    if [[ -n "${!constant_name:-}" ]]; then
        echo "${!constant_name}"
    else
        echo "$default_value"
    fi
}

# Export helper function
export -f constants::get