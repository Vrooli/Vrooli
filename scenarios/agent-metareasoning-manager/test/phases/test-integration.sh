#!/bin/bash
# Executes integration-tagged Go tests.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

if testing::phase::check "Go integration test suite" bash -c 'cd api && go test -v -tags=integration ./...'; then
  :
fi

testing::phase::end_with_summary "Integration validation completed"
