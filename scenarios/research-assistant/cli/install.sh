#!/usr/bin/env bash
set -euo pipefail

# T using cached value or compute once (3 levels up: scenarios/research-assistant/cli/install.sh)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scenarios/research-assistant/cli"
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

install_cli "$CLI_DIR/research-assistant" "research-assistant"