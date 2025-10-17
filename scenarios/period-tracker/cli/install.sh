#!/bin/bash

# Period Tracker CLI Installation Script
set -e

echo "🩸 Installing Period Tracker CLI..."

# Build the CLI binary
echo "📦 Building CLI binary..."
go mod download
go build -o period-tracker main.go

# Make it executable
chmod +x period-tracker

# Create ~/.local/bin if it doesn't exist
mkdir -p ~/.local/bin

# Create symlink in ~/.local/bin
echo "🔗 Installing CLI to ~/.local/bin/period-tracker..."
ln -sf "$(pwd)/period-tracker" ~/.local/bin/period-tracker

# Check if ~/.local/bin is in PATH
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo "⚠️  Note: ~/.local/bin is not in your PATH"
    echo "   Add this to your shell profile (.bashrc, .zshrc, etc.):"
    echo "   export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
    echo "   Or run commands with full path: ~/.local/bin/period-tracker"
else
    echo "✅ CLI installed successfully!"
fi

echo ""
echo "🔒 Privacy Notice:"
echo "   All health data is encrypted and stored locally only."
echo "   No data is transmitted to external servers."
echo ""
echo "📚 Usage:"
echo "   period-tracker help"
echo "   period-tracker status"
echo ""
echo "🔧 Configuration:"
echo "   Set PERIOD_TRACKER_USER_ID environment variable for multi-tenant support"
echo "   Set PERIOD_TRACKER_API_HOST/PORT to customize API connection"