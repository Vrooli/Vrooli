#!/bin/bash
# Script to run the heat transfer example

set -euo pipefail

# Configuration
readonly ELMER_PORT="${ELMER_FEM_PORT:-8192}"
readonly API_URL="http://localhost:${ELMER_PORT}"

echo "Running Heat Transfer Example..."
echo "================================"

# Wait for service to be ready
echo "Checking service health..."
if ! timeout 5 curl -sf "${API_URL}/health" > /dev/null; then
    echo "ERROR: Elmer FEM service not responding at ${API_URL}"
    exit 1
fi

# Create a test case
echo "Creating heat transfer case..."
CASE_RESPONSE=$(curl -s -X POST "${API_URL}/case/create" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "heat_transfer_test",
        "type": "heat_transfer",
        "parameters": {
            "conductivity": 50.0,
            "max_iterations": 500
        }
    }')

CASE_ID=$(echo "$CASE_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['case_id'])")
echo "Created case: $CASE_ID"

# Solve the case
echo "Running solver..."
SOLVE_RESPONSE=$(curl -s -X POST "${API_URL}/case/${CASE_ID}/solve" \
    -H "Content-Type: application/json" \
    -d '{"mpi_processes": 1}')

STATUS=$(echo "$SOLVE_RESPONSE" | python3 -c "import sys, json; print(json.load(sys.stdin)['status'])" 2>/dev/null || echo "failed")

if [[ "$STATUS" == "completed" ]]; then
    echo "✓ Solver completed successfully"
else
    echo "✗ Solver failed"
    echo "Response: $SOLVE_RESPONSE"
    exit 1
fi

# Check results
echo "Checking results..."
RESULTS_RESPONSE=$(curl -s "${API_URL}/case/${CASE_ID}/results")
RESULTS_COUNT=$(echo "$RESULTS_RESPONSE" | python3 -c "import sys, json; print(len(json.load(sys.stdin)['results']))" 2>/dev/null || echo "0")

if [[ "$RESULTS_COUNT" -gt 0 ]]; then
    echo "✓ Found $RESULTS_COUNT result files"
    echo "$RESULTS_RESPONSE" | python3 -m json.tool
else
    echo "✗ No results generated"
    exit 1
fi

echo ""
echo "Heat Transfer Example Completed Successfully!"
echo "Results saved to: /workspace/results/${CASE_ID}"