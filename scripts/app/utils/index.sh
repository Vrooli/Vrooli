#!/usr/bin/env bash
# Vrooli-specific utilities index - provides monorepo-specific functionality
# This file makes it easy to source all Vrooli utilities at once

VROOLI_UTILS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${VROOLI_UTILS_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"

# Then source Vrooli-specific utilities with error handling
for util in ci docker env jwt pnpm_tools proxy repository vault secrets service-config keyless_ssh reverseProxy; do
    if [[ -f "${VROOLI_UTILS_DIR}/${util}.sh" ]]; then
        # shellcheck disable=SC1091
        source "${VROOLI_UTILS_DIR}/${util}.sh" || {
            log::warning "Failed to source Vrooli utility: ${util}.sh"
        }
    fi
done