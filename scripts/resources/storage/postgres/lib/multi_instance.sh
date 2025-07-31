#!/usr/bin/env bash
# PostgreSQL Multi-Instance Operations
# Functions for batch operations across multiple instances

#######################################
# Execute operation on multiple instances
# Arguments:
#   $1 - operation function name
#   $2 - instance pattern ("all" or specific names)
#   $@ - additional arguments to pass to operation
# Returns: 0 if all succeed, 1 if any fail
#######################################
postgres::multi::execute() {
    local operation="$1"
    local instance_pattern="$2"
    shift 2
    local additional_args=("$@")
    
    if [[ -z "$operation" || -z "$instance_pattern" ]]; then
        log::error "Operation and instance pattern are required"
        return 1
    fi
    
    # Get list of instances to operate on
    local instances=()
    if [[ "$instance_pattern" == "all" ]]; then
        instances=($(postgres::common::list_instances))
    else
        # Split comma-separated instance names
        IFS=',' read -ra instances <<< "$instance_pattern"
    fi
    
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::warn "No instances found matching pattern: $instance_pattern"
        return 0
    fi
    
    log::info "Executing '$operation' on ${#instances[@]} instance(s)..."
    
    local success_count=0
    local failure_count=0
    local failed_instances=()
    
    for instance in "${instances[@]}"; do
        # Clean instance name
        instance=$(echo "$instance" | tr -d ' ')
        
        log::info "  → Processing instance: $instance"
        
        # Execute operation on instance
        if "$operation" "$instance" "${additional_args[@]}"; then
            log::success "    ✓ Success: $instance"
            ((success_count++))
        else
            log::error "    ✗ Failed: $instance"
            ((failure_count++))
            failed_instances+=("$instance")
        fi
    done
    
    # Summary
    log::info ""
    log::info "Multi-instance operation summary:"
    log::info "  Total instances: ${#instances[@]}"
    log::info "  Successful: $success_count"
    log::info "  Failed: $failure_count"
    
    if [[ $failure_count -gt 0 ]]; then
        log::error "Failed instances: ${failed_instances[*]}"
        return 1
    else
        log::success "All instances processed successfully"
        return 0
    fi
}

#######################################
# Start multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
# Returns: 0 if all succeed, 1 if any fail
#######################################
postgres::multi::start() {
    local instance_pattern="${1:-all}"
    
    postgres::multi::execute "postgres::instance::start" "$instance_pattern"
}

#######################################
# Stop multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
# Returns: 0 if all succeed, 1 if any fail
#######################################
postgres::multi::stop() {
    local instance_pattern="${1:-all}"
    
    postgres::multi::execute "postgres::instance::stop" "$instance_pattern"
}

#######################################
# Restart multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
# Returns: 0 if all succeed, 1 if any fail
#######################################
postgres::multi::restart() {
    local instance_pattern="${1:-all}"
    
    postgres::multi::execute "postgres::instance::restart" "$instance_pattern"
}

#######################################
# Run migrations on multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
#   $2 - migrations directory path
#   $3 - database name (optional)
# Returns: 0 if all succeed, 1 if any fail
#######################################
postgres::multi::migrate() {
    local instance_pattern="$1"
    local migrations_dir="$2"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    
    if [[ -z "$migrations_dir" ]]; then
        log::error "Migrations directory is required"
        return 1
    fi
    
    postgres::multi::execute "postgres::migration::run" "$instance_pattern" "$migrations_dir" "$database"
}

#######################################
# Create database on multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
#   $2 - database name
#   $3 - owner (optional)
# Returns: 0 if all succeed, 1 if any fail
#######################################
postgres::multi::create_database() {
    local instance_pattern="$1"
    local database="$2"
    local owner="${3:-$POSTGRES_DEFAULT_USER}"
    
    if [[ -z "$database" ]]; then
        log::error "Database name is required"
        return 1
    fi
    
    postgres::multi::execute "postgres::database::create" "$instance_pattern" "$database" "$owner"
}

#######################################
# Create user on multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
#   $2 - username
#   $3 - password
# Returns: 0 if all succeed, 1 if any fail
#######################################
postgres::multi::create_user() {
    local instance_pattern="$1"
    local username="$2"
    local password="$3"
    
    if [[ -z "$username" || -z "$password" ]]; then
        log::error "Username and password are required"
        return 1
    fi
    
    postgres::multi::execute "postgres::database::create_user" "$instance_pattern" "$username" "$password"
}

