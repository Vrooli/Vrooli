#!/usr/bin/env bash
# Convert existing BATS tests to use shared mock infrastructure
# This script updates BATS test files to use centralized mocks instead of duplicated ones

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$(dirname "$SCRIPT_DIR")"

# Track what we've updated
UPDATED_FILES=()
CONVERSION_LOG="$SCRIPT_DIR/conversion.log"

# Initialize log
echo "BATS Test Conversion Log - $(date)" > "$CONVERSION_LOG"
echo "=======================================" >> "$CONVERSION_LOG"

log_message() {
    echo "$1" | tee -a "$CONVERSION_LOG"
}

# Function to calculate relative path from test file to shared mocks
get_relative_path() {
    local test_file="$1"
    local test_dir="$(dirname "$test_file")"
    local mocks_dir="$SCRIPT_DIR/bats-fixtures"
    
    # Count directory depth from resources root
    local rel_path="../../../tests/bats-fixtures/common_setup.bash"
    
    # Adjust path based on how deep the test file is
    local depth=$(echo "$test_file" | sed "s|$RESOURCES_DIR/||" | tr '/' '\n' | wc -l)
    
    case $depth in
        3) echo "../../../tests/bats-fixtures/common_setup.bash" ;;  # ai/ollama/lib/
        4) echo "../../../../tests/bats-fixtures/common_setup.bash" ;;  # automation/n8n/lib/
        *) echo "../../../tests/bats-fixtures/common_setup.bash" ;;  # Default
    esac
}

