#!/bin/bash
set -e
echo "Running Date Night Planner tests"
cd "$(dirname "$0")/phases"
for test in test-*.sh; do
  if [ -f "$test" ]; then
    echo "Running $test"
    ./"$test"
  fi
done
echo "All tests passed"