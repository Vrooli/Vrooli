#!/bin/bash
# Airbyte Schedule Management Library
# Provides cron-based scheduling for sync jobs

set -euo pipefail

# Schedule storage directory
SCHEDULE_DIR="${DATA_DIR}/schedules"
CRON_FILE="${SCHEDULE_DIR}/airbyte.cron"
SCHEDULE_REGISTRY="${SCHEDULE_DIR}/registry.json"

# Initialize schedule storage
init_schedules() {
    if [[ ! -d "$SCHEDULE_DIR" ]]; then
        mkdir -p "$SCHEDULE_DIR"
        chmod 755 "$SCHEDULE_DIR"
    fi
    
    if [[ ! -f "$SCHEDULE_REGISTRY" ]]; then
        echo "[]" > "$SCHEDULE_REGISTRY"
    fi
    
    if [[ ! -f "$CRON_FILE" ]]; then
        touch "$CRON_FILE"
    fi
}

# Create or update a schedule
create_schedule() {
    local name="$1"
    local connection_id="$2"
    local cron_expression="$3"
    local enabled="${4:-true}"
    
    init_schedules
    
    # Validate cron expression
    if ! validate_cron "$cron_expression"; then
        log_error "Invalid cron expression: $cron_expression"
        return 1
    fi
    
    # Create schedule entry
    local schedule_entry=$(jq -n \
        --arg name "$name" \
        --arg connection_id "$connection_id" \
        --arg cron "$cron_expression" \
        --arg enabled "$enabled" \
        --arg created "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        '{
            name: $name,
            connection_id: $connection_id,
            cron_expression: $cron,
            enabled: ($enabled == "true"),
            created: $created,
            last_run: null,
            next_run: null
        }')
    
    # Update registry
    local updated_registry=$(cat "$SCHEDULE_REGISTRY" | jq \
        --argjson entry "$schedule_entry" \
        'map(select(.name != $entry.name)) + [$entry]')
    
    echo "$updated_registry" > "$SCHEDULE_REGISTRY"
    
    # Update crontab if enabled
    if [[ "$enabled" == "true" ]]; then
        add_to_cron "$name" "$connection_id" "$cron_expression"
    fi
    
    # Calculate next run time
    local next_run=$(calculate_next_run "$cron_expression")
    update_schedule_next_run "$name" "$next_run"
    
    log_info "Schedule created: $name"
    echo "$schedule_entry"
}

# Validate cron expression
validate_cron() {
    local cron="$1"
    
    # Basic validation - check field count and format
    local field_count=$(echo "$cron" | awk '{print NF}')
    
    if [[ $field_count -ne 5 ]]; then
        return 1
    fi
    
    # Validate each field
    echo "$cron" | awk '
    {
        # minute (0-59)
        if ($1 !~ /^(\*|[0-9]|[0-5][0-9])(\/[0-9]+)?$/) exit 1
        
        # hour (0-23)
        if ($2 !~ /^(\*|[0-9]|1[0-9]|2[0-3])(\/[0-9]+)?$/) exit 1
        
        # day of month (1-31)
        if ($3 !~ /^(\*|[1-9]|[12][0-9]|3[01])(\/[0-9]+)?$/) exit 1
        
        # month (1-12)
        if ($4 !~ /^(\*|[1-9]|1[0-2])(\/[0-9]+)?$/) exit 1
        
        # day of week (0-7)
        if ($5 !~ /^(\*|[0-7])(\/[0-9]+)?$/) exit 1
    }'
    
    return $?
}

# Add schedule to crontab
add_to_cron() {
    local name="$1"
    local connection_id="$2"
    local cron_expression="$3"
    
    # Remove existing entry if present
    remove_from_cron "$name"
    
    # Add new cron entry
    local cron_command="${RESOURCE_DIR}/cli.sh schedule execute --name \"$name\" --connection-id \"$connection_id\""
    echo "$cron_expression $cron_command # airbyte-schedule:$name" >> "$CRON_FILE"
    
    # Install crontab
    install_cron
}

# Remove schedule from crontab
remove_from_cron() {
    local name="$1"
    
    if [[ -f "$CRON_FILE" ]]; then
        grep -v "# airbyte-schedule:$name$" "$CRON_FILE" > "${CRON_FILE}.tmp" || true
        mv "${CRON_FILE}.tmp" "$CRON_FILE"
        install_cron
    fi
}

# Install crontab
install_cron() {
    # Check if any schedules are enabled
    if [[ -s "$CRON_FILE" ]]; then
        # Install the crontab
        crontab "$CRON_FILE"
        log_info "Crontab updated"
    else
        # Remove crontab if no schedules
        crontab -r 2>/dev/null || true
    fi
}

# Get schedule by name
get_schedule() {
    local name="$1"
    
    init_schedules
    
    cat "$SCHEDULE_REGISTRY" | jq --arg name "$name" '.[] | select(.name == $name)'
}

