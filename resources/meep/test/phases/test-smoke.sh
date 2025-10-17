#!/usr/bin/env bash
# MEEP Smoke Test Phase

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${SCRIPT_DIR}/lib/test.sh"

meep::test_smoke "$@"