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

jwt::do_keys_exist() {
    # Check existence of staging and production JWT key pairs; report if missing or partial
    local any_missing=0
    for env in staging production; do
        local priv_var="var_${env^^}_JWT_PRIV_KEY_FILE"
        local pub_var="var_${env^^}_JWT_PUB_KEY_FILE"
        local priv_file="${!priv_var}"
        local pub_file="${!pub_var}"
        if [[ -f "$priv_file" && -f "$pub_file" ]]; then
            log::info "[$env] JWT keys already exist."
        elif [[ -f "$priv_file" || -f "$pub_file" ]]; then
            log::error "[$env] One JWT key exists, but not the other. Delete them both and try again."
            exit $ERROR_JWT_FILE_MISSING
        else
            any_missing=1
        fi
    done
    return $any_missing
}

jwt::generate_key_pair() {
    if jwt::do_keys_exist; then
        return 0
    fi

    log::header "Generating JWT key pairs for staging & production"
    for env in staging production; do
        local priv_var="var_${env^^}_JWT_PRIV_KEY_FILE"
        local pub_var="var_${env^^}_JWT_PUB_KEY_FILE"
        local priv_file="${!priv_var}"
        local pub_file="${!pub_var}"

        if [[ -f "$priv_file" && -f "$pub_file" ]]; then
            log::info "[$env] JWT keys already exist at $priv_file and $pub_file"
            continue
        fi

        log::info "[$env] Generating JWT key pair..."
        openssl genpkey -algorithm RSA -out "$priv_file" -pkeyopt rsa_keygen_bits:2048
        openssl rsa -pubout -in "$priv_file" -out "$pub_file"
        log::info "[$env] JWT keys generated: $priv_file, $pub_file"
    done
}
