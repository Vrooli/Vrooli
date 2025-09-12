#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_DIR="${APP_ROOT}/resources/twilio"
TWILIO_LIB_DIR="${TWILIO_DIR}/lib"
TWILIO_DATA_DIR="${TWILIO_DIR}/data"
TWILIO_TEMPLATES_FILE="${TWILIO_DATA_DIR}/templates.json"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Ensure templates directory exists
twilio::templates::ensure_data_dir() {
    if [[ ! -d "$TWILIO_DATA_DIR" ]]; then
        mkdir -p "$TWILIO_DATA_DIR"
    fi
    
    # Initialize templates file if it doesn't exist
    if [[ ! -f "$TWILIO_TEMPLATES_FILE" ]]; then
        echo '{"templates": []}' > "$TWILIO_TEMPLATES_FILE"
    fi
}

# Create a new template
twilio::templates::create() {
    local name="${1:-}"
    local template="${2:-}"
    local description="${3:-}"
    
    if [[ -z "$name" ]] || [[ -z "$template" ]]; then
        log::error "Usage: resource-twilio content template-create <name> <template> [description]"
        log::error "Variables: Use {{variable}} for substitution"
        log::error "Example: 'Hi {{name}}, your order {{order_id}} is ready!'"
        return 1
    fi
    
    twilio::templates::ensure_data_dir
    
    # Check if template already exists
    local existing
    existing=$(jq ".templates[] | select(.name == \"$name\")" "$TWILIO_TEMPLATES_FILE" 2>/dev/null || echo "")
    
    if [[ -n "$existing" ]]; then
        log::error "Template '$name' already exists. Use template-update to modify it."
        return 1
    fi
    
    # Extract variables from template
    local variables
    variables=$(echo "$template" | grep -oE '\{\{[^}]+\}\}' | sed 's/[{}]//g' | sort -u | jq -R . | jq -s .)
    
    # Create template entry
    local template_entry
    template_entry=$(jq -n \
        --arg name "$name" \
        --arg template "$template" \
        --arg description "$description" \
        --arg created "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        --argjson variables "$variables" \
        '{
            name: $name,
            template: $template,
            description: $description,
            created: $created,
            variables: $variables
        }')
    
    # Add to templates file
    jq ".templates += [$template_entry]" "$TWILIO_TEMPLATES_FILE" > "${TWILIO_TEMPLATES_FILE}.tmp" && \
        mv "${TWILIO_TEMPLATES_FILE}.tmp" "$TWILIO_TEMPLATES_FILE"
    
    log::success "Template '$name' created successfully"
    
    # Show detected variables
    local variables
    variables=$(echo "$template_entry" | jq -r '.variables[]' 2>/dev/null)
    if [[ -n "$variables" ]]; then
        log::info "Detected variables:"
        while IFS= read -r var; do
            log::info "  - {{$var}}"
        done <<< "$variables"
    fi
}

# List all templates
twilio::templates::list() {
    twilio::templates::ensure_data_dir
    
    local count
    count=$(jq '.templates | length' "$TWILIO_TEMPLATES_FILE")
    
    if [[ $count -eq 0 ]]; then
        log::info "No templates found"
        return 0
    fi
    
    log::info "📝 Available templates ($count):"
    
    jq -r '.templates[] | "\n  Name: \(.name)\n  Template: \(.template)\n  Description: \(.description // "N/A")\n  Variables: \(.variables | join(", "))\n  Created: \(.created)"' "$TWILIO_TEMPLATES_FILE"
}

# Get a specific template
twilio::templates::get() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Template name is required"
        return 1
    fi
    
    twilio::templates::ensure_data_dir
    
    local template
    template=$(jq ".templates[] | select(.name == \"$name\")" "$TWILIO_TEMPLATES_FILE" 2>/dev/null)
    
    if [[ -z "$template" ]]; then
        log::error "Template '$name' not found"
        return 1
    fi
    
    echo "$template" | jq .
}

