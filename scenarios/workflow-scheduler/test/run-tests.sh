#!/bin/bash
set -e
SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
echo "=== Running standard scenario tests for $(basename $SCRIPT_DIR) ==="
cd "$SCRIPT_DIR/phases"
for phase in test-*.sh; do
  if [ -f "$phase" ]; then
    echo "Running $phase..."
    bash "$phase"
  else
    echo "Phase $phase not found, skipping"
  fi
done
echo "✅ All standard tests completed"