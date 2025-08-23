#!/usr/bin/env bash
# Judge0 Languages Module
# Handles programming language management and queries

#######################################
# List all available languages
#######################################
judge0::languages::list() {
    log::info "$JUDGE0_MSG_LANG_LIST"
    
    if ! judge0::is_running; then
        log::error "Judge0 is not running. Showing default languages..."
        judge0::languages::list_defaults
        return 0
    fi
    
    # Get languages from API
    local languages=$(judge0::api::get_languages 2>/dev/null)
    
    if [[ -z "$languages" ]] || [[ "$languages" == "[]" ]]; then
        log::error "Failed to retrieve languages from API"
        judge0::languages::list_defaults
        return 1
    fi
    
    # Display languages in a formatted table
    echo
    printf "%-5s %-20s %-30s %s\n" "ID" "Name" "Version" "Compiler"
    printf "%-5s %-20s %-30s %s\n" "---" "----" "-------" "--------"
    
    echo "$languages" | jq -r '.[] | 
        "\(.id)\t\(.name)\t\(.source_file // "N/A")\t\(.compile_cmd // .run_cmd // "N/A")"' | \
    while IFS=$'\t' read -r id name source_file cmd; do
        # Extract version from name if available
        local version=""
        if [[ "$name" =~ \((.*)\) ]]; then
            version="${BASH_REMATCH[1]}"
            name="${name%% (*}"
        fi
        
        # Truncate long commands
        if [[ ${#cmd} -gt 30 ]]; then
            cmd="${cmd:0:27}..."
        fi
        
        printf "%-5s %-20s %-30s %s\n" "$id" "$name" "$version" "$cmd"
    done | sort -n -k1
    
    echo
    local count=$(echo "$languages" | jq 'length')
    log::info "Total languages available: $count"
}

#######################################
# List default languages (offline)
#######################################
judge0::languages::list_defaults() {
    echo
    echo "Default languages (may not reflect actual availability):"
    echo
    printf "%-15s %s\n" "Language" "ID"
    printf "%-15s %s\n" "--------" "--"
    
    for lang_entry in "${JUDGE0_DEFAULT_LANGUAGES[@]}"; do
        local name="${lang_entry%%:*}"
        local id="${lang_entry##*:}"
        printf "%-15s %s\n" "$name" "$id"
    done
    echo
}

#######################################
# Get detailed language info
# Arguments:
#   $1 - Language name or ID
#######################################
judge0::languages::info() {
    local lang="$1"
    
    if ! judge0::is_running; then
        log::error "Judge0 is not running"
        return 1
    fi
    
    # Try to get language ID
    local language_id=""
    
    # Check if input is a number (ID)
    if [[ "$lang" =~ ^[0-9]+$ ]]; then
        language_id="$lang"
    else
        # Try to find by name
        language_id=$(judge0::get_language_id "$lang")
        
        if [[ -z "$language_id" ]]; then
            # Try API search
            local languages=$(judge0::api::get_languages 2>/dev/null || echo "[]")
            language_id=$(echo "$languages" | jq -r ".[] | select(.name | ascii_downcase | contains(\"$lang\" | ascii_downcase)) | .id" | head -1)
        fi
    fi
    
    if [[ -z "$language_id" ]]; then
        log::error "$(printf "$JUDGE0_MSG_LANG_NOT_FOUND" "$lang")"
        return 1
    fi
    
    # Get language details
    local lang_info=$(judge0::api::get_language "$language_id" 2>/dev/null)
    
    if [[ -z "$lang_info" ]] || [[ "$lang_info" == "{}" ]]; then
        log::error "Failed to retrieve language information"
        return 1
    fi
    
    # Display language info
    echo
    echo "📋 Language Information:"
    echo "$lang_info" | jq -r '
        "  ID: \(.id)",
        "  Name: \(.name)",
        "  Source File: \(.source_file // "N/A")",
        "  Compile Command: \(.compile_cmd // "N/A")",
        "  Run Command: \(.run_cmd // "N/A")"
    '
    echo
}

#######################################
# Test language with sample code
# Arguments:
#   $1 - Language name or ID
#######################################
judge0::languages::test() {
    local lang="$1"
    
    # Sample programs for different languages
    declare -A sample_programs=(
        ["javascript"]='console.log("Hello from JavaScript!");'
        ["python"]='print("Hello from Python!")'
        ["go"]='package main\nimport "fmt"\nfunc main() {\n    fmt.Println("Hello from Go!")\n}'
        ["rust"]='fn main() {\n    println!("Hello from Rust!");\n}'
        ["java"]='public class Main {\n    public static void main(String[] args) {\n        System.out.println("Hello from Java!");\n    }\n}'
        ["cpp"]='#include <iostream>\nint main() {\n    std::cout << "Hello from C++!" << std::endl;\n    return 0;\n}'
        ["c"]='#include <stdio.h>\nint main() {\n    printf("Hello from C!\\n");\n    return 0;\n}'
        ["ruby"]='puts "Hello from Ruby!"'
        ["php"]='<?php\necho "Hello from PHP!\\n";'
        ["bash"]='echo "Hello from Bash!"'
    )
    
    # Get sample code
    local sample_code="${sample_programs[$lang]}"
    
    if [[ -z "$sample_code" ]]; then
        # Try generic print statement
        sample_code="print(\"Hello from $lang!\")"
    fi
    
    log::info "Testing $lang with sample code..."
    echo
    echo "📝 Code:"
    echo -e "$sample_code"
    echo
    
    # Submit and execute
    judge0::api::submit "$sample_code" "$lang"
}

#######################################
# Install language support
# Arguments:
#   $1 - Language name
#######################################
judge0::languages::install() {
    local lang="$1"
    
    log::info "$(printf "$JUDGE0_MSG_LANG_INSTALL" "$lang")"
    
    # Note: Judge0 comes with all languages pre-installed
    # This function is mainly for consistency with other resources
    
    if judge0::validate_language "$lang"; then
        log::success "Language '$lang' is already available"
    else
        log::error "Language '$lang' is not supported by Judge0"
        echo
        echo "Run '$0 --action languages' to see available languages"
    fi
}

#######################################
# Search for languages by keyword
# Arguments:
#   $1 - Search keyword
#######################################
judge0::languages::search() {
    local keyword="$1"
    
    if ! judge0::is_running; then
        log::error "Judge0 is not running"
        return 1
    fi
    
    log::info "Searching for languages matching '$keyword'..."
    
    local languages=$(judge0::api::get_languages 2>/dev/null || echo "[]")
    
    if [[ "$languages" == "[]" ]]; then
        log::error "Failed to retrieve languages"
        return 1
    fi
    
    # Filter languages
    local matches=$(echo "$languages" | jq -r ".[] | select(.name | ascii_downcase | contains(\"$keyword\" | ascii_downcase))")
    
    if [[ -z "$matches" ]]; then
        log::warning "No languages found matching '$keyword'"
        return 0
    fi
    
    # Display matches
    echo
    printf "%-5s %-30s\n" "ID" "Name"
    printf "%-5s %-30s\n" "---" "----"
    
    echo "$matches" | jq -r '"\(.id)\t\(.name)"' | \
    while IFS=$'\t' read -r id name; do
        printf "%-5s %-30s\n" "$id" "$name"
    done
    echo
}

#######################################
# Get language statistics
#######################################
judge0::languages::stats() {
    if ! judge0::is_running; then
        log::error "Judge0 is not running"
        return 1
    fi
    
    log::info "📊 Language Statistics:"
    
    local languages=$(judge0::api::get_languages 2>/dev/null || echo "[]")
    
    if [[ "$languages" == "[]" ]]; then
        log::error "Failed to retrieve languages"
        return 1
    fi
    
    # Count languages by category
    local total=$(echo "$languages" | jq 'length')
    local compiled=$(echo "$languages" | jq '[.[] | select(.compile_cmd != null)] | length')
    local interpreted=$((total - compiled))
    
    echo
    echo "  Total Languages: $total"
    echo "  Compiled: $compiled"
    echo "  Interpreted: $interpreted"
    echo
    
    # Popular languages check
    echo "  Popular Languages Available:"
    local popular=("JavaScript" "Python" "Java" "C++" "Go" "Rust" "TypeScript" "Ruby" "PHP" "Swift")
    
    for lang in "${popular[@]}"; do
        if echo "$languages" | jq -e ".[] | select(.name | contains(\"$lang\"))" >/dev/null 2>&1; then
            echo "    ✅ $lang"
        else
            echo "    ❌ $lang"
        fi
    done
    echo
}