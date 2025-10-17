#!/bin/bash
set -e
echo "=== Structure Tests ==="
# Verify scenario follows required file structure and standards

SCENARIO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_ROOT"

echo "Checking required files and directories..."

# Required files from PRD
REQUIRED_FILES=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
  "Makefile"
  "api/main.go"
  "api/go.mod"
  "cli/main.go"
  "cli/install.sh"
  "ui/index.html"
  "initialization/storage/postgres/schema.sql"
)

MISSING_COUNT=0
for file in "${REQUIRED_FILES[@]}"; do
  if [ -f "$file" ]; then
    echo "✅ $file"
  else
    echo "❌ Missing: $file"
    MISSING_COUNT=$((MISSING_COUNT + 1))
  fi
done

if [ $MISSING_COUNT -gt 0 ]; then
  echo "❌ Structure validation failed: $MISSING_COUNT missing files"
  exit 1
fi

# Check service.json structure
echo ""
echo "Validating service.json structure..."
if [ -f ".vrooli/service.json" ]; then
  # Check required fields
  if jq -e '.service.name' .vrooli/service.json > /dev/null 2>&1; then
    echo "✅ service.name present"
  else
    echo "❌ Missing: service.name"
    MISSING_COUNT=$((MISSING_COUNT + 1))
  fi

  if jq -e '.ports.api' .vrooli/service.json > /dev/null 2>&1; then
    echo "✅ ports.api configured"
  else
    echo "❌ Missing: ports.api"
    MISSING_COUNT=$((MISSING_COUNT + 1))
  fi

  if jq -e '.lifecycle.health' .vrooli/service.json > /dev/null 2>&1; then
    echo "✅ lifecycle.health configured"
  else
    echo "⚠️  lifecycle.health not configured"
  fi
fi

# Check Makefile has required targets
echo ""
echo "Validating Makefile targets..."
REQUIRED_TARGETS=("start" "stop" "test" "logs" "clean")
for target in "${REQUIRED_TARGETS[@]}"; do
  if grep -q "^${target}:" Makefile 2>/dev/null; then
    echo "✅ make $target"
  else
    echo "❌ Missing make target: $target"
    MISSING_COUNT=$((MISSING_COUNT + 1))
  fi
done

# Check CLI is installed
echo ""
echo "Checking CLI installation..."
if command -v elo-swipe > /dev/null 2>&1; then
  echo "✅ elo-swipe CLI installed"
  CLI_VERSION=$(elo-swipe version 2>/dev/null || echo "unknown")
  echo "   Version: $CLI_VERSION"
else
  echo "⚠️  elo-swipe CLI not in PATH"
fi

# Check test structure
echo ""
echo "Validating test structure..."
if [ -d "test/phases" ]; then
  TEST_PHASE_COUNT=$(find test/phases -name "test-*.sh" | wc -l)
  echo "✅ test/phases directory exists ($TEST_PHASE_COUNT test phases)"
else
  echo "❌ Missing: test/phases directory"
  MISSING_COUNT=$((MISSING_COUNT + 1))
fi

if [ $MISSING_COUNT -gt 0 ]; then
  echo ""
  echo "❌ Structure tests failed with $MISSING_COUNT issues"
  exit 1
fi

echo ""
echo "✅ Structure tests completed successfully"
