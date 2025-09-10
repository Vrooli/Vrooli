#!/bin/bash
# JupyterHub Integration Tests - End-to-end functionality

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Use the CLI to run integration tests
"${RESOURCE_DIR}/cli.sh" test integration