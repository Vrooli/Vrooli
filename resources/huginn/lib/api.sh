#!/usr/bin/env bash
# Huginn API and Data Management Functions
# Agent, scenario, and event operations via Rails API

#######################################
# List all agents
# Arguments:
#   $1 - format: "table" or "json" (optional, defaults to "table")
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::list_agents() {
    local format="${1:-table}"
    
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    local agents_code='
    agents = Agent.order(:name)
    
    if agents.any?
      agents.each do |agent|
        status = agent.disabled? ? "❌" : "✅"
        last_run = agent.last_check_at ? agent.last_check_at.strftime("%m/%d %H:%M") : "Never"
        events = agent.events.count
        schedule = agent.schedule || "Manual"
        
        puts "#{status} [#{agent.id}] #{agent.name}"
        puts "   Type: #{agent.type.split("::")[-1]} | Events: #{events}"
        puts "   Last Run: #{last_run} | Schedule: #{schedule}"
        puts ""
      end
    else
      puts "📭 No agents found"
    end
    '
    
    huginn::show_agents_header
    huginn::rails_runner "$agents_code" || {
        log::error "Failed to retrieve agents"
        return 1
    }
}

#######################################
# Export scenario to JSON file
# Arguments:
#   $1 - scenario ID or "all" for all scenarios
#   $2 - output file path (optional, defaults to scenario-{id}.json)
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::export_scenario() {
    local scenario_id="${1:-}"
    local output_file="${2:-}"
    
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    # Set default output file if not provided
    if [[ -z "$output_file" ]]; then
        if [[ "$scenario_id" == "all" ]]; then
            output_file="huginn-scenarios-export-$(date +%Y%m%d-%H%M%S).json"
        else
            output_file="scenario-${scenario_id}.json"
        fi
    fi
    
    local export_code
    if [[ "$scenario_id" == "all" ]]; then
        export_code='
        scenarios = Scenario.includes(:agents)
        export_data = {
          "version" => "1.0",
          "exported_at" => Time.now.iso8601,
          "scenarios" => scenarios.map do |scenario|
            {
              "id" => scenario.id,
              "name" => scenario.name,
              "description" => scenario.description,
              "guid" => scenario.guid,
              "tag_bg_color" => scenario.tag_bg_color,
              "tag_fg_color" => scenario.tag_fg_color,
              "icon" => scenario.icon,
              "agents" => scenario.agents.map do |agent|
                {
                  "type" => agent.type,
                  "name" => agent.name,
                  "guid" => agent.guid,
                  "disabled" => agent.disabled,
                  "options" => agent.options,
                  "schedule" => agent.schedule,
                  "keep_events_for" => agent.keep_events_for,
                  "propagate_immediately" => agent.propagate_immediately,
                  "memory" => agent.memory
                }
              end
            }
          end
        }
        puts export_data.to_json
        '
    else
        if ! huginn::validate_scenario_id "$scenario_id"; then
            log::error "Invalid scenario ID format: $scenario_id"
            return 1
        fi
        
        export_code="
        scenario = Scenario.includes(:agents).find($scenario_id) rescue nil
        if scenario.nil?
          puts \"ERROR: Scenario $scenario_id not found\"
          exit 1
        else
          export_data = {
            \"version\" => \"1.0\",
            \"exported_at\" => Time.now.iso8601,
            \"scenario\" => {
              \"id\" => scenario.id,
              \"name\" => scenario.name,
              \"description\" => scenario.description,
              \"guid\" => scenario.guid,
              \"tag_bg_color\" => scenario.tag_bg_color,
              \"tag_fg_color\" => scenario.tag_fg_color,
              \"icon\" => scenario.icon,
              \"agents\" => scenario.agents.map do |agent|
                {
                  \"type\" => agent.type,
                  \"name\" => agent.name,
                  \"guid\" => agent.guid,
                  \"disabled\" => agent.disabled,
                  \"options\" => agent.options,
                  \"schedule\" => agent.schedule,
                  \"keep_events_for\" => agent.keep_events_for,
                  \"propagate_immediately\" => agent.propagate_immediately,
                  \"memory\" => agent.memory,
                  \"source_agent_guids\" => agent.sources.pluck(:guid),
                  \"receiver_agent_guids\" => agent.receivers.pluck(:guid)
                }
              end
            }
          }
          puts export_data.to_json
        end
        "
    fi
    
    log::info "📦 Exporting scenario(s) to $output_file..."
    
    local result
    if result=$(huginn::rails_runner "$export_code" 2>&1); then
        if [[ "$result" == "ERROR:"* ]]; then
            log::error "${result#ERROR: }"
            return 1
        fi
        
        echo "$result" > "$output_file"
        log::success "✅ Scenario(s) exported successfully to $output_file"
        log::info "📊 File size: $(du -h "$output_file" | cut -f1)"
        return 0
    else
        log::error "Failed to export scenario(s)"
        return 1
    fi
}

