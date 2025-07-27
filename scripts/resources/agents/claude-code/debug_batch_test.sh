#\!/bin/bash

source config/defaults.sh
source lib/common.sh
source lib/execute.sh

# Mock functions
claude_code::is_installed() { return 0; }
eval() { 
    echo "DEBUG: eval called with: $*"
    return 1
}
log::header() { echo "HEADER: $*"; }
log::info() { echo "INFO: $*"; }
log::success() { echo "SUCCESS: $*"; }
log::error() { echo "ERROR: $*"; }

# Set variables
export PROMPT='Test batch'
export ALLOWED_TOOLS=''
export TIMEOUT=600
export MAX_TURNS=5

echo "=== Running claude_code::batch ==="
claude_code::batch
exit_code=$?
echo "=== Exit code: $exit_code ==="
