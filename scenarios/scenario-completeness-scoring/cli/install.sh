#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scenarios/scenario-completeness-scoring/cli"

if ! command -v go >/dev/null 2>&1; then
    echo "Go toolchain is required to build the scenario-completeness-scoring CLI."
    exit 1
fi

echo "Building and installing scenario-completeness-scoring CLI..."
(
    cd "$APP_ROOT/packages/cli-core"
    go run ./cmd/cli-installer --module "$CLI_DIR" --name scenario-completeness-scoring --install-dir "${HOME}/.vrooli/bin"
)