# List all schedules
list_schedules() {
    init_schedules
    
    cat "$SCHEDULE_REGISTRY" | jq '.'
}

# Update schedule status
update_schedule() {
    local name="$1"
    local field="$2"
    local value="$3"
    
    init_schedules
    
    local updated=$(cat "$SCHEDULE_REGISTRY" | jq \
        --arg name "$name" \
        --arg field "$field" \
        --arg value "$value" \
        'map(if .name == $name then .[$field] = $value else . end)')
    
    echo "$updated" > "$SCHEDULE_REGISTRY"
}

# Enable schedule
enable_schedule() {
    local name="$1"
    
    local schedule=$(get_schedule "$name")
    if [[ -z "$schedule" ]]; then
        log_error "Schedule not found: $name"
        return 1
    fi
    
    local connection_id=$(echo "$schedule" | jq -r '.connection_id')
    local cron_expression=$(echo "$schedule" | jq -r '.cron_expression')
    
    update_schedule "$name" "enabled" "true"
    add_to_cron "$name" "$connection_id" "$cron_expression"
    
    log_info "Schedule enabled: $name"
}

# Disable schedule
disable_schedule() {
    local name="$1"
    
    update_schedule "$name" "enabled" "false"
    remove_from_cron "$name"
    
    log_info "Schedule disabled: $name"
}

# Delete schedule
delete_schedule() {
    local name="$1"
    
    init_schedules
    
    # Remove from cron
    remove_from_cron "$name"
    
    # Remove from registry
    local updated=$(cat "$SCHEDULE_REGISTRY" | jq \
        --arg name "$name" \
        'map(select(.name != $name))')
    
    echo "$updated" > "$SCHEDULE_REGISTRY"
    
    log_info "Schedule deleted: $name"
}

# Execute scheduled sync
execute_scheduled_sync() {
    local name="$1"
    local connection_id="$2"
    
    log_info "Executing scheduled sync: $name (connection: $connection_id)"
    
    # Update last run time
    update_schedule "$name" "last_run" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    
    # Trigger sync
    local result=$(content_execute --connection-id "$connection_id" --wait)
    
    # Update next run time
    local schedule=$(get_schedule "$name")
    local cron_expression=$(echo "$schedule" | jq -r '.cron_expression')
    local next_run=$(calculate_next_run "$cron_expression")
    update_schedule_next_run "$name" "$next_run"
    
    return $?
}

# Calculate next run time from cron expression
calculate_next_run() {
    local cron="$1"
    
    # Use python to calculate next run time
    python3 -c "
import croniter
import datetime
import sys

try:
    cron = croniter.croniter('$cron', datetime.datetime.now())
    next_run = cron.get_next(datetime.datetime)
    print(next_run.strftime('%Y-%m-%dT%H:%M:%SZ'))
except Exception as e:
    print('', end='')
" 2>/dev/null || echo ""
}

# Update next run time
update_schedule_next_run() {
    local name="$1"
    local next_run="$2"
    
    if [[ -n "$next_run" ]]; then
        update_schedule "$name" "next_run" "$next_run"
    fi
}

# Get schedule status
schedule_status() {
    local name="$1"
    
    local schedule=$(get_schedule "$name")
    if [[ -z "$schedule" ]]; then
        echo "Schedule not found: $name"
        return 1
    fi
    
    echo "Schedule: $name"
    echo "  Connection ID: $(echo "$schedule" | jq -r '.connection_id')"
    echo "  Cron Expression: $(echo "$schedule" | jq -r '.cron_expression')"
    echo "  Enabled: $(echo "$schedule" | jq -r '.enabled')"
    echo "  Last Run: $(echo "$schedule" | jq -r '.last_run // "Never"')"
    echo "  Next Run: $(echo "$schedule" | jq -r '.next_run // "Not scheduled"')"
    echo "  Created: $(echo "$schedule" | jq -r '.created')"
}

# Validate all schedules
validate_schedules() {
    init_schedules
    
    local schedules=$(cat "$SCHEDULE_REGISTRY" | jq -r '.[] | @json')
    local valid=0
    local invalid=0
    
    while IFS= read -r schedule_json; do
        if [[ -z "$schedule_json" ]]; then
            continue
        fi
        
        local schedule=$(echo "$schedule_json" | jq '.')
        local name=$(echo "$schedule" | jq -r '.name')
        local cron=$(echo "$schedule" | jq -r '.cron_expression')
        
        if validate_cron "$cron"; then
            ((valid++))
        else
            ((invalid++))
            log_error "Invalid schedule: $name (cron: $cron)"
        fi
    done <<< "$schedules"
    
    echo "Schedule validation complete: $valid valid, $invalid invalid"
    return $([ $invalid -eq 0 ] && echo 0 || echo 1)
}