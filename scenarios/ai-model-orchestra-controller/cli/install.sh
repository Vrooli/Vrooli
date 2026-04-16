#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
REPO_ROOT="$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)"

"${REPO_ROOT}/packages/cli-core/install.sh" "scenarios/ai-model-orchestra-controller/cli" --name "ai-model-orchestra-controller" --manifest "scenarios/ai-model-orchestra-controller/.vrooli/service.json"