# Update a template
twilio::templates::update() {
    local name="${1:-}"
    local new_template="${2:-}"
    local new_description="${3:-}"
    
    if [[ -z "$name" ]] || [[ -z "$new_template" ]]; then
        log::error "Usage: resource-twilio content template-update <name> <new_template> [new_description]"
        return 1
    fi
    
    twilio::templates::ensure_data_dir
    
    # Check if template exists
    local existing
    existing=$(jq ".templates[] | select(.name == \"$name\")" "$TWILIO_TEMPLATES_FILE" 2>/dev/null || echo "")
    
    if [[ -z "$existing" ]]; then
        log::error "Template '$name' not found"
        return 1
    fi
    
    # Extract variables from new template
    local variables
    variables=$(echo "$new_template" | grep -oE '\{\{[^}]+\}\}' | sed 's/[{}]//g' | sort -u | jq -R . | jq -s .)
    
    # Update template
    local updated_template
    if [[ -n "$new_description" ]]; then
        updated_template=$(jq \
            --arg name "$name" \
            --arg template "$new_template" \
            --arg description "$new_description" \
            --argjson variables "$variables" \
            --arg updated "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
            '.templates |= map(if .name == $name then .template = $template | .description = $description | .variables = $variables | .updated = $updated else . end)' \
            "$TWILIO_TEMPLATES_FILE")
    else
        updated_template=$(jq \
            --arg name "$name" \
            --arg template "$new_template" \
            --argjson variables "$variables" \
            --arg updated "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
            '.templates |= map(if .name == $name then .template = $template | .variables = $variables | .updated = $updated else . end)' \
            "$TWILIO_TEMPLATES_FILE")
    fi
    
    echo "$updated_template" > "${TWILIO_TEMPLATES_FILE}.tmp" && \
        mv "${TWILIO_TEMPLATES_FILE}.tmp" "$TWILIO_TEMPLATES_FILE"
    
    log::success "Template '$name' updated successfully"
}

# Delete a template
twilio::templates::delete() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Template name is required"
        return 1
    fi
    
    twilio::templates::ensure_data_dir
    
    # Check if template exists
    local existing
    existing=$(jq ".templates[] | select(.name == \"$name\")" "$TWILIO_TEMPLATES_FILE" 2>/dev/null || echo "")
    
    if [[ -z "$existing" ]]; then
        log::error "Template '$name' not found"
        return 1
    fi
    
    # Delete template
    jq ".templates |= map(select(.name != \"$name\"))" "$TWILIO_TEMPLATES_FILE" > "${TWILIO_TEMPLATES_FILE}.tmp" && \
        mv "${TWILIO_TEMPLATES_FILE}.tmp" "$TWILIO_TEMPLATES_FILE"
    
    log::success "Template '$name' deleted"
}

# Send SMS using a template
twilio::templates::send() {
    local template_name="${1:-}"
    local to="${2:-}"
    local from="${3:-}"
    shift 3
    local variables=("$@")
    
    if [[ -z "$template_name" ]] || [[ -z "$to" ]]; then
        log::error "Usage: resource-twilio content template-send <template_name> <to> [from] [var1=value1] [var2=value2] ..."
        log::error "Example: resource-twilio content template-send welcome +1234567890 +0987654321 name=John order_id=12345"
        return 1
    fi
    
    twilio::templates::ensure_data_dir
    
    # Get template
    local template_data
    template_data=$(jq ".templates[] | select(.name == \"$template_name\")" "$TWILIO_TEMPLATES_FILE" 2>/dev/null)
    
    if [[ -z "$template_data" ]]; then
        log::error "Template '$template_name' not found"
        return 1
    fi
    
    local template_text
    template_text=$(echo "$template_data" | jq -r '.template')
    
    # Process variables
    local message="$template_text"
    for var_pair in "${variables[@]}"; do
        if [[ "$var_pair" =~ ^([^=]+)=(.*)$ ]]; then
            local var_name="${BASH_REMATCH[1]}"
            local var_value="${BASH_REMATCH[2]}"
            message="${message//\{\{$var_name\}\}/$var_value}"
        fi
    done
    
    # Check for unsubstituted variables
    if [[ "$message" =~ \{\{[^}]+\}\} ]]; then
        log::warn "Warning: Message contains unsubstituted variables"
        log::info "Message preview: $message"
        echo -n "Continue sending? (y/n): "
        read -r confirm
        if [[ "$confirm" != "y" ]]; then
            log::info "Send cancelled"
            return 1
        fi
    fi
    
    log::info "📝 Using template: $template_name"
    log::info "   Message: $message"
    
    # Check for test mode
    if [[ "${TWILIO_ACCOUNT_SID:-}" =~ test ]] || [[ "${TWILIO_ACCOUNT_SID:-}" == "AC_test_"* ]]; then
        log::info "Test mode detected - simulating template SMS send"
        log::success "Would send templated message to $to"
        return 0
    fi
    
    # Send the message
    source "$TWILIO_LIB_DIR/sms.sh"
    twilio::send_sms "$to" "$message" "$from"
}

