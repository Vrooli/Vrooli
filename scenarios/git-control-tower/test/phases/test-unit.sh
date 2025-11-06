#!/bin/bash
# Unit test phase - <60 seconds
# Runs language-specific unit tests with coverage reporting
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/core.sh"
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

testing::phase::init --target-time "60s"

# Detect languages and run appropriate tests
languages=$(testing::core::detect_languages)

if [ -z "$languages" ]; then
    echo "ℹ️  No supported languages detected for unit testing"
    testing::phase::add_test skipped
else
    echo "📋 Detected languages: $languages"

    # Build arguments for the universal runner
    runner_args=(
        "--coverage-warn" "80"
        "--coverage-error" "70"
    )

    # Skip languages not present
    for lang in go node python; do
        if ! echo "$languages" | grep -q "$lang"; then
            runner_args+=("--skip-$lang")
        fi
    done

    # Run the universal test runner
    testing::unit::run_all_tests "${runner_args[@]}"
fi

testing::phase::end_with_summary "Unit tests completed"
