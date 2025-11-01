#!/bin/bash
# Phased test orchestrator for token-economy scenario
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

TEST_LOG_DIR="$TEST_DIR/artifacts"
SCENARIO_NAME="token-economy"

# Initialise shared runner; auto-manage runtime disabled by default (phases opt-in)
testing::runner::init \
  --scenario-name "$SCENARIO_NAME" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_LOG_DIR"

PHASES_DIR="$TEST_DIR/phases"

testing::runner::register_phase \
  --name structure \
  --script "$PHASES_DIR/test-structure.sh" \
  --timeout 30 \
  --cache-ttl 3600 \
  --cache-key-from ".vrooli/service.json,PRD.md,README.md"

testing::runner::register_phase \
  --name dependencies \
  --script "$PHASES_DIR/test-dependencies.sh" \
  --timeout 90 \
  --cache-ttl 1800 \
  --cache-key-from "api/go.mod,api/go.sum,ui/package.json"

testing::runner::register_phase \
  --name unit \
  --script "$PHASES_DIR/test-unit.sh" \
  --timeout 120

testing::runner::register_phase \
  --name integration \
  --script "$PHASES_DIR/test-integration.sh" \
  --timeout 180

testing::runner::register_phase \
  --name business \
  --script "$PHASES_DIR/test-business.sh" \
  --timeout 120

testing::runner::register_phase \
  --name performance \
  --script "$PHASES_DIR/test-performance.sh" \
  --timeout 120

# Presets for quick iteration
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure dependencies integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

testing::runner::execute "$@"
