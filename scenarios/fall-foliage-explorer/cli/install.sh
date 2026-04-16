#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
REPO_ROOT="$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)"

"${REPO_ROOT}/packages/cli-core/install.sh" "scenarios/fall-foliage-explorer/cli" --name "fall-foliage-explorer" --manifest "scenarios/fall-foliage-explorer/.vrooli/service.json"
