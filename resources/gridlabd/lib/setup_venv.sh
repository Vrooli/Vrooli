#!/usr/bin/env bash
# Setup Python virtual environment for GridLAB-D API

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VENV_DIR="${GRIDLABD_VENV_DIR:-${SCRIPT_DIR}/venv}"

setup_python_venv() {
    echo "Setting up Python virtual environment..."
    
    # Check Python availability
    if ! command -v python3 &> /dev/null; then
        echo "Error: Python3 not found. Please install Python 3.8+"
        return 1
    fi
    
    # Create virtual environment
    if [ ! -d "$VENV_DIR" ]; then
        echo "Creating virtual environment at $VENV_DIR..."
        python3 -m venv "$VENV_DIR"
    fi
    
    # Activate and install packages
    echo "Installing Python packages..."
    source "$VENV_DIR/bin/activate"
    
    # Upgrade pip first
    pip install --upgrade pip &>/dev/null || true
    
    # Install required packages
    pip install flask flask-cors &>/dev/null || {
        echo "Warning: Flask installation failed. API will use basic HTTP server."
    }
    
    # Install optional packages for enhanced functionality
    pip install numpy pandas matplotlib plotly &>/dev/null || {
        echo "Warning: Some optional packages failed to install"
    }
    
    echo "Python environment setup complete"
    
    # Create wrapper script for API
    cat > "${SCRIPT_DIR}/lib/run_api.sh" << 'EOF'
#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VENV_DIR="${GRIDLABD_VENV_DIR:-${SCRIPT_DIR}/venv}"

# Try to use venv Python, fall back to system Python
if [ -f "$VENV_DIR/bin/python" ]; then
    exec "$VENV_DIR/bin/python" "${SCRIPT_DIR}/lib/flask_server.py" "$@"
else
    exec python3 "${SCRIPT_DIR}/lib/flask_server.py" "$@"
fi
EOF
    chmod +x "${SCRIPT_DIR}/lib/run_api.sh"
}

# Main
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    setup_python_venv
fi