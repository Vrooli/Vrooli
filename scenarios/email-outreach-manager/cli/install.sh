#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
APP_ROOT="${VROOLI_ROOT:-$(builtin cd "$SCRIPT_DIR/../../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

install_cli "$SCRIPT_DIR/email-outreach-manager" "email-outreach-manager"
