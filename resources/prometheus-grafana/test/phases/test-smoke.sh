#!/usr/bin/env bash
# Smoke tests for Prometheus + Grafana - Quick health validation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

source "${RESOURCE_DIR}/lib/test.sh"

# Run smoke tests with 30-second timeout (v2.0 requirement)
timeout 30 run_smoke_tests