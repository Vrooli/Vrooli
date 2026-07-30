#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
if [[ -z "${SCRIPT_DIR}" ]]; then
  echo "failed to resolve the CLI script directory" >&2
  exit 1
fi
REPO_ROOT="$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)"

"${REPO_ROOT}/packages/cli-core/install.sh" "scenarios/agent-manager/cli" --name "agent-manager" --manifest "scenarios/agent-manager/.vrooli/service.json"
