#!/bin/bash
# Ai Content Assistant Example Test - New Framework Version
# Replaces 600+ lines of boilerplate with declarative testing

set -euo pipefail

# Resolve paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCENARIO_DIR="${APP_ROOT}/scenarios/campaign-content-studio"
FRAMEWORK_DIR="${APP_ROOT}/scripts/scenarios/validation"

echo "🚀 Testing Ai Content Assistant Example Business Scenario"
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
    echo "🎉 Ai Content Assistant Example scenario validation complete!"
    echo "   Ready for production deployment."
else
    echo "❌ Ai Content Assistant Example scenario validation failed."
    echo "   Please check resource availability and configuration."
fi

exit $exit_code
