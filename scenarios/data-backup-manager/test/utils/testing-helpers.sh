#!/bin/bash
# Compatibility wrapper functions for testing::phase:: namespace
# These functions provide a consistent interface for test logging

# Wrapper for log output
testing::phase::log() {
    echo "ℹ️  $*"
}

# Wrapper for success messages
testing::phase::success() {
    echo "✅ $*"
}

# Wrapper for error messages
testing::phase::error() {
    echo "❌ $*" >&2
    TESTING_PHASE_ERROR_COUNT=$((TESTING_PHASE_ERROR_COUNT + 1))
}

# Wrapper for warning messages
testing::phase::warn() {
    echo "⚠️  $*"
    TESTING_PHASE_WARNING_COUNT=$((TESTING_PHASE_WARNING_COUNT + 1))
}

# Export functions for subshells
export -f testing::phase::log
export -f testing::phase::success
export -f testing::phase::error
export -f testing::phase::warn
