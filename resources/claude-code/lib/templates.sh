#!/usr/bin/env bash
# Claude Code Composable Prompt Template System
# Provides reusable prompt templates for common development tasks

# Set CLAUDE_CODE_SCRIPT_DIR if not already set (for BATS test compatibility)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
CLAUDE_CODE_SCRIPT_DIR="${CLAUDE_CODE_SCRIPT_DIR:-${APP_ROOT}/resources/claude-code}"

# Template directory
TEMPLATES_DIR="${CLAUDE_CODE_SCRIPT_DIR}/templates/prompts"

#######################################
# List available prompt templates
# Arguments:
#   $1 - Format (text|json)
# Outputs: List of available templates
#######################################
claude_code::templates_list() {
    local format="${1:-text}"
    
    if [[ ! -d "$TEMPLATES_DIR" ]]; then
        if [[ "$format" == "json" ]]; then
            echo "[]"
        else
            echo "No templates directory found"
        fi
        return 1
    fi
    
    local templates=()
    
    # Find all .md template files
    while IFS= read -r -d '' template_file; do
        local template_name
        template_name=$(basename "$template_file" .md)
        templates+=("$template_name")
    done < <(find "$TEMPLATES_DIR" -name "*.md" -type f -print0 2>/dev/null)
    
    case "$format" in
        "json")
            if [[ ${#templates[@]} -eq 0 ]]; then
                echo "[]"
            else
                printf '%s\n' "${templates[@]}" | jq -R . | jq -s '.'
            fi
            ;;
        "text")
            if [[ ${#templates[@]} -eq 0 ]]; then
                echo "No templates found"
            else
                echo "Available templates:"
                printf '  - %s\n' "${templates[@]}"
            fi
            ;;
    esac
}

#######################################
# Load and process a prompt template
# Arguments:
#   $1 - Template name (without .md extension)
#   $@ - Variable substitutions in format "key=value"
# Outputs: Processed template content
#######################################
claude_code::template_load() {
    local template_name="$1"
    shift
    
    local template_file="$TEMPLATES_DIR/${template_name}.md"
    
    if [[ ! -f "$template_file" ]]; then
        log::error "Template not found: $template_name"
        log::info "Available templates:"
        claude_code::templates_list text
        return 1
    fi
    
    # Read template content
    local template_content
    template_content=$(cat "$template_file")
    
    # Process variable substitutions
    for var in "$@"; do
        if [[ "$var" == *"="* ]]; then
            local key="${var%%=*}"
            local value="${var#*=}"
            
            # Replace {key} placeholders with values
            template_content="${template_content//\{$key\}/$value}"
            template_content="${template_content//\{\{$key\}\}/$value}"  # Support double braces too
        else
            log::warn "Invalid variable format: $var (expected key=value)"
        fi
    done
    
    # Check for unreplaced variables
    local unreplaced
    unreplaced=$(echo "$template_content" | grep -o '{[^}]*}' | sort -u || true)
    
    if [[ -n "$unreplaced" ]]; then
        log::warn "Unreplaced template variables found:"
        echo "$unreplaced" | while read -r var; do
            log::warn "  $var"
        done
    fi
    
    echo "$template_content"
}

#######################################
# Execute Claude Code with a template
# Arguments:
#   $1 - Template name
#   $2 - Max turns (optional, default 20)
#   $3 - Allowed tools (optional, default "Read,Edit,Write")
#   $@ - Variable substitutions in format "key=value"
# Returns: 0 on success, 1 on failure
#######################################
claude_code::template_run() {
    local template_name="$1"
    local max_turns="${2:-20}"
    local allowed_tools="${3:-Read,Edit,Write}"
    shift 3
    
    log::header "🎯 Running Template: $template_name"
    log::info "Max turns: $max_turns"
    log::info "Allowed tools: $allowed_tools"
    
    if [[ $# -gt 0 ]]; then
        log::info "Variables:"
        for var in "$@"; do
            if [[ "$var" == *"="* ]]; then
                log::info "  ${var%%=*} = ${var#*=}"
            fi
        done
    fi
    
    # Load and process template
    local prompt
    if ! prompt=$(claude_code::template_load "$template_name" "$@"); then
        return 1
    fi
    
    echo
    log::info "Generated prompt preview:"
    echo "─────────────────────────────────────────"
    echo "$prompt" | head -5
    if [[ $(echo "$prompt" | wc -l) -gt 5 ]]; then
        echo "... ($(echo "$prompt" | wc -l) total lines)"
    fi
    echo "─────────────────────────────────────────"
    echo
    
    # Execute with Claude Code
    claude_code::run_automation "$prompt" "$allowed_tools" "$max_turns"
}

#######################################
# Create a new template from user input
# Arguments:
#   $1 - Template name
#   $2 - Template description (optional)
# Returns: 0 on success, 1 on failure
#######################################
claude_code::template_create() {
    local template_name="$1"
    local description="${2:-Custom template}"
    
    if [[ -z "$template_name" ]]; then
        log::error "Template name is required"
        return 1
    fi
    
    # Validate template name
    if [[ ! "$template_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        log::error "Template name must contain only letters, numbers, hyphens, and underscores"
        return 1
    fi
    
    local template_file="$TEMPLATES_DIR/${template_name}.md"
    
    if [[ -f "$template_file" ]]; then
        log::error "Template already exists: $template_name"
        if ! confirm "Overwrite existing template?"; then
            return 1
        fi
    fi
    
    # Create templates directory if it doesn't exist
    mkdir -p "$TEMPLATES_DIR"
    
    log::header "📝 Creating Template: $template_name"
    log::info "Description: $description"
    log::info "Template file: $template_file"
    
    # Create template with basic structure
    cat > "$template_file" <<EOF
# $template_name Template

$description

## Variables
This template supports the following variables:
- \`{files}\` - Files to process (e.g., "src/**/*.ts")
- \`{task}\` - Specific task description
- \`{context}\` - Additional context information

## Template Content

You are an expert software engineer working on {files}.

Task: {task}

Context: {context}

Please analyze the code and provide detailed recommendations for improvement.

Focus on:
1. Code quality and maintainability
2. Performance optimization opportunities
3. Security considerations
4. Best practices adherence

Provide specific examples and actionable suggestions.
EOF
    
    log::success "✅ Template created: $template_file"
    log::info "Edit the template file to customize the prompt content"
    log::info "Use variables like {variable_name} for dynamic content"
    
    return 0
}

#######################################
# Show template information and preview
# Arguments:
#   $1 - Template name
# Outputs: Template metadata and preview
#######################################
claude_code::template_info() {
    local template_name="$1"
    
    if [[ -z "$template_name" ]]; then
        log::error "Template name is required"
        return 1
    fi
    
    local template_file="$TEMPLATES_DIR/${template_name}.md"
    
    if [[ ! -f "$template_file" ]]; then
        log::error "Template not found: $template_name"
        return 1
    fi
    
    log::header "📋 Template Information: $template_name"
    log::info "File: $template_file"
    log::info "Size: $(wc -c < "$template_file") bytes"
    log::info "Lines: $(wc -l < "$template_file")"
    
    # Extract variables from template
    local variables
    variables=$(grep -o '{[^}]*}' "$template_file" | sort -u | tr '\n' ' ' || echo "none")
    log::info "Variables: $variables"
    
    echo
    log::info "Template preview:"
    echo "─────────────────────────────────────────"
    head -20 "$template_file"
    if [[ $(wc -l < "$template_file") -gt 20 ]]; then
        echo
        echo "... ($(wc -l < "$template_file") total lines, showing first 20)"
    fi
    echo "─────────────────────────────────────────"
}

#######################################
# Validate a template for syntax and structure
# Arguments:
#   $1 - Template name
# Returns: 0 if valid, 1 if invalid
#######################################
claude_code::template_validate() {
    local template_name="$1"
    
    if [[ -z "$template_name" ]]; then
        log::error "Template name is required"
        return 1
    fi
    
    local template_file="$TEMPLATES_DIR/${template_name}.md"
    
    if [[ ! -f "$template_file" ]]; then
        log::error "Template not found: $template_name"
        return 1
    fi
    
    log::header "✅ Validating Template: $template_name"
    
    local issues=0
    
    # Check file is readable
    if [[ ! -r "$template_file" ]]; then
        log::error "Template file is not readable"
        issues=$((issues + 1))
    fi
    
    # Check file is not empty
    if [[ ! -s "$template_file" ]]; then
        log::error "Template file is empty"
        issues=$((issues + 1))
    fi
    
    # Check for balanced braces
    local open_braces close_braces
    open_braces=$(grep -o '{' "$template_file" | wc -l || echo 0)
    close_braces=$(grep -o '}' "$template_file" | wc -l || echo 0)
    
    if [[ $open_braces -ne $close_braces ]]; then
        log::error "Unbalanced braces: $open_braces open, $close_braces close"
        issues=$((issues + 1))
    fi
    
    # Check for malformed variables
    local malformed
    malformed=$(grep -o '{[^}]*{' "$template_file" || true)
    if [[ -n "$malformed" ]]; then
        log::error "Malformed variables found (nested braces):"
        echo "$malformed" | while read -r var; do
            log::error "  $var"
        done
        issues=$((issues + 1))
    fi
    
    # Check for reasonable content length
    local char_count
    char_count=$(wc -c < "$template_file")
    if [[ $char_count -lt 50 ]]; then
        log::warn "Template is very short ($char_count characters)"
    elif [[ $char_count -gt 10000 ]]; then
        log::warn "Template is very long ($char_count characters)"
    fi
    
    # List variables for review
    local variables
    variables=$(grep -o '{[^}]*}' "$template_file" | sort -u || true)
    if [[ -n "$variables" ]]; then
        log::info "Template variables found:"
        echo "$variables" | while read -r var; do
            log::info "  $var"
        done
    else
        log::info "No template variables found"
    fi
    
    if [[ $issues -eq 0 ]]; then
        log::success "✅ Template validation passed"
        return 0
    else
        log::error "❌ Template validation failed with $issues issues"
        return 1
    fi
}

#######################################
# Wrapper function for template management
# Arguments:
#   $1 - Action (list|load|run|create|info|validate)
#   $@ - Action-specific arguments
#######################################
claude_code::template_manage() {
    local action="$1"
    shift
    
    case "$action" in
        "list")
            claude_code::templates_list "$@"
            ;;
        "load")
            claude_code::template_load "$@"
            ;;
        "run")
            claude_code::template_run "$@"
            ;;
        "create")
            claude_code::template_create "$@"
            ;;
        "info")
            claude_code::template_info "$@"
            ;;
        "validate")
            claude_code::template_validate "$@"
            ;;
        *)
            log::error "Unknown template action: $action"
            echo "Available actions: list, load, run, create, info, validate"
            return 1
            ;;
    esac
}

# Export functions for external use
export -f claude_code::templates_list
export -f claude_code::template_load
export -f claude_code::template_run
export -f claude_code::template_create
export -f claude_code::template_info
export -f claude_code::template_validate
export -f claude_code::template_manage