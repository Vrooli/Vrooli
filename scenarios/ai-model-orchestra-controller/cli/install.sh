#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scenarios/ai-model-orchestra-controller/cli"
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

install_cli "$CLI_DIR/ai-model-orchestra-controller" "ai-model-orchestra-controller"
