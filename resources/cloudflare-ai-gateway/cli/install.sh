#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
REPO_ROOT="$(builtin cd "${SCRIPT_DIR}/../../.." && builtin pwd)"

"${REPO_ROOT}/packages/cli-core/install.sh" "resources/cloudflare-ai-gateway/cli" --name "resource-cloudflare-ai-gateway" --manifest "resources/cloudflare-ai-gateway/resource.json"
