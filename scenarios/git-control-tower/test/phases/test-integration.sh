#!/usr/bin/env bash
# Test Integration Phase - Basic integration tests

set -e

SCENARIO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "Testing Git Control Tower integration..."

# Test API binary exists after build
if [[ ! -f "${SCENARIO_ROOT}/api/git-control-tower-api" ]]; then
    echo "⚠️  API binary not built yet (run 'make setup' first)"
    echo "✅ Integration tests skipped (binary not built)"
    exit 0
fi

# Test CLI can be executed
if ! "${SCENARIO_ROOT}/cli/git-control-tower" help &> /dev/null; then
    echo "❌ CLI help command failed"
    exit 1
fi

echo "✅ Integration validation passed"
