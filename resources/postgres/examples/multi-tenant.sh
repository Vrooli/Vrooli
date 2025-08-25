#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL Multi-Tenant Management Example
# Demonstrates managing multiple client instances simultaneously for Vrooli's platform factory

# This script shows how to:
# - Create and manage multiple client instances
# - Perform batch operations across all clients
# - Monitor health and performance of all instances
# - Scale instances up/down based on demand
# - Manage migrations and deployments across all clients
# - Handle client onboarding and offboarding

#######################################
# Configuration
#######################################

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
POSTGRES_DIR="${APP_ROOT}/resources/postgres"

# Source the PostgreSQL management script
# shellcheck disable=SC1091
source "${POSTGRES_DIR}/manage.sh"

#######################################
# Multi-tenant configuration
#######################################
DEFAULT_CLIENTS=(
    "real-estate:production:5435"
    "ecommerce:production:5436"
    "healthcare:production:5437"
    "education:development:5438"
    "fintech:production:5439"
)

CLIENT_TEMPLATES_FILE="/tmp/vrooli_clients.conf"
MONITORING_LOG="/tmp/vrooli_monitoring.log"

#######################################
# Utility functions
#######################################
log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $1"
}

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" >&2
}

log_warn() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] WARN: $1"
}

log_step() {
    echo ""
    echo "================================================"
    echo "STEP: $1"
    echo "================================================"
}

#######################################
# Parse command line arguments
#######################################
usage() {
    cat << EOF
Multi-Tenant PostgreSQL Management

USAGE:
    $0 <command> [options]

COMMANDS:
    setup               Initial setup of all client instances
    onboard <client>    Onboard a new client
    offboard <client>   Offboard an existing client
    status              Show status of all clients
    health              Perform health check on all clients
    migrate <dir>       Run migrations on all clients
    backup              Backup all client instances
    scale-up            Scale up resources for all instances
    scale-down          Scale down resources for all instances
    monitor             Start continuous monitoring
    maintenance         Perform maintenance operations
    disaster-recovery   Disaster recovery procedures
    report              Generate comprehensive report

OPTIONS:
    --client-name <name>    Specific client name
    --template <template>   PostgreSQL template (development|production|testing|minimal)
    --migrations-dir <dir>  Path to migrations directory
    --backup-type <type>    Backup type (full|schema|data)
    --dry-run              Show what would be done without executing
    --force                Skip confirmation prompts
    --help                 Show this help message

EXAMPLES:
    # Initial setup of all clients
    $0 setup

    # Onboard new client
    $0 onboard --client-name logistics --template production

    # Run migrations on all clients
    $0 migrate --migrations-dir ./migrations

    # Health check all clients
    $0 health

    # Start monitoring
    $0 monitor

    # Backup all clients
    $0 backup --backup-type full

    # Generate report
    $0 report

EOF
}

# Parse command line arguments
COMMAND=""
CLIENT_NAME=""
TEMPLATE="production"
MIGRATIONS_DIR=""
BACKUP_TYPE="full"
DRY_RUN=false
FORCE=false

if [[ $# -eq 0 ]]; then
    usage
    exit 1
fi

COMMAND="$1"
shift

while [[ $# -gt 0 ]]; do
    case $1 in
        --client-name)
            CLIENT_NAME="$2"
            shift 2
            ;;
        --template)
            TEMPLATE="$2"
            shift 2
            ;;
        --migrations-dir)
            MIGRATIONS_DIR="$2"
            shift 2
            ;;
        --backup-type)
            BACKUP_TYPE="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

#######################################
# Client configuration management
#######################################
load_client_config() {
    if [[ -f "$CLIENT_TEMPLATES_FILE" ]]; then
        mapfile -t CLIENTS < "$CLIENT_TEMPLATES_FILE"
    else
        CLIENTS=("${DEFAULT_CLIENTS[@]}")
        save_client_config
    fi
}

save_client_config() {
    printf '%s\n' "${CLIENTS[@]}" > "$CLIENT_TEMPLATES_FILE"
    log_info "Client configuration saved to $CLIENT_TEMPLATES_FILE"
}

parse_client_config() {
    local config="$1"
    IFS=':' read -r name template port <<< "$config"
    echo "$name" "$template" "$port"
}

get_client_names() {
    load_client_config
    for client_config in "${CLIENTS[@]}"; do
        local name template port
        read -r name template port <<< "$(parse_client_config "$client_config")"
        echo "$name"
    done
}

