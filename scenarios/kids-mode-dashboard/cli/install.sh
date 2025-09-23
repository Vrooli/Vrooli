#!/bin/bash
set -e

echo "Installing kids-mode-dashboard..."

# Initialize Go module for API if needed
if [ ! -f "../api/go.mod" ]; then
  cd ../api
  go mod init kids-mode-dashboard
  cd ../cli
fi

# Build API
cd ../api
go mod tidy
go build -o kids-mode-dashboard-api .

# Return to cli and make executable ready (assuming build later)
cd ../cli
echo "Installation complete. Run 'make build' or similar to build CLI."
