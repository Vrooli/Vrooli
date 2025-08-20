#!/bin/bash

# Wiki.js Basic Usage Example
# This example demonstrates common Wiki.js operations

set -euo pipefail

echo "=== Wiki.js Basic Usage Example ==="
echo ""

# Get the resource CLI path
WIKIJS_CLI="resource-wikijs"

# Check if Wiki.js is installed
echo "1. Checking Wiki.js status..."
$WIKIJS_CLI status
echo ""

# If not installed, show how to install
if ! $WIKIJS_CLI status 2>/dev/null | grep -q "Running: Yes"; then
    echo "Wiki.js is not running. To install and start:"
    echo "  $WIKIJS_CLI install"
    echo "  $WIKIJS_CLI start"
    echo ""
    exit 0
fi

# Create a test page
echo "2. Creating a test page..."
$WIKIJS_CLI api create-page \
    --title "Test Page" \
    --content "This is a test page created via the CLI" \
    --path "test-page"
echo ""

# List all pages
echo "3. Listing all pages..."
$WIKIJS_CLI api list-pages
echo ""

# Search for content
echo "4. Searching for 'test'..."
$WIKIJS_CLI search "test"
echo ""

# Show how to inject a markdown file
echo "5. Injecting a markdown file..."
echo "To inject a markdown file into Wiki.js:"
echo "  $WIKIJS_CLI inject --file /path/to/file.md --title 'Page Title'"
echo ""

# Show backup command
echo "6. Creating a backup..."
echo "To create a backup:"
echo "  $WIKIJS_CLI backup"
echo ""

# Show logs command
echo "7. Viewing logs..."
echo "To view Wiki.js logs:"
echo "  $WIKIJS_CLI logs --lines 20"
echo ""

echo "=== Example Complete ==="
echo "Wiki.js is running at: http://localhost:$(scripts/resources/port_registry.sh get wikijs)"