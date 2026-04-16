#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"

test -f "${RESOURCE_DIR}/resource.json"

cd "${RESOURCE_DIR}/cli"
go run . help >/dev/null
go run . content help >/dev/null
go run . show-config --json >/dev/null
