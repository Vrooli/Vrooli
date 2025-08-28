#!/usr/bin/env bash
set -euo pipefail

# Unstructured.io Data Injection Adapter
# This script handles injection of parsers and configurations into Unstructured.io
# Part of the Vrooli resource data injection system

export DESCRIPTION="Inject parsers and configurations into Unstructured.io document processing service"

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/unstructured-io"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

# Source Unstructured.io configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh"
fi

# Default Unstructured.io settings
readonly DEFAULT_UNSTRUCTURED_HOST="http://localhost:8005"
readonly DEFAULT_UNSTRUCTURED_DATA_DIR="${HOME}/.unstructured"
readonly DEFAULT_UNSTRUCTURED_MODELS_DIR="${DEFAULT_UNSTRUCTURED_DATA_DIR}/models"

# Unstructured.io settings (can be overridden by environment)
UNSTRUCTURED_HOST="${UNSTRUCTURED_HOST:-$DEFAULT_UNSTRUCTURED_HOST}"
UNSTRUCTURED_DATA_DIR="${UNSTRUCTURED_DATA_DIR:-$DEFAULT_UNSTRUCTURED_DATA_DIR}"
UNSTRUCTURED_MODELS_DIR="${UNSTRUCTURED_MODELS_DIR:-$DEFAULT_UNSTRUCTURED_MODELS_DIR}"

