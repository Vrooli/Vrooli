#!/bin/bash
# Placeholder performance gate for future benchmarks
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

testing::phase::add_warning "Performance benchmarks not yet implemented; add wrk/Lighthouse thresholds when available."
testing::phase::add_test skipped

testing::phase::end_with_summary "Performance phase placeholder"