#######################################
# Export agents to JSON file
# Arguments:
#   $1 - agent ID, comma-separated IDs, or "all"
#   $2 - output file path (optional)
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::export_agents() {
    local agent_ids="${1:-}"
    local output_file="${2:-}"
    
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    # Set default output file if not provided
    if [[ -z "$output_file" ]]; then
        if [[ "$agent_ids" == "all" ]]; then
            output_file="huginn-agents-export-$(date +%Y%m%d-%H%M%S).json"
        else
            output_file="agents-export-$(date +%Y%m%d-%H%M%S).json"
        fi
    fi
    
    local export_code
    if [[ "$agent_ids" == "all" ]]; then
        export_code='
        agents = Agent.all
        export_data = {
          "version" => "1.0",
          "exported_at" => Time.now.iso8601,
          "agents" => agents.map do |agent|
            {
              "id" => agent.id,
              "type" => agent.type,
              "name" => agent.name,
              "guid" => agent.guid,
              "disabled" => agent.disabled,
              "options" => agent.options,
              "schedule" => agent.schedule,
              "keep_events_for" => agent.keep_events_for,
              "propagate_immediately" => agent.propagate_immediately,
              "memory" => agent.memory,
              "source_agent_guids" => agent.sources.pluck(:guid),
              "receiver_agent_guids" => agent.receivers.pluck(:guid)
            }
          end
        }
        puts export_data.to_json
        '
    else
        # Convert comma-separated IDs to array
        local ids_array="[${agent_ids//,/, }]"
        export_code="
        agent_ids = $ids_array
        agents = Agent.where(id: agent_ids)
        
        if agents.empty?
          puts \"ERROR: No agents found with IDs: #{agent_ids.join(', ')}\"
          exit 1
        else
          export_data = {
            \"version\" => \"1.0\",
            \"exported_at\" => Time.now.iso8601,
            \"agents\" => agents.map do |agent|
              {
                \"id\" => agent.id,
                \"type\" => agent.type,
                \"name\" => agent.name,
                \"guid\" => agent.guid,
                \"disabled\" => agent.disabled,
                \"options\" => agent.options,
                \"schedule\" => agent.schedule,
                \"keep_events_for\" => agent.keep_events_for,
                \"propagate_immediately\" => agent.propagate_immediately,
                \"memory\" => agent.memory,
                \"source_agent_guids\" => agent.sources.pluck(:guid),
                \"receiver_agent_guids\" => agent.receivers.pluck(:guid)
              }
            end
          }
          puts export_data.to_json
        end
        "
    fi
    
    log::info "📦 Exporting agent(s) to $output_file..."
    
    local result
    if result=$(huginn::rails_runner "$export_code" 2>&1); then
        if [[ "$result" == "ERROR:"* ]]; then
            log::error "${result#ERROR: }"
            return 1
        fi
        
        echo "$result" > "$output_file"
        log::success "✅ Agent(s) exported successfully to $output_file"
        log::info "📊 File size: $(du -h "$output_file" | cut -f1)"
        return 0
    else
        log::error "Failed to export agent(s)"
        return 1
    fi
}

