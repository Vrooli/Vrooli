#!/bin/bash
# Phased test orchestrator for resource-experimenter scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

# Initialise shared runner. Manage-runtime defaults to false so callers can
# opt in via CLI flags; integration/business phases request runtime explicitly.
testing::runner::init \
  --scenario-name "resource-experimenter" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_DIR/artifacts"

PHASE_DIR="$TEST_DIR/phases"

testing::runner::register_phase \
  --name structure \
  --script "$PHASE_DIR/test-structure.sh" \
  --timeout 30 \
  --cache-ttl 3600 \
  --cache-key-from ".vrooli/service.json,README.md,PRD.md"

testing::runner::register_phase \
  --name dependencies \
  --script "$PHASE_DIR/test-dependencies.sh" \
  --timeout 60 \
  --cache-ttl 1800 \
  --cache-key-from "api/go.mod,api/go.sum,ui/package.json"

testing::runner::register_phase \
  --name unit \
  --script "$PHASE_DIR/test-unit.sh" \
  --timeout 120 \
  --cache-ttl 900 \
  --cache-key-from "api/go.mod,api/go.sum"

testing::runner::register_phase \
  --name integration \
  --script "$PHASE_DIR/test-integration.sh" \
  --timeout 180 \
  --requires-runtime true

testing::runner::register_phase \
  --name business \
  --script "$PHASE_DIR/test-business.sh" \
  --timeout 180 \
  --requires-runtime true

testing::runner::register_phase \
  --name performance \
  --script "$PHASE_DIR/test-performance.sh" \
  --timeout 60 \
  --requires-runtime true \
  --optional true

# Quick iteration presets
testing::runner::define_preset quick "structure dependencies unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset full "structure dependencies unit integration business performance"

testing::runner::execute "$@"
