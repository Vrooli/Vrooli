#!/usr/bin/env bash
set -euo pipefail
DESCRIPTION="Append SSH public key to authorized_keys. Run as the deploy user on your staging or production server."

# Source var.sh first with relative path
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Now source everything else using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"

authorize_key::main() {
  log::header "Authorize SSH Public Key"
  log::info "This will add a new public key to ~/.ssh/authorized_keys."
  log::info "Please paste the public key now, then press Ctrl-D when finished:"
  
  # Ensure .ssh directory exists with secure permissions and proper ownership
  sudo::mkdir_as_user ~/.ssh 700
  
  # Append the pasted key to authorized_keys
  cat >> ~/.ssh/authorized_keys
  
  # Secure the authorized_keys file
  chmod 600 ~/.ssh/authorized_keys
  
  # Restore ownership if running under sudo
  sudo::restore_owner ~/.ssh/authorized_keys
  log::success "✅ Public key successfully added to ~/.ssh/authorized_keys"
}

# If help flag is passed, show usage
if [[ "${1-}" == "-h" || "${1-}" == "--help" ]]; then
  echo "$DESCRIPTION"
  echo
  echo "Usage: $(basename "$0")"
  exit 0
fi

authorize_key::main 