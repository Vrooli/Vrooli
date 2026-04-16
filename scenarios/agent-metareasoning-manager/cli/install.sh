#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
REPO_ROOT="$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)"

"${REPO_ROOT}/packages/cli-core/install.sh" "scenarios/agent-metareasoning-manager/cli" --name "agent-metareasoning-manager" --manifest "scenarios/agent-metareasoning-manager/.vrooli/service.json"
