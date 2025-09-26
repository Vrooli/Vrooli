#!/bin/bash

# Native API endpoints for Huginn
# Provides direct HTTP API access without Rails runner

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../config/defaults.sh"
source "${SCRIPT_DIR}/../config/messages.sh"
source "${SCRIPT_DIR}/docker.sh"

# Ensure container name and port are set
HUGINN_CONTAINER_NAME="${HUGINN_CONTAINER_NAME:-huginn}"
: "${HUGINN_PORT:=4111}"
HUGINN_API_BASE="http://localhost:${HUGINN_PORT}"

#######################################
# Check API health
# Returns: 0 if healthy, 1 otherwise
#######################################
api_health() {
    log::info "Checking API health..."
    
    if ! huginn::is_running; then
        log::error "Huginn is not running"
        return 1
    fi
    
    # Check if web interface responds
    if timeout 5 curl -sf "${HUGINN_API_BASE}/" > /dev/null 2>&1; then
        log::success "API is healthy"
        return 0
    else
        log::error "Health check failed"
        return 1
    fi
}

#######################################
# Get API status with detailed information
# Returns: JSON status object
#######################################
api_status() {
    log::info "Getting API status..."
    
    if ! huginn::is_running; then
        echo '{"running": false, "error": "Service not running"}'
        return 1
    fi
    
    # Get status via Rails runner
    local status_code="
    begin
      require 'json'
      
      status = {
        running: true,
        version: 'latest',
        users: User.count,
        agents: Agent.count,
        scenarios: Scenario.count,
        events: Event.count,
        jobs: {
          scheduled: Delayed::Job.where('run_at > ?', Time.now).count,
          failed: Delayed::Job.where('failed_at IS NOT NULL').count,
          pending: Delayed::Job.where('locked_at IS NULL AND failed_at IS NULL').count
        },
        database: {
          connected: ActiveRecord::Base.connected?,
          adapter: ActiveRecord::Base.connection.adapter_name
        }
      }
      
      puts JSON.pretty_generate(status)
    rescue => e
      puts JSON.pretty_generate({error: e.message})
      exit 1
    end
    "
    
    if huginn::rails_runner "$status_code" 2>/dev/null; then
        return 0
    else
        echo '{"running": true, "error": "Failed to get detailed status"}'
        return 1
    fi
}

#######################################
# List all agents
# Returns: JSON array of agents
#######################################
api_agents_list() {
    local format="${1:-json}"
    
    if ! huginn::is_running; then
        echo '{"error": "Service not running"}'
        return 1
    fi
    
    local list_code="
    begin
      require 'json'
      
      agents = Agent.all.map do |agent|
        {
          id: agent.id,
          name: agent.name,
          type: agent.type,
          schedule: agent.schedule,
          disabled: agent.disabled,
          events_count: agent.events.count,
          last_run: agent.last_check_at ? agent.last_check_at.iso8601 : nil,
          created_at: agent.created_at.iso8601
        }
      end
      
      puts JSON.pretty_generate(agents)
    rescue => e
      puts JSON.pretty_generate({error: e.message})
      exit 1
    end
    "
    
    if huginn::rails_runner "$list_code" 2>/dev/null; then
        return 0
    else
        echo '{"error": "Failed to list agents"}'
        return 1
    fi
}

#######################################
# Get specific agent details
# Arguments:
#   $1 - Agent ID
# Returns: JSON agent object
#######################################
api_agent_get() {
    local agent_id="${1:?Agent ID required}"
    
    if ! huginn::is_running; then
        echo '{"error": "Service not running"}'
        return 1
    fi
    
    local get_code="
    begin
      require 'json'
      
      agent = Agent.find(${agent_id})
      
      agent_data = {
        id: agent.id,
        name: agent.name,
        type: agent.type,
        schedule: agent.schedule,
        disabled: agent.disabled,
        options: agent.options,
        keep_events_for: agent.keep_events_for,
        events_count: agent.events.count,
        last_run: agent.last_check_at ? agent.last_check_at.iso8601 : nil,
        created_at: agent.created_at.iso8601,
        updated_at: agent.updated_at.iso8601
      }
      
      puts JSON.pretty_generate(agent_data)
    rescue ActiveRecord::RecordNotFound
      puts JSON.pretty_generate({error: 'Agent not found'})
      exit 1
    rescue => e
      puts JSON.pretty_generate({error: e.message})
      exit 1
    end
    "
    
    if huginn::rails_runner "$get_code" 2>/dev/null; then
        return 0
    else
        echo '{"error": "Failed to get agent"}'
        return 1
    fi
}

