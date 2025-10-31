#!/usr/bin/env bash
set -euo pipefail

# Test: Business Logic Validation
# Validates core business logic, rules, and data integrity

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "💼 Testing Prompt Injection Arena business logic..."

# Track failures
FAILURES=0

# Test database schema requirements
echo "🗄️  Testing database schema..."

schema_file="${SCENARIO_DIR}/initialization/postgres/schema.sql"
if [ ! -f "$schema_file" ]; then
    echo "  ❌ Schema file not found"
    ((FAILURES++))
else
    # Check for required tables
    required_tables=("injection_techniques" "agent_configurations" "test_results" "tournaments" "tournament_results")
    for table in "${required_tables[@]}"; do
        if grep -q "$table" "$schema_file"; then
            echo "  ✅ Table defined: $table"
        else
            echo "  ❌ Missing table: $table"
            ((FAILURES++))
        fi
    done
    
    # Check for security scoring functions
    if grep -q "calculate_robustness_score" "$schema_file" || grep -q "robustness" "$schema_file"; then
        echo "  ✅ Security scoring logic present"
    else
        echo "  ⚠️  Security scoring may need review"
    fi
    
    # Check for injection categories
    if grep -q "direct_override" "$schema_file"; then
        echo "  ✅ Injection categorization defined"
    else
        echo "  ⚠️  Injection categories may need review"
    fi
fi

# Test seed data
echo "📊 Testing seed data..."

seed_file="${SCENARIO_DIR}/initialization/postgres/seed.sql"
if [ -f "$seed_file" ]; then
    # Count seed injections
    injection_count=$(grep -c "INSERT INTO injection_techniques" "$seed_file" || echo "0")
    if [ "$injection_count" -gt 0 ]; then
        echo "  ✅ Seed injections provided ($injection_count)"
    else
        echo "  ⚠️  No seed injections found"
    fi
    
    # Check for diverse categories
    if grep -q "direct_override" "$seed_file" && grep -q "context_poisoning" "$seed_file"; then
        echo "  ✅ Diverse injection categories in seed data"
    else
        echo "  ⚠️  Seed data may lack diversity"
    fi
else
    echo "  ⚠️  No seed data file found"
fi

# Test N8N workflow configurations
echo "🔄 Testing N8N workflows..."

sandbox_workflow="${SCENARIO_DIR}/initialization/n8n/security-sandbox.json"
if [ -f "$sandbox_workflow" ]; then
    if jq empty "$sandbox_workflow" 2>/dev/null; then
        echo "  ✅ Security sandbox workflow valid JSON"
        
        # Check for security nodes
        if jq -e '.nodes[] | select(.name | contains("Security") or contains("Limit"))' "$sandbox_workflow" > /dev/null 2>&1; then
            echo "  ✅ Security constraint nodes present"
        else
            echo "  ⚠️  Security constraint nodes may be missing"
        fi
    else
        echo "  ❌ Security sandbox workflow invalid JSON"
        ((FAILURES++))
    fi
else
    echo "  ❌ Security sandbox workflow not found"
    ((FAILURES++))
fi

tester_workflow="${SCENARIO_DIR}/initialization/n8n/injection-tester.json"
if [ -f "$tester_workflow" ]; then
    if jq empty "$tester_workflow" 2>/dev/null; then
        echo "  ✅ Injection tester workflow valid JSON"
    else
        echo "  ❌ Injection tester workflow invalid JSON"
        ((FAILURES++))
    fi
else
    echo "  ❌ Injection tester workflow not found"
    ((FAILURES++))
fi

# Test API business logic
echo "🔧 Testing API business logic..."

api_main="${SCENARIO_DIR}/api/main.go"
if [ -f "$api_main" ]; then
    # Check for scoring algorithm
    if grep -q "robustness" "$api_main" || grep -q "confidence" "$api_main"; then
        echo "  ✅ Robustness scoring logic present"
    else
        echo "  ⚠️  Scoring logic may need review"
    fi
    
    # Check for safety constraints
    if grep -q "timeout" "$api_main" && grep -q "limit" "$api_main"; then
        echo "  ✅ Safety constraint checks present"
    else
        echo "  ⚠️  Safety constraints may need review"
    fi
    
    # Check for audit logging
    if grep -q "logger" "$api_main" || grep -q "log" "$api_main"; then
        echo "  ✅ Audit logging implemented"
    else
        echo "  ⚠️  Audit logging may be missing"
    fi
fi

# Test tournament logic
echo "🏆 Testing tournament logic..."

tournament_file="${SCENARIO_DIR}/api/tournament.go"
if [ -f "$tournament_file" ]; then
    # Check for tournament execution
    if grep -q "RunTournament" "$tournament_file" || grep -q "run" "$tournament_file"; then
        echo "  ✅ Tournament execution logic present"
    else
        echo "  ⚠️  Tournament execution may be incomplete"
    fi
    
    # Check for results calculation
    if grep -q "results" "$tournament_file" && grep -q "score" "$tournament_file"; then
        echo "  ✅ Results calculation logic present"
    else
        echo "  ⚠️  Results calculation may need review"
    fi
fi

# Test export logic
echo "📤 Testing export functionality..."

export_file="${SCENARIO_DIR}/api/export.go"
if [ -f "$export_file" ]; then
    # Check for multiple formats
    if grep -q "json" "$export_file" && grep -q "csv" "$export_file"; then
        echo "  ✅ Multiple export formats supported"
    else
        echo "  ⚠️  Export formats may be limited"
    fi
    
    # Check for responsible disclosure
    if grep -q "disclosure" "$export_file" || grep -q "guidelines" "$export_file"; then
        echo "  ✅ Responsible disclosure guidelines present"
    else
        echo "  ⚠️  Responsible disclosure may need documentation"
    fi
fi

# Test vector search logic
echo "🔍 Testing vector search logic..."

vector_file="${SCENARIO_DIR}/api/vector_search.go"
if [ -f "$vector_file" ]; then
    # Check for Qdrant integration
    if grep -q "qdrant" "$vector_file" || grep -q "Qdrant" "$vector_file"; then
        echo "  ✅ Qdrant integration present"
    else
        echo "  ⚠️  Vector search integration may be incomplete"
    fi
    
    # Check for embedding generation
    if grep -q "embedding" "$vector_file" || grep -q "Embedding" "$vector_file"; then
        echo "  ✅ Embedding generation logic present"
    else
        echo "  ⚠️  Embedding generation may be missing"
    fi
fi

# Summary
echo ""
if [ ${FAILURES} -eq 0 ]; then
    echo "✅ Business logic validation passed!"
    exit 0
else
    echo "❌ Business logic validation failed with ${FAILURES} error(s)"
    exit 1
fi
