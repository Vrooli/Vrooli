#!/bin/bash
set -e
echo "=== Running all test phases ==="
if [ -d "test/phases" ]; then
  cd test/phases
  for test in test-*.sh; do
    if [ -f "$test" ]; then
      echo "Running $test"
      ./"$test"
    fi
  done
  cd ../..
else
  echo "No test phases directory found"
fi
echo "✅ All tests completed"