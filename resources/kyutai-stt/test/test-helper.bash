#!/usr/bin/env bash

set -euo pipefail

VROOLI_ROOT="${VROOLI_ROOT:-${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}}"

# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

if ! declare -F trash::safe_remove >/dev/null 2>&1; then
    trash::safe_remove() {
        rm -rf "$1"
    }
fi

vrooli_setup_service_test() {
    local service_name="${1:-kyutai-stt}"
    export VROOLI_TEST_SERVICE="$service_name"
    export VROOLI_TEST_TMPDIR="${BATS_TEST_TMPDIR:-$(mktemp -d /tmp/kyutai-stt-bats.XXXXXX)}"
    mkdir -p "$VROOLI_TEST_TMPDIR"
}

vrooli_auto_setup() {
    export VROOLI_ROOT
    export KYUTAI_STT_TEST_TMPDIR="${VROOLI_TEST_TMPDIR:-$(mktemp -d /tmp/kyutai-stt-bats.XXXXXX)}"
    mkdir -p "$KYUTAI_STT_TEST_TMPDIR"

    unset _VAR_SH_SOURCED \
        _LOG_SH_SOURCED \
        _SYSTEM_COMMANDS_SH_SOURCED \
        _DOCKER_UTILS_SOURCED \
        _DOCKER_RESOURCE_UTILS_SOURCED \
        _RESOURCES_COMMON_SH_SOURCED 2>/dev/null || true
}

vrooli_cleanup_test() {
    if [[ -n "${KYUTAI_STT_TEST_TMPDIR:-}" ]] && [[ -d "${KYUTAI_STT_TEST_TMPDIR}" ]]; then
        rm -rf "${KYUTAI_STT_TEST_TMPDIR}"
    fi
}

vrooli_make_temp_dir() {
    local prefix="${1:-kyutai-stt-test}"
    mktemp -d "${BATS_TEST_TMPDIR:-/tmp}/${prefix}.XXXXXX"
}
