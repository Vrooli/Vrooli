#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"

if ! command -v node >/dev/null 2>&1; then
    echo "node is required to run playbook helper tests" >&2
    exit 1
fi

if ! command -v bats >/dev/null 2>&1; then
    echo "bats is required to run playbook helper tests" >&2
    exit 1
fi

NODE_OPTIONS="${NODE_OPTIONS:-}" node --test "$SCRIPT_DIR/build-registry.test.mjs"
BATS_TEST_DIRNAME="$SCRIPT_DIR" bats "$SCRIPT_DIR/reset-seed-state.bats"
