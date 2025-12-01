#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scenarios/scenario-completeness-scoring/cli"
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

if ! command -v go >/dev/null 2>&1; then
    echo "Go toolchain is required to build the scenario-completeness-scoring CLI."
    exit 1
fi

echo "Building scenario-completeness-scoring CLI..."
(
    cd "$CLI_DIR"
    go build -o scenario-completeness-scoring .
)

install_cli "$CLI_DIR/scenario-completeness-scoring" "scenario-completeness-scoring"
