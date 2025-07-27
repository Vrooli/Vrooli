#\!/bin/bash
source ../config/defaults.sh
source common.sh
source execute.sh

claude_code::is_installed() { return 0; }
PROMPT='Test prompt'
MAX_TURNS=5
OUTPUT_FORMAT='text'
ALLOWED_TOOLS=''
eval() { echo "Command: $*"; }
log::header() { :; }
log::info() { :; }

echo "=== Running claude_code::run ==="
output=$(claude_code::run 2>&1)
echo "=== Output received ==="
echo "$output"
echo "=== End output ==="
echo "=== Testing regex ==="
if [[ "$output" =~ "Command: claude --prompt \"Test prompt\" --max-turns 5" ]]; then
    echo "MATCH: Found expected pattern"
else
    echo "NO MATCH: Pattern not found"
fi
