#!/usr/bin/env bash
# Huginn Installation and Setup Functions
# Installation, uninstallation, and configuration management

#######################################
# Install Huginn
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::install() {
    huginn::show_installing
    
    # Pre-installation checks
    if ! huginn::pre_install_checks; then
        return 1
    fi
    
    # Handle existing installation
    if huginn::is_installed; then
        if [[ "${FORCE:-no}" != "yes" ]]; then
            huginn::show_already_installed
            return 1
        else
            log::info "🔄 Forcing reinstallation..."
            huginn::uninstall_internal
        fi
    fi
    
    # Clean up any existing resources
    huginn::cleanup_old_resources
    
    # Create necessary directories
    if ! huginn::ensure_data_directories; then
        return 1
    fi
    
    # Create Docker network
    if ! huginn::ensure_network; then
        return 1
    fi
    
    # Pull required images
    if ! huginn::pull_images; then
        return 1
    fi
    
    # Start the services
    if ! huginn::start; then
        log::error "Failed to start Huginn services"
        return 1
    fi
    
    # Post-installation setup
    if ! huginn::post_install_setup; then
        return 1
    fi
    
    # Update Vrooli configuration
    huginn::update_vrooli_config
    
    huginn::show_installation_complete
    return 0
}

#######################################
# Uninstall Huginn
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::uninstall() {
    huginn::show_uninstalling
    huginn::uninstall_internal
    huginn::show_uninstall_complete
    return 0
}

#######################################
# Internal uninstall function (no messages)
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::uninstall_internal() {
    # Stop containers
    huginn::stop >/dev/null 2>&1 || true
    
    # Set removal flag for cleanup
    export REMOVE_VOLUMES="${REMOVE_VOLUMES:-no}"
    
    # Clean up Docker resources
    huginn::cleanup_docker_resources
    
    # Remove data directories if requested
    if [[ "${REMOVE_DATA:-no}" == "yes" ]]; then
        log::info "Removing data directories..."
        rm -rf "$HUGINN_DATA_DIR" 2>/dev/null || true
    fi
    
    return 0
}

#######################################
# Pre-installation checks
# Returns: 0 if all checks pass, 1 otherwise
#######################################
huginn::pre_install_checks() {
    # Check Docker availability
    if ! huginn::check_docker; then
        return 1
    fi
    
    # Check if port is available
    if resources::is_service_running "$HUGINN_PORT"; then
        log::error "Port $HUGINN_PORT is already in use"
        log::info "Stop the service using that port or set HUGINN_CUSTOM_PORT"
        return 1
    fi
    
    # Check disk space (require at least 2GB free)
    local available_space
    available_space=$(df "${HOME}" | awk 'NR==2 {print $4}')
    if [[ $available_space -lt 2097152 ]]; then  # 2GB in KB
        log::warn "⚠️  Low disk space detected (less than 2GB available)"
        log::info "Huginn requires at least 2GB of free space"
        if [[ "${YES:-no}" != "yes" ]]; then
            log::info "Continue installation? (y/N)"
            read -r response
            if [[ "$response" != "y" && "$response" != "Y" ]]; then
                return 1
            fi
        fi
    fi
    
    # Check system resources
    huginn::check_system_resources
    
    return 0
}

