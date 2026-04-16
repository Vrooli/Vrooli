#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
REPO_ROOT="$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)"

"${REPO_ROOT}/packages/cli-core/install.sh" "scenarios/ai-chatbot-manager/cli" --name "ai-chatbot-manager" --manifest "scenarios/ai-chatbot-manager/.vrooli/service.json"
