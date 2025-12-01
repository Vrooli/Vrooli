#!/usr/bin/env bash
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"

source "${SCENARIO_DIR}/test/lib/orchestrator.sh"

tg::orchestrator::run "$@"
