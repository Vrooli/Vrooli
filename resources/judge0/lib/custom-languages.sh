#!/usr/bin/env bash
# Judge0 Custom Language Support Module
# Enables adding and managing custom language configurations

# Source shared utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/judge0/lib/common.sh"

# Custom languages configuration directory
export JUDGE0_CUSTOM_LANGS_DIR="${JUDGE0_CONFIG_DIR}/custom_languages"

#######################################
# Initialize custom languages directory
#######################################
judge0::custom_langs::init() {
    if [[ ! -d "$JUDGE0_CUSTOM_LANGS_DIR" ]]; then
        mkdir -p "$JUDGE0_CUSTOM_LANGS_DIR"
    fi
    
    # Create registry if it doesn't exist
    local registry_file="${JUDGE0_CUSTOM_LANGS_DIR}/registry.json"
    if [[ ! -f "$registry_file" ]]; then
        echo '{"languages": [], "next_id": 1000}' > "$registry_file"
    fi
}

#######################################
# Add custom language configuration
# Arguments:
#   $1 - Language name
#   $2 - Compiler/interpreter command
#   $3 - File extension
#   $4 - Optional: compile command
# Outputs:
#   Language ID assigned
#######################################
judge0::custom_langs::add() {
    local name="$1"
    local run_command="$2"
    local extension="$3"
    local compile_command="${4:-}"
    
    if [[ -z "$name" ]] || [[ -z "$run_command" ]] || [[ -z "$extension" ]]; then
        log::error "Language name, run command, and extension are required"
        return 1
    fi
    
    judge0::custom_langs::init
    
    local registry_file="${JUDGE0_CUSTOM_LANGS_DIR}/registry.json"
    local registry=$(cat "$registry_file")
    
    # Check if language already exists
    local exists=$(echo "$registry" | jq -r --arg name "$name" '.languages[] | select(.name == $name) | .name')
    if [[ -n "$exists" ]]; then
        log::error "Language already exists: $name"
        return 1
    fi
    
    # Get next available ID
    local lang_id=$(echo "$registry" | jq -r '.next_id')
    
    # Create language configuration
    local config=$(jq -n \
        --arg id "$lang_id" \
        --arg name "$name" \
        --arg ext "$extension" \
        --arg run "$run_command" \
        --arg compile "$compile_command" \
        --arg created "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        '{
            id: ($id | tonumber),
            name: $name,
            extension: $ext,
            run_command: $run,
            compile_command: $compile,
            time_limit: 10,
            memory_limit: 262144,
            created: $created,
            enabled: true
        }')
    
    # Save language configuration
    local config_file="${JUDGE0_CUSTOM_LANGS_DIR}/${lang_id}.json"
    echo "$config" > "$config_file"
    
    # Update registry
    registry=$(echo "$registry" | jq \
        --argjson lang "$config" \
        '.languages += [$lang] | .next_id += 1')
    echo "$registry" > "$registry_file"
    
    log::success "Custom language added: $name (ID: $lang_id)"
    echo "$lang_id"
    return 0
}

#######################################
# List custom languages
# Outputs:
#   List of configured custom languages
#######################################
judge0::custom_langs::list() {
    judge0::custom_langs::init
    
    local registry_file="${JUDGE0_CUSTOM_LANGS_DIR}/registry.json"
    local registry=$(cat "$registry_file")
    
    local count=$(echo "$registry" | jq -r '.languages | length')
    
    if [[ "$count" -eq 0 ]]; then
        log::info "No custom languages configured"
        return 0
    fi
    
    log::header "Custom Languages ($count)"
    
    echo "$registry" | jq -r '.languages[] | "ID: \(.id)\nName: \(.name)\nExtension: \(.extension)\nRun: \(.run_command)\nCompile: \(.compile_command // "N/A")\nEnabled: \(.enabled)\n"'
    
    return 0
}

