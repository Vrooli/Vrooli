#!/usr/bin/env bash
################################################################################
# n8n Resource CLI - v2.0 Universal Contract Compliant
# 
# Workflow automation platform with visual editor and 400+ integrations
#
# Usage:
#   resource-n8n <command> [options]
#   resource-n8n <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    N8N_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${N8N_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
N8N_CLI_DIR="${APP_ROOT}/resources/n8n"

# Source standard variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source v2.0 CLI Command Framework
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source n8n configuration
# shellcheck disable=SC1091
source "${N8N_CLI_DIR}/config/defaults.sh"
n8n::export_config 2>/dev/null || true

# Source n8n libraries (only what exists)
for lib in core docker status health api inject auto-credentials credential-registry content recovery test secrets; do
    lib_file="${N8N_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        if ! source "$lib_file" 2>/dev/null; then
            log::warn "Failed to source ${lib}.sh - some functions may not be available"
        fi
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "n8n" "n8n workflow automation platform" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="n8n::install_with_backup"
CLI_COMMAND_HANDLERS["manage::uninstall"]="n8n::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="n8n::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="n8n::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="n8n::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="n8n::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="n8n::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="n8n::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="n8n::test::all"

# Content handlers - n8n workflow/credential management
CLI_COMMAND_HANDLERS["content::add"]="n8n::content::add"
CLI_COMMAND_HANDLERS["content::list"]="n8n::content::list"
CLI_COMMAND_HANDLERS["content::get"]="n8n::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="n8n::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="n8n::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed n8n status" "n8n::status"
cli::register_command "logs" "Show n8n logs" "n8n::logs"

# ==============================================================================
# N8N-SPECIFIC COMMANDS - Critical workflow automation functionality
# ==============================================================================

# Workflow management commands - CRITICAL for n8n
cli::register_subcommand "content" "inject" "Inject workflow/credentials from file" "n8n::cli_inject" "modifies-system"
cli::register_subcommand "content" "activate" "Activate workflow by ID" "n8n::cli_activate_workflow" "modifies-system"
cli::register_subcommand "content" "deactivate" "Deactivate workflow by ID" "n8n::cli_deactivate_workflow" "modifies-system"
cli::register_subcommand "content" "activate-all" "Activate all workflows" "n8n::activate_all_workflows" "modifies-system"
cli::register_subcommand "content" "activate-pattern" "Activate workflows by pattern" "n8n::cli_activate_pattern" "modifies-system"
cli::register_subcommand "content" "delete" "Delete workflow by ID" "n8n::cli_delete_workflow" "modifies-system"
cli::register_subcommand "content" "delete-all" "Delete all workflows" "n8n::delete_all_workflows" "modifies-system"
cli::register_subcommand "content" "workflows" "List all workflows" "n8n::list_workflows"

# Credential management commands - CRITICAL for n8n
cli::register_subcommand "content" "auto-credentials" "Auto-discover and create credentials" "n8n::auto_manage_credentials" "modifies-system"
cli::register_subcommand "content" "list-credentials" "List existing n8n credentials" "n8n::validate_auto_credentials"
cli::register_subcommand "content" "list-discoverable" "List discoverable resources" "n8n::list_discoverable_resources"
cli::register_subcommand "content" "credential-registry" "Show credential registry info" "n8n::cli_credential_registry"

# Information and integration commands
cli::register_command "credentials" "Show n8n credentials for integration" "n8n::cli_credentials"

# Memory monitoring commands - CRITICAL for preventing OOM crashes
cli::register_subcommand "test" "memory" "Check memory health and usage" "n8n::cli_memory_check"
cli::register_command "memory" "Memory monitoring and alerts" "n8n::cli_memory_status"

# Backup and recovery commands - CRITICAL for data protection
cli::register_subcommand "manage" "create-backup" "Create n8n backup with API key" "n8n::cli_create_backup" "modifies-system"
cli::register_subcommand "manage" "restore-backup" "Restore from backup" "n8n::cli_restore_backup" "modifies-system"
cli::register_subcommand "manage" "safety-check" "Pre-deployment safety analysis" "n8n::cli_safety_check"
cli::register_subcommand "manage" "migrate-volume" "Migrate to named Docker volume" "n8n::cli_migrate_volume" "modifies-system"
cli::register_command "memory-start" "Start memory monitoring" "n8n::cli_memory_start"
cli::register_command "memory-stop" "Stop memory monitoring" "n8n::cli_memory_stop"

# ==============================================================================
# CLI WRAPPER FUNCTIONS - For commands requiring argument handling
# ==============================================================================

# Inject workflow or credentials into n8n
n8n::cli_inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-n8n content inject <file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-n8n content inject workflow.json"
        echo "  resource-n8n content inject shared:initialization/automation/n8n/workflow.json"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${var_VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Use n8n injection framework
    if command -v n8n::inject &>/dev/null; then
        n8n::inject "$file"
    elif command -v n8n::main &>/dev/null; then
        n8n::main "$file"
    else
        log::error "n8n injection functions not available"
        return 1
    fi
}

# Activate a single workflow
n8n::cli_activate_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: resource-n8n content activate <workflow_id>"
        echo ""
        echo "To find workflow IDs, run: resource-n8n content workflows"
        return 1
    fi
    
    n8n::activate_workflow "$workflow_id"
}

# Deactivate a single workflow
n8n::cli_deactivate_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: resource-n8n content deactivate <workflow_id>"
        echo ""
        echo "To find workflow IDs, run: resource-n8n content workflows"
        return 1
    fi
    
    n8n::deactivate_workflow "$workflow_id"
}

# Activate workflows by pattern
n8n::cli_activate_pattern() {
    local pattern="${1:-}"
    
    if [[ -z "$pattern" ]]; then
        log::error "Pattern is required"
        echo "Usage: resource-n8n content activate-pattern <pattern>"
        echo ""
        echo "Examples:"
        echo "  resource-n8n content activate-pattern \"embedding*\""
        echo "  resource-n8n content activate-pattern \"*generator*\""
        echo "  resource-n8n content activate-pattern \"reasoning-chain\""
        return 1
    fi
    
    n8n::activate_workflows_by_pattern "$pattern"
}

# Delete workflow
n8n::cli_delete_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: resource-n8n content delete <workflow_id>"
        echo ""
        echo "Examples:"
        echo "  resource-n8n content delete wbu5QO9dVCD521pe"
        return 1
    fi
    
    n8n::delete_workflow "$workflow_id"
}

# Show credential registry statistics
n8n::cli_credential_registry() {
    local action="${1:-stats}"
    
    # Source the registry library
    # shellcheck disable=SC1091
    source "${N8N_CLI_DIR}/lib/credential-registry.sh"
    
    case "$action" in
        stats)
            credential_registry::stats
            ;;
        list)
            echo "📋 Registered Credentials:"
            local credentials
            credentials=$(credential_registry::list)
            if [[ -n "$credentials" ]]; then
                echo "$credentials" | jq -r '"  • \(.name) (\(.type)) - \(.resource) - ID: \(.id)"'
            else
                echo "  No credentials registered"
            fi
            ;;
        backup)
            credential_registry::backup
            ;;
        file)
            echo "Registry file: $(credential_registry::get_file_path)"
            ;;
        *)
            log::error "Unknown registry action: $action"
            echo "Available actions: stats, list, backup, file"
            return 1
            ;;
    esac
}

# Show credentials for n8n integration
n8n::cli_credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    credentials::parse_args "$@" || return $?
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "${N8N_CONTAINER_NAME:-n8n}")
    
    # Build connections array
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Get API key for n8n
        local api_key
        api_key=$(n8n::resolve_api_key 2>/dev/null || echo "")
        
        # n8n API connection
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${N8N_PORT:-5678}" \
            --arg path "/api/v1" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local auth_obj="{}"
        if [[ -n "$api_key" ]]; then
            auth_obj=$(jq -n \
                --arg header_name "X-N8N-API-KEY" \
                --arg header_value "$api_key" \
                '{
                    header_name: $header_name,
                    header_value: $header_value
                }')
        fi
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "n8n workflow automation API" \
            --arg web_url "http://localhost:${N8N_PORT:-5678}" \
            --arg api_version "v1" \
            '{
                description: $description,
                web_interface: $web_url,
                api_version: $api_version
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "api" \
            "n8n API" \
            "httpHeaderAuth" \
            "$connection_obj" \
            "$auth_obj" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "n8n" "$status" "$connections_array")
    
    credentials::format_output "$response"
}

# Memory monitoring CLI functions
n8n::cli_memory_check() {
    log::header "🧠 n8n Memory Health Check"
    
    # Source memory monitor
    # shellcheck disable=SC1091
    source "${N8N_CLI_DIR}/lib/memory-monitor.sh" 2>/dev/null || {
        log::error "Memory monitoring not available"
        return 1
    }
    
    local detailed_stats
    detailed_stats=$(n8n::memory::get_detailed_stats)
    
    if echo "$detailed_stats" | grep -q '"error"'; then
        log::error "Failed to get memory stats"
        echo "$detailed_stats" | jq -r '.error'
        return 1
    fi
    
    # Parse and display stats
    local usage_mb limit_mb percent
    usage_mb=$(echo "$detailed_stats" | jq -r '.usage_mb')
    limit_mb=$(echo "$detailed_stats" | jq -r '.limit_mb')
    percent=$(echo "$detailed_stats" | jq -r '.percent')
    
    echo ""
    echo "📊 Current Memory Status:"
    echo "   Usage: ${usage_mb}MB / ${limit_mb}MB (${percent}%)"
    echo ""
    echo "🎯 Thresholds:"
    echo "   Warning:   $(echo "$detailed_stats" | jq -r '.thresholds.warning')%"
    echo "   Critical:  $(echo "$detailed_stats" | jq -r '.thresholds.critical')%"
    echo "   Emergency: $(echo "$detailed_stats" | jq -r '.thresholds.emergency')%"
    echo ""
    
    # Check health
    local health_code
    n8n::check_memory_health
    health_code=$?
    
    case $health_code in
        0)
            log::success "✅ Memory usage is healthy"
            ;;
        1)
            log::warn "⚠️  Memory usage is approaching warning threshold"
            ;;
        2)
            log::error "🔴 Memory usage is critical - consider action soon"
            ;;
        3)
            log::error "🚨 EMERGENCY: Memory usage is dangerously high!"
            ;;
    esac
    
    return $health_code
}

n8n::cli_memory_status() {
    log::header "🧠 n8n Memory Monitor Status"
    
    # Source memory monitor
    # shellcheck disable=SC1091
    source "${N8N_CLI_DIR}/lib/memory-monitor.sh" 2>/dev/null || {
        log::error "Memory monitoring not available"
        return 1
    }
    
    local status
    status=$(n8n::memory::get_monitor_status)
    
    local is_running pid current_usage
    is_running=$(echo "$status" | jq -r '.monitor_running')
    pid=$(echo "$status" | jq -r '.monitor_pid // ""')
    current_usage=$(echo "$status" | jq -r '.current_usage')
    
    echo ""
    if [[ "$is_running" == "true" ]]; then
        log::success "✅ Memory monitor is running (PID: $pid)"
    else
        log::warn "⚠️  Memory monitor is not running"
        echo "   Start with: resource-n8n memory-start"
    fi
    
    echo ""
    echo "📊 Current Usage: ${current_usage}%"
    
    # Show recent history
    local history_count
    history_count=$(echo "$status" | jq '.history | length')
    if [[ "$history_count" -gt 0 ]]; then
        echo ""
        echo "📈 Recent History:"
        echo "$status" | jq -r '.history[]' | tail -5 | while read -r usage; do
            echo "   ${usage}%"
        done
    fi
}

n8n::cli_memory_start() {
    log::header "🚀 Starting n8n Memory Monitor"
    
    # Source memory monitor
    # shellcheck disable=SC1091
    source "${N8N_CLI_DIR}/lib/memory-monitor.sh" 2>/dev/null || {
        log::error "Memory monitoring not available"
        return 1
    }
    
    n8n::start_memory_monitoring
    local result=$?
    
    if [[ $result -eq 0 ]]; then
        log::success "✅ Memory monitor started successfully"
        echo ""
        echo "Monitor will check every ${N8N_MEM_CHECK_INTERVAL:-30} seconds"
        echo "View status with: resource-n8n memory"
    else
        log::error "Failed to start memory monitor"
    fi
    
    return $result
}

n8n::cli_memory_stop() {
    log::header "🛑 Stopping n8n Memory Monitor"
    
    # Source memory monitor
    # shellcheck disable=SC1091
    source "${N8N_CLI_DIR}/lib/memory-monitor.sh" 2>/dev/null || {
        log::error "Memory monitoring not available"
        return 1
    }
    
    n8n::memory::stop_monitor
    log::success "✅ Memory monitor stopped"
}

# Backup and recovery CLI handlers
n8n::cli_create_backup() {
    local label="${1:-manual}"
    log::header "🛡️ Creating n8n Backup"
    log::info "Creating n8n backup with label: $label"
    
    if backup_path=$(n8n::create_backup "$label"); then
        log::success "✅ Backup created successfully"
        log::info "Backup ID: $(basename "$backup_path")"
        echo "Backup created: $(basename "$backup_path")"
    else
        log::error "❌ Backup creation failed"
        return 1
    fi
}

n8n::cli_restore_backup() {
    local backup_id="${1:-}"
    if [[ -z "$backup_id" ]]; then
        log::header "📋 Available n8n Backups"
        if backup::list "n8n" 2>/dev/null; then
            echo ""
            log::error "Usage: resource-n8n manage restore-backup <backup-id>"
        else
            log::warn "No backups found. Create one with: resource-n8n manage create-backup"
        fi
        return 1
    fi
    
    log::header "🔄 n8n Backup Restore"
    log::warn "⚠️ This will replace current n8n data with backup: $backup_id"
    read -p "Continue? (y/N): " -r
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log::info "Restore cancelled"
        return 0
    fi
    
    n8n::recover_from_specific_backup "$backup_id"
}

# Deployment safety CLI handlers
n8n::cli_safety_check() {
    log::header "🛡️ n8n Deployment Safety Analysis"
    n8n::deployment_safety_check
}

n8n::cli_migrate_volume() {
    log::header "📦 n8n Volume Migration"
    n8n::migrate_to_named_volume
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi