#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

"${APP_ROOT}/packages/cli-core/install.sh" "resources/mail-in-a-box/cli" --name "resource-mail-in-a-box"
