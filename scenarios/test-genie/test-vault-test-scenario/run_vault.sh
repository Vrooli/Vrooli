#!/bin/bash
set -e

VAULT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_NAME=$(basename "$VAULT_DIR" | sed 's/test-vault-//')

echo "=== Test Vault Execution for $SCENARIO_NAME ==="

# Load vault configuration
if [ -f "$VAULT_DIR/vault.yaml" ]; then
    echo "Loading vault configuration..."
else
    echo "Error: vault.yaml not found"
    exit 1
fi

# Execute phases in order
for phase_file in "$VAULT_DIR"/phases/*.yaml; do
    if [ -f "$phase_file" ]; then
        phase_name=$(basename "$phase_file" .yaml)
        echo ""
        echo "--- Executing Phase: $phase_name ---"
        
        # Here you would integrate with your actual test execution system
        # For now, we'll simulate phase execution
        echo "Phase $phase_name: STARTED"
        sleep 2
        echo "Phase $phase_name: COMPLETED"
    fi
done

echo ""
echo "=== Vault Execution Completed ==="
