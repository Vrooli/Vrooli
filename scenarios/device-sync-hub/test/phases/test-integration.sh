#!/usr/bin/env bash
# Integration tests for Device Sync Hub
# Comprehensive tests for API endpoints, WebSocket, and file operations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Run the existing comprehensive integration test
exec "${SCENARIO_DIR}/test/integration.sh"
