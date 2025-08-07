#!/usr/bin/env bash
# Debug script for syntax validator

set -euo pipefail

# Source required components
source "./tests/framework/parsers/contract-parser.sh"
source "./tests/framework/parsers/script-analyzer.sh"

echo "=== Debug Syntax Validator ==="

# Initialize contract parser
echo "1. Initializing contract parser..."
contract_parser_init "./contracts"

# Test get_required_actions
echo "2. Getting required actions for AI..."
required_actions=$(get_required_actions "ai")
echo "Required actions: $required_actions"

# Test extract_script_actions
echo "3. Extracting actions from ollama script..."
implemented_actions=$(extract_script_actions "./ai/ollama/manage.sh")
echo "Implemented actions: $implemented_actions"

# Test comparison logic
echo "4. Comparing actions..."
missing_actions=()
found_actions=()

while IFS= read -r action; do
    [[ -z "$action" ]] && continue
    
    if echo "$implemented_actions" | grep -q "^$action$"; then
        found_actions+=("$action")
        echo "✅ Found: $action"
    else
        missing_actions+=("$action")
        echo "❌ Missing: $action"
    fi
done <<< "$required_actions"

echo "Found actions: ${found_actions[*]}"
echo "Missing actions: ${missing_actions[*]}"

echo "=== Debug Complete ==="