#!/bin/bash
set -euo pipefail

echo "💼 Running Secure Document Processing business logic tests"

SCENARIO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/../.." &amp;&amp; pwd )"

# Business workflow validation
echo "🔍 Validating core business workflows..."

# Check if key business files exist and have content
business_files=(
    "PRD.md"
    "IMPLEMENTATION_PLAN.md"
    "initialization/automation/n8n/document-intake.json"
    "initialization/automation/n8n/process-orchestrator.json"
    "api/main.go"
)

for file in "${business_files[@]}"; do
    if [ -f "$SCENARIO_DIR/$file" ] &amp;&amp; [ -s "$SCENARIO_DIR/$file" ]; then
        echo "✅ Business file present and non-empty: $file"
    else
        echo "❌ Missing or empty business file: $file"
        exit 1
    fi
done

# Validate n8n workflows (basic JSON syntax)
n8n_workflows=(document-intake process-orchestrator prompt-processor semantic-indexer workflow-executor)
for workflow in "${n8n_workflows[@]}"; do
    workflow_file="initialization/automation/n8n/$workflow.json"
    if [ -f "$SCENARIO_DIR/$workflow_file" ]; then
        if jq empty "$SCENARIO_DIR/$workflow_file" 2&gt;/dev/null; then
            echo "✅ n8n workflow valid JSON: $workflow"
        else
            echo "❌ Invalid JSON in n8n workflow: $workflow"
            exit 1
        fi
    fi
done

# Check database schema for business entities
if [ -f "$SCENARIO_DIR/initialization/storage/postgres/schema.sql" ]; then
    if grep -q -E "(documents|jobs|workflows|audit_logs)" "$SCENARIO_DIR/initialization/storage/postgres/schema.sql"; then
        echo "✅ Database schema includes business entities"
    else
        echo "⚠️  Database schema may be missing business tables"
    fi
fi

# API business endpoints check (assuming service running)
# For static check, verify API has business routes
if grep -q -E "(documents|jobs|process|workflow)" "$SCENARIO_DIR/api/main.go"; then
    echo "✅ API source includes business endpoints"
else
    echo "⚠️  API may be missing business route implementations"
fi

# Security and compliance business requirements
if [ -f "$SCENARIO_DIR/initialization/configuration/encryption-config.json" ]; then
    echo "✅ Encryption configuration present"
fi

if [ -f "$SCENARIO_DIR/initialization/configuration/compliance-config.json" ]; then
    echo "✅ Compliance configuration present"
fi

echo "✅ All business logic validation passed"
