#!/usr/bin/env bash
set -euo pipefail
RESOURCE_DIR="$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../.." && builtin pwd)"
if ! (cd "${RESOURCE_DIR}/cli" && go run . models list --json >/dev/null 2>&1); then
  echo "SKIP: claude-code live model catalog unavailable"
  exit 0
fi
(cd "${RESOURCE_DIR}/cli" && go run . policy validate --against-live --json >/dev/null)
model="$(cd "${RESOURCE_DIR}/cli" && go run . policy resolve --role code.default --json | jq -r .model)"
(cd "${RESOURCE_DIR}/cli" && go run . models resolve --model "${model}" --json >/dev/null)
echo "PASS: claude-code model discovery and live policy validation"