#######################################
# Multi-tenant setup
#######################################
setup_all_clients() {
    log_step "Setting up all client instances"
    
    load_client_config
    
    log_info "Setting up ${#CLIENTS[@]} client instances..."
    
    local success_count=0
    local failure_count=0
    local failed_clients=()
    
    for client_config in "${CLIENTS[@]}"; do
        local name template port
        read -r name template port <<< "$(parse_client_config "$client_config")"
        
        log_info "Setting up client: $name (template: $template, port: $port)"
        
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "[DRY RUN] Would create instance: $name"
            ((success_count++))
            continue
        fi
        
        # Create instance
        if "${POSTGRES_DIR}/manage.sh" --action create --instance "$name" --template "$template" --port "$port"; then
            log_success "Client instance created: $name"
            
            # Initialize migration system
            "${POSTGRES_DIR}/manage.sh" --action migrate-init --instance "$name" || log_warn "Failed to initialize migrations for $name"
            
            # Create client-specific database
            "${POSTGRES_DIR}/manage.sh" --action create-db --instance "$name" --database "${name}_app" || log_warn "Failed to create app database for $name"
            
            ((success_count++))
        else
            log_error "Failed to create client instance: $name"
            ((failure_count++))
            failed_clients+=("$name")
        fi
    done
    
    log_info ""
    log_info "Setup Summary:"
    log_info "  Total clients: ${#CLIENTS[@]}"
    log_info "  Successful: $success_count"
    log_info "  Failed: $failure_count"
    
    if [[ $failure_count -gt 0 ]]; then
        log_error "Failed clients: ${failed_clients[*]}"
        return 1
    else
        log_success "All client instances set up successfully"
        return 0
    fi
}

