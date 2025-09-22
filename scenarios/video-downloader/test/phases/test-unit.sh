#!/bin/bash

set -euo pipefail

printf "=== Unit Tests ===\\n"

# API unit tests

if [[ -d "api" ]]; then

  pushd api &gt;&amp; go test ./... -v -short || { printf "❌ API unit tests failed\\n"; exit 1; } &amp;&amp; popd

  printf "✅ API unit tests passed\\n"

else

  printf "⚠️ No API for unit tests\\n"

fi

# UI unit tests (assuming Jest or similar)

if [[ -d "ui" ]]; then

  pushd ui &gt;&amp; npm test -- --passWithNoTests || { printf "❌ UI unit tests failed\\n"; exit 1; } &amp;&amp; popd

  printf "✅ UI unit tests passed\\n"

else

  printf "⚠️ No UI for unit tests\\n"

fi

printf "✅ Unit tests completed\\n"