#!/usr/bin/env bash
set -euo pipefail
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SV_LIB_DIR="${APP_ROOT}/resources/speaker-verification/lib"
# shellcheck disable=SC1091
source "${SV_LIB_DIR}/test.sh"
speaker_verification::test::smoke
