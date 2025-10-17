#!/usr/bin/env bash
# Huginn Installation and Setup Functions
# Installation, uninstallation, and configuration management

# Source required utilities using unique directory variable
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HUGINN_LIB_DIR="${APP_ROOT}/resources/huginn/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

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
    
    # Auto-install CLI if available
    # shellcheck disable=SC1091
    "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "${APP_ROOT}/resources/huginn" 2>/dev/null || true
    
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
        trash::safe_remove "$HUGINN_DATA_DIR" --no-confirm 2>/dev/null || true
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
    
    # Only pull PostgreSQL image, Huginn will be built custom
    log::info "Pulling $POSTGRES_IMAGE..."
    if ! docker pull "$POSTGRES_IMAGE" >/dev/null 2>&1; then
        log::error "Failed to pull image: $POSTGRES_IMAGE"
        return 1
    fi
    
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
    
    # Register agent for tracking if agent tracking is available
    local tracking_agent_id=""
    if type -t agents::register &>/dev/null; then
        tracking_agent_id=$(agents::generate_id)
        local command_string="huginn::import_scenario_file $scenario_file"
        if agents::register "$tracking_agent_id" $$ "$command_string"; then
            huginn::setup_agent_cleanup "$tracking_agent_id"
        else
            tracking_agent_id=""
        fi
    fi
    
    # Read and validate JSON
    local scenario_json
    if ! scenario_json=$(cat "$scenario_file" 2>/dev/null); then
        # Cleanup tracking agent on error
        if [[ -n "$tracking_agent_id" ]] && type -t agents::unregister &>/dev/null; then
            agents::unregister "$tracking_agent_id" >/dev/null 2>&1
        fi
        log::error "Failed to read scenario file: $scenario_file"
        return 1
    fi
    
    # Validate JSON format
    if ! echo "$scenario_json" | jq . >/dev/null 2>&1; then
        # Cleanup tracking agent on error
        if [[ -n "$tracking_agent_id" ]] && type -t agents::unregister &>/dev/null; then
            agents::unregister "$tracking_agent_id" >/dev/null 2>&1
        fi
        log::error "Invalid JSON in scenario file: $scenario_file"
        return 1
    fi
    
    # Escape JSON for Ruby
    scenario_json=$(echo "$scenario_json" | sed "s/'/\\\\'/g")
    
    # Import scenario via Rails with complete agent reconstruction
    local import_code="
    require 'json'
    
    scenario_data = '$scenario_json'
    
    begin
      data = JSON.parse(scenario_data)
      user = User.find_by(username: 'admin') || User.first
      
      # Handle single scenario or multiple scenarios
      scenarios_to_import = []
      if data['scenario']
        scenarios_to_import << data['scenario']
      elsif data['scenarios']
        scenarios_to_import = data['scenarios']
      else
        puts \"❌ Invalid format: missing 'scenario' or 'scenarios' key\"
        exit 1
      end
      
      imported_count = 0
      failed_count = 0
      
      scenarios_to_import.each do |scenario_data|
        # Create scenario
        scenario = Scenario.new(
          name: scenario_data['name'] || 'Imported Scenario',
          description: scenario_data['description'],
          user: user,
          tag_bg_color: scenario_data['tag_bg_color'],
          tag_fg_color: scenario_data['tag_fg_color'],
          icon: scenario_data['icon']
        )
        
        if scenario.save
          puts \"✅ Created scenario: #{scenario.name}\"
          
          # Create agents and track GUID to ID mapping
          guid_to_agent = {}
          
          if scenario_data['agents'] && scenario_data['agents'].any?
            scenario_data['agents'].each do |agent_data|
              # Skip if agent type is not available
              unless Object.const_defined?(agent_data['type'])
                puts \"   ⚠️  Skipping agent #{agent_data['name']}: Unknown type #{agent_data['type']}\"
                next
              end
              
              agent_class = agent_data['type'].constantize
              agent = agent_class.new(
                user: user,
                name: agent_data['name'],
                disabled: agent_data['disabled'] || false,
                options: agent_data['options'] || {},
                schedule: agent_data['schedule'],
                keep_events_for: agent_data['keep_events_for'] || 0,
                propagate_immediately: agent_data['propagate_immediately'] || false,
                memory: agent_data['memory'] || {}
              )
              
              # Assign to scenario
              agent.scenario_ids = [scenario.id]
              
              if agent.save
                guid_to_agent[agent_data['guid']] = agent
                puts \"   ✅ Created agent: #{agent.name} (#{agent.type.split('::')[-1]})\"
              else
                puts \"   ❌ Failed to create agent #{agent_data['name']}: #{agent.errors.full_messages.join(', ')}\"
              end
            end
            
            # Now establish agent relationships
            scenario_data['agents'].each do |agent_data|
              next unless guid_to_agent[agent_data['guid']]
              agent = guid_to_agent[agent_data['guid']]
              
              # Connect source agents
              if agent_data['source_agent_guids'] && agent_data['source_agent_guids'].any?
                agent_data['source_agent_guids'].each do |source_guid|
                  if source_agent = guid_to_agent[source_guid]
                    agent.sources << source_agent
                    puts \"   🔗 Connected #{source_agent.name} → #{agent.name}\"
                  end
                end
              end
              
              # Connect receiver agents
              if agent_data['receiver_agent_guids'] && agent_data['receiver_agent_guids'].any?
                agent_data['receiver_agent_guids'].each do |receiver_guid|
                  if receiver_agent = guid_to_agent[receiver_guid]
                    agent.receivers << receiver_agent
                    puts \"   🔗 Connected #{agent.name} → #{receiver_agent.name}\"
                  end
                end
              end
              
              agent.save
            end
          end
          
          imported_count += 1
        else
          puts \"❌ Failed to import scenario: #{scenario.errors.full_messages.join(', ')}\"
          failed_count += 1
        end
      end
      
      puts \"\"
      puts \"📊 Import Summary:\"
      puts \"   ✅ Imported: #{imported_count} scenario(s)\"
      puts \"   ❌ Failed: #{failed_count} scenario(s)\" if failed_count > 0
      
      exit(failed_count > 0 && imported_count == 0 ? 1 : 0)
      
    rescue => e
      puts \"❌ Import error: #{e.message}\"
      puts e.backtrace.first(5).join(\"\\n\")
      exit 1
    end
    "
    
    log::info "📦 Importing scenario from $scenario_file..."
    
    local result
    result=$(huginn::rails_runner "$import_code" 2>&1)
    
    # Cleanup tracking agent
    if [[ -n "$tracking_agent_id" ]] && type -t agents::unregister &>/dev/null; then
        agents::unregister "$tracking_agent_id" >/dev/null 2>&1
    fi
    
    echo "$result"
    
    if [[ "$result" == *"✅ Imported:"* ]] || [[ "$result" == *"✅ Created scenario:"* ]]; then
        return 0
    else
        return 1
    fi
}