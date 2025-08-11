#!/usr/bin/env bash
set -euo pipefail

# Judge0 Data Injection Adapter
# This script handles injection of languages, submissions, and configurations into Judge0
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject languages, submissions, and configurations into Judge0 code execution service"

JUDGE0_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${JUDGE0_SCRIPT_DIR}/../../../lib/utils/var.sh"

# Source common utilities using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Source Judge0 configuration if available
if [[ -f "${JUDGE0_SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${JUDGE0_SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default Judge0 settings
readonly DEFAULT_JUDGE0_HOST="http://localhost:2358"
readonly DEFAULT_JUDGE0_DATA_DIR="${HOME}/.judge0"
readonly DEFAULT_JUDGE0_SUBMISSIONS_DIR="${DEFAULT_JUDGE0_DATA_DIR}/submissions"
readonly DEFAULT_JUDGE0_TEMPLATES_DIR="${DEFAULT_JUDGE0_DATA_DIR}/templates"

# Judge0 settings (can be overridden by environment)
JUDGE0_HOST="${JUDGE0_HOST:-$DEFAULT_JUDGE0_HOST}"
JUDGE0_DATA_DIR="${JUDGE0_DATA_DIR:-$DEFAULT_JUDGE0_DATA_DIR}"
JUDGE0_SUBMISSIONS_DIR="${JUDGE0_SUBMISSIONS_DIR:-$DEFAULT_JUDGE0_SUBMISSIONS_DIR}"
JUDGE0_TEMPLATES_DIR="${JUDGE0_TEMPLATES_DIR:-$DEFAULT_JUDGE0_TEMPLATES_DIR}"

# Operation tracking
declare -a JUDGE0_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
judge0_inject::usage() {
    cat << EOF
Judge0 Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects code templates, test submissions, and configurations into Judge0 based on 
    scenario configuration. Supports validation, injection, status checks, 
    and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the data injection
    --status      Check status of injected data
    --rollback    Rollback injected data
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "languages": [
        {
          "id": 71,
          "name": "python",
          "version": "3.10",
          "enable": true
        }
      ],
      "templates": [
        {
          "name": "python_starter",
          "language_id": 71,
          "file": "path/to/template.py",
          "description": "Python starter template"
        }
      ],
      "test_submissions": [
        {
          "name": "hello_world",
          "language_id": 71,
          "source_code_file": "path/to/code.py",
          "stdin": "test input",
          "expected_output": "Hello, World!"
        }
      ],
      "configurations": [
        {
          "key": "cpu_time_limit",
          "value": 2.0
        },
        {
          "key": "memory_limit",
          "value": 128000
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"languages": [{"id": 71, "name": "python", "enable": true}]}'
    
    # Inject templates and test submissions
    $0 --inject '{"templates": [{"name": "test", "language_id": 71, "file": "test.py"}]}'
    
    # Configure execution limits
    $0 --inject '{"configurations": [{"key": "cpu_time_limit", "value": 5.0}]}'

EOF
}

#######################################
# Check if Judge0 is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
judge0_inject::check_accessibility() {
    # Check if Judge0 is running
    if curl -s --max-time 5 "${JUDGE0_HOST}/about" 2>/dev/null | grep -q "Judge0"; then
        log::debug "Judge0 is accessible at $JUDGE0_HOST"
        return 0
    else
        log::error "Judge0 is not accessible at $JUDGE0_HOST"
        log::info "Ensure Judge0 is running: ./scripts/resources/execution/judge0/manage.sh --action start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
judge0_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    JUDGE0_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Judge0 rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
judge0_inject::execute_rollback() {
    if [[ ${#JUDGE0_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Judge0 rollback actions to execute"
        return 0
    fi
    
    log::info "Executing Judge0 rollback actions..."
    
    local success_count=0
    local total_count=${#JUDGE0_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#JUDGE0_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${JUDGE0_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "Judge0 rollback completed: $success_count/$total_count actions successful"
    JUDGE0_ROLLBACK_ACTIONS=()
}

#######################################
# Validate language configuration
# Arguments:
#   $1 - languages configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
judge0_inject::validate_languages() {
    local languages_config="$1"
    
    log::debug "Validating language configurations..."
    
    # Check if languages is an array
    local languages_type
    languages_type=$(echo "$languages_config" | jq -r 'type')
    
    if [[ "$languages_type" != "array" ]]; then
        log::error "Languages configuration must be an array, got: $languages_type"
        return 1
    fi
    
    # Validate each language
    local language_count
    language_count=$(echo "$languages_config" | jq 'length')
    
    for ((i=0; i<language_count; i++)); do
        local language
        language=$(echo "$languages_config" | jq -c ".[$i]")
        
        # Check required fields
        local id name
        id=$(echo "$language" | jq -r '.id // empty')
        name=$(echo "$language" | jq -r '.name // empty')
        
        if [[ -z "$id" ]]; then
            log::error "Language at index $i missing required 'id' field"
            return 1
        fi
        
        if [[ -z "$name" ]]; then
            log::error "Language at index $i missing required 'name' field"
            return 1
        fi
        
        # Validate language ID is numeric
        if ! [[ "$id" =~ ^[0-9]+$ ]]; then
            log::error "Language ID must be numeric, got: $id"
            return 1
        fi
        
        log::debug "Language '$name' (ID: $id) configuration is valid"
    done
    
    log::success "All language configurations are valid"
    return 0
}

#######################################
# Validate template configuration
# Arguments:
#   $1 - templates configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
judge0_inject::validate_templates() {
    local templates_config="$1"
    
    log::debug "Validating template configurations..."
    
    # Check if templates is an array
    local templates_type
    templates_type=$(echo "$templates_config" | jq -r 'type')
    
    if [[ "$templates_type" != "array" ]]; then
        log::error "Templates configuration must be an array, got: $templates_type"
        return 1
    fi
    
    # Validate each template
    local template_count
    template_count=$(echo "$templates_config" | jq 'length')
    
    for ((i=0; i<template_count; i++)); do
        local template
        template=$(echo "$templates_config" | jq -c ".[$i]")
        
        # Check required fields
        local name language_id file
        name=$(echo "$template" | jq -r '.name // empty')
        language_id=$(echo "$template" | jq -r '.language_id // empty')
        file=$(echo "$template" | jq -r '.file // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Template at index $i missing required 'name' field"
            return 1
        fi
        
        if [[ -z "$language_id" ]]; then
            log::error "Template '$name' missing required 'language_id' field"
            return 1
        fi
        
        if [[ -z "$file" ]]; then
            log::error "Template '$name' missing required 'file' field"
            return 1
        fi
        
        # Check if file exists
        local template_path="$VROOLI_PROJECT_ROOT/$file"
        if [[ ! -f "$template_path" ]]; then
            log::error "Template file not found: $template_path"
            return 1
        fi
        
        log::debug "Template '$name' configuration is valid"
    done
    
    log::success "All template configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
judge0_inject::validate_config() {
    local config="$1"
    
    log::info "Validating Judge0 injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Judge0 injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_languages has_templates has_submissions has_configurations
    has_languages=$(echo "$config" | jq -e '.languages' >/dev/null 2>&1 && echo "true" || echo "false")
    has_templates=$(echo "$config" | jq -e '.templates' >/dev/null 2>&1 && echo "true" || echo "false")
    has_submissions=$(echo "$config" | jq -e '.test_submissions' >/dev/null 2>&1 && echo "true" || echo "false")
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_languages" == "false" && "$has_templates" == "false" && "$has_submissions" == "false" && "$has_configurations" == "false" ]]; then
        log::error "Judge0 injection configuration must have 'languages', 'templates', 'test_submissions', or 'configurations'"
        return 1
    fi
    
    # Validate languages if present
    if [[ "$has_languages" == "true" ]]; then
        local languages
        languages=$(echo "$config" | jq -c '.languages')
        
        if ! judge0_inject::validate_languages "$languages"; then
            return 1
        fi
    fi
    
    # Validate templates if present
    if [[ "$has_templates" == "true" ]]; then
        local templates
        templates=$(echo "$config" | jq -c '.templates')
        
        if ! judge0_inject::validate_templates "$templates"; then
            return 1
        fi
    fi
    
    # Validate test submissions if present
    if [[ "$has_submissions" == "true" ]]; then
        local submissions
        submissions=$(echo "$config" | jq -c '.test_submissions')
        
        local submission_count
        submission_count=$(echo "$submissions" | jq 'length')
        
        for ((i=0; i<submission_count; i++)); do
            local submission
            submission=$(echo "$submissions" | jq -c ".[$i]")
            
            local name language_id source_code_file
            name=$(echo "$submission" | jq -r '.name // empty')
            language_id=$(echo "$submission" | jq -r '.language_id // empty')
            source_code_file=$(echo "$submission" | jq -r '.source_code_file // empty')
            
            if [[ -z "$name" ]]; then
                log::error "Test submission at index $i missing required 'name' field"
                return 1
            fi
            
            if [[ -z "$language_id" ]]; then
                log::error "Test submission '$name' missing required 'language_id' field"
                return 1
            fi
            
            if [[ -z "$source_code_file" ]]; then
                log::error "Test submission '$name' missing required 'source_code_file' field"
                return 1
            fi
            
            # Check if file exists
            local code_path="$VROOLI_PROJECT_ROOT/$source_code_file"
            if [[ ! -f "$code_path" ]]; then
                log::error "Source code file not found: $code_path"
                return 1
            fi
        done
    fi
    
    log::success "Judge0 injection configuration is valid"
    return 0
}

#######################################
# Configure language
# Arguments:
#   $1 - language configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
judge0_inject::configure_language() {
    local language_config="$1"
    
    local id name version enable
    id=$(echo "$language_config" | jq -r '.id')
    name=$(echo "$language_config" | jq -r '.name')
    version=$(echo "$language_config" | jq -r '.version // ""')
    enable=$(echo "$language_config" | jq -r '.enable // true')
    
    log::info "Configuring language: $name (ID: $id)"
    
    # Create language configs directory
    local langs_dir="${JUDGE0_DATA_DIR}/languages"
    mkdir -p "$langs_dir"
    
    # Save language configuration
    local lang_file="${langs_dir}/${id}.json"
    cat > "$lang_file" << EOF
{
  "id": $id,
  "name": "$name",
  "version": "$version",
  "enabled": $enable,
  "configured": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    log::success "Configured language: $name"
    
    # Add rollback action
    judge0_inject::add_rollback_action \
        "Remove language configuration: $name" \
        "trash::safe_remove '${lang_file}' --temp"
    
    return 0
}

#######################################
# Install template
# Arguments:
#   $1 - template configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
judge0_inject::install_template() {
    local template_config="$1"
    
    local name language_id file description
    name=$(echo "$template_config" | jq -r '.name')
    language_id=$(echo "$template_config" | jq -r '.language_id')
    file=$(echo "$template_config" | jq -r '.file')
    description=$(echo "$template_config" | jq -r '.description // ""')
    
    log::info "Installing template: $name"
    
    # Resolve file path
    local template_path="$VROOLI_PROJECT_ROOT/$file"
    
    # Create templates directory with language subdirectory
    local lang_dir="${JUDGE0_TEMPLATES_DIR}/lang_${language_id}"
    mkdir -p "$lang_dir"
    
    # Determine file extension
    local extension="${file##*.}"
    local dest_file="${lang_dir}/${name}.${extension}"
    
    # Copy template file
    cp "$template_path" "$dest_file"
    
    # Create metadata file
    local metadata_file="${lang_dir}/${name}.meta.json"
    cat > "$metadata_file" << EOF
{
  "name": "$name",
  "language_id": $language_id,
  "description": "$description",
  "extension": "$extension",
  "installed": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    log::success "Installed template: $name"
    
    # Add rollback actions
    judge0_inject::add_rollback_action \
        "Remove template: $name" \
        "trash::safe_remove '${dest_file}' --temp; trash::safe_remove '${metadata_file}' --temp"
    
    return 0
}

#######################################
# Create test submission
# Arguments:
#   $1 - submission configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
judge0_inject::create_submission() {
    local submission_config="$1"
    
    local name language_id source_code_file stdin expected_output
    name=$(echo "$submission_config" | jq -r '.name')
    language_id=$(echo "$submission_config" | jq -r '.language_id')
    source_code_file=$(echo "$submission_config" | jq -r '.source_code_file')
    stdin=$(echo "$submission_config" | jq -r '.stdin // ""')
    expected_output=$(echo "$submission_config" | jq -r '.expected_output // ""')
    
    log::info "Creating test submission: $name"
    
    # Resolve file path
    local code_path="$VROOLI_PROJECT_ROOT/$source_code_file"
    
    # Read source code
    local source_code
    source_code=$(cat "$code_path")
    
    # Create submissions directory
    mkdir -p "$JUDGE0_SUBMISSIONS_DIR"
    
    # Create submission file
    local submission_file="${JUDGE0_SUBMISSIONS_DIR}/${name}.json"
    cat > "$submission_file" << EOF
{
  "name": "$name",
  "language_id": $language_id,
  "source_code": $(echo "$source_code" | jq -Rs .),
  "stdin": $(echo "$stdin" | jq -Rs .),
  "expected_output": $(echo "$expected_output" | jq -Rs .),
  "created": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
    
    # If Judge0 is running, submit the test
    if judge0_inject::check_accessibility; then
        log::info "Submitting test to Judge0..."
        
        local submission_response
        submission_response=$(curl -s -X POST "${JUDGE0_HOST}/submissions?wait=true" \
            -H "Content-Type: application/json" \
            -d "{
                \"language_id\": $language_id,
                \"source_code\": $(echo "$source_code" | jq -Rs .),
                \"stdin\": $(echo "$stdin" | jq -Rs .),
                \"expected_output\": $(echo "$expected_output" | jq -Rs .)
            }" 2>/dev/null || echo "{}")
        
        # Save submission token if received
        local token
        token=$(echo "$submission_response" | jq -r '.token // empty')
        
        if [[ -n "$token" ]]; then
            echo "$token" > "${submission_file}.token"
            log::success "Test submission created with token: $token"
        fi
    fi
    
    log::success "Created test submission: $name"
    
    # Add rollback action
    judge0_inject::add_rollback_action \
        "Remove test submission: $name" \
        "trash::safe_remove '${submission_file}' --temp; trash::safe_remove '${submission_file}.token' --temp"
    
    return 0
}

#######################################
# Inject languages
# Arguments:
#   $1 - languages configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
judge0_inject::inject_languages() {
    local languages_config="$1"
    
    log::info "Injecting Judge0 languages..."
    
    local language_count
    language_count=$(echo "$languages_config" | jq 'length')
    
    if [[ "$language_count" -eq 0 ]]; then
        log::info "No languages to inject"
        return 0
    fi
    
    local failed_languages=()
    
    for ((i=0; i<language_count; i++)); do
        local language
        language=$(echo "$languages_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$language" | jq -r '.name')
        
        if ! judge0_inject::configure_language "$language"; then
            failed_languages+=("$name")
        fi
    done
    
    if [[ ${#failed_languages[@]} -eq 0 ]]; then
        log::success "All languages injected successfully"
        return 0
    else
        log::error "Failed to inject languages: ${failed_languages[*]}"
        return 1
    fi
}

#######################################
# Inject templates
# Arguments:
#   $1 - templates configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
judge0_inject::inject_templates() {
    local templates_config="$1"
    
    log::info "Injecting Judge0 templates..."
    
    local template_count
    template_count=$(echo "$templates_config" | jq 'length')
    
    if [[ "$template_count" -eq 0 ]]; then
        log::info "No templates to inject"
        return 0
    fi
    
    local failed_templates=()
    
    for ((i=0; i<template_count; i++)); do
        local template
        template=$(echo "$templates_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$template" | jq -r '.name')
        
        if ! judge0_inject::install_template "$template"; then
            failed_templates+=("$name")
        fi
    done
    
    if [[ ${#failed_templates[@]} -eq 0 ]]; then
        log::success "All templates injected successfully"
        return 0
    else
        log::error "Failed to inject templates: ${failed_templates[*]}"
        return 1
    fi
}

#######################################
# Inject test submissions
# Arguments:
#   $1 - submissions configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
judge0_inject::inject_submissions() {
    local submissions_config="$1"
    
    log::info "Injecting Judge0 test submissions..."
    
    local submission_count
    submission_count=$(echo "$submissions_config" | jq 'length')
    
    if [[ "$submission_count" -eq 0 ]]; then
        log::info "No test submissions to inject"
        return 0
    fi
    
    local failed_submissions=()
    
    for ((i=0; i<submission_count; i++)); do
        local submission
        submission=$(echo "$submissions_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$submission" | jq -r '.name')
        
        if ! judge0_inject::create_submission "$submission"; then
            failed_submissions+=("$name")
        fi
    done
    
    if [[ ${#failed_submissions[@]} -eq 0 ]]; then
        log::success "All test submissions injected successfully"
        return 0
    else
        log::error "Failed to inject test submissions: ${failed_submissions[*]}"
        return 1
    fi
}

#######################################
# Apply configurations
# Arguments:
#   $1 - configurations JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
judge0_inject::apply_configurations() {
    local configurations="$1"
    
    log::info "Applying Judge0 configurations..."
    
    local config_count
    config_count=$(echo "$configurations" | jq 'length')
    
    if [[ "$config_count" -eq 0 ]]; then
        log::info "No configurations to apply"
        return 0
    fi
    
    # Save configuration settings
    local config_file="${JUDGE0_DATA_DIR}/config.json"
    echo "$configurations" > "$config_file"
    
    log::success "Saved configuration settings"
    log::warn "Note: Some configurations require Judge0 restart to take effect"
    
    # Add rollback action
    judge0_inject::add_rollback_action \
        "Remove configuration file" \
        "trash::safe_remove '${config_file}' --temp"
    
    return 0
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
judge0_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into Judge0"
    
    # Check Judge0 accessibility (optional for file-based operations)
    judge0_inject::check_accessibility || true
    
    # Clear previous rollback actions
    JUDGE0_ROLLBACK_ACTIONS=()
    
    # Inject languages if present
    local has_languages
    has_languages=$(echo "$config" | jq -e '.languages' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_languages" == "true" ]]; then
        local languages
        languages=$(echo "$config" | jq -c '.languages')
        
        if ! judge0_inject::inject_languages "$languages"; then
            log::error "Failed to inject languages"
            judge0_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject templates if present
    local has_templates
    has_templates=$(echo "$config" | jq -e '.templates' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_templates" == "true" ]]; then
        local templates
        templates=$(echo "$config" | jq -c '.templates')
        
        if ! judge0_inject::inject_templates "$templates"; then
            log::error "Failed to inject templates"
            judge0_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject test submissions if present
    local has_submissions
    has_submissions=$(echo "$config" | jq -e '.test_submissions' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_submissions" == "true" ]]; then
        local submissions
        submissions=$(echo "$config" | jq -c '.test_submissions')
        
        if ! judge0_inject::inject_submissions "$submissions"; then
            log::error "Failed to inject test submissions"
            judge0_inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply configurations if present
    local has_configurations
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_configurations" == "true" ]]; then
        local configurations
        configurations=$(echo "$config" | jq -c '.configurations')
        
        if ! judge0_inject::apply_configurations "$configurations"; then
            log::error "Failed to apply configurations"
            judge0_inject::execute_rollback
            return 1
        fi
    fi
    
    log::success "✅ Judge0 data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
judge0_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking Judge0 injection status"
    
    # Check Judge0 accessibility
    local is_running=false
    if judge0_inject::check_accessibility; then
        is_running=true
        log::success "✅ Judge0 is running"
        
        # Get Judge0 info
        local about_info
        about_info=$(curl -s "${JUDGE0_HOST}/about" 2>/dev/null || echo "{}")
        
        local version
        version=$(echo "$about_info" | jq -r '.version // "unknown"')
        log::info "  - Version: $version"
    else
        log::warn "⚠️  Judge0 is not running"
    fi
    
    # Check language configurations
    local langs_dir="${JUDGE0_DATA_DIR}/languages"
    if [[ -d "$langs_dir" ]]; then
        local lang_count
        lang_count=$(find "$langs_dir" -name "*.json" 2>/dev/null | wc -l)
        
        if [[ "$lang_count" -gt 0 ]]; then
            log::info "Found $lang_count language configurations"
        else
            log::info "No language configurations found"
        fi
    else
        log::info "Language configurations directory does not exist"
    fi
    
    # Check templates
    if [[ -d "$JUDGE0_TEMPLATES_DIR" ]]; then
        local template_count
        template_count=$(find "$JUDGE0_TEMPLATES_DIR" -type f ! -name "*.meta.json" 2>/dev/null | wc -l)
        
        if [[ "$template_count" -gt 0 ]]; then
            log::info "Found $template_count templates"
            
            # Count templates by language
            local lang_dirs
            lang_dirs=$(find "$JUDGE0_TEMPLATES_DIR" -maxdepth 1 -type d -name "lang_*" 2>/dev/null | wc -l)
            
            if [[ "$lang_dirs" -gt 0 ]]; then
                log::info "  - Templates for $lang_dirs languages"
            fi
        else
            log::info "No templates found"
        fi
    else
        log::info "Templates directory does not exist"
    fi
    
    # Check test submissions
    if [[ -d "$JUDGE0_SUBMISSIONS_DIR" ]]; then
        local submission_count
        submission_count=$(find "$JUDGE0_SUBMISSIONS_DIR" -name "*.json" ! -name "*.token" 2>/dev/null | wc -l)
        
        if [[ "$submission_count" -gt 0 ]]; then
            log::info "Found $submission_count test submissions"
            
            # Count submissions with tokens
            local token_count
            token_count=$(find "$JUDGE0_SUBMISSIONS_DIR" -name "*.token" 2>/dev/null | wc -l)
            
            if [[ "$token_count" -gt 0 ]]; then
                log::info "  - $token_count submissions have been executed"
            fi
        else
            log::info "No test submissions found"
        fi
    else
        log::info "Submissions directory does not exist"
    fi
    
    # Test API if running
    if [[ "$is_running" == true ]]; then
        log::info "Testing Judge0 API..."
        
        # Get languages
        local languages
        languages=$(curl -s "${JUDGE0_HOST}/languages" 2>/dev/null || echo "[]")
        
        local lang_count
        lang_count=$(echo "$languages" | jq 'length // 0')
        
        if [[ "$lang_count" -gt 0 ]]; then
            log::success "✅ Judge0 API is responding"
            log::info "  - Available languages: $lang_count"
        else
            log::error "❌ Judge0 API not responding properly"
        fi
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
judge0_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        judge0_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            judge0_inject::validate_config "$config"
            ;;
        "--inject")
            judge0_inject::inject_data "$config"
            ;;
        "--status")
            judge0_inject::check_status "$config"
            ;;
        "--rollback")
            judge0_inject::execute_rollback
            ;;
        "--help")
            judge0_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            judge0_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        judge0_inject::usage
        exit 1
    fi
    
    judge0_inject::main "$@"
fi