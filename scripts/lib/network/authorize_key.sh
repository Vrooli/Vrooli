#!/usr/bin/env bash
set -euo pipefail
DESCRIPTION="Append SSH public key to authorized_keys. Run as the deploy user on your staging or production server."

LIB_NETWORK_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${LIB_NETWORK_DIR}/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

authorize_key::main() {
  log::header "Authorize SSH Public Key"
  log::info "This will add a new public key to ~/.ssh/authorized_keys."
  log::info "Please paste the public key now, then press Ctrl-D when finished:"
  
  # Ensure .ssh directory exists with secure permissions
  mkdir -p ~/.ssh
  chmod 700 ~/.ssh
  
  # Append the pasted key to authorized_keys
  cat >> ~/.ssh/authorized_keys
  
  # Secure the authorized_keys file
  chmod 600 ~/.ssh/authorized_keys
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