#!/usr/bin/env bash
# Simple script to convert BATS tests to use shared mocks
# This script uses sed-based replacements for reliability

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$(dirname "$SCRIPT_DIR")"

# Function to convert a single BATS file
convert_file() {
    local file="$1"
    local backup_file="${file}.backup.$(date +%Y%m%d_%H%M%S)"
    
    echo "Converting: $file"
    
    # Create backup
    cp "$file" "$backup_file"
    echo "  Backup: $backup_file"
    
    # Get relative path to shared mocks based on file location
    local rel_path="../../../tests/bats-fixtures/common_setup.bash"
    if [[ "$file" =~ /lib/[^/]*\.bats$ ]]; then
        # File is in a lib subdirectory
        rel_path="../../../tests/bats-fixtures/common_setup.bash"
    elif [[ "$file" =~ /[^/]*\.bats$ ]]; then
        # File is directly in resource directory
        rel_path="../../tests/bats-fixtures/common_setup.bash"
    fi
    
    # Use a Python script for more reliable text processing
    python3 << EOF
import re
import sys

def convert_bats_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()
    
    # Check if already converted
    if 'common_setup.bash' in content:
        print("  Already converted")
        return False
    
    # Pattern to match setup() function
    setup_pattern = r'(setup\(\) \{\s*\n)(.*?)(\n\})'
    
    def replace_setup(match):
        setup_start = match.group(1)
        setup_body = match.group(2)
        setup_end = match.group(3)
        
        # Remove common duplicated mocks
        lines = setup_body.split('\n')
        new_lines = []
        skip_function = False
        brace_count = 0
        
        i = 0
        while i < len(lines):
            line = lines[i]
            stripped = line.strip()
            
            # Check for mock function definitions to remove
            if re.match(r'^\s*(log::(info|success|error|warn|header|debug)|system::is_command|docker|curl)\(\)', line):
                skip_function = True
                brace_count = 0
                # Count opening braces in this line
                brace_count += line.count('{')
                brace_count -= line.count('}')
                if brace_count <= 0:
                    skip_function = False
                i += 1
                continue
            
            # Skip lines inside mock functions
            if skip_function:
                brace_count += line.count('{')
                brace_count -= line.count('}')
                if brace_count <= 0:
                    skip_function = False
                i += 1
                continue
            
            # Skip simple pass-through mocks
            if re.match(r'^\s*(jq|wc|tr|sed|cat|echo|date|which|command)\(\) \{ /.*bin/.* "\$@"; \}', line):
                i += 1
                continue
            
            new_lines.append(line)
            i += 1
        
        # Insert shared mock loading at the beginning
        shared_setup = [
            "    # Load shared test infrastructure",
            "    source \"\$(dirname \"\${BATS_TEST_FILENAME}\")/${rel_path}\"",
            "    ",
            "    # Setup standard mocks",
            "    setup_standard_mocks",
            "    "
        ]
        
        return setup_start + '\n'.join(shared_setup + new_lines) + setup_end
    
    # Apply the replacement
    new_content = re.sub(setup_pattern, replace_setup, content, flags=re.DOTALL)
    
    if new_content != content:
        with open(filepath, 'w') as f:
            f.write(new_content)
        print("  ✓ Converted")
        return True
    else:
        print("  No changes needed")
        return False

# Convert the file
convert_bats_file('${file}')
EOF

}

# Convert specific files or all if no arguments
if [[ $# -gt 0 ]]; then
    # Convert specific files
    for file in "$@"; do
        if [[ -f "$file" ]]; then
            convert_file "$file"
        else
            echo "File not found: $file"
        fi
    done
else
    echo "Please provide BATS files to convert"
    echo "Usage: $0 file1.bats [file2.bats ...]"
fi