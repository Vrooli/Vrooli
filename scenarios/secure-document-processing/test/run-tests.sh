#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &amp;&amp; pwd )"
PHASES_DIR="$SCRIPT_DIR/phases"

echo "🧪 Running Secure Document Processing Test Suite"
echo "📁 Working directory: $SCRIPT_DIR"

if [ ! -d "$PHASES_DIR" ]; then
    echo "❌ Test phases directory not found: $PHASES_DIR"
    exit 1
fi

failed_phases=()
passed_phases=()

for phase in "$PHASES_DIR"/test-*.sh; do
    if [ -f "$phase" ]; then
        phase_name=$(basename "$phase")
        echo ""
        echo "=== Running $phase_name ==="
        if bash "$phase" 2&gt;&amp;1; then
            echo "✅ $phase_name passed"
            passed_phases+=("$phase_name")
        else
            echo "❌ $phase_name failed"
            failed_phases+=("$phase_name")
        fi
    fi
done

echo ""
echo "📊 Test Summary:"
echo "   Passed: ${#passed_phases[@]} phases"
for phase in "${passed_phases[@]}"; do
    echo "     ✓ $phase"
done
if [ ${#failed_phases[@]} -gt 0 ]; then
    echo "   Failed: ${#failed_phases[@]} phases"
    for phase in "${failed_phases[@]}"; do
        echo "     ✗ $phase"
    done
    exit 1
fi

echo "🎉 All tests passed!"
