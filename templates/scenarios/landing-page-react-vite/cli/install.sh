#!/usr/bin/env bash
set -euo pipefail

MODULE_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
REPO_ROOT="$(builtin cd "${MODULE_DIR}/../../.." && builtin pwd)"

"${REPO_ROOT}/packages/cli-core/install.sh" "${MODULE_DIR}" --manifest "scenarios/{{SCENARIO_ID}}/.vrooli/service.json"
