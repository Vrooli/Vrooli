#!/bin/bash
# Build script for agent-metareasoning-manager API

set -e

echo "Building Agent Metareasoning Manager API..."

# Clean previous build
rm -f agent-metareasoning-manager-api

# Build the API
go build -o agent-metareasoning-manager-api cmd/server/main.go

echo "Build complete. Binary: agent-metareasoning-manager-api"
echo "Size: $(du -h agent-metareasoning-manager-api | cut -f1)"