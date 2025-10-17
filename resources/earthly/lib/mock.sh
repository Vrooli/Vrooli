#!/bin/bash
# Earthly Resource - Mock Implementation for Testing
# Provides simulated responses when Earthly binary is not available

set -euo pipefail

# Mock Earthly version
mock_earthly_version() {
    echo "earthly version 0.8.15+mock"
}

# Mock build execution
mock_earthly_build() {
    echo "[Mock] Executing build target: $*"
    echo "[Mock] Build would run with Earthly binary"
    echo "[Mock] Success: Build completed"
    return 0
}

# Mock health check
mock_health_check() {
    echo "[Mock] Health check passed (simulated)"
    return 0
}

# Check if we should use mock
should_use_mock() {
    if ! command -v earthly &>/dev/null && [[ ! -f "${HOME}/.local/bin/earthly" ]]; then
        return 0  # Use mock
    else
        return 1  # Use real Earthly
    fi
}