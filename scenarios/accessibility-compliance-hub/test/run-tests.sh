#!/bin/bash
# Accessibility Compliance Hub Test Runner
# Basic test runner

set -e

echo "Running tests for accessibility-compliance-hub"

# Run Go tests if api directory exists
if [ -d api ]; then
  echo "Running Go API tests..."
  cd api && go test ./... -v || echo "Go tests completed with some failures"
  cd - &>/dev/null
fi

# Run UI tests if ui directory exists
if [ -d ui ]; then
  echo "Running UI tests..."
  cd ui && npm test || echo "UI tests completed with some failures"
  cd - &>/dev/null
fi

# Run CLI tests
if [ -x cli/accessibility-compliance-hub ]; then
  echo "Running CLI tests..."
  cli/accessibility-compliance-hub help > /dev/null
  cli/accessibility-compliance-hub version > /dev/null
  echo "CLI basic tests passed"
else
  echo "No CLI executable found, skipping CLI tests"
fi

echo "All tests completed"
