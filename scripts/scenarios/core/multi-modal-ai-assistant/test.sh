#!/bin/bash
# Multi-Modal AI Assistant Test - New Framework Version
# Replaces 1000+ lines of boilerplate with 20 lines of declarative testing

set -euo pipefail

# Resolve paths
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRAMEWORK_DIR="$(cd "$SCENARIO_DIR/../../framework" && pwd)"

echo "🤖 Testing Multi-Modal AI Assistant Business Scenario"
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
    echo "🎉 Multi-Modal AI Assistant scenario validation complete!"
    echo "   Ready for AI-powered voice-to-visual-to-action workflows."
else
    echo "❌ Multi-Modal AI Assistant scenario validation failed."
    echo "   Please check resource availability and configuration."
fi

exit $exit_code