#!/bin/bash
# Resume Screening Assistant Test - New Framework Version
# Replaces 600+ lines of boilerplate with declarative testing

set -euo pipefail

# Source var.sh first with proper relative path
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh"

# Resolve paths
SCENARIO_DIR="${APP_ROOT}/scenarios/resume-screening-assistant"
FRAMEWORK_DIR="$var_SCRIPTS_SCENARIOS_DIR/framework"

echo "🚀 Testing Resume Screening Assistant Business Scenario"
echo "📁 Scenario: $(basename "$SCENARIO_DIR")"
echo "🔧 Framework: $FRAMEWORK_DIR"
echo

# Run declarative tests using the new framework
"$FRAMEWORK_DIR/scenario-test-runner.sh" \
  --scenario "$SCENARIO_DIR" \
  --config "scenario-test.yaml" \
  --verbose \
  "$@"

exit_code=$?

echo
if [[ $exit_code -eq 0 ]]; then
    echo "🎉 Resume Screening Assistant scenario validation complete!"
    echo "   Ready for production deployment."
else
    echo "❌ Resume Screening Assistant scenario validation failed."
    echo "   Please check resource availability and configuration."
fi

exit $exit_code
