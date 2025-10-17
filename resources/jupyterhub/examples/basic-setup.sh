#!/bin/bash
# Basic JupyterHub Setup Example

set -euo pipefail

echo "🚀 JupyterHub Basic Setup Example"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Install JupyterHub
echo "1. Installing JupyterHub..."
resource-jupyterhub manage install

# Start the service
echo "2. Starting JupyterHub..."
resource-jupyterhub manage start --wait

# Check status
echo "3. Checking status..."
resource-jupyterhub status

# Show credentials
echo "4. Access credentials:"
resource-jupyterhub credentials

echo ""
echo "✅ Setup complete!"
echo "Access JupyterHub at: http://localhost:8000"
echo ""
echo "Next steps:"
echo "  - Create users: resource-jupyterhub content add --type user --name <username>"
echo "  - Start a notebook: resource-jupyterhub content spawn --user <username>"
echo "  - Install extensions: resource-jupyterhub content add --type extension --name <extension>"