#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scripts/scenarios/templates/full/cli"
source "$CLI_DIR/../scripts/lib/utils/cli-install.sh"

install_cli "$CLI_DIR/CLI_NAME_PLACEHOLDER" "CLI_NAME_PLACEHOLDER"