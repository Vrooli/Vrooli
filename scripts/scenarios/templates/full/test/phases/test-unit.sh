#!/bin/bash
# Orchestrates Go/Node/Python unit tests with coverage thresholds.
# Configuration is read from .vrooli/testing.json
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/unit.sh"

testing::unit::validate_all
