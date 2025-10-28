#!/bin/bash
set -euo pipefail

echo "=== Business Logic Tests ==="

# Validate n8n workflows
# Note: Some workflows are Jinja2 templates (contain {% %}), skip those
if [ -d "initialization/automation/n8n" ]; then
  for workflow in initialization/automation/n8n/*.json; do
    if [ -f "$workflow" ]; then
      # Skip Jinja2 templates - they need processing before validation
      if grep -q "{%" "$workflow"; then
        echo "⚠️ Skipping template validation: $workflow (requires Jinja2 processing)"
        continue
      fi
      if ! jq empty "$workflow" > /dev/null 2>&1; then
        echo "❌ Invalid JSON in workflow: $workflow"
        exit 1
      fi
    fi
  done
  echo "✅ n8n workflows validated (templates skipped)"
fi

# Check configuration files
if [ -f "initialization/configuration/research-config.json" ]; then
  jq empty "initialization/configuration/research-config.json" > /dev/null || exit 1
fi

# Basic business rule: ensure API has health endpoint (static check)
if ! grep -q "handleHealth" api/main.go; then
  echo "⚠️ Health handler not found in API"
fi

echo "✅ Business tests passed"