#!/usr/bin/env bash
set -euo pipefail

# Get the directory containing this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Find Vrooli root
VROOLI_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Source the CLI install utility
source "${VROOLI_ROOT}/scripts/lib/utils/cli-install.sh"

# Install the CLI
install_cli "${SCRIPT_DIR}/math-tools" "math-tools"
