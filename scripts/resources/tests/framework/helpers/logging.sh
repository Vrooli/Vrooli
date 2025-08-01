#!/bin/bash
# ====================================================================
# Shared Logging Functions
# ====================================================================
#
# Common logging functions used across all test scripts. Provides
# consistent output formatting and color coding.
#
# Functions:
#   - log_header()    - Header messages
#   - log_info()      - Info messages  
#   - log_success()   - Success messages
#   - log_warning()   - Warning messages
#   - log_error()     - Error messages
#   - log_debug()     - Debug messages (only shown if VERBOSE=true)
#
# ====================================================================

# Colors for output (only if terminal supports color)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    PURPLE='\033[0;35m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m' # No Color
else
    RED='' GREEN='' YELLOW='' BLUE='' PURPLE='' CYAN='' BOLD='' NC=''
fi

# Logging functions
log_header() {
    echo -e "${BOLD}${BLUE}[HEADER]${NC} ${BOLD}$1${NC}"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC}    $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} ✅ $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC}    ⚠️  $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC}   ❌ $1"
}

log_debug() {
    if [[ "${VERBOSE:-false}" == "true" ]]; then
        echo -e "${PURPLE}[DEBUG]${NC}   🔍 $1"
    fi
}

# Step logging for test progress
log_step() {
    local step="$1"
    local description="$2"
    echo -e "${CYAN}[STEP $step]${NC} $description"
}