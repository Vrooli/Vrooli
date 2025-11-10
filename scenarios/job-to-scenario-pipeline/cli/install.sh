#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/scenarios/job-scenario-pipeline/cli"
source "${APP_ROOT}/scripts/lib/utils/cli-install.sh"

install_cli "$CLI_DIR/job-scenario-pipeline" "job-scenario-pipeline"
