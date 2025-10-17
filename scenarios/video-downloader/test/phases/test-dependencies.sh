#!/bin/bash

set -euo pipefail

printf "=== Test Dependencies ===\\n"

# Check Go dependencies

if [[ -f "api/go.mod" ]]; then

  pushd api &gt;&amp; go mod tidy &gt;/dev/null 2&gt;&amp;1 &amp;&amp; popd

  printf "✅ Go dependencies verified\\n"

else

  printf "⚠️ No Go module found\\n"

fi

# Check Node dependencies

if [[ -f "ui/package.json" ]]; then

  # Dry run install to check deps

  pushd ui &gt;&amp; npm install --dry-run --silent &amp;&amp; popd

  printf "✅ UI dependencies check passed\\n"

else

  printf "⚠️ No UI package.json found\\n"

fi

printf "✅ Dependencies tests passed\\n"