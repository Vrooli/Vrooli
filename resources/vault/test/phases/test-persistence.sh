#!/usr/bin/env bash
# Vault Resource Persistence Test - validates the current Go CLI and lifecycle
# keep secrets durable across a resource restart.

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/log.sh"

vault::test::persistence() {
    log::info "Running Vault persistence regression test..."

    if ! command -v resource-vault >/dev/null 2>&1; then
        log::error "resource-vault is not on PATH"
        return 1
    fi
    if ! command -v vrooli >/dev/null 2>&1; then
        log::error "vrooli is not on PATH"
        return 1
    fi

    local status_json storage_type persistence_safe
    status_json="$(resource-vault status --json)"
    storage_type="$(printf '%s' "$status_json" | jq -r '.storage_type')"
    persistence_safe="$(printf '%s' "$status_json" | jq -r '.persistence_safe')"

    if [[ "$storage_type" != "file" || "$persistence_safe" != "true" ]]; then
        log::error "Vault is not in persistent file-backed mode"
        printf '%s\n' "$status_json"
        return 1
    fi

    VAULT_PERSISTENCE_TEST_PATH="secret/test/vrooli-persistence-$$-$(date +%s)"
    local test_value="survives-restart-$$"

    cleanup() {
        if [[ -n "${VAULT_PERSISTENCE_TEST_PATH:-}" ]]; then
            resource-vault content delete --path "$VAULT_PERSISTENCE_TEST_PATH" >/dev/null 2>&1 || true
        fi
    }
    trap cleanup EXIT

    resource-vault content set --path "$VAULT_PERSISTENCE_TEST_PATH" --key value --value "$test_value" >/dev/null
    vrooli --no-stale-check resource restart vault >/dev/null

    local actual
    actual="$(resource-vault content get --path "$VAULT_PERSISTENCE_TEST_PATH" --key value --format raw)"
    if [[ "$actual" != "$test_value" ]]; then
        log::error "Vault secret did not survive restart"
        log::error "Expected: $test_value"
        log::error "Actual: $actual"
        return 1
    fi

    log::success "Vault persistence regression test passed"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    vault::test::persistence "$@"
fi
