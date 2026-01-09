#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CLI_CORE="${REPO_ROOT}/packages/cli-core/install.sh"

if [[ ! -x "${CLI_CORE}" ]]; then
  echo "cli-core installer not found at ${CLI_CORE}" >&2
  exit 1
fi

"${CLI_CORE}" "${SCRIPT_DIR}" --name resource-sqlite "$@"
