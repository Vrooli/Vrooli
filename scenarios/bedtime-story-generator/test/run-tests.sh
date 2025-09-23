#!/bin/bash
set -e
echo "Running tests for bedtime-story-generator..."
# Test the CLI
if /home/matthalloran8/Vrooli/scenarios/bedtime-story-generator/cli/bedtime-story-generator generate | grep -q "Once upon a time"; then
  echo "CLI test passed"
else
  echo "CLI test failed"
  exit 1
fi
echo "All tests passed!"
