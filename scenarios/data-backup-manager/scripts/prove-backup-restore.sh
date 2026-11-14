#!/usr/bin/env bash
set -euo pipefail
# This proof intentionally never contacts the running DBM service, Kopia,
# credential authority, or production catalog. All inputs are synthetic.
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}/../api"
exec go test ./internal/sources ./internal/engine ./internal/restores -run 'TestCheckpoint|TestFilesystemRestore|TestProduction|TestVerify' -count=1
