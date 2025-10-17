#!/usr/bin/env bash
# Apache Airflow Resource - Main Test Runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"

# Delegate to the main test implementation
"${RESOURCE_DIR}/lib/test.sh" "$@"