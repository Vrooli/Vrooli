#!/bin/bash
# Orchestrates language unit tests with coverage thresholds.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/unit.sh"

# ==============================================================================
# BROWSER-AUTOMATION-STUDIO SPECIFIC: Shared Playbook Helper Tests
# ==============================================================================
# This block is intentionally unique to browser-automation-studio's unit phase.
# Unlike typical scenarios, BAS is responsible for validating the shared testing
# infrastructure (workflow-runner.sh, resolve-workflow.py, build-registry.mjs)
# that ALL scenarios depend on for their e2e tests.
#
# Why this belongs here:
# 1. BAS is the test harness for e2e testing across all scenarios. So while the workflow runner 
#    and related utilities are technically outside of the scenario/browser-automation-studio/ folder, 
#    they are directly relying on the browser-automation-studio API to power workflow execution.
# 2. Validating them here ensures the entire testing ecosystem works before other
#    scenarios rely on them
# 3. These are testing utilities without their own standalone test infrastructure
#
# This coupling is intentional: if shared testing tools break, BAS unit tests
# should fail to prevent cascading failures across all scenario e2e tests.
# ==============================================================================
PLAYBOOK_HELPER_TESTS="${APP_ROOT}/scripts/scenarios/testing/playbooks/tests/run.sh"
if [ -x "$PLAYBOOK_HELPER_TESTS" ]; then
    echo "[INFO]    Running playbook helper tests"
    if ! bash "$PLAYBOOK_HELPER_TESTS"; then
        testing::phase::add_error "Playbook helper tests failed"
        exit 1
    fi
fi

testing::unit::validate_all
