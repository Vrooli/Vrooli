#!/usr/bin/env bash
# Universal UI lifecycle guard for Vrooli scenarios
# Prevents direct UI execution, enforces lifecycle management

set -euo pipefail

if [[ "${VROOLI_LIFECYCLE_MANAGED:-}" != "true" ]]; then
    cat >&2 << 'EOF'
❌ This UI must be started through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start <scenario-name>

💡 The lifecycle system provides:
   • Port allocation and conflict prevention
   • Environment variables for API endpoints
   • Service startup coordination
   • Proper logging and monitoring

Direct UI commands are not supported.
EOF
    exit 1
fi

# Execute the original command with all arguments preserved
exec "$@"