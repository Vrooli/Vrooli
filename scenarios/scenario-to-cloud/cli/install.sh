#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

INSTALL_DIR="${INSTALL_DIR:-${HOME}/.vrooli/bin}"
REAL_NAME="scenario-to-cloud.bin"

"${APP_ROOT}/packages/cli-core/install.sh" scenarios/scenario-to-cloud/cli --name "${REAL_NAME}" --install-dir "${INSTALL_DIR}"

cat > "${INSTALL_DIR}/scenario-to-cloud" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

BIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export VROOLI_LIFECYCLE_MANAGED=true
exec "${BIN_DIR}/scenario-to-cloud.bin" "$@"
EOF

chmod +x "${INSTALL_DIR}/scenario-to-cloud"
