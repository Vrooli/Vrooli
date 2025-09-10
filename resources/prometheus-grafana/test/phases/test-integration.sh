#!/usr/bin/env bash
# Integration tests for Prometheus + Grafana - End-to-end functionality

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

source "${RESOURCE_DIR}/lib/test.sh"

# Run integration tests with 120-second timeout
timeout 120 run_integration_tests