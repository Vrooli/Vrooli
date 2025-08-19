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
        log::error "Failed to run agent"
        return 1
    }
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