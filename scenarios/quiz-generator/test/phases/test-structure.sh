#!/bin/bash
# Structure testing phase for quiz-generator scenario
# Validates file structure and configuration

set -euo pipefail

# Determine APP_ROOT
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with target time
testing::phase::init --target-time "30s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

echo "📁 Testing Quiz Generator Structure..."
echo ""

STRUCTURE_OK=true

# Required files
REQUIRED_FILES=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
  "Makefile"
  "api/main.go"
  "api/go.mod"
  "cli/quiz-generator"
  "cli/install.sh"
  "initialization/storage/postgres/schema.sql"
  "ui/package.json"
  "ui/src/main.tsx"
  "test/run-tests.sh"
)

echo "Checking required files..."
for file in "${REQUIRED_FILES[@]}"; do
  if [ -f "$file" ]; then
    echo "✅ $file"
  else
    echo "❌ Missing: $file"
    STRUCTURE_OK=false
  fi
done

# Required directories
echo ""
echo "Checking required directories..."
REQUIRED_DIRS=(
  "api"
  "cli"
  "ui"
  "initialization"
  "initialization/storage/postgres"
  "test"
  "test/phases"
)

for dir in "${REQUIRED_DIRS[@]}"; do
  if [ -d "$dir" ]; then
    echo "✅ $dir/"
  else
    echo "❌ Missing: $dir/"
    STRUCTURE_OK=false
  fi
done

# Validate service.json
echo ""
echo "Validating service.json..."
if jq . .vrooli/service.json >/dev/null 2>&1; then
  echo "✅ service.json is valid JSON"

  # Check required fields
  if jq -e '.service.name' .vrooli/service.json >/dev/null 2>&1; then
    echo "✅ service.name is present"
  else
    echo "❌ service.name is missing"
    STRUCTURE_OK=false
  fi
else
  echo "❌ service.json is invalid"
  STRUCTURE_OK=false
fi

# Validate API binary
echo ""
echo "Checking API binary..."
if [ -f "api/quiz-generator-api" ] && [ -x "api/quiz-generator-api" ]; then
  echo "✅ API binary exists and is executable"
else
  echo "⚠️  API binary not found (may need to run setup)"
fi

# Validate CLI
echo ""
echo "Checking CLI installation..."
if command -v quiz-generator &>/dev/null; then
  echo "✅ CLI is installed globally"
else
  echo "⚠️  CLI not installed globally (may need to run cli/install.sh)"
fi

echo ""
if [ "$STRUCTURE_OK" = true ]; then
  echo "✅ All structure requirements met"
  testing::phase::end_with_summary "Structure tests passed"
else
  echo "❌ Some structure requirements are missing"
  testing::phase::end_with_summary "Structure tests failed"
  exit 1
fi
