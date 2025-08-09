#!/bin/bash
# Resume Screening Assistant Test - New Framework Version
# Replaces 600+ lines of boilerplate with declarative testing

set -euo pipefail

# Source var.sh first with proper relative path
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../lib/utils/var.sh"

# Resolve paths
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
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
