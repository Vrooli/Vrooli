#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

"${APP_ROOT}/packages/cli-core/install.sh" "scenarios/landing-page-business-suite/cli" --name "landing-page-business-suite"