#######################################
# Show detailed agent information
# Arguments:
#   $1 - agent ID
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::show_agent() {
    local agent_id="$1"
    
    if ! huginn::validate_agent_id "$agent_id"; then
        log::error "Invalid agent ID format: $agent_id"
        return 1
    fi
    
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    local show_code="
    agent = Agent.find($agent_id) rescue nil
    if agent.nil?
      puts \"❌ Agent $agent_id not found\"
    else
      puts \"🔍 Agent Details: #{agent.name}\"
      puts \"=\" * 50
      puts \"ID: #{agent.id}\"
      puts \"Type: #{agent.type}\"
      puts \"Owner: #{agent.user.username}\"
      puts \"Created: #{agent.created_at.strftime('%Y-%m-%d %H:%M')}\"
      puts \"Last Check: #{agent.last_check_at&.strftime('%Y-%m-%d %H:%M') || 'Never'}\"
      puts \"Schedule: #{agent.schedule || 'Manual'}\"
      puts \"Status: #{agent.disabled? ? 'Disabled' : 'Active'}\"
      puts \"Events Generated: #{agent.events.count}\"
      puts \"\"
      puts \"📋 Configuration:\"
      agent.options.each { |k, v| puts \"   #{k}: #{v.to_s[0..50]}#{v.to_s.length > 50 ? '...' : ''}\" }
      puts \"\"
      puts \"🔗 Connections:\"
      puts \"   Sources: #{agent.sources.map(&:name).join(', ')}\"
      puts \"   Receivers: #{agent.receivers.map(&:name).join(', ')}\"
      puts \"\"
      puts \"📊 Recent Events (last 3):\"
      agent.events.order(created_at: :desc).limit(3).each do |event|
        puts \"   #{event.created_at.strftime('%m/%d %H:%M')}: #{event.payload.to_s[0..60]}...\"
      end
    end
    "
    
    huginn::rails_runner "$show_code" 2>/dev/null || {
        log::error "Failed to retrieve agent information"
        return 1
    }
}

#######################################
# Run an agent manually
# Arguments:
#   $1 - agent ID
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::run_agent() {
    local agent_id="$1"
    
    if ! huginn::validate_agent_id "$agent_id"; then
        log::error "Invalid agent ID format: $agent_id"
        return 1
    fi
    
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    # Register agent for tracking if agent tracking is available
    local tracking_agent_id=""
    if type -t agents::register &>/dev/null; then
        tracking_agent_id=$(agents::generate_id)
        local command_string="huginn::run_agent $agent_id"
        if agents::register "$tracking_agent_id" $$ "$command_string"; then
            huginn::setup_agent_cleanup "$tracking_agent_id"
        else
            tracking_agent_id=""
        fi
    fi
    
    log::info "🚀 Running agent $agent_id..."
    
    local run_code="
    agent = Agent.find($agent_id) rescue nil
    if agent.nil?
      puts \"❌ Agent $agent_id not found\"
    else
      puts \"🚀 Running: #{agent.name}\"
      begin
        agent.check
        puts \"✅ Execution successful\"
        puts \"📊 Total events: #{agent.events.count}\"
      rescue => e
        puts \"❌ Execution failed: #{e.message}\"
      end
    end
    "
    
    huginn::rails_runner "$run_code" 2>/dev/null || {
        # Cleanup tracking agent on error
        if [[ -n "$tracking_agent_id" ]] && type -t agents::unregister &>/dev/null; then
            agents::unregister "$tracking_agent_id" >/dev/null 2>&1
        fi
        log::error "Failed to run agent"
        return 1
    }
    
    # Cleanup tracking agent on success
    if [[ -n "$tracking_agent_id" ]] && type -t agents::unregister &>/dev/null; then
        agents::unregister "$tracking_agent_id" >/dev/null 2>&1
    fi
}

#######################################
# List all scenarios
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::list_scenarios() {
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    local scenarios_code='
    scenarios = Scenario.includes(:agents).order(:name)
    
    if scenarios.any?
      scenarios.each do |scenario|
        puts "📁 [#{scenario.id}] #{scenario.name}"
        puts "   Description: #{scenario.description || \"No description\"}"
        puts "   Agents: #{scenario.agents.count}"
        puts "   Created: #{scenario.created_at.strftime(\"%Y-%m-%d\")}"
        puts ""
      end
    else
      puts "📭 No scenarios found"
    end
    '
    
    huginn::show_scenarios_header
    huginn::rails_runner "$scenarios_code" 2>/dev/null || {
        log::error "Failed to retrieve scenarios"
        return 1
    }
}

#######################################
# Show detailed scenario information
# Arguments:
#   $1 - scenario ID
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::show_scenario() {
    local scenario_id="$1"
    
    if ! huginn::validate_scenario_id "$scenario_id"; then
        log::error "Invalid scenario ID format: $scenario_id"
        return 1
    fi
    
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    local show_code="
    scenario = Scenario.find($scenario_id) rescue nil
    if scenario.nil?
      puts \"❌ Scenario $scenario_id not found\"
    else
      puts \"📂 Scenario Details: #{scenario.name}\"
      puts \"=\" * 50
      puts \"ID: #{scenario.id}\"
      puts \"Description: #{scenario.description || 'No description'}\"
      puts \"Owner: #{scenario.user.username}\"
      puts \"Created: #{scenario.created_at.strftime('%Y-%m-%d %H:%M')}\"
      puts \"Agents: #{scenario.agents.count}\"
      puts \"\"
      puts \"🤖 Included Agents:\"
      scenario.agents.each do |agent|
        status = agent.disabled? ? '❌' : '✅'
        puts \"   #{status} #{agent.name} (#{agent.type.split('::')[-1]})\"
      end
    end
    "
    
    huginn::rails_runner "$show_code" 2>/dev/null || {
        log::error "Failed to retrieve scenario information"
        return 1
    }
}

#######################################
# Show recent events
# Arguments:
#   $1 - count (optional, defaults to 10)
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::show_recent_events() {
    local count="${1:-10}"
    
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    log::info "📊 Recent Events (last $count):"
    
    local events_code="
    events = Event.includes(:agent).order(created_at: :desc).limit($count)
    
    if events.any?
      events.each do |event|
        agent_name = event.agent&.name || 'Unknown Agent'
        timestamp = event.created_at.strftime('%m/%d %H:%M')
        payload_preview = event.payload.to_s[0..60] + '...'
        puts \"📄 #{timestamp} - #{agent_name}:\"
        puts \"   #{payload_preview}\"
        puts \"\"
      end
    else
      puts \"📭 No events found\"
    end
    "
    
    huginn::rails_runner "$events_code" 2>/dev/null || {
        log::error "Failed to retrieve events"
        return 1
    }
}

#######################################
# Show events for specific agent
# Arguments:
#   $1 - agent ID
#   $2 - count (optional, defaults to 10)
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::show_agent_events() {
    local agent_id="$1"
    local count="${2:-10}"
    
    if ! huginn::validate_agent_id "$agent_id"; then
        log::error "Invalid agent ID format: $agent_id"
        return 1
    fi
    
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    local events_code="
    agent = Agent.find($agent_id) rescue nil
    if agent.nil?
      puts \"❌ Agent $agent_id not found\"
    else
      puts \"📊 Events for Agent: #{agent.name}\"
      puts \"=\" * 40
      
      events = agent.events.order(created_at: :desc).limit($count)
      if events.any?
        events.each do |event|
          timestamp = event.created_at.strftime('%m/%d %H:%M')
          payload_preview = event.payload.to_s[0..80] + '...'
          puts \"📄 #{timestamp}: #{payload_preview}\"
        end
      else
        puts \"📭 No events found for this agent\"
      end
    end
    "
    
    huginn::rails_runner "$events_code" 2>/dev/null || {
        log::error "Failed to retrieve agent events"
        return 1
    }
}

#######################################
# Import agent from JSON
# Arguments:
#   $1 - JSON content of the agent
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::api_import_agent() {
    local agent_json="$1"
    
    if ! huginn::is_running; then
        log::error "Huginn is not running"
        return 1
    fi
    
    # Escape JSON for Ruby
    local escaped_json
    escaped_json=$(echo "$agent_json" | sed "s/'/\\\\'/g")
    
    local import_code="
    begin
      require 'json'
      
      # Parse the agent JSON
      agent_data = JSON.parse('$escaped_json')
      
      # Create new agent
      agent = Agent.build_for_type(
        agent_data['type'],
        User.first,
        agent_data.slice('name', 'schedule', 'disabled', 'options', 'keep_events_for', 'propagate_immediately')
      )
      
      if agent.save
        puts \"SUCCESS: Agent '#{agent.name}' created with ID #{agent.id}\"
      else
        puts \"ERROR: Failed to create agent: #{agent.errors.full_messages.join(', ')}\"
        exit 1
      end
    rescue => e
      puts \"ERROR: #{e.message}\"
      exit 1
    end
    "
    
    if huginn::rails_runner "$import_code" 2>/dev/null; then
        return 0
    else
        log::error "Failed to import agent"
        return 1
    fi
}

#######################################
# Import scenario from JSON
# Arguments:
#   $1 - JSON content of the scenario
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::api_import_scenario() {
    local scenario_json="$1"
    
    if ! huginn::is_running; then
        log::error "Huginn is not running"
        return 1
    fi
    
    # Escape JSON for Ruby
    local escaped_json
    escaped_json=$(echo "$scenario_json" | sed "s/'/\\\\'/g")
    
    local import_code="
    begin
      require 'json'
      
      # Parse the scenario JSON
      scenario_data = JSON.parse('$escaped_json')
      
      # Create new scenario
      scenario = Scenario.new(
        user: User.first,
        name: scenario_data['name'],
        description: scenario_data['description'] || '',
        public: scenario_data['public'] || false,
        tag_fg_color: scenario_data['tag_fg_color'],
        tag_bg_color: scenario_data['tag_bg_color']
      )
      
      if scenario.save
        puts \"SUCCESS: Scenario '#{scenario.name}' created with ID #{scenario.id}\"
        
        # Import agents if present
        if scenario_data['agents'].is_a?(Array)
          scenario_data['agents'].each do |agent_data|
            agent = Agent.build_for_type(
              agent_data['type'],
              User.first,
              agent_data.slice('name', 'schedule', 'disabled', 'options', 'keep_events_for', 'propagate_immediately')
            )
            agent.scenario_ids = [scenario.id]
            
            if agent.save
              puts \"  - Agent '#{agent.name}' added to scenario\"
            else
              puts \"  - WARNING: Failed to add agent '#{agent.name}': #{agent.errors.full_messages.join(', ')}\"
            end
          end
        end
      else
        puts \"ERROR: Failed to create scenario: #{scenario.errors.full_messages.join(', ')}\"
        exit 1
      end
    rescue => e
      puts \"ERROR: #{e.message}\"
      exit 1
    end
    "
    
    if huginn::rails_runner "$import_code" 2>/dev/null; then
        return 0
    else
        log::error "Failed to import scenario"
        return 1
    fi
}

#######################################
# Show monitoring dashboard
# Displays real-time agent activity and system metrics
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::monitoring_dashboard() {
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    log::header "📊 Huginn Monitoring Dashboard"
    echo ""
    
    # System overview
    local overview_code='
    total_agents = Agent.count
    active_agents = Agent.where(disabled: false).count
    disabled_agents = Agent.where(disabled: true).count
    total_events = Event.count
    recent_events = Event.where("created_at > ?", 24.hours.ago).count
    failed_jobs = Agent.where("last_error_log_at > ?", 24.hours.ago).count
    
    puts "📈 System Overview"
    puts "=" * 40
    puts "👥 Total Agents: #{total_agents} (Active: #{active_agents}, Disabled: #{disabled_agents})"
    puts "📊 Total Events: #{total_events} (Last 24h: #{recent_events})"
    puts "⚠️  Failed Jobs (24h): #{failed_jobs}"
    puts ""
    
    # Active agents
    puts "🔥 Recently Active Agents"
    puts "=" * 40
    active = Agent.where("last_check_at > ?", 1.hour.ago)
                  .where(disabled: false)
                  .order(last_check_at: :desc)
                  .limit(5)
    
    if active.any?
      active.each do |agent|
        last_run = agent.last_check_at.strftime("%H:%M")
        events = agent.events.where("created_at > ?", 1.hour.ago).count
        status = agent.last_error_log_at && agent.last_error_log_at > 1.hour.ago ? "⚠️" : "✅"
        puts "#{status} #{agent.name} (#{agent.type.split("::")[-1]})"
        puts "   Last Run: #{last_run} | Recent Events: #{events}"
      end
    else
      puts "📭 No recently active agents"
    end
    puts ""
    
    # Recent errors
    puts "❌ Recent Errors"
    puts "=" * 40
    errors = Agent.where("last_error_log_at > ?", 24.hours.ago)
                  .order(last_error_log_at: :desc)
                  .limit(3)
    
    if errors.any?
      errors.each do |agent|
        error_time = agent.last_error_log_at.strftime("%H:%M")
        error_msg = agent.last_error || "Unknown error"
        puts "⚠️  [#{error_time}] #{agent.name}: #{error_msg[0..60]}..."
      end
    else
      puts "✅ No recent errors"
    end
    puts ""
    
    # Performance metrics
    puts "⚡ Performance Metrics"
    puts "=" * 40
    
    # Calculate average event processing time (simplified)
    recent_checks = Agent.where("last_check_at > ?", 1.hour.ago)
                         .where(disabled: false)
    
    if recent_checks.any?
      avg_events = (recent_events.to_f / recent_checks.count).round(1)
      puts "📊 Avg Events/Agent (1h): #{avg_events}"
    end
    
    # Memory/database stats
    db_size = ActiveRecord::Base.connection.execute("SELECT pg_database_size(current_database())").first["pg_database_size"].to_i rescue 0
    if db_size > 0
      db_mb = (db_size / 1024.0 / 1024.0).round(1)
      puts "💾 Database Size: #{db_mb} MB"
    end
    
    event_retention = Event.where("created_at < ?", 30.days.ago).count
    if event_retention > 0
      puts "🗑️  Old Events (>30d): #{event_retention}"
    end
    '
    
    if ! huginn::rails_runner "$overview_code" 2>/dev/null; then
        log::error "Failed to retrieve monitoring data"
        return 1
    fi
    
    return 0
}

#######################################
# Backup Huginn data
# Creates a full backup of database and configuration
# Arguments:
#   $1 - backup directory (optional, defaults to /tmp/huginn-backup-{timestamp})
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::backup() {
    local backup_dir="${1:-/tmp/huginn-backup-$(date +%Y%m%d-%H%M%S)}"
    
    if ! huginn::is_running; then
        log::warn "⚠️  Service not running - backing up configuration only"
    fi
    
    log::info "📦 Creating Huginn backup..."
    log::info "   Backup directory: $backup_dir"
    
    # Create backup directory
    mkdir -p "$backup_dir" || {
        log::error "Failed to create backup directory"
        return 1
    }
    
    # Export all agents and scenarios
    log::info "   Exporting agents and scenarios..."
    local export_code='
    require "json"
    
    backup_data = {
      version: "1.0",
      timestamp: Time.now.iso8601,
      agents: [],
      scenarios: [],
      users: []
    }
    
    # Export all agents with full details
    Agent.all.each do |agent|
      backup_data[:agents] << {
        id: agent.id,
        type: agent.type,
        name: agent.name,
        guid: agent.guid,
        disabled: agent.disabled,
        options: agent.options,
        schedule: agent.schedule,
        keep_events_for: agent.keep_events_for,
        propagate_immediately: agent.propagate_immediately,
        memory: agent.memory,
        source_agent_guids: agent.sources.pluck(:guid),
        receiver_agent_guids: agent.receivers.pluck(:guid)
      }
    end
    
    # Export scenarios if they exist
    if defined?(Scenario)
      Scenario.all.each do |scenario|
        backup_data[:scenarios] << {
          id: scenario.id,
          name: scenario.name,
          description: scenario.description,
          agent_ids: scenario.agents.pluck(:id)
        }
      end
    end
    
    # Export user info (sanitized)
    User.all.each do |user|
      backup_data[:users] << {
        id: user.id,
        username: user.username,
        email: user.email
      }
    end
    
    puts backup_data.to_json
    '
    
    if huginn::rails_runner "$export_code" > "$backup_dir/data.json" 2>/dev/null; then
        log::success "✅ Data exported successfully"
    else
        log::warn "⚠️  Could not export data (service may not be running)"
    fi
    
    # Backup configuration files
    log::info "   Backing up configuration..."
    cp -r "${HUGINN_DIR}/config" "$backup_dir/" 2>/dev/null || true
    
    # Create backup metadata
    cat > "$backup_dir/metadata.txt" <<EOF
Huginn Backup
Created: $(date)
Version: $(docker inspect huginn --format='{{.Config.Image}}' 2>/dev/null || echo "unknown")
Container: huginn
Data File: data.json
Configuration: config/
EOF
    
    # Compress backup
    local backup_archive="${backup_dir}.tar.gz"
    log::info "   Compressing backup..."
    tar -czf "$backup_archive" -C "$(dirname "$backup_dir")" "$(basename "$backup_dir")" || {
        log::error "Failed to compress backup"
        return 1
    }
    
    # Clean up directory
    rm -rf "$backup_dir"
    
    log::success "✅ Backup completed: $backup_archive"
    log::info "   Size: $(du -h "$backup_archive" | cut -f1)"
    
    return 0
}

#######################################
# Restore Huginn from backup
# Restores database and configuration from backup
# Arguments:
#   $1 - backup file path (tar.gz archive)
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::restore() {
    local backup_file="$1"
    
    if [[ -z "$backup_file" ]]; then
        log::error "Backup file path required"
        echo "Usage: huginn restore <backup-file.tar.gz>"
        return 1
    fi
    
    if [[ ! -f "$backup_file" ]]; then
        log::error "Backup file not found: $backup_file"
        return 1
    fi
    
    log::warn "⚠️  This will replace all current Huginn data!"
    echo -n "Continue? (y/N): "
    read -r confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        log::info "Restore cancelled"
        return 0
    fi
    
    log::info "📦 Restoring Huginn from backup..."
    
    # Extract backup
    local temp_dir="/tmp/huginn-restore-$$"
    mkdir -p "$temp_dir"
    
    log::info "   Extracting backup..."
    tar -xzf "$backup_file" -C "$temp_dir" || {
        log::error "Failed to extract backup"
        rm -rf "$temp_dir"
        return 1
    }
    
    # Find the backup directory
    local backup_dir
    backup_dir=$(find "$temp_dir" -name "data.json" -type f | head -1 | xargs dirname)
    
    if [[ -z "$backup_dir" || ! -f "$backup_dir/data.json" ]]; then
        log::error "Invalid backup file - missing data.json"
        rm -rf "$temp_dir"
        return 1
    fi
    
    # Import data
    log::info "   Importing data..."
    if [[ -f "$backup_dir/data.json" ]]; then
        # Use the existing import functionality
        if huginn::import_scenario "$backup_dir/data.json"; then
            log::success "✅ Data imported successfully"
        else
            log::error "Failed to import data"
            rm -rf "$temp_dir"
            return 1
        fi
    fi
    
    # Clean up
    rm -rf "$temp_dir"
    
    log::success "✅ Restore completed successfully"
    log::info "   You may need to restart Huginn for all changes to take effect"
    
    return 0
}

#######################################
# List available agent types
# Returns: 0 if successful, 1 otherwise
#######################################
huginn::list_agent_types() {
    if ! huginn::is_running; then
        huginn::show_not_running
        return 1
    fi
    
    log::info "🔧 Available Agent Types:"
    
    local types_code='
    Agent.types.sort.each do |type|
      short_name = type.split("::")[-1]
      description = case short_name
      when "WebsiteAgent" then "Monitor websites and scrape content"
      when "DigestAgent" then "Aggregate events into periodic summaries"
      when "WebhookAgent" then "Send HTTP requests to external services"
      when "EventFormattingAgent" then "Transform and format event data"
      when "JavaScriptAgent" then "Execute custom JavaScript code"
      when "RssAgent" then "Monitor RSS feeds for new items"
      when "EmailAgent" then "Send immediate email notifications"
      when "TriggerAgent" then "Trigger actions based on conditions"
      when "DeDuplicationAgent" then "Remove duplicate events"
      when "WeatherAgent" then "Collect weather data"
      else "Advanced agent type"
      end
      puts "✅ #{short_name}"
      puts "   #{description}"
      puts ""
    end
    '
    
    huginn::rails_runner "$types_code" 2>/dev/null || {
        log::error "Failed to retrieve agent types"
        return 1
    }
}