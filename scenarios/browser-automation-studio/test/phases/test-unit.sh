#!/bin/bash
set -e

echo "=== Test Unit ==="

# Run Go unit tests
if [[ -f api/go.mod ]]; then
  cd api
  if go test ./... -v -short 2&gt;&amp;1 | tee /dev/stderr | grep -q "FAIL"; then
    echo "❌ Go unit tests failed"
    exit 1
  else
    echo "✅ Go unit tests passed"
  fi
  cd ..
else
  echo "⚠️ No Go unit tests to run"
fi

# Run UI unit tests if test script exists
if [[ -f ui/package.json ]] &amp;&amp; grep -q '"test"' ui/package.json; then
  cd ui
  if npm test; then
    echo "✅ UI unit tests passed"
  else
    echo "❌ UI unit tests failed"
    exit 1
  fi
  cd ..
else
  echo "⚠️ No UI unit tests configured, skipping"
fi

echo "✅ Unit tests passed"