#!/bin/bash
#
# Test Orchestrator for knowledge-observatory
# Executes all test phases in sequence
#

set -e

echo "=== Running all test phases for knowledge-observatory ==="

# Change to test phases directory
if [ -d "test/phases" ]; then
  cd test/phases

  # Run each test phase in order
  for test in test-*.sh; do
    if [ -f "$test" ]; then
      echo ""
      echo "=========================================="
      echo "Running $test"
      echo "=========================================="
      ./"$test"
    fi
  done

  cd ../..
else
  echo "❌ No test phases directory found at test/phases"
  exit 1
fi

echo ""
echo "=========================================="
echo "✅ All test phases completed successfully"
echo "=========================================="