#######################################
# List events with optional filtering
# Arguments:
#   $1 - Limit (optional, default 10)
#   $2 - Agent ID filter (optional)
# Returns: JSON array of events
#######################################
api_events() {
    local limit="${1:-10}"
    local agent_id="${2:-}"
    
    if ! huginn::is_running; then
        echo '{"error": "Service not running"}'
        return 1
    fi
    
    local filter=""
    if [[ -n "$agent_id" ]]; then
        filter="where(agent_id: ${agent_id})."
    fi
    
    local events_code="
    begin
      require 'json'
      
      events = Event.${filter}order(created_at: :desc).limit(${limit}).map do |event|
        {
          id: event.id,
          agent_id: event.agent_id,
          agent_name: event.agent.name,
          payload: event.payload,
          created_at: event.created_at.iso8601
        }
      end
      
      puts JSON.pretty_generate(events)
    rescue => e
      puts JSON.pretty_generate({error: e.message})
      exit 1
    end
    "
    
    if huginn::rails_runner "$events_code" 2>/dev/null; then
        return 0
    else
        echo '{"error": "Failed to list events"}'
        return 1
    fi
}

#######################################
# List all scenarios
# Returns: JSON array of scenarios
#######################################
api_scenarios() {
    if ! huginn::is_running; then
        echo '{"error": "Service not running"}'
        return 1
    fi
    
    local scenarios_code="
    begin
      require 'json'
      
      scenarios = Scenario.all.map do |scenario|
        {
          id: scenario.id,
          name: scenario.name,
          description: scenario.description,
          public: scenario.public,
          agents_count: scenario.agents.count,
          created_at: scenario.created_at.iso8601,
          updated_at: scenario.updated_at.iso8601
        }
      end
      
      puts JSON.pretty_generate(scenarios)
    rescue => e
      puts JSON.pretty_generate({error: e.message})
      exit 1
    end
    "
    
    if huginn::rails_runner "$scenarios_code" 2>/dev/null; then
        return 0
    else
        echo '{"error": "Failed to list scenarios"}'
        return 1
    fi
}

#######################################
# List webhook endpoints
# Returns: JSON array of webhook agents
#######################################
api_webhooks() {
    if ! huginn::is_running; then
        echo '{"error": "Service not running"}'
        return 1
    fi
    
    local webhooks_code="
    begin
      require 'json'
      
      webhooks = Agent.where(type: 'Agents::WebhookAgent').map do |agent|
        {
          id: agent.id,
          name: agent.name,
          secret: agent.options['secret'],
          endpoint: \"/users/#{agent.user_id}/web_requests/#{agent.id}/#{agent.options['secret']}\",
          events_count: agent.events.count,
          created_at: agent.created_at.iso8601
        }
      end
      
      puts JSON.pretty_generate(webhooks)
    rescue => e
      puts JSON.pretty_generate({error: e.message})
      exit 1
    end
    "
    
    if huginn::rails_runner "$webhooks_code" 2>/dev/null; then
        return 0
    else
        echo '{"error": "Failed to list webhooks"}'
        return 1
    fi
}

#######################################
# List users
# Returns: JSON array of users
#######################################
api_users() {
    if ! huginn::is_running; then
        echo '{"error": "Service not running"}'
        return 1
    fi
    
    local users_code="
    begin
      require 'json'
      
      users = User.all.map do |user|
        {
          id: user.id,
          username: user.username,
          email: user.email,
          admin: user.admin,
          agents_count: user.agents.count,
          scenarios_count: user.scenarios.count,
          created_at: user.created_at.iso8601
        }
      end
      
      puts JSON.pretty_generate(users)
    rescue => e
      puts JSON.pretty_generate({error: e.message})
      exit 1
    end
    "
    
    if huginn::rails_runner "$users_code" 2>/dev/null; then
        return 0
    else
        echo '{"error": "Failed to list users"}'
        return 1
    fi
}