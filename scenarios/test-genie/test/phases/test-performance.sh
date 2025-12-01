#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/runtime.sh"

tg::require_command go "Go toolchain is required for performance benchmarks"

tg::log_info "Building Go API binary for performance baseline"
tmp_bin="$(mktemp)"
start_ts="$(date +%s)"
if ! tg::run_in_dir "${TEST_GENIE_SCENARIO_DIR}/api" go build -o "$tmp_bin" ./...; then
  rm -f "$tmp_bin"
  tg::fail "Go build failed"
fi
duration=$(( $(date +%s) - start_ts ))
rm -f "$tmp_bin"

tg::log_info "Go build completed in ${duration}s"
if (( duration > 90 )); then
  tg::fail "Go build exceeded 90 seconds (${duration}s)"
fi

tg::log_success "Performance validation complete"