# Operation tracking
declare -a UNSTRUCTURED_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
unstructured_inject::usage() {
    cat << EOF
Unstructured.io Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects parsers, models, and configurations into Unstructured.io based on 
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
      "parsers": [
        {
          "type": "pdf",
          "enable": true,
          "options": {
            "ocr": true,
            "strategy": "hi_res"
          }
        },
        {
          "type": "docx",
          "enable": true
        }
      ],
      "models": [
        {
          "name": "detectron2",
          "download": true
        }
      ],
      "test_documents": [
        {
          "file": "path/to/document.pdf",
          "name": "test_doc"
        }
      ],
      "configurations": [
        {
          "key": "max_file_size",
          "value": "50MB"
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"parsers": [{"type": "pdf", "enable": true}]}'
    
    # Enable parsers and download models
    $0 --inject '{"parsers": [{"type": "pdf", "enable": true}], "models": [{"name": "detectron2", "download": true}]}'
    
    # Upload test documents
    $0 --inject '{"test_documents": [{"file": "samples/test.pdf", "name": "test"}]}'

EOF
}

#######################################
# Check if Unstructured.io is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
unstructured_inject::check_accessibility() {
    # Check if Unstructured.io is running
    if curl -s --max-time 5 "${UNSTRUCTURED_HOST}/healthcheck" 2>/dev/null | grep -q "ok"; then
        log::debug "Unstructured.io is accessible at $UNSTRUCTURED_HOST"
        return 0
    else
        log::error "Unstructured.io is not accessible at $UNSTRUCTURED_HOST"
        log::info "Ensure Unstructured.io is running: resource-unstructured-io manage start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
unstructured_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    UNSTRUCTURED_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Unstructured.io rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
unstructured_inject::execute_rollback() {
    if [[ ${#UNSTRUCTURED_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Unstructured.io rollback actions to execute"
        return 0
    fi
    
    log::info "Executing Unstructured.io rollback actions..."
    
    local success_count=0
    local total_count=${#UNSTRUCTURED_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#UNSTRUCTURED_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${UNSTRUCTURED_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "Unstructured.io rollback completed: $success_count/$total_count actions successful"
    UNSTRUCTURED_ROLLBACK_ACTIONS=()
}

#######################################
# Validate parser configuration
# Arguments:
#   $1 - parsers configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
unstructured_inject::validate_parsers() {
    local parsers_config="$1"
    
    log::debug "Validating parser configurations..."
    
    # Check if parsers is an array
    local parsers_type
    parsers_type=$(echo "$parsers_config" | jq -r 'type')
    
    if [[ "$parsers_type" != "array" ]]; then
        log::error "Parsers configuration must be an array, got: $parsers_type"
        return 1
    fi
    
    # Validate each parser
    local parser_count
    parser_count=$(echo "$parsers_config" | jq 'length')
    
    local valid_parsers=("pdf" "docx" "doc" "pptx" "ppt" "xlsx" "xls" "csv" "txt" "html" "xml" "json" "md" "rst" "rtf" "odt" "epub")
    
    for ((i=0; i<parser_count; i++)); do
        local parser
        parser=$(echo "$parsers_config" | jq -c ".[$i]")
        
        # Check required fields
        local type enable
        type=$(echo "$parser" | jq -r '.type // empty')
        enable=$(echo "$parser" | jq -r '.enable // false')
        
        if [[ -z "$type" ]]; then
            log::error "Parser at index $i missing required 'type' field"
            return 1
        fi
        
        # Validate parser type
        local is_valid=false
        for valid_type in "${valid_parsers[@]}"; do
            if [[ "$type" == "$valid_type" ]]; then
                is_valid=true
                break
            fi
        done
        
        if [[ "$is_valid" == false ]]; then
            log::error "Invalid parser type: $type. Valid types: ${valid_parsers[*]}"
            return 1
        fi
        
        log::debug "Parser '$type' configuration is valid"
    done
    
    log::success "All parser configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
unstructured_inject::validate_config() {
    local config="$1"
    
    log::info "Validating Unstructured.io injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Unstructured.io injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_parsers has_models has_documents has_configurations
    has_parsers=$(echo "$config" | jq -e '.parsers' >/dev/null 2>&1 && echo "true" || echo "false")
    has_models=$(echo "$config" | jq -e '.models' >/dev/null 2>&1 && echo "true" || echo "false")
    has_documents=$(echo "$config" | jq -e '.test_documents' >/dev/null 2>&1 && echo "true" || echo "false")
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_parsers" == "false" && "$has_models" == "false" && "$has_documents" == "false" && "$has_configurations" == "false" ]]; then
        log::error "Unstructured.io injection configuration must have 'parsers', 'models', 'test_documents', or 'configurations'"
        return 1
    fi
    
    # Validate parsers if present
    if [[ "$has_parsers" == "true" ]]; then
        local parsers
        parsers=$(echo "$config" | jq -c '.parsers')
        
        if ! unstructured_inject::validate_parsers "$parsers"; then
            return 1
        fi
    fi
    
    # Validate test documents if present
    if [[ "$has_documents" == "true" ]]; then
        local documents
        documents=$(echo "$config" | jq -c '.test_documents')
        
        local doc_count
        doc_count=$(echo "$documents" | jq 'length')
        
        for ((i=0; i<doc_count; i++)); do
            local doc
            doc=$(echo "$documents" | jq -c ".[$i]")
            
            local file
            file=$(echo "$doc" | jq -r '.file // empty')
            
            if [[ -z "$file" ]]; then
                log::error "Test document at index $i missing required 'file' field"
                return 1
            fi
            
            # Check if file exists
            local doc_path="$var_ROOT_DIR/$file"
            if [[ ! -f "$doc_path" ]]; then
                log::error "Document file not found: $doc_path"
                return 1
            fi
        done
    fi
    
    log::success "Unstructured.io injection configuration is valid"
    return 0
}

#######################################
# Enable parser
# Arguments:
#   $1 - parser configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
unstructured_inject::enable_parser() {
    local parser_config="$1"
    
    local type enable options
    type=$(echo "$parser_config" | jq -r '.type')
    enable=$(echo "$parser_config" | jq -r '.enable // true')
    # shellcheck disable=SC2034
    options=$(echo "$parser_config" | jq -r '.options // {}')
    
    if [[ "$enable" != "true" ]]; then
        log::info "Skipping disabled parser: $type"
        return 0
    fi
    
    log::info "Enabling parser: $type"
    
    # Create parser configuration directory if needed
    local parser_config_dir="${UNSTRUCTURED_DATA_DIR}/parsers"
    mkdir -p "$parser_config_dir"
    
    # Save parser configuration
    echo "$parser_config" > "${parser_config_dir}/${type}.json"
    
    log::success "Enabled parser: $type"
    
    # Add rollback action
    unstructured_inject::add_rollback_action \
        "Remove parser configuration: $type" \
        "trash::safe_remove '${parser_config_dir}/${type}.json' --temp"
    
    return 0
}

#######################################
# Download model
# Arguments:
#   $1 - model configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
unstructured_inject::download_model() {
    local model_config="$1"
    
    local name download
    name=$(echo "$model_config" | jq -r '.name')
    download=$(echo "$model_config" | jq -r '.download // false')
    
    if [[ "$download" != "true" ]]; then
        log::info "Skipping model: $name (download not requested)"
        return 0
    fi
    
    log::info "Downloading model: $name"
    
    # Create models directory if it doesn't exist
    mkdir -p "$UNSTRUCTURED_MODELS_DIR"
    
    # Model downloading for Unstructured.io is typically automatic
    # This is a placeholder for specific model handling
    case "$name" in
        "detectron2")
            log::info "Detectron2 model will be downloaded on first use"
            touch "${UNSTRUCTURED_MODELS_DIR}/.detectron2_enabled"
            ;;
        "tessera")
            log::info "Tesseract OCR model will be downloaded on first use"
            touch "${UNSTRUCTURED_MODELS_DIR}/.tesseract_enabled"
            ;;
        *)
            log::warn "Unknown model: $name"
            ;;
    esac
    
    log::success "Model '$name' prepared for download"
    
    # Add rollback action
    unstructured_inject::add_rollback_action \
        "Remove model marker: $name" \
        "trash::safe_remove '${UNSTRUCTURED_MODELS_DIR}/.${name}_enabled' --temp"
    
    return 0
}

#######################################
# Upload test document
# Arguments:
#   $1 - document configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
unstructured_inject::upload_document() {
    local doc_config="$1"
    
    local file name
    file=$(echo "$doc_config" | jq -r '.file')
    name=$(echo "$doc_config" | jq -r '.name // empty')
    
    # Resolve file path
    local doc_path="$var_ROOT_DIR/$file"
    
    if [[ -z "$name" ]]; then
        name=$(basename "$file")
    fi
    
    log::info "Uploading test document: $name"
    
    # Create test documents directory if it doesn't exist
    local test_docs_dir="${UNSTRUCTURED_DATA_DIR}/test_documents"
    mkdir -p "$test_docs_dir"
    
    # Copy document to test directory
    cp "$doc_path" "$test_docs_dir/$name"
    
    log::success "Uploaded test document: $name"
    
    # Add rollback action
    unstructured_inject::add_rollback_action \
        "Remove test document: $name" \
        "trash::safe_remove '${test_docs_dir}/${name}' --temp"
    
    return 0
}

#######################################
# Inject parsers
# Arguments:
#   $1 - parsers configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
unstructured_inject::inject_parsers() {
    local parsers_config="$1"
    
    log::info "Injecting Unstructured.io parsers..."
    
    local parser_count
    parser_count=$(echo "$parsers_config" | jq 'length')
    
    if [[ "$parser_count" -eq 0 ]]; then
        log::info "No parsers to inject"
        return 0
    fi
    
    local failed_parsers=()
    
    for ((i=0; i<parser_count; i++)); do
        local parser
        parser=$(echo "$parsers_config" | jq -c ".[$i]")
        
        local type
        type=$(echo "$parser" | jq -r '.type')
        
        if ! unstructured_inject::enable_parser "$parser"; then
            failed_parsers+=("$type")
        fi
    done
    
    if [[ ${#failed_parsers[@]} -eq 0 ]]; then
        log::success "All parsers injected successfully"
        return 0
    else
        log::error "Failed to inject parsers: ${failed_parsers[*]}"
        return 1
    fi
}

#######################################
# Inject models
# Arguments:
#   $1 - models configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
unstructured_inject::inject_models() {
    local models_config="$1"
    
    log::info "Injecting Unstructured.io models..."
    
    local model_count
    model_count=$(echo "$models_config" | jq 'length')
    
    if [[ "$model_count" -eq 0 ]]; then
        log::info "No models to inject"
        return 0
    fi
    
    local failed_models=()
    
    for ((i=0; i<model_count; i++)); do
        local model
        model=$(echo "$models_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$model" | jq -r '.name')
        
        if ! unstructured_inject::download_model "$model"; then
            failed_models+=("$name")
        fi
    done
    
    if [[ ${#failed_models[@]} -eq 0 ]]; then
        log::success "All models injected successfully"
        return 0
    else
        log::error "Failed to inject models: ${failed_models[*]}"
        return 1
    fi
}

#######################################
# Inject test documents
# Arguments:
#   $1 - test documents configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
unstructured_inject::inject_documents() {
    local docs_config="$1"
    
    log::info "Injecting test documents..."
    
    local doc_count
    doc_count=$(echo "$docs_config" | jq 'length')
    
    if [[ "$doc_count" -eq 0 ]]; then
        log::info "No test documents to inject"
        return 0
    fi
    
    local failed_docs=()
    
    for ((i=0; i<doc_count; i++)); do
        local doc
        doc=$(echo "$docs_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$doc" | jq -r '.name // empty')
        
        if [[ -z "$name" ]]; then
            name=$(echo "$doc" | jq -r '.file' | xargs basename)
        fi
        
        if ! unstructured_inject::upload_document "$doc"; then
            failed_docs+=("$name")
        fi
    done
    
    if [[ ${#failed_docs[@]} -eq 0 ]]; then
        log::success "All test documents injected successfully"
        return 0
    else
        log::error "Failed to inject test documents: ${failed_docs[*]}"
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
unstructured_inject::apply_configurations() {
    local configurations="$1"
    
    log::info "Applying Unstructured.io configurations..."
    
    local config_count
    config_count=$(echo "$configurations" | jq 'length')
    
    if [[ "$config_count" -eq 0 ]]; then
        log::info "No configurations to apply"
        return 0
    fi
    
    # Save configuration settings
    local config_file="${UNSTRUCTURED_DATA_DIR}/config.json"
    echo "$configurations" > "$config_file"
    
    log::success "Saved configuration settings"
    
    # Add rollback action
    unstructured_inject::add_rollback_action \
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
unstructured_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into Unstructured.io"
    
    # Check Unstructured.io accessibility
    if ! unstructured_inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    UNSTRUCTURED_ROLLBACK_ACTIONS=()
    
    # Inject parsers if present
    local has_parsers
    has_parsers=$(echo "$config" | jq -e '.parsers' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_parsers" == "true" ]]; then
        local parsers
        parsers=$(echo "$config" | jq -c '.parsers')
        
        if ! unstructured_inject::inject_parsers "$parsers"; then
            log::error "Failed to inject parsers"
            unstructured_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject models if present
    local has_models
    has_models=$(echo "$config" | jq -e '.models' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_models" == "true" ]]; then
        local models
        models=$(echo "$config" | jq -c '.models')
        
        if ! unstructured_inject::inject_models "$models"; then
            log::error "Failed to inject models"
            unstructured_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject test documents if present
    local has_documents
    has_documents=$(echo "$config" | jq -e '.test_documents' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_documents" == "true" ]]; then
        local documents
        documents=$(echo "$config" | jq -c '.test_documents')
        
        if ! unstructured_inject::inject_documents "$documents"; then
            log::error "Failed to inject test documents"
            unstructured_inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply configurations if present
    local has_configurations
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_configurations" == "true" ]]; then
        local configurations
        configurations=$(echo "$config" | jq -c '.configurations')
        
        if ! unstructured_inject::apply_configurations "$configurations"; then
            log::error "Failed to apply configurations"
            unstructured_inject::execute_rollback
            return 1
        fi
    fi
    
    log::success "✅ Unstructured.io data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
unstructured_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking Unstructured.io injection status"
    
    # Check Unstructured.io accessibility
    if ! unstructured_inject::check_accessibility; then
        return 1
    fi
    
    # Check parser configurations
    local parser_config_dir="${UNSTRUCTURED_DATA_DIR}/parsers"
    
    if [[ -d "$parser_config_dir" ]]; then
        local parser_count
        parser_count=$(find "$parser_config_dir" -name "*.json" 2>/dev/null | wc -l)
        
        if [[ "$parser_count" -gt 0 ]]; then
            log::info "Found $parser_count parser configurations"
        else
            log::info "No parser configurations found"
        fi
    else
        log::info "Parser configuration directory does not exist"
    fi
    
    # Check models
    if [[ -d "$UNSTRUCTURED_MODELS_DIR" ]]; then
        local model_markers
        model_markers=$(find "$UNSTRUCTURED_MODELS_DIR" -name ".*_enabled" 2>/dev/null | wc -l)
        
        if [[ "$model_markers" -gt 0 ]]; then
            log::info "Found $model_markers model markers"
        else
            log::info "No model markers found"
        fi
    fi
    
    # Check test documents
    local test_docs_dir="${UNSTRUCTURED_DATA_DIR}/test_documents"
    
    if [[ -d "$test_docs_dir" ]]; then
        local doc_count
        doc_count=$(find "$test_docs_dir" -type f 2>/dev/null | wc -l)
        
        if [[ "$doc_count" -gt 0 ]]; then
            log::info "Found $doc_count test documents"
        else
            log::info "No test documents found"
        fi
    fi
    
    # Test API endpoint
    log::info "Testing Unstructured.io API..."
    
    if curl -s "${UNSTRUCTURED_HOST}/healthcheck" 2>/dev/null | grep -q "ok"; then
        log::success "✅ Unstructured.io API is healthy"
    else
        log::error "❌ Unstructured.io API health check failed"
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
unstructured_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        unstructured_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            unstructured_inject::validate_config "$config"
            ;;
        "--inject")
            unstructured_inject::inject_data "$config"
            ;;
        "--status")
            unstructured_inject::check_status "$config"
            ;;
        "--rollback")
            unstructured_inject::execute_rollback
            ;;
        "--help")
            unstructured_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            unstructured_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        unstructured_inject::usage
        exit 1
    fi
    
    unstructured_inject::main "$@"
fi