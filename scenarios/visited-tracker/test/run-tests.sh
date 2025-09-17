#!/bin/bash
# Thin wrapper around the shared scenario test runner
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

SCENARIO_NAME="visited-tracker"
LOG_DIR="$TEST_DIR/artifacts"
mkdir -p "$LOG_DIR"

# Initialise shared runner
testing::runner::init \
    --scenario-name "$SCENARIO_NAME" \
    --scenario-dir "$SCENARIO_DIR" \
    --test-dir "$TEST_DIR" \
    --log-dir "$LOG_DIR" \
    --default-manage-runtime false

# Phase registrations with caching configuration
# Cache TTL (Time To Live) values:
#   - structure: 1 hour (3600s) - rarely changes
#   - dependencies: 1 hour (3600s) - changes only with package updates
#   - unit: 30 minutes (1800s) - changes with code updates
#   - integration/business/performance: no cache (always run fresh)
# Cache keys are based on relevant files that would invalidate the cache
PHASES_DIR="$TEST_DIR/phases"

testing::runner::register_phase \
    --name "structure" \
    --script "$PHASES_DIR/test-structure.sh" \
    --timeout 15 \
    --display "phase-structure" \
    --cache-ttl 3600 \
    --cache-key-from ".vrooli/service.json,README.md,PRD.md"

testing::runner::register_phase \
    --name "dependencies" \
    --script "$PHASES_DIR/test-dependencies.sh" \
    --timeout 30 \
    --display "phase-dependencies" \
    --cache-ttl 3600 \
    --cache-key-from ".vrooli/service.json,api/go.mod,ui/package.json"

testing::runner::register_phase \
    --name "unit" \
    --script "$PHASES_DIR/test-unit.sh" \
    --timeout 60 \
    --display "phase-unit" \
    --cache-ttl 1800 \
    --cache-key-from "api/go.mod,api/go.sum,ui/package.json,ui/package-lock.json"

testing::runner::register_phase \
    --name "integration" \
    --script "$PHASES_DIR/test-integration.sh" \
    --timeout 120 \
    --requires-runtime true \
    --display "phase-integration"

testing::runner::register_phase \
    --name "business" \
    --script "$PHASES_DIR/test-business.sh" \
    --timeout 180 \
    --requires-runtime true \
    --display "phase-business"

testing::runner::register_phase \
    --name "performance" \
    --script "$PHASES_DIR/test-performance.sh" \
    --timeout 60 \
    --requires-runtime true \
    --display "phase-performance"

# Test type registrations
UNIT_DIR="$TEST_DIR/unit"
CLI_DIR="$TEST_DIR/cli"
UI_DIR="$TEST_DIR/ui"

testing::runner::register_test_type \
    --name "go" \
    --handler "$UNIT_DIR/run-unit-tests.sh --skip-node --skip-python" \
    --kind command \
    --display "test-go"

testing::runner::register_test_type \
    --name "node" \
    --handler "$UNIT_DIR/run-unit-tests.sh --skip-go --skip-python" \
    --kind command \
    --display "test-node"

testing::runner::register_test_type \
    --name "bats" \
    --handler "$CLI_DIR/run-cli-tests.sh" \
    --kind script \
    --requires-runtime true \
    --display "test-bats"

testing::runner::register_test_type \
    --name "ui" \
    --handler "$UI_DIR/run-ui-tests.sh" \
    --kind script \
    --requires-runtime true \
    --display "test-ui"

# Define parallel execution groups
# Structure and dependencies can run in parallel as they don't interfere
testing::runner::define_parallel_group "validation" "structure" "dependencies"

# Unit and performance tests can run in parallel as they don't share resources
testing::runner::define_parallel_group "quality" "unit" "performance"

# Presets
testing::runner::define_preset quick "structure dependencies unit"
testing::runner::define_preset smoke "structure dependencies"
testing::runner::define_preset core "structure dependencies unit integration"

# Execute with provided arguments
testing::runner::execute "$@"
