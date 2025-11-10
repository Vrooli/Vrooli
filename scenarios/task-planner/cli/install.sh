#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scenarios/task-planner/cli"
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

install_cli "$CLI_DIR/task-planner" "task-planner"
