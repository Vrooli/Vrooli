#!/bin/bash
# Framework Demo Test - Minimal scenario test using new framework

set -euo pipefail

# Resolve paths
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRAMEWORK_DIR="$(cd "$SCENARIO_DIR/../../framework" && pwd)"

echo "🚀 Testing New Scenario Framework"
echo "Scenario: $(basename "$SCENARIO_DIR")"
echo "Framework: $FRAMEWORK_DIR"
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
    echo "🎉 Framework demo completed successfully!"
    echo "   The new testing framework is working correctly."
else
    echo "❌ Framework demo failed."
    echo "   Please check the output above for details."
fi

exit $exit_code