# Send bulk SMS using a template
twilio::templates::send_bulk() {
    local template_name="${1:-}"
    local csv_file="${2:-}"
    local from="${3:-}"
    
    if [[ -z "$template_name" ]] || [[ -z "$csv_file" ]]; then
        log::error "Usage: resource-twilio content template-send-bulk <template_name> <csv_file> [from]"
        log::error "CSV format: phone_number,var1,var2,..."
        log::error "First row should be header with variable names"
        return 1
    fi
    
    if [[ ! -f "$csv_file" ]]; then
        log::error "CSV file not found: $csv_file"
        return 1
    fi
    
    twilio::templates::ensure_data_dir
    
    # Get template
    local template_data
    template_data=$(jq ".templates[] | select(.name == \"$template_name\")" "$TWILIO_TEMPLATES_FILE" 2>/dev/null)
    
    if [[ -z "$template_data" ]]; then
        log::error "Template '$template_name' not found"
        return 1
    fi
    
    local template_text
    template_text=$(echo "$template_data" | jq -r '.template')
    
    log::info "📝 Using template: $template_name"
    log::info "   Processing CSV: $csv_file"
    
    # Read CSV header
    local header
    IFS=',' read -r -a header < "$csv_file"
    
    # Process each row
    local line_count=0
    local success_count=0
    local fail_count=0
    
    while IFS=',' read -r -a values; do
        ((line_count++))
        
        # Skip header
        if [[ $line_count -eq 1 ]]; then
            continue
        fi
        
        # Get phone number (first column)
        local phone_number="${values[0]}"
        phone_number=$(echo "$phone_number" | tr -d '"' | tr -d ' ')
        
        if [[ -z "$phone_number" ]]; then
            continue
        fi
        
        # Build message from template
        local message="$template_text"
        for i in "${!header[@]}"; do
            if [[ $i -gt 0 ]]; then  # Skip phone number column
                local var_name="${header[$i]}"
                local var_value="${values[$i]:-}"
                var_name=$(echo "$var_name" | tr -d '"' | tr -d ' ')
                var_value=$(echo "$var_value" | tr -d '"')
                message="${message//\{\{$var_name\}\}/$var_value}"
            fi
        done
        
        log::info "   Sending to: $phone_number"
        
        # Send the message
        source "$TWILIO_LIB_DIR/sms.sh"
        if twilio::send_sms "$phone_number" "$message" "$from" &>/dev/null; then
            ((success_count++))
        else
            ((fail_count++))
            log::warn "   Failed to send to: $phone_number"
        fi
        
        # Rate limiting
        sleep 0.1
    done < "$csv_file"
    
    log::info "📊 Bulk Template SMS Results:"
    log::success "   Successful: $success_count"
    if [[ $fail_count -gt 0 ]]; then
        log::warn "   Failed: $fail_count"
    fi
    
    if [[ $fail_count -eq 0 ]] && [[ $success_count -gt 0 ]]; then
        log::success "All messages sent successfully!"
        return 0
    elif [[ $success_count -gt 0 ]]; then
        log::warn "Partial success: $success_count messages sent"
        return 1
    else
        log::error "Failed to send any messages"
        return 1
    fi
}