#######################################
# Backup multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
#   $2 - backup name prefix (optional, defaults to timestamp)
#   $3 - backup type (optional, default: "full")
# Returns: 0 if all succeed, 1 if any fail
#######################################
postgres::multi::backup() {
    local instance_pattern="$1"
    local backup_prefix="${2:-$(date +%Y%m%d_%H%M%S)}"
    local backup_type="${3:-full}"
    
    # Custom function to handle backup naming per instance
    postgres::multi::backup_instance() {
        local instance_name="$1"
        local backup_name="${backup_prefix}_${instance_name}"
        
        postgres::backup::create "$instance_name" "$backup_name" "$backup_type"
    }
    
    postgres::multi::execute "postgres::multi::backup_instance" "$instance_pattern"
}

#######################################
# Show status of multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
# Returns: 0 if all succeed, 1 if any fail
#######################################
postgres::multi::status() {
    local instance_pattern="${1:-all}"
    
    # Get list of instances
    local instances=()
    if [[ "$instance_pattern" == "all" ]]; then
        instances=($(postgres::common::list_instances))
    else
        IFS=',' read -ra instances <<< "$instance_pattern"
    fi
    
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::warn "No instances found matching pattern: $instance_pattern"
        return 0
    fi
    
    
    log::info "Status of PostgreSQL instances:"
    log::info "==============================="
    printf "%-20s %-12s %-8s %-12s %s\\n" "Instance" "Status" "Health" "Port" "Template"
    printf "%-20s %-12s %-8s %-12s %s\\n" "--------------------" "------------" "--------" "------------" "------------"
    
    local running_count=0
    local healthy_count=0
    
    for instance in "${instances[@]}"; do
        instance=$(echo "$instance" | tr -d ' ')
        
        local status="stopped"
        local health="unknown"
        local port="N/A"
        local template="N/A"
        
        if postgres::common::container_exists "$instance"; then
            port=$(postgres::common::get_instance_config "$instance" "port" 2>/dev/null || echo "N/A")
            template=$(postgres::common::get_instance_config "$instance" "template" 2>/dev/null || echo "N/A")
            
            if postgres::common::is_running "$instance"; then
                status="running"
                running_count=$((running_count + 1))
                
                if postgres::common::health_check "$instance"; then
                    health="healthy"
                    healthy_count=$((healthy_count + 1))
                else
                    health="unhealthy"
                fi
            fi
        else
            status="missing"
        fi
        
        # Color coding for status
        local status_display="$status"
        case "$status" in
            "running")
                if [[ "$health" == "healthy" ]]; then
                    status_display="✓ running"
                else
                    status_display="⚠ running"
                fi
                ;;
            "stopped")
                status_display="○ stopped"
                ;;
            "missing")
                status_display="✗ missing"
                ;;
        esac
        
        printf "%-20s %-12s %-8s %-12s %s\\n" "$instance" "$status_display" "$health" "$port" "$template"
    done
    
    log::info ""
    log::info "Summary: $running_count/${#instances[@]} running, $healthy_count/${#instances[@]} healthy"
    
    return 0
}

