#!/usr/bin/env bash
# This script generates a public/private key pair for JWT signing
set -euo pipefail

UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${UTILS_DIR}/var.sh"

ci::do_keys_exist() {
    # Check existence of staging and production key pairs; report if missing or partial
    local any_missing=0
    for env in staging production; do
        local priv_var="${env^^}_CI_SSH_PRIV_KEY_FILE"
        local pub_var="${env^^}_CI_SSH_PUB_KEY_FILE"
        local priv_file="${!priv_var}"
        local pub_file="${!pub_var}"
        if [[ -f "$priv_file" && -f "$pub_file" ]]; then
            log::info "[$env] SSH keys already exist."
        elif [[ -f "$priv_file" || -f "$pub_file" ]]; then
            log::error "[$env] One SSH key exists, but not the other. Delete them both and try again."
            exit $ERROR_SSH_FILE_MISSING
        else
            any_missing=1
        fi
    done
    return $any_missing
}

ci::generate_key_pair() {
    if ci::do_keys_exist; then
        return 0
    fi

    log::header "Generating SSH key pairs for CI (staging & production)"

    for env in staging production; do
        local priv_var="${env^^}_CI_SSH_PRIV_KEY_FILE"
        local pub_var="${env^^}_CI_SSH_PUB_KEY_FILE"
        local priv_file="${!priv_var}"
        local pub_file="${!pub_var}"

        if [[ -f "$priv_file" && -f "$pub_file" ]]; then
            log::info "[$env] SSH keys already exist at $priv_file and $pub_file"
            continue
        fi

        log::info "[$env] Generating key pair..."
        ssh-keygen -t ed25519 -f "$priv_file" -N "" -C "github-actions deploy key ($env)"
        mv "${priv_file}.pub" "$pub_file"
        chmod 600 "$priv_file" && chmod 644 "$pub_file"
        log::info "[$env] Generated keys: $priv_file, $pub_file"
    done

    log::header "Next Steps:"
    cat <<EOF
1. SSH into your staging server and authorize the staging public key:
   ssh <staging-user>@<staging-host>
   bash scripts/main/authorize_key.sh
   # Paste the contents of $var_STAGING_CI_SSH_PUB_KEY_FILE when prompted, then press Ctrl-D

2. SSH into your production server and authorize the production public key:
   ssh <production-user>@<production-host>
   bash scripts/main/authorize_key.sh
   # Paste the contents of $var_PRODUCTION_CI_SSH_PUB_KEY_FILE when prompted, then press Ctrl-D

3. Add private keys to GitHub Actions environment secrets:
   - For staging: set VPS_SSH_PRIVATE_KEY = contents of $var_STAGING_CI_SSH_PRIV_KEY_FILE
   - For production: set VPS_SSH_PRIVATE_KEY = contents of $var_PRODUCTION_CI_SSH_PRIV_KEY_FILE

4. Ensure each environment secret set also includes:
   VPS_DEPLOY_USER, VPS_DEPLOY_HOST, JWT_PRIV_PEM, JWT_PUB_PEM, ENV_FILE_CONTENT

5. Re-run the GitHub Actions pipeline and confirm both "Verify SSH connection" steps pass.
EOF
}
