#!/usr/bin/env bash
# Example: Run waveguide simulation

set -euo pipefail

echo "Running waveguide simulation example..."

# Check if MEEP is running
if ! curl -sf http://localhost:8193/health > /dev/null; then
    echo "Error: MEEP service is not running"
    echo "Start it with: vrooli resource meep manage start --wait"
    exit 1
fi

# Create simulation
echo "Creating simulation..."
response=$(curl -sf -X POST http://localhost:8193/simulation/create \
    -H "Content-Type: application/json" \
    -d '{
        "template": "waveguide",
        "resolution": 50,
        "runtime": 100
    }')

sim_id=$(echo "$response" | jq -r '.simulation_id')
echo "Simulation ID: $sim_id"

# Run simulation
echo "Starting simulation..."
curl -sf -X POST "http://localhost:8193/simulation/${sim_id}/run"

# Wait for completion
echo "Waiting for simulation to complete..."
for i in {1..30}; do
    status=$(curl -sf "http://localhost:8193/simulation/${sim_id}/status" | jq -r '.status')
    if [[ "$status" == "completed" ]]; then
        echo "Simulation completed!"
        break
    elif [[ "$status" == "failed" ]]; then
        echo "Simulation failed!"
        exit 1
    fi
    echo -n "."
    sleep 2
done

# Get results
echo "Fetching results..."
spectra=$(curl -sf "http://localhost:8193/simulation/${sim_id}/spectra")

echo "Transmission spectrum:"
echo "$spectra" | jq '.transmission[:10]'

echo
echo "Results saved. To download field data:"
echo "curl -o fields.h5 http://localhost:8193/simulation/${sim_id}/fields"