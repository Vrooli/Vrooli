#!/bin/bash
# JupyterHub Smoke Tests - Quick validation (<30s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Use the CLI to run smoke tests
"${RESOURCE_DIR}/cli.sh" test smoke