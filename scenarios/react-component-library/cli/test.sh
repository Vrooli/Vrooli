#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEST_SCRIPT="${ROOT_DIR}/test/cli/run-cli-tests.sh"

if ! command -v bats >/dev/null 2>&1; then
  echo "bats not installed; skipping CLI tests" >&2
  exit 0
fi

if [ ! -f "${TEST_SCRIPT}" ]; then
  echo "CLI test script not found at ${TEST_SCRIPT}" >&2
  exit 1
fi

cd "${ROOT_DIR}/test/cli"
bats "${TEST_SCRIPT}"
