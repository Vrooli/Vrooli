#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
# Navigate to Vrooli root (cli -> crypto-tools -> scenarios -> Vrooli)
APP_ROOT="${VROOLI_ROOT:-$(builtin cd "$SCRIPT_DIR/../../.." && builtin pwd)}"
CLI_DIR="$SCRIPT_DIR"

source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

install_cli "$CLI_DIR/crypto-tools" "crypto-tools"