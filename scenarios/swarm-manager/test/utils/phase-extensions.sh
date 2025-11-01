#!/usr/bin/env bash
# Additional helpers layered on top of the shared phase utilities so legacy
# phase scripts can log progress without reimplementing counters.
set -euo pipefail

# Informational log that does not mutate counters.
testing::phase::log() {
    local message="${1:-}"
    if [ -n "$message" ]; then
        log::info "$message"
    fi
}

# Successful checkpoint log.
testing::phase::success() {
    local message="${1:-}"
    if [ -n "$message" ]; then
        log::success "$message"
    fi
}

# Warning helper that also increments the warning counter.
testing::phase::warn() {
    local message="${1:-}"
    testing::phase::add_warning "$message"
}

# Error helper that records an error without immediately exiting.
testing::phase::error() {
    local message="${1:-}"
    testing::phase::add_error "$message"
}

# Required binary check used by the legacy dependency script.
testing::phase::check_binary() {
    local binary="$1"
    local description="${2:-}"

    if command -v "$binary" >/dev/null 2>&1; then
        local detail="${description:+ - ${description}}"
        log::success "Binary '$binary' available${detail}"
        return 0
    fi

    local detail="${description:+ (${description})}"
    testing::phase::add_error "Required binary '$binary' not found${detail}"
    testing::phase::end_with_summary "Missing required binary: $binary"
}

export -f testing::phase::log
export -f testing::phase::success
export -f testing::phase::warn
export -f testing::phase::error
export -f testing::phase::check_binary
