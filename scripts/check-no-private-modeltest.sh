#!/usr/bin/env bash
set -euo pipefail

found="$(find scenarios templates -type d -name modeltest -not -path 'packages/*' -print)"
if [[ -n "$found" ]]; then
  echo "private modeltest copies found; import @vrooli/flow-runtime instead:" >&2
  echo "$found" >&2
  exit 1
fi

if rg -n '(/|\")modeltest/(formal|matrix|traces)' scenarios templates --glob '*.go' --glob '*.ts' --glob '*.tsx' --glob '!**/node_modules/**' --glob '!**/docs/**' >/dev/null; then
  echo "private modeltest import paths found; use the published flow runtime" >&2
  exit 1
fi
