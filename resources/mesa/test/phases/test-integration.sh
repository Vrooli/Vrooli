#!/usr/bin/env bash
# Mesa Integration Tests
# Full functionality validation per v2.0 contract (<120s)

set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly TEST_DIR
readonly MESA_URL="http://localhost:9512"

echo "=== Mesa Integration Tests ==="
echo "Testing complete functionality..."

# Test 1: Model execution with deterministic output
echo -n "1. Testing Schelling model execution... "
response=$(curl -sf -X POST "${MESA_URL}/simulate" \
    -H "Content-Type: application/json" \
    -d '{"model": "schelling", "steps": 10, "seed": 42}' 2>/dev/null || echo "failed")

if [[ "$response" != "failed" ]] && echo "$response" | grep -q "results"; then
    echo "✓"
else
    echo "✗ (Model execution failed)"
    exit 1
fi

# Test 2: Metrics export functionality
echo -n "2. Testing metrics export... "
metrics=$(curl -sf "${MESA_URL}/metrics/latest" 2>/dev/null || echo "failed")

if [[ "$metrics" != "failed" ]]; then
    echo "✓"
else
    echo "✗ (Metrics export failed)"
    exit 1
fi

# Test 3: Batch simulation with parameter sweep
echo -n "3. Testing batch simulation... "
batch=$(curl -sf -X POST "${MESA_URL}/batch" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "schelling",
        "parameters": {
            "similarity": [0.3, 0.5, 0.7],
            "density": [0.8]
        },
        "steps": 5,
        "runs": 2,
        "seed": 42
    }' 2>/dev/null || echo "failed")

if [[ "$batch" != "failed" ]] && echo "$batch" | grep -q "batch_id"; then
    echo "✓"
else
    echo "✗ (Batch simulation failed)"
    exit 1
fi

# Test 4: State snapshot export
echo -n "4. Testing state snapshot export... "
snapshot=$(curl -sf -X POST "${MESA_URL}/simulate" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "schelling",
        "steps": 5,
        "export_snapshots": true,
        "seed": 42
    }' 2>/dev/null || echo "failed")

if [[ "$snapshot" != "failed" ]] && echo "$snapshot" | grep -q "snapshots"; then
    echo "✓"
else
    echo "✗ (Snapshot export failed)"
    exit 1
fi

# Test 5: Virus model execution
echo -n "5. Testing virus epidemiology model... "
virus=$(curl -sf -X POST "${MESA_URL}/simulate" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "virus",
        "steps": 20,
        "parameters": {
            "num_agents": 50,
            "avg_node_degree": 3,
            "initial_infection": 0.1,
            "virus_spread_chance": 0.4,
            "recovery_chance": 0.3
        },
        "seed": 42
    }' 2>/dev/null || echo "failed")

if [[ "$virus" != "failed" ]] && echo "$virus" | grep -q "infected"; then
    echo "✓"
else
    echo "✗ (Virus model failed)"
    exit 1
fi

# Test 6: Results retrieval
echo -n "6. Testing results retrieval... "
results=$(curl -sf "${MESA_URL}/results" 2>/dev/null || echo "failed")

if [[ "$results" != "failed" ]]; then
    echo "✓"
else
    echo "✗ (Results retrieval failed)"
    exit 1
fi

# Test 7: Deterministic execution validation
echo -n "7. Testing deterministic execution... "
run1=$(curl -sf -X POST "${MESA_URL}/simulate" \
    -H "Content-Type: application/json" \
    -d '{"model": "schelling", "steps": 5, "seed": 123}' 2>/dev/null)

run2=$(curl -sf -X POST "${MESA_URL}/simulate" \
    -H "Content-Type: application/json" \
    -d '{"model": "schelling", "steps": 5, "seed": 123}' 2>/dev/null)

if [[ "$run1" == "$run2" ]]; then
    echo "✓"
else
    echo "✗ (Non-deterministic results)"
    exit 1
fi

echo ""
echo "=== Integration Tests Passed ==="
echo "All Mesa functionality validated!"
exit 0