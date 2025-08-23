#!/usr/bin/env bash
# Basic setup script for Web Scraper Manager

set -euo pipefail

echo "🔧 Setting up Web Scraper Manager environment..."

# Create necessary directories
mkdir -p data/exports data/screenshots data/assets

# Set permissions
chmod 755 data data/exports data/screenshots data/assets

echo "✅ Basic setup completed"