#######################################
# Health check multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
# Returns: 0 if all healthy, 1 if any unhealthy
#######################################
postgres::multi::health_check() {
    local instance_pattern="${1:-all}"
    
    # Get list of instances
    local instances=()
    if [[ "$instance_pattern" == "all" ]]; then
        instances=($(postgres::common::list_instances))
    else
        IFS=',' read -ra instances <<< "$instance_pattern"
    fi
    
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::warn "No instances found matching pattern: $instance_pattern"
        return 0
    fi
    
    log::info "Health check for ${#instances[@]} instance(s)..."
    
    local healthy_count=0
    local unhealthy_instances=()
    
    for instance in "${instances[@]}"; do
        instance=$(echo "$instance" | tr -d ' ')
        
        if postgres::common::container_exists "$instance" && postgres::common::is_running "$instance"; then
            if postgres::common::health_check "$instance"; then
                log::success "  ✓ $instance: Healthy"
                ((healthy_count++))
            else
                log::error "  ✗ $instance: Unhealthy"
                unhealthy_instances+=("$instance")
            fi
        else
            log::warn "  ○ $instance: Not running"
            unhealthy_instances+=("$instance")
        fi
    done
    
    log::info ""
    log::info "Health check summary: $healthy_count/${#instances[@]} healthy"
    
    if [[ ${#unhealthy_instances[@]} -gt 0 ]]; then
        log::error "Unhealthy instances: ${unhealthy_instances[*]}"
        return 1
    else
        log::success "All instances are healthy"
        return 0
    fi
}

#######################################
# Execute custom SQL on multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
#   $2 - SQL command
#   $3 - database name (optional)
# Returns: 0 if all succeed, 1 if any fail
#######################################
postgres::multi::execute_sql() {
    local instance_pattern="$1"
    local sql_command="$2"
    local database="${3:-$POSTGRES_DEFAULT_DB}"
    
    if [[ -z "$sql_command" ]]; then
        log::error "SQL command is required"
        return 1
    fi
    
    postgres::multi::execute "postgres::database::execute" "$instance_pattern" "$sql_command" "$database"
}

#######################################
# Update configuration template for multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
#   $2 - new template name
# Returns: 0 if all succeed, 1 if any fail
#######################################
postgres::multi::update_template() {
    local instance_pattern="$1"
    local new_template="$2"
    
    if [[ -z "$new_template" ]]; then
        log::error "Template name is required"
        return 1
    fi
    
    # Check if template exists
    local template_file="${POSTGRES_TEMPLATE_DIR}/${new_template}.conf"
    if [[ ! -f "$template_file" ]]; then
        log::error "Template not found: $template_file"
        return 1
    fi
    
    # Custom function to update template for each instance
    postgres::multi::update_instance_template() {
        local instance_name="$1"
        local template="$2"
        
        if ! postgres::common::container_exists "$instance_name"; then
            log::error "Instance '$instance_name' does not exist"
            return 1
        fi
        
        local config_dir="${POSTGRES_INSTANCES_DIR}/${instance_name}/config"
        mkdir -p "$config_dir"
        
        # Copy new template
        if cp "$template_file" "$config_dir/postgresql.conf"; then
            # Update instance configuration
            postgres::common::set_instance_config "$instance_name" "template" "$template"
            
            log::info "Updated template for instance '$instance_name' to '$template'"
            log::info "Restart instance to apply changes"
            return 0
        else
            log::error "Failed to update template for instance '$instance_name'"
            return 1
        fi
    }
    
    postgres::multi::execute "postgres::multi::update_instance_template" "$instance_pattern" "$new_template"
}

#######################################
# Show resource usage for multiple instances
# Arguments:
#   $1 - instance pattern ("all" or specific names)
# Returns: 0 on success, 1 on failure
#######################################
postgres::multi::resource_usage() {
    local instance_pattern="${1:-all}"
    
    # Get list of instances
    local instances=()
    if [[ "$instance_pattern" == "all" ]]; then
        instances=($(postgres::common::list_instances))
    else
        IFS=',' read -ra instances <<< "$instance_pattern"
    fi
    
    if [[ ${#instances[@]} -eq 0 ]]; then
        log::warn "No instances found matching pattern: $instance_pattern"
        return 0
    fi
    
    log::info "Resource Usage for PostgreSQL instances:"
    log::info "========================================"
    printf "%-20s %-12s %-12s %-12s %s\\n" "Instance" "CPU %" "Memory" "Disk I/O" "Status"
    printf "%-20s %-12s %-12s %-12s %s\\n" "--------------------" "------------" "------------" "------------" "------------"
    
    for instance in "${instances[@]}"; do
        instance=$(echo "$instance" | tr -d ' ')
        
        if postgres::common::is_running "$instance"; then
            local container_name="${POSTGRES_CONTAINER_PREFIX}-${instance}"
            
            # Get container stats (simplified)
            local stats=$(docker stats "$container_name" --no-stream --format "table {{.CPUPerc}}\t{{.MemUsage}}\t{{.BlockIO}}" 2>/dev/null | tail -n 1)
            
            if [[ -n "$stats" ]]; then
                IFS=$'\t' read -r cpu_perc mem_usage block_io <<< "$stats"
                printf "%-20s %-12s %-12s %-12s %s\\n" "$instance" "$cpu_perc" "$mem_usage" "$block_io" "running"
            else
                printf "%-20s %-12s %-12s %-12s %s\\n" "$instance" "N/A" "N/A" "N/A" "running"
            fi
        else
            printf "%-20s %-12s %-12s %-12s %s\\n" "$instance" "N/A" "N/A" "N/A" "stopped"
        fi
    done
    
    return 0
}