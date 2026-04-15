#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
REPO_ROOT="$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)"
CLI_DIR="${REPO_ROOT}/scenarios/scenario-to-extension/cli"
source "${REPO_ROOT}/scripts/lib/utils/cli-install.sh"

install_cli "$CLI_DIR/scenario-to-extension" "scenario-to-extension"
