#!/usr/bin/env bash
set -euo pipefail

CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$CLI_DIR/../scripts/lib/utils/cli-install.sh"

install_cli "$CLI_DIR/CLI_NAME_PLACEHOLDER" "CLI_NAME_PLACEHOLDER"