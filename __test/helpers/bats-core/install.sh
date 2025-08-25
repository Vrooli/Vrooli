#!/bin/bash
# Simple BATS install script - logs installation attempts
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_LOG_DIR="$(dirname "$SCRIPT_DIR")"
echo "INSTALL_CALLED with prefix: $1" > "$INSTALL_LOG_DIR/install_result"
