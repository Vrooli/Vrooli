#!/usr/bin/env bash
# Unit tests for Prometheus + Grafana - Library function validation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

source "${RESOURCE_DIR}/lib/test.sh"

# Run unit tests with 60-second timeout
timeout 60 run_unit_tests