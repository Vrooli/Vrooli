#!/usr/bin/env bash

# Simple Claude Code Prompts Example
# Demonstrates basic usage of the management script API
# Note: This script uses the corrected CLI interface (fixed January 2025)

set -euo pipefail

echo "=== Claude Code Simple Prompts Example ==="
echo

# Check if Claude Code is available
if ! command -v claude &> /dev/null; then
    echo "❌ Claude Code not found. Please install first:"
    echo "   ./manage.sh --action install"
    exit 1
fi

# Verify Claude Code status
echo "📋 Checking Claude Code status..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANAGE_SCRIPT="$SCRIPT_DIR/../../manage.sh"
if ! "$MANAGE_SCRIPT" --action status &> /dev/null; then
    echo "❌ Claude Code not properly installed"
    exit 1
fi
echo "✅ Claude Code is ready"
echo

# Example 1: Simple code explanation
echo "🔍 Example 1: Code Explanation"
echo "Creating a sample JavaScript function..."

cat > sample-function.js << 'EOF'
function fibonacci(n) {
    if (n <= 1) return n;
    return fibonacci(n - 1) + fibonacci(n - 2);
}
EOF

echo "Sample code created: sample-function.js"
echo "Asking Claude to explain this function..."

"$MANAGE_SCRIPT" --action run \
  --prompt "Explain this JavaScript function and suggest improvements" \
  --allowed-tools "Read" \
  --max-turns 3

echo
echo "✅ Code explanation completed"
echo

# Example 2: Security analysis
echo "🔒 Example 2: Security Analysis"
echo "Creating a sample with potential security issues..."

cat > sample-security.js << 'EOF'
const express = require('express');
const app = express();

app.get('/user/:id', (req, res) => {
    const userId = req.params.id;
    const query = `SELECT * FROM users WHERE id = ${userId}`;
    // Execute query (SQL injection vulnerable)
    res.json({ query: query });
});
EOF

echo "Sample code created: sample-security.js"
echo "Asking Claude to identify security issues..."

"$MANAGE_SCRIPT" --action run \
  --prompt "Review this Express.js code for security vulnerabilities" \
  --allowed-tools "Read" \
  --max-turns 2

echo
echo "✅ Security analysis completed"
echo

# Example 3: Documentation generation
echo "📝 Example 3: Documentation Generation"
echo "Asking Claude to generate documentation for our functions..."

"$MANAGE_SCRIPT" --action run \
  --prompt "Generate JSDoc documentation for all JavaScript functions in the current directory" \
  --allowed-tools "Read,Write" \
  --max-turns 5

echo
echo "✅ Documentation generation completed"
echo

# Cleanup
echo "🧹 Cleaning up example files..."
rm -f sample-function.js sample-security.js

echo
echo "🎉 All examples completed successfully!"
echo
echo "💡 Next steps:"
echo "   - Try interactive mode: claude"
echo "   - Explore more complex workflows in other example directories"
echo "   - Read the API documentation: docs/API.md"