#!/bin/bash

# Installation functions for SageMath

# Define APP_ROOT using cached pattern  
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Universal Contract v2.0 handler for manage::install
sagemath::install::execute() {
    local verbose="${1:-}"
    
    echo "Installing SageMath resource..."
    
    # Ensure directories exist
    sagemath_ensure_directories
    
    # Pull Docker image
    echo "Pulling SageMath Docker image..."
    if ! docker pull "$SAGEMATH_IMAGE"; then
        echo "Error: Failed to pull SageMath image" >&2
        return 1
    fi
    
    # Create container
    echo "Creating SageMath container..."
    if sagemath_container_exists; then
        echo "Container already exists"
    else
        docker create \
            --name "$SAGEMATH_CONTAINER_NAME" \
            -p "${SAGEMATH_PORT_JUPYTER}:8888" \
            -p "${SAGEMATH_PORT_API}:8889" \
            -v "$SAGEMATH_SCRIPTS_DIR:/home/sage/scripts" \
            -v "$SAGEMATH_NOTEBOOKS_DIR:/home/sage/notebooks" \
            -v "$SAGEMATH_OUTPUTS_DIR:/home/sage/outputs" \
            -e SAGE_JUPYTER_SERVER=yes \
            "$SAGEMATH_IMAGE" \
            sage -n jupyter --no-browser --ip='0.0.0.0' --port=8888
    fi
    
    # Register CLI
    echo "Registering SageMath CLI..."
    "${APP_ROOT}/scripts/lib/resources/install-resource-cli.sh" "execution/sagemath"
    
    # Create initial test script
    cat > "$SAGEMATH_SCRIPTS_DIR/test.sage" << 'EOF'
# Test SageMath functionality
print("SageMath Test Suite")
print("=" * 30)

# Basic arithmetic
assert 2 + 2 == 4
print("✓ Basic arithmetic")

# Symbolic computation
x = var('x')
eq = x^2 + 2*x - 3 == 0
solutions = solve(eq, x)
assert len(solutions) == 2
print("✓ Symbolic solving")

# Calculus
f = sin(x)
df = diff(f, x)
assert df == cos(x)
print("✓ Differentiation")

# Linear algebra
A = matrix([[1, 2], [3, 4]])
det_A = A.det()
assert det_A == -2
print("✓ Linear algebra")

print("\nAll tests passed!")
EOF
    
    echo "SageMath installation complete"
    return 0
}

# Universal Contract v2.0 handler for manage::uninstall
sagemath::install::uninstall() {
    echo "Uninstalling SageMath resource..."
    
    # Stop container if running
    if sagemath_container_running; then
        sagemath_stop
    fi
    
    # Remove container
    if sagemath_container_exists; then
        echo "Removing SageMath container..."
        docker rm -f "$SAGEMATH_CONTAINER_NAME"
    fi
    
    # Unregister CLI
    echo "Unregistering SageMath CLI..."
    "${APP_ROOT}/scripts/lib/resources/uninstall-resource-cli.sh" sagemath
    
    echo "SageMath uninstallation complete"
    return 0
}