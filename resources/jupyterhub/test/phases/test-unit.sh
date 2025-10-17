#!/bin/bash
# JupyterHub Unit Tests - Library function validation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Use the CLI to run unit tests
"${RESOURCE_DIR}/cli.sh" test unit