#\!/bin/bash

source config/defaults.sh
source lib/common.sh
source lib/execute.sh

# Mock functions with debugging
claude_code::is_installed() { return 0; }
eval() { 
    echo "DEBUG: eval called with: $*"
    local cmd="$*"
    if [[ "$cmd" =~ \>.*(/tmp/claude-batch-[0-9]+\.json) ]]; then
        local output_file="${BASH_REMATCH[1]}"
        echo "DEBUG: Found output file: $output_file"
        echo "DEBUG: File exists before rm: $([ -f "$output_file" ] && echo yes || echo no)"
        rm -f "$output_file" 2>/dev/null
        echo "DEBUG: File exists after rm: $([ -f "$output_file" ] && echo yes || echo no)"
    fi
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
