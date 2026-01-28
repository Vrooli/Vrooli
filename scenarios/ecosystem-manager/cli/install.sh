#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scenarios/ecosystem-manager/cli"
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

if [[ $# -gt 0 ]]; then
  "${CLI_DIR}/ecosystem-manager" "$@"
  exit $?
fi

install_cli "$CLI_DIR/ecosystem-manager" "ecosystem-manager"
