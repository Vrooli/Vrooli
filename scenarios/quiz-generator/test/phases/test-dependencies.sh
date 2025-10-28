#!/bin/bash
# Dependency testing phase for quiz-generator scenario
# Validates all required and optional dependencies

set -euo pipefail

# Determine APP_ROOT
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with target time
testing::phase::init --target-time "60s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🔗 Testing Quiz Generator Dependencies..."
echo ""

# Required dependencies
DEPENDENCIES_OK=true

# Test 1: PostgreSQL
echo "Test 1: PostgreSQL (required)"
if resource-postgres status &>/dev/null; then
  echo "✅ PostgreSQL is running"
else
  echo "❌ PostgreSQL is not running"
  DEPENDENCIES_OK=false
fi

# Test 2: Ollama
echo "Test 2: Ollama (required for AI generation)"
if resource-ollama status &>/dev/null; then
  echo "✅ Ollama is running"
else
  echo "⚠️  Ollama is not running (AI generation will fail)"
fi

# Optional dependencies
echo ""
echo "Optional Dependencies:"

# Test 3: Redis
echo "Test 3: Redis (optional - caching)"
if resource-redis status &>/dev/null; then
  echo "✅ Redis is available"
else
  echo "ℹ️  Redis not available (using in-memory cache)"
fi

# Test 4: Qdrant
echo "Test 4: Qdrant (optional - semantic search)"
if resource-qdrant status &>/dev/null; then
  echo "✅ Qdrant is available"
else
  echo "ℹ️  Qdrant not available (using PostgreSQL full-text search)"
fi

# Test 5: Database schema
echo ""
echo "Test 5: Database schema validation"
SCHEMA_CHECK=$(resource-postgres query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='quiz_generator' AND table_name IN ('quizzes', 'questions', 'quiz_results')" 2>/dev/null || echo "0")

if echo "$SCHEMA_CHECK" | grep -q "3"; then
  echo "✅ Database schema is initialized"
else
  echo "❌ Database schema is incomplete"
  DEPENDENCIES_OK=false
fi

echo ""
if [ "$DEPENDENCIES_OK" = true ]; then
  echo "✅ All required dependencies are satisfied"
  testing::phase::end_with_summary "Dependency tests passed"
else
  echo "❌ Some required dependencies are missing"
  testing::phase::end_with_summary "Dependency tests failed"
  exit 1
fi
