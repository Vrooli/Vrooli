#!/bin/bash
set -euo pipefail

echo "=== Test Business Logic ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Check core business requirements
# For Vrooli Bridge: project management, doc injection, etc.

# Minimal: check if key files exist
if [[ ! -f "${SCENARIO_DIR}/initialization/templates/CLAUDE_ADDITIONS.md.template" ]]; then
  echo "❌ Missing CLAUDE_ADDITIONS template"
  exit 1
fi

if [[ ! -f "${SCENARIO_DIR}/initialization/templates/VROOLI_INTEGRATION.md.template" ]]; then
  echo "❌ Missing VROOLI_INTEGRATION template"
  exit 1
fi

echo "✅ Business logic structure tests passed"