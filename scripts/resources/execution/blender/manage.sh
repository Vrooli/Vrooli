#!/bin/bash
# Blender management script - thin wrapper over CLI

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Forward all commands to CLI
"${SCRIPT_DIR}/cli.sh" "$@"