#######################################
# Remove custom language
# Arguments:
#   $1 - Language ID or name
# Outputs:
#   Removal status
#######################################
judge0::custom_langs::remove() {
    local identifier="$1"
    
    if [[ -z "$identifier" ]]; then
        log::error "Language ID or name is required"
        return 1
    fi
    
    judge0::custom_langs::init
    
    local registry_file="${JUDGE0_CUSTOM_LANGS_DIR}/registry.json"
    local registry=$(cat "$registry_file")
    
    # Find language by ID or name
    local lang_id
    if [[ "$identifier" =~ ^[0-9]+$ ]]; then
        lang_id="$identifier"
    else
        lang_id=$(echo "$registry" | jq -r --arg name "$identifier" '.languages[] | select(.name == $name) | .id')
    fi
    
    if [[ -z "$lang_id" ]]; then
        log::error "Language not found: $identifier"
        return 1
    fi
    
    # Remove configuration file
    rm -f "${JUDGE0_CUSTOM_LANGS_DIR}/${lang_id}.json"
    
    # Update registry
    registry=$(echo "$registry" | jq --arg id "$lang_id" '.languages |= map(select(.id != ($id | tonumber)))')
    echo "$registry" > "$registry_file"
    
    log::success "Custom language removed: $identifier"
    return 0
}

#######################################
# Enable/disable custom language
# Arguments:
#   $1 - Language ID or name
#   $2 - Action (enable/disable)
# Outputs:
#   Update status
#######################################
judge0::custom_langs::toggle() {
    local identifier="$1"
    local action="$2"
    
    if [[ -z "$identifier" ]] || [[ -z "$action" ]]; then
        log::error "Language identifier and action (enable/disable) are required"
        return 1
    fi
    
    judge0::custom_langs::init
    
    local registry_file="${JUDGE0_CUSTOM_LANGS_DIR}/registry.json"
    local registry=$(cat "$registry_file")
    
    # Find language
    local lang_id
    if [[ "$identifier" =~ ^[0-9]+$ ]]; then
        lang_id="$identifier"
    else
        lang_id=$(echo "$registry" | jq -r --arg name "$identifier" '.languages[] | select(.name == $name) | .id')
    fi
    
    if [[ -z "$lang_id" ]]; then
        log::error "Language not found: $identifier"
        return 1
    fi
    
    # Update enabled status
    local enabled="true"
    if [[ "$action" == "disable" ]]; then
        enabled="false"
    fi
    
    # Update configuration file
    local config_file="${JUDGE0_CUSTOM_LANGS_DIR}/${lang_id}.json"
    if [[ -f "$config_file" ]]; then
        local config=$(cat "$config_file")
        config=$(echo "$config" | jq --arg enabled "$enabled" '.enabled = ($enabled == "true")')
        echo "$config" > "$config_file"
    fi
    
    # Update registry
    registry=$(echo "$registry" | jq \
        --arg id "$lang_id" \
        --arg enabled "$enabled" \
        '.languages |= map(if .id == ($id | tonumber) then .enabled = ($enabled == "true") else . end)')
    echo "$registry" > "$registry_file"
    
    log::success "Language $action""d: $identifier"
    return 0
}

#######################################
# Test custom language configuration
# Arguments:
#   $1 - Language ID or name
#   $2 - Test code (optional)
# Outputs:
#   Test result
#######################################
judge0::custom_langs::test() {
    local identifier="$1"
    local test_code="${2:-}"
    
    if [[ -z "$identifier" ]]; then
        log::error "Language ID or name is required"
        return 1
    fi
    
    judge0::custom_langs::init
    
    local registry_file="${JUDGE0_CUSTOM_LANGS_DIR}/registry.json"
    local registry=$(cat "$registry_file")
    
    # Find language
    local lang_config
    if [[ "$identifier" =~ ^[0-9]+$ ]]; then
        lang_config=$(echo "$registry" | jq --arg id "$identifier" '.languages[] | select(.id == ($id | tonumber))')
    else
        lang_config=$(echo "$registry" | jq --arg name "$identifier" '.languages[] | select(.name == $name)')
    fi
    
    if [[ -z "$lang_config" ]]; then
        log::error "Language not found: $identifier"
        return 1
    fi
    
    local name=$(echo "$lang_config" | jq -r '.name')
    local extension=$(echo "$lang_config" | jq -r '.extension')
    local run_command=$(echo "$lang_config" | jq -r '.run_command')
    local compile_command=$(echo "$lang_config" | jq -r '.compile_command // ""')
    
    log::header "Testing Custom Language: $name"
    
    # Use default test code if not provided
    if [[ -z "$test_code" ]]; then
        case "$extension" in
            .sh|.bash)
                test_code='echo "Hello from custom language!"'
                ;;
            .pl)
                test_code='print "Hello from custom language!\n";'
                ;;
            .rb)
                test_code='puts "Hello from custom language!"'
                ;;
            .lua)
                test_code='print("Hello from custom language!")'
                ;;
            *)
                test_code='Hello from custom language!'
                ;;
        esac
    fi
    
    # Create temporary test file
    local test_file="/tmp/judge0_test_${RANDOM}${extension}"
    echo "$test_code" > "$test_file"
    
    # Run compile command if needed
    if [[ -n "$compile_command" ]]; then
        log::info "Compiling..."
        local compile_result=$(eval "${compile_command/\{file\}/$test_file}" 2>&1)
        if [[ $? -ne 0 ]]; then
            log::error "Compilation failed: $compile_result"
            rm -f "$test_file"
            return 1
        fi
    fi
    
    # Run the code
    log::info "Running..."
    local run_result=$(eval "${run_command/\{file\}/$test_file}" 2>&1)
    local exit_code=$?
    
    # Clean up
    rm -f "$test_file" "${test_file%.${extension}}"
    
    if [[ $exit_code -eq 0 ]]; then
        log::success "Test successful!"
        echo "Output: $run_result"
    else
        log::error "Test failed (exit code: $exit_code)"
        echo "Error: $run_result"
        return 1
    fi
    
    return 0
}

#######################################
# Import custom languages from file
# Arguments:
#   $1 - Import file path (JSON)
# Outputs:
#   Import status
#######################################
judge0::custom_langs::import() {
    local import_file="$1"
    
    if [[ ! -f "$import_file" ]]; then
        log::error "Import file not found: $import_file"
        return 1
    fi
    
    judge0::custom_langs::init
    
    log::header "Importing Custom Languages"
    
    local imported=0
    local skipped=0
    
    while IFS= read -r lang; do
        local name=$(echo "$lang" | jq -r '.name')
        local run_command=$(echo "$lang" | jq -r '.run_command')
        local extension=$(echo "$lang" | jq -r '.extension')
        local compile_command=$(echo "$lang" | jq -r '.compile_command // ""')
        
        log::info "Importing: $name"
        
        if judge0::custom_langs::add "$name" "$run_command" "$extension" "$compile_command" >/dev/null 2>&1; then
            ((imported++))
        else
            log::warning "Skipped: $name (already exists or invalid)"
            ((skipped++))
        fi
    done < <(jq -c '.languages[]' "$import_file" 2>/dev/null || jq -c '.[]' "$import_file")
    
    log::success "Import complete: $imported added, $skipped skipped"
    return 0
}

#######################################
# Export custom languages to file
# Arguments:
#   $1 - Export file path (optional)
# Outputs:
#   Export file path
#######################################
judge0::custom_langs::export() {
    local export_file="${1:-${JUDGE0_DATA_DIR}/custom_languages_$(date +%Y%m%d_%H%M%S).json}"
    
    judge0::custom_langs::init
    
    local registry_file="${JUDGE0_CUSTOM_LANGS_DIR}/registry.json"
    local registry=$(cat "$registry_file")
    
    # Export languages array
    echo "$registry" | jq '.languages' > "$export_file"
    
    log::success "Custom languages exported to: $export_file"
    echo "$export_file"
    return 0
}

#######################################
# Add common custom languages presets
#######################################
judge0::custom_langs::add_presets() {
    log::header "Adding Custom Language Presets"
    
    # Shell script
    judge0::custom_langs::add "Shell Script" "bash {file}" ".sh" "" >/dev/null 2>&1 || true
    
    # Perl
    judge0::custom_langs::add "Perl" "perl {file}" ".pl" "" >/dev/null 2>&1 || true
    
    # Lua
    judge0::custom_langs::add "Lua" "lua {file}" ".lua" "" >/dev/null 2>&1 || true
    
    # R
    judge0::custom_langs::add "R" "Rscript {file}" ".r" "" >/dev/null 2>&1 || true
    
    # Julia
    judge0::custom_langs::add "Julia" "julia {file}" ".jl" "" >/dev/null 2>&1 || true
    
    # Dart
    judge0::custom_langs::add "Dart" "dart {file}" ".dart" "" >/dev/null 2>&1 || true
    
    # Kotlin Script
    judge0::custom_langs::add "Kotlin Script" "kotlinc -script {file}" ".kts" "" >/dev/null 2>&1 || true
    
    log::success "Presets added (existing languages skipped)"
    judge0::custom_langs::list
}