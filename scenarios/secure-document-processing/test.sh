#!/bin/bash
# Secure Document Processing Test - New Framework Version
# Replaces 600+ lines of boilerplate with declarative testing

set -euo pipefail

# Resolve paths
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "$(cd "$SCENARIO_DIR" && cd ../../lib/utils && pwd)/var.sh"

FRAMEWORK_DIR="$(cd "$SCENARIO_DIR/../../framework" && pwd)"

echo "🚀 Testing Secure Document Processing Business Scenario"
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
    echo "🎉 Secure Document Processing scenario validation complete!"
    echo "   Ready for production deployment."
else
    echo "❌ Secure Document Processing scenario validation failed."
    echo "   Please check resource availability and configuration."
fi

exit $exit_code