# Function to convert a single BATS file
convert_bats_file() {
    local file="$1"
    local backup_file="${file}.backup.$(date +%Y%m%d_%H%M%S)"
    
    log_message "Converting: $file"
    
    # Create backup
    cp "$file" "$backup_file"
    log_message "  Backup created: $backup_file"
    
    # Get relative path to shared mocks
    local rel_path=$(get_relative_path "$file")
    
    # Create temporary file for modifications
    local temp_file=$(mktemp)
    
    # Process the file line by line
    local in_setup=false
    local setup_started=false
    local found_duplicated_mocks=false
    
    while IFS= read -r line; do
        # Check if we're entering setup function
        if [[ "$line" =~ ^setup\(\)[[:space:]]*\{? ]]; then
            in_setup=true
            setup_started=true
            echo "$line" >> "$temp_file"
            
            # Add shared mock loading after setup() {
            if [[ "$line" =~ \{ ]]; then
                echo "    # Load shared test infrastructure" >> "$temp_file"
                echo "    source \"\$(dirname \"\${BATS_TEST_FILENAME}\")/$rel_path\"" >> "$temp_file"
                echo "    " >> "$temp_file"
                echo "    # Setup standard mocks" >> "$temp_file"
                echo "    setup_standard_mocks" >> "$temp_file"
                echo "    " >> "$temp_file"
            fi
            continue
        fi
        
        # Check if we're exiting setup function
        if [[ "$in_setup" == "true" && "$line" =~ ^\} ]]; then
            in_setup=false
            echo "$line" >> "$temp_file"
            continue
        fi
        
        # Skip duplicated mock definitions in setup
        if [[ "$in_setup" == "true" ]]; then
            # Skip common duplicated mocks
            if [[ "$line" =~ ^[[:space:]]*log::(info|success|error|warn|header|debug)\(\) ]] ||
               [[ "$line" =~ ^[[:space:]]*system::is_command\(\) ]] ||
               [[ "$line" =~ ^[[:space:]]*docker\(\) ]] ||
               [[ "$line" =~ ^[[:space:]]*curl\(\) ]] && [[ ! "$line" =~ "Override curl mock" ]]; then
                found_duplicated_mocks=true
                # Skip this line and any following lines that are part of the function
                continue
            fi
            
            # Skip lines that are part of mock function bodies
            if [[ "$found_duplicated_mocks" == "true" ]] && 
               [[ "$line" =~ ^[[:space:]]*echo || 
                  "$line" =~ ^[[:space:]]*return || 
                  "$line" =~ ^[[:space:]]*case ||
                  "$line" =~ ^[[:space:]]*\} ||
                  "$line" =~ ^[[:space:]]*esac ||
                  "$line" =~ ^[[:space:]]*\*\) ||
                  "$line" =~ ^[[:space:]]*;; ||
                  "$line" =~ ^[[:space:]]*if ||
                  "$line" =~ ^[[:space:]]*fi ||
                  "$line" =~ ^[[:space:]]*while ||
                  "$line" =~ ^[[:space:]]*done ||
                  "$line" =~ ^[[:space:]]*# ]]; then
                continue
            fi
            
            # Reset flag when we hit a non-mock line
            if [[ ! "$line" =~ ^[[:space:]]*$ ]] && 
               [[ ! "$line" =~ ^[[:space:]]*# ]] &&
               [[ ! "$line" =~ ^[[:space:]]*echo ]] &&
               [[ ! "$line" =~ ^[[:space:]]*return ]] &&
               [[ ! "$line" =~ ^[[:space:]]*case ]] &&
               [[ ! "$line" =~ ^[[:space:]]*\} ]] &&
               [[ ! "$line" =~ ^[[:space:]]*esac ]] &&
               [[ ! "$line" =~ ^[[:space:]]*\*\) ]] &&
               [[ ! "$line" =~ ^[[:space:]]*;; ]]; then
                found_duplicated_mocks=false
            fi
        fi
        
        # Skip obviously duplicated system utility mocks
        if [[ "$line" =~ ^[[:space:]]*(jq|wc|tr|sed|cat|echo|date|which|command|systemctl|nc|ping|sudo|lsof)\(\) ]] &&
           [[ "$line" =~ /usr/bin/\|/bin/ ]]; then
            # Skip pass-through mocks like: jq() { /usr/bin/jq "$@"; }
            continue
        fi
        
        # Add all other lines
        echo "$line" >> "$temp_file"
        
    done < "$file"
    
    # Replace original file with modified version
    mv "$temp_file" "$file"
    
    UPDATED_FILES+=("$file")
    log_message "  ✓ Converted successfully"
}

# Function to find all BATS files that need conversion
find_bats_files() {
    find "$RESOURCES_DIR" -name "*.bats" -type f | \
        grep -v "/tests/" | \
        sort
}

# Function to check if a file needs conversion
needs_conversion() {
    local file="$1"
    
    # Check if it already uses shared mocks
    if grep -q "common_setup.bash" "$file"; then
        return 1  # Already converted
    fi
    
    # Check if it has duplicated mocks
    if grep -q "log::info()" "$file" || 
       grep -q "system::is_command()" "$file" ||
       grep -q "^[[:space:]]*docker()" "$file"; then
        return 0  # Needs conversion
    fi
    
    return 1  # Doesn't need conversion
}

# Main conversion process
main() {
    log_message "Starting BATS test conversion to shared mocks..."
    log_message "Scanning for BATS files that need conversion..."
    
    local total_files=0
    local converted_files=0
    local skipped_files=0
    
    while IFS= read -r file; do
        ((total_files++))
        
        if needs_conversion "$file"; then
            convert_bats_file "$file"
            ((converted_files++))
        else
            log_message "Skipping: $file (already uses shared mocks or no duplicated mocks found)"
            ((skipped_files++))
        fi
    done < <(find_bats_files)
    
    log_message ""
    log_message "======================================="
    log_message "Conversion Summary:"
    log_message "  Total files scanned: $total_files"
    log_message "  Files converted: $converted_files"
    log_message "  Files skipped: $skipped_files"
    log_message ""
    
    if [[ $converted_files -gt 0 ]]; then
        log_message "Files that were converted:"
        for file in "${UPDATED_FILES[@]}"; do
            log_message "  - $file"
        done
        log_message ""
        log_message "Backups were created with .backup.TIMESTAMP extension"
        log_message ""
        log_message "Next steps:"
        log_message "1. Run tests to verify conversions: pnpm test:shell"
        log_message "2. If tests pass, remove backup files"
        log_message "3. If tests fail, restore from backups and check conversion script"
    else
        log_message "No files required conversion."
    fi
    
    log_message "Conversion completed!"
}

# Run the conversion if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi