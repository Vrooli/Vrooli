#!/usr/bin/env bash
# Huginn User Messages and Help Text
# All user-facing messages, prompts, and documentation

#######################################
# Display usage information
#######################################
huginn::usage() {
    args::usage "$RESOURCE_DESC"
    echo
    echo "Examples:"
    echo "  # Install Huginn with default settings"
    echo "  $0 --action install"
    echo
    echo "  # Check system status and health"
    echo "  $0 --action status"
    echo
    echo "  # List all agents in the system"
    echo "  $0 --action agents --operation list"
    echo
    echo "  # Show details for specific agent"
    echo "  $0 --action agents --operation show --agent-id 5"
    echo
    echo "  # Run an agent manually"
    echo "  $0 --action agents --operation run --agent-id 10"
    echo
    echo "  # List all scenarios"
    echo "  $0 --action scenarios --operation list"
    echo
    echo "  # Export scenario as JSON"
    echo "  $0 --action scenarios --operation export --scenario-id 2"
    echo
    echo "  # View recent events"
    echo "  $0 --action events --operation recent --count 20"
    echo
    echo "  # Create system backup"
    echo "  $0 --action backup"
    echo
    echo "  # Test cross-resource integration"
    echo "  $0 --action integration --operation test"
    echo
}

#######################################
# Installation messages
#######################################
huginn::show_installing() {
    log::header "🤖 Installing Huginn Automation Platform"
    log::info "Setting up agent-based workflow automation system..."
}

huginn::show_installation_complete() {
    log::success "✅ Huginn installation completed successfully"
    echo
    log::info "🌐 Web Interface: $HUGINN_BASE_URL"
    log::info "👤 Username: $DEFAULT_ADMIN_USERNAME"
    log::info "🔑 Password: $DEFAULT_ADMIN_PASSWORD"
    echo
    log::info "📚 Quick Start:"
    log::info "   • View system status: $0 --action status"
    log::info "   • List agents: $0 --action agents --operation list"
    log::info "   • Create your first automation workflow in the web interface"
    echo
}

huginn::show_uninstalling() {
    log::header "🗑️  Uninstalling Huginn"
    log::info "Removing containers, volumes, and data..."
}

huginn::show_uninstall_complete() {
    log::success "✅ Huginn uninstalled successfully"
    log::info "All containers, volumes, and network resources have been removed"
}

#######################################
# Status and health messages
#######################################
huginn::show_status_header() {
    log::header "🤖 Huginn System Status"
}

huginn::show_waiting_message() {
    log::info "⏳ Waiting for Huginn to be ready..."
    log::info "   This may take 30-60 seconds for first startup"
}

huginn::show_health_check_failed() {
    log::error "❌ Huginn health check failed"
    log::info "   Check logs with: $0 --action logs"
}

huginn::show_not_installed() {
    log::warn "⚠️  Huginn is not installed"
    log::info "   Run: $0 --action install"
}

huginn::show_not_running() {
    log::warn "⚠️  Huginn is not running"
    log::info "   Start with: $0 --action start"
}

huginn::show_already_installed() {
    log::warn "⚠️  Huginn is already installed"
    log::info "   Use --force yes to reinstall"
}

#######################################
# Operation messages
#######################################
huginn::show_starting() {
    log::info "🚀 Starting Huginn containers..."
}

huginn::show_stopping() {
    log::info "⏹️  Stopping Huginn containers..."
}

huginn::show_restarting() {
    log::info "🔄 Restarting Huginn containers..."
}

#######################################
# Agent management messages
#######################################
huginn::show_agents_header() {
    log::header "🤖 Huginn Agent Management"
}

huginn::show_agent_not_found() {
    local agent_id="$1"
    log::error "❌ Agent with ID '$agent_id' not found"
}

huginn::show_agent_execution_success() {
    local agent_name="$1"
    local event_count="$2"
    log::success "✅ Agent '$agent_name' executed successfully"
    log::info "📊 Generated $event_count total events"
}

huginn::show_agent_execution_failed() {
    local agent_name="$1"
    local error="$2"
    log::error "❌ Agent '$agent_name' execution failed"
    log::info "   Error: $error"
}

#######################################
# Scenario management messages
#######################################
huginn::show_scenarios_header() {
    log::header "📂 Huginn Scenario Management"
}

huginn::show_scenario_not_found() {
    local scenario_id="$1"
    log::error "❌ Scenario with ID '$scenario_id' not found"
}

#######################################
# Integration messages
#######################################
huginn::show_integration_header() {
    log::header "🔗 Huginn Integration Status"
}

huginn::show_integration_test_success() {
    log::success "✅ Cross-resource integration test passed"
}

huginn::show_integration_test_failed() {
    log::error "❌ Cross-resource integration test failed"
}

#######################################
# Backup messages
#######################################
huginn::show_backup_header() {
    log::header "💾 Huginn System Backup"
}

huginn::show_backup_complete() {
    local backup_file="$1"
    local backup_size="$2"
    log::success "✅ Backup completed successfully"
    log::info "📁 File: $backup_file"
    log::info "📊 Size: $backup_size"
}

#######################################
# Error messages
#######################################
huginn::show_docker_error() {
    log::error "❌ Docker is not available or not running"
    log::info "   Please ensure Docker is installed and running"
}

huginn::show_network_error() {
    log::error "❌ Failed to create Docker network"
    log::info "   Check Docker permissions and network configuration"
}

huginn::show_database_error() {
    log::error "❌ Database connection failed"
    log::info "   Check PostgreSQL container status and logs"
}

huginn::show_rails_error() {
    log::error "❌ Rails application failed to start"
    log::info "   Check Huginn container logs for detailed error information"
}

#######################################
# Interrupt handling
#######################################
huginn::show_interrupt_message() {
    echo
    log::info "🛑 Huginn operation interrupted by user"
    log::info "   System may be in an incomplete state"
    log::info "   Run status check: $0 --action status"
}