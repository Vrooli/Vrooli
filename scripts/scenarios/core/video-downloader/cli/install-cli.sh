#!/bin/bash

# Install Video Downloader CLI

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_NAME="video-downloader"
INSTALL_DIR="/usr/local/bin"

# Check if running with sufficient permissions
if [ ! -w "$INSTALL_DIR" ]; then
    echo "Installing Video Downloader CLI (requires sudo)..."
    sudo cp "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/$CLI_NAME"
else
    echo "Installing Video Downloader CLI..."
    cp "$SCRIPT_DIR/$CLI_NAME" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$CLI_NAME"
fi

if [ $? -eq 0 ]; then
    echo "✓ Video Downloader CLI installed successfully"
    echo "  You can now use: video-downloader --help"
else
    echo "✗ Failed to install Video Downloader CLI"
    exit 1
fi