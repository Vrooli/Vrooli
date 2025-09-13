#!/bin/bash

# OpenFOAM Example: Lid-Driven Cavity Flow
# Classic CFD benchmark case

echo "OpenFOAM Cavity Flow Example"
echo "============================"

# Start OpenFOAM if not running
vrooli resource openfoam status | grep -q "Running" || {
    echo "Starting OpenFOAM..."
    vrooli resource openfoam manage start --wait
}

# Create cavity case
echo ""
echo "Creating cavity flow case..."
vrooli resource openfoam content add cavity_example

# Execute simulation
echo ""
echo "Running simulation..."
vrooli resource openfoam content execute cavity_example

echo ""
echo "Simulation complete!"
echo "Results available in: ~/.vrooli/openfoam/cases/cavity_example/"
echo ""
echo "To visualize results:"
echo "  1. Install ParaView"
echo "  2. Open ~/.vrooli/openfoam/cases/cavity_example/VTK/cavity_example.vtk"