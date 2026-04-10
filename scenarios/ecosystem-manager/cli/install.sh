#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  cat <<'EOF'
Usage: install.sh

Installs the ecosystem-manager CLI binary into the Vrooli bin directory.
EOF
  exit 0
fi

if [[ "${1:-}" == "version" || "${1:-}" == "--version" || "${1:-}" == "-v" ]]; then
  echo "ecosystem-manager-cli 1.0.0"
  exit 0
fi

if [[ "$#" -gt 0 ]]; then
  echo "Error: unknown argument(s): $*" >&2
  echo "Run 'install.sh --help' for usage." >&2
  exit 1
fi

"${APP_ROOT}/packages/cli-core/install.sh" "scenarios/ecosystem-manager/cli" --name "ecosystem-manager"