#######################################
# Pull required Docker images
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::pull_images() {
    log::info "📦 Pulling Docker images..."
    
    local images=("$HUGINN_IMAGE" "$POSTGRES_IMAGE")
    
    for image in "${images[@]}"; do
        log::info "Pulling $image..."
        if ! docker pull "$image" >/dev/null 2>&1; then
            log::error "Failed to pull image: $image"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Post-installation setup
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::post_install_setup() {
    log::info "🔧 Running post-installation setup..."
    
    # Wait for the application to be fully ready
    sleep 5
    
    # Verify database connection
    if ! huginn::check_database; then
        log::error "Database connection verification failed"
        return 1
    fi
    
    # Install default scenarios if available
    huginn::install_default_scenarios
    
    # Set up basic monitoring
    huginn::setup_basic_monitoring
    
    return 0
}

#######################################
# Install default scenarios from examples
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::install_default_scenarios() {
    local scenarios_dir="${SCRIPT_DIR}/examples/scenarios"
    
    if [[ ! -d "$scenarios_dir" ]]; then
        log::info "No default scenarios found to install"
        return 0
    fi
    
    log::info "📋 Installing default scenarios..."
    
    local installed_count=0
    
    for scenario_file in "$scenarios_dir"/*.json; do
        if [[ -f "$scenario_file" ]]; then
            local scenario_name
            scenario_name=$(basename "$scenario_file" .json)
            
            log::info "Installing scenario: $scenario_name"
            
            if huginn::import_scenario_file "$scenario_file"; then
                installed_count=$((installed_count + 1))
            else
                log::warn "Failed to install scenario: $scenario_name"
            fi
        fi
    done
    
    if [[ $installed_count -gt 0 ]]; then
        log::success "✅ Installed $installed_count default scenarios"
    fi
    
    return 0
}

#######################################
# Set up basic system monitoring
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::setup_basic_monitoring() {
    log::info "🔍 Setting up basic monitoring..."
    
    # Create a system health monitoring agent
    local monitoring_agent_code='
    user = User.find_by(username: "admin") || User.first
    
    # Create system health monitor
    health_agent = Agents::JavaScriptAgent.new(
      name: "System Health Monitor",
      options: {
        "language" => "JavaScript",
        "code" => "
          var stats = {
            timestamp: new Date().toISOString(),
            container_status: \"healthy\",
            agents_count: this.options.agents_count || 0,
            events_count: this.options.events_count || 0,
            system_uptime: process.uptime()
          };
          this.createEvent(stats);
        ",
        "expected_receive_period_in_days" => 2
      },
      schedule: "every_1h"
    )
    health_agent.user = user
    
    if health_agent.save
      puts "✅ System health monitor created"
    else
      puts "❌ Failed to create system health monitor"
    end
    '
    
    huginn::rails_runner "$monitoring_agent_code" >/dev/null 2>&1
    
    return 0
}

#######################################
# Update Vrooli configuration
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::update_vrooli_config() {
    log::info "📝 Updating Vrooli configuration..."
    
    local config_data="{
        \"enabled\": true,
        \"baseUrl\": \"$HUGINN_BASE_URL\",
        \"port\": $HUGINN_PORT,
        \"version\": \"$(huginn::get_version)\",
        \"credentials\": {
            \"username\": \"$DEFAULT_ADMIN_USERNAME\",
            \"password\": \"$DEFAULT_ADMIN_PASSWORD\"
        },
        \"features\": {
            \"agents\": true,
            \"scenarios\": true,
            \"events\": true,
            \"webhooks\": true,
            \"integrations\": true
        },
        \"health\": {
            \"endpoint\": \"$HUGINN_BASE_URL\",
            \"timeout\": $HUGINN_HEALTH_CHECK_TIMEOUT
        }
    }"
    
    # Update configuration using common function
    resources::update_config "automation" "huginn" "$config_data"
    
    return 0
}

#######################################
# Check system resources
# Arguments: none
# Returns: 0 always (informational only)
#######################################
huginn::check_system_resources() {
    # Check memory
    local total_mem
    total_mem=$(free -m | awk 'NR==2{print $2}')
    
    if [[ $total_mem -lt 2048 ]]; then
        log::warn "⚠️  System has less than 2GB RAM ($total_mem MB)"
        log::info "Huginn may run slowly with limited memory"
    fi
    
    # Check CPU cores
    local cpu_cores
    cpu_cores=$(nproc)
    
    if [[ $cpu_cores -lt 2 ]]; then
        log::warn "⚠️  System has only $cpu_cores CPU core(s)"
        log::info "Multi-core system recommended for better performance"
    fi
    
    return 0
}

#######################################
# Import scenario from JSON file
# Arguments:
#   $1 - path to scenario JSON file
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::import_scenario_file() {
    local scenario_file="$1"
    
    if [[ ! -f "$scenario_file" ]]; then
        log::error "Scenario file not found: $scenario_file"
        return 1
    fi
    
    # Read and validate JSON
    local scenario_json
    if ! scenario_json=$(cat "$scenario_file" 2>/dev/null); then
        log::error "Failed to read scenario file: $scenario_file"
        return 1
    fi
    
    # Validate JSON format
    if ! echo "$scenario_json" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in scenario file: $scenario_file"
        return 1
    fi
    
    # Import scenario via Rails
    local import_code="
    scenario_data = '$scenario_json'
    
    begin
      data = JSON.parse(scenario_data)
      user = User.find_by(username: 'admin') || User.first
      
      # Create scenario
      scenario = Scenario.new(
        name: data['name'],
        description: data['description'],
        user: user
      )
      
      if scenario.save
        puts \"✅ Scenario imported: #{scenario.name}\"
      else
        puts \"❌ Failed to import scenario: #{scenario.errors.full_messages.join(', ')}\"
      end
    rescue => e
      puts \"❌ Import error: #{e.message}\"
    end
    "
    
    local result
    result=$(huginn::rails_runner "$import_code" 2>/dev/null)
    
    if [[ "$result" == *"✅"* ]]; then
        return 0
    else
        return 1
    fi
}