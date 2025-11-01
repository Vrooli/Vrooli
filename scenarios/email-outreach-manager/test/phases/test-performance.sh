#!/bin/bash
# Performance gate stub to track future benchmarks
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

testing::phase::add_warning "Performance benchmarks not defined; skipping"
testing::phase::add_test skipped

testing::phase::end_with_summary "Performance phase placeholder"