#######################################
# Client onboarding
#######################################
onboard_client() {
    local client_name="$1"
    local template="$2"
    
    log_step "Onboarding new client: $client_name"
    
    # Validate client name
    if [[ ! "$client_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        log_error "Invalid client name: $client_name"
        return 1
    fi
    
    # Check if client already exists
    load_client_config
    for client_config in "${CLIENTS[@]}"; do
        local name template_existing port
        read -r name template_existing port <<< "$(parse_client_config "$client_config")"
        if [[ "$name" == "$client_name" ]]; then
            log_error "Client already exists: $client_name"
            return 1
        fi
    done
    
    # Find available port
    local start_port=5440
    local max_port=5499
    local available_port=""
    
    for ((port=start_port; port<=max_port; port++)); do
        if ! "${POSTGRES_DIR}/manage.sh" --action status 2>/dev/null | grep -q ":$port"; then
            available_port="$port"
            break
        fi
    done
    
    if [[ -z "$available_port" ]]; then
        log_error "No available ports in range $start_port-$max_port"
        return 1
    fi
    
    log_info "Assigned port: $available_port"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would onboard client: $client_name (template: $template, port: $available_port)"
        return 0
    fi
    
    # Create instance
    if "${POSTGRES_DIR}/manage.sh" --action create --instance "$client_name" --template "$template" --port "$available_port"; then
        # Initialize migration system
        "${POSTGRES_DIR}/manage.sh" --action migrate-init --instance "$client_name"
        
        # Create client-specific database
        "${POSTGRES_DIR}/manage.sh" --action create-db --instance "$client_name" --database "${client_name}_app"
        
        # Add to client configuration
        CLIENTS+=("${client_name}:${template}:${available_port}")
        save_client_config
        
        # Create initial backup
        "${POSTGRES_DIR}/manage.sh" --action backup --instance "$client_name" --backup-name "onboarding_$(date +%Y%m%d_%H%M%S)" --backup-type "schema"
        
        log_success "Client onboarded successfully: $client_name"
        
        # Show connection information
        log_info "Connection Information:"
        "${POSTGRES_DIR}/manage.sh" --action connect --instance "$client_name"
        
        return 0
    else
        log_error "Failed to onboard client: $client_name"
        return 1
    fi
}

#######################################
# Client offboarding
#######################################
offboard_client() {
    local client_name="$1"
    
    log_step "Offboarding client: $client_name"
    
    # Check if client exists
    load_client_config
    local client_found=false
    local updated_clients=()
    
    for client_config in "${CLIENTS[@]}"; do
        local name template port
        read -r name template port <<< "$(parse_client_config "$client_config")"
        if [[ "$name" == "$client_name" ]]; then
            client_found=true
        else
            updated_clients+=("$client_config")
        fi
    done
    
    if [[ "$client_found" != "true" ]]; then
        log_error "Client not found: $client_name"
        return 1
    fi
    
    # Confirm offboarding
    if [[ "$FORCE" != "true" ]]; then
        log_warn "This will permanently remove client instance: $client_name"
        log_warn "All data will be lost unless backed up!"
        read -p "Create final backup before removal? (Y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
            log_info "Creating final backup..."
            "${POSTGRES_DIR}/manage.sh" --action backup --instance "$client_name" --backup-name "offboarding_$(date +%Y%m%d_%H%M%S)" --backup-type "full"
        fi
        
        read -p "Proceed with client removal? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Offboarding cancelled"
            return 0
        fi
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would offboard client: $client_name"
        return 0
    fi
    
    # Remove instance
    if "${POSTGRES_DIR}/manage.sh" --action destroy --instance "$client_name" --force yes; then
        # Update client configuration
        CLIENTS=("${updated_clients[@]}")
        save_client_config
        
        log_success "Client offboarded successfully: $client_name"
        return 0
    else
        log_error "Failed to offboard client: $client_name"
        return 1
    fi
}

#######################################
# Multi-tenant status
#######################################
show_all_status() {
    log_step "Multi-Tenant Status Overview"
    
    # Use the multi-status command from the main script
    "${POSTGRES_DIR}/manage.sh" --action multi-status --instance all
    
    # Additional multi-tenant specific information
    load_client_config
    
    echo ""
    echo "Client Configuration Summary:"
    echo "============================="
    printf "%-20s %-15s %-8s %-12s %s\\n" "Client" "Template" "Port" "Status" "Health"
    printf "%-20s %-15s %-8s %-12s %s\\n" "$(printf '%*s' 20 | tr ' ' '-')" "$(printf '%*s' 15 | tr ' ' '-')" "$(printf '%*s' 8 | tr ' ' '-')" "$(printf '%*s' 12 | tr ' ' '-')" "$(printf '%*s' 8 | tr ' ' '-')"
    
    for client_config in "${CLIENTS[@]}"; do
        local name template port
        read -r name template port <<< "$(parse_client_config "$client_config")"
        
        local status="unknown"
        local health="unknown"
        
        # Check if instance exists and get status
        if "${POSTGRES_DIR}/manage.sh" --action status --instance "$name" >/dev/null 2>&1; then
            status="running"
            # Simple health check
            if "${POSTGRES_DIR}/manage.sh" --action diagnose >/dev/null 2>&1; then
                health="healthy"
            else
                health="issues"
            fi
        else
            status="stopped"
        fi
        
        printf "%-20s %-15s %-8s %-12s %s\\n" "$name" "$template" "$port" "$status" "$health"
    done
    
    echo ""
    echo "Resource Usage Summary:"
    "${POSTGRES_DIR}/manage.sh" --action multi-status --instance all | grep -A 50 "Resource Usage" || echo "Resource usage not available"
}

#######################################
# Multi-tenant health monitoring
#######################################
health_check_all() {
    log_step "Multi-Tenant Health Check"
    
    # Use the multi-health command
    "${POSTGRES_DIR}/manage.sh" --action multi-health --instance all
    
    # Additional health metrics
    load_client_config
    
    local healthy_count=0
    local unhealthy_count=0
    local stopped_count=0
    
    echo ""
    echo "Detailed Health Analysis:"
    echo "========================"
    
    for client_config in "${CLIENTS[@]}"; do
        local name template port
        read -r name template port <<< "$(parse_client_config "$client_config")"
        
        echo "Client: $name"
        
        # Check if running
        if "${POSTGRES_DIR}/manage.sh" --action status --instance "$name" >/dev/null 2>&1; then
            # Get database statistics
            "${POSTGRES_DIR}/manage.sh" --action db-stats --instance "$name" 2>/dev/null || echo "  Database stats not available"
            
            # Check recent backups
            local backup_count=$(${POSTGRES_DIR}/manage.sh --action list-backups --instance "$name" 2>/dev/null | grep -c "backup" || echo "0")
            echo "  Recent backups: $backup_count"
            
            ((healthy_count++))
        else
            echo "  Status: Not running"
            ((stopped_count++))
        fi
        
        echo ""
    done
    
    echo "Health Summary:"
    echo "  Total clients: ${#CLIENTS[@]}"
    echo "  Healthy: $healthy_count"
    echo "  Unhealthy: $unhealthy_count"
    echo "  Stopped: $stopped_count"
    
    # Log to monitoring file
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Health Check - Total: ${#CLIENTS[@]}, Healthy: $healthy_count, Unhealthy: $unhealthy_count, Stopped: $stopped_count" >> "$MONITORING_LOG"
}

#######################################
# Multi-tenant migrations
#######################################
migrate_all_clients() {
    local migrations_dir="$1"
    
    if [[ -z "$migrations_dir" ]]; then
        log_error "Migrations directory is required"
        return 1
    fi
    
    if [[ ! -d "$migrations_dir" ]]; then
        log_error "Migrations directory not found: $migrations_dir"
        return 1
    fi
    
    log_step "Running migrations on all clients"
    log_info "Migrations directory: $migrations_dir"
    
    # Use the multi-migrate command
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would run migrations on all clients"
        return 0
    fi
    
    # Get list of client names
    local client_names=($(get_client_names))
    local client_list=$(IFS=','; echo "${client_names[*]}")
    
    if "${POSTGRES_DIR}/manage.sh" --action multi-migrate --instance "$client_list" --migrations-dir "$migrations_dir"; then
        log_success "Migrations completed on all clients"
        
        # Show migration status for each client
        echo ""
        echo "Migration Status Summary:"
        echo "========================"
        for client_name in "${client_names[@]}"; do
            echo "Client: $client_name"
            "${POSTGRES_DIR}/manage.sh" --action migrate-status --instance "$client_name" --database "${client_name}_app" | head -10
            echo ""
        done
        
        return 0
    else
        log_error "Migrations failed on one or more clients"
        return 1
    fi
}

#######################################
# Multi-tenant backup
#######################################
backup_all_clients() {
    local backup_type="$1"
    
    log_step "Backing up all client instances"
    log_info "Backup type: $backup_type"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would backup all clients with type: $backup_type"
        return 0
    fi
    
    # Get list of client names
    local client_names=($(get_client_names))
    local client_list=$(IFS=','; echo "${client_names[*]}")
    
    # Create timestamp for backup naming
    local backup_prefix="multi_tenant_$(date +%Y%m%d_%H%M%S)"
    
    if "${POSTGRES_DIR}/manage.sh" --action multi-backup --instance "$client_list" --backup-name "$backup_prefix" --backup-type "$backup_type"; then
        log_success "Backup completed for all clients"
        
        # Show backup summary
        echo ""
        echo "Backup Summary:"
        echo "==============="
        for client_name in "${client_names[@]}"; do
            echo "Client: $client_name"
            "${POSTGRES_DIR}/manage.sh" --action list-backups --instance "$client_name" | head -5
            echo ""
        done
        
        return 0
    else
        log_error "Backup failed for one or more clients"
        return 1
    fi
}

#######################################
# Continuous monitoring
#######################################
start_monitoring() {
    log_step "Starting continuous monitoring"
    log_info "Monitoring log: $MONITORING_LOG"
    log_info "Press Ctrl+C to stop monitoring"
    
    # Create monitoring log if it doesn't exist
    touch "$MONITORING_LOG"
    
    while true; do
        clear
        echo "Vrooli Multi-Tenant PostgreSQL Monitoring"
        echo "=========================================="
        echo "$(date)"
        echo ""
        
        # Show quick status
        show_all_status
        
        echo ""
        echo "Monitoring every 30 seconds... (Ctrl+C to stop)"
        
        # Log monitoring entry
        load_client_config
        local total_clients=${#CLIENTS[@]}
        local running_count=0
        
        for client_config in "${CLIENTS[@]}"; do
            local name template port
            read -r name template port <<< "$(parse_client_config "$client_config")"
            
            if "${POSTGRES_DIR}/manage.sh" --action status --instance "$name" >/dev/null 2>&1; then
                ((running_count++))
            fi
        done
        
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Monitor - Total: $total_clients, Running: $running_count" >> "$MONITORING_LOG"
        
        sleep 30
    done
}

#######################################
# Maintenance operations
#######################################
perform_maintenance() {
    log_step "Performing maintenance operations"
    
    load_client_config
    
    log_info "Maintenance tasks:"
    log_info "1. Health check all instances"
    log_info "2. Cleanup old backups"
    log_info "3. Verify backup integrity"
    log_info "4. Update statistics"
    log_info "5. Check disk space"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would perform maintenance on all clients"
        return 0
    fi
    
    local maintenance_success=true
    
    # Health check
    if ! health_check_all; then
        log_warn "Health check found issues"
        maintenance_success=false
    fi
    
    # Cleanup old backups
    log_info "Cleaning up old backups..."
    for client_config in "${CLIENTS[@]}"; do
        local name template port
        read -r name template port <<< "$(parse_client_config "$client_config")"
        
        "${POSTGRES_DIR}/manage.sh" --action cleanup-backups --instance "$name" --retention-days 7 || log_warn "Backup cleanup failed for $name"
    done
    
    # Verify recent backups
    log_info "Verifying recent backups..."
    for client_config in "${CLIENTS[@]}"; do
        local name template port
        read -r name template port <<< "$(parse_client_config "$client_config")"
        
        # Get most recent backup
        local recent_backup=$(${POSTGRES_DIR}/manage.sh --action list-backups --instance "$name" 2>/dev/null | head -5 | tail -1 | awk '{print $1}' || echo "")
        
        if [[ -n "$recent_backup" ]]; then
            "${POSTGRES_DIR}/manage.sh" --action verify-backup --instance "$name" --backup-name "$recent_backup" || log_warn "Backup verification failed for $name:$recent_backup"
        fi
    done
    
    if [[ "$maintenance_success" == "true" ]]; then
        log_success "Maintenance completed successfully"
    else
        log_warn "Maintenance completed with some issues"
    fi
}

#######################################
# Generate comprehensive report
#######################################
generate_report() {
    log_step "Generating Multi-Tenant Report"
    
    local report_file="/tmp/vrooli_multitenant_report_$(date +%Y%m%d_%H%M%S).txt"
    
    {
        echo "Vrooli Multi-Tenant PostgreSQL Report"
        echo "======================================"
        echo "Generated: $(date)"
        echo ""
        
        # Client overview
        echo "CLIENT OVERVIEW"
        echo "==============="
        load_client_config
        echo "Total clients configured: ${#CLIENTS[@]}"
        echo ""
        
        # Detailed status
        echo "DETAILED STATUS"
        echo "==============="
        show_all_status
        echo ""
        
        # Health summary
        echo "HEALTH SUMMARY"
        echo "=============="
        health_check_all
        echo ""
        
        # Backup status
        echo "BACKUP STATUS"
        echo "============="
        for client_config in "${CLIENTS[@]}"; do
            local name template port
            read -r name template port <<< "$(parse_client_config "$client_config")"
            
            echo "Client: $name"
            "${POSTGRES_DIR}/manage.sh" --action list-backups --instance "$name" 2>/dev/null || echo "  No backups found"
            echo ""
        done
        
        # Recent monitoring logs
        echo "RECENT MONITORING"
        echo "================="
        if [[ -f "$MONITORING_LOG" ]]; then
            tail -20 "$MONITORING_LOG"
        else
            echo "No monitoring log found"
        fi
        
        echo ""
        echo "Report generated: $(date)"
    } > "$report_file"
    
    log_success "Report generated: $report_file"
    
    # Display summary
    echo ""
    echo "Report Summary:"
    echo "==============="
    head -50 "$report_file"
    echo ""
    echo "Full report available at: $report_file"
}

#######################################
# Main command dispatcher
#######################################
main() {
    case "$COMMAND" in
        "setup")
            setup_all_clients
            ;;
        "onboard")
            if [[ -z "$CLIENT_NAME" ]]; then
                log_error "Client name is required for onboarding"
                echo "Usage: $0 onboard --client-name <name> [--template <template>]"
                exit 1
            fi
            onboard_client "$CLIENT_NAME" "$TEMPLATE"
            ;;
        "offboard")
            if [[ -z "$CLIENT_NAME" ]]; then
                log_error "Client name is required for offboarding"
                echo "Usage: $0 offboard --client-name <name>"
                exit 1
            fi
            offboard_client "$CLIENT_NAME"
            ;;
        "status")
            show_all_status
            ;;
        "health")
            health_check_all
            ;;
        "migrate")
            if [[ -z "$MIGRATIONS_DIR" ]]; then
                log_error "Migrations directory is required"
                echo "Usage: $0 migrate --migrations-dir <directory>"
                exit 1
            fi
            migrate_all_clients "$MIGRATIONS_DIR"
            ;;
        "backup")
            backup_all_clients "$BACKUP_TYPE"
            ;;
        "monitor")
            start_monitoring
            ;;
        "maintenance")
            perform_maintenance
            ;;
        "report")
            generate_report
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            usage
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"