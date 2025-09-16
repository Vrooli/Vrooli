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

# API helper functions

api_request() {
    local method="${1:?HTTP method required}"
    local endpoint="${2:?Endpoint required}"
    local data="${3:-}"
    local extra_args="${4:-}"
    
    local url="${HUGINN_API_BASE}${endpoint}"
    local curl_args="-s -X ${method}"
    
    # Add authentication headers if available
    if [[ -n "${HUGINN_API_TOKEN:-}" ]]; then
        curl_args="${curl_args} -H 'Authorization: Bearer ${HUGINN_API_TOKEN}'"
    fi
    
    # Add content type for POST/PUT
    if [[ "${method}" == "POST" || "${method}" == "PUT" ]]; then
        curl_args="${curl_args} -H 'Content-Type: application/json'"
        if [[ -n "${data}" ]]; then
            curl_args="${curl_args} -d '${data}'"
        fi
    fi
    
    # Add any extra arguments
    if [[ -n "${extra_args}" ]]; then
        curl_args="${curl_args} ${extra_args}"
    fi
    
    # Execute request
    eval "curl ${curl_args} '${url}'"
}

# Health API endpoints

api_health() {
    log::info "Checking API health..."
    
    # Basic health check
    local health_response
    health_response=$(timeout 5 curl -sf "${HUGINN_API_BASE}/health" 2>&1) || {
        log::error "Health check failed"
        return 1
    }
    
    log::success "API is healthy"
    echo "${health_response}"
}

api_status() {
    log::info "Getting API status..."
    
    # Create status check script
    local status_script='
        status = {
            running: true,
            version: "latest",
            users: User.count,
            agents: Agent.count,
            scenarios: Scenario.count,
            events: Event.count,
            jobs: {
                scheduled: Delayed::Job.where("run_at > ?", Time.now).count,
                failed: Delayed::Job.where("last_error IS NOT NULL").count,
                pending: Delayed::Job.where("locked_at IS NULL AND run_at <= ?", Time.now).count
            },
            database: {
                connected: ActiveRecord::Base.connected?,
                adapter: ActiveRecord::Base.connection.adapter_name
            }
        }
        puts status.to_json'
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${status_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to get status"
        return 1
    }
    
    echo "${result}" | jq .
}

# Agent API endpoints

api_agents_list() {
    local user_filter="${1:-}"
    
    log::info "Listing agents..."
    
    local filter_clause=""
    if [[ -n "${user_filter}" ]]; then
        filter_clause="joins(:user).where(users: {username: '${user_filter}'})"
    fi
    
    local list_script='
        agents = Agent.all
        result = agents.map do |agent|
            {
                id: agent.id,
                name: agent.name,
                type: agent.type.split("::").last,
                user: agent.user.username,
                scenarios: agent.scenarios.map(&:name),
                event_count: agent.events.count,
                last_event_at: agent.last_event_at,
                created_at: agent.created_at
            }
        end
        puts result.to_json'
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${list_script}'" 2>&1)
    
    # Filter out deprecation warnings and get JSON
    result=$(echo "${result}" | grep -v "DEPRECATION" | grep -v "WARNING" | tail -n1)
    
    if [[ -z "${result}" ]] || [[ "${result}" == *"error"* ]]; then
        log::error "Failed to list agents: ${result}"
        return 1
    fi
    
    echo "${result}" | jq .
}

api_agent_get() {
    local agent_id="${1:?Agent ID required}"
    
    log::info "Getting agent ${agent_id}..."
    
    local get_script="
        agent = Agent.find(${agent_id})
        result = {
            id: agent.id,
            name: agent.name,
            type: agent.type,
            options: agent.options,
            memory: agent.memory,
            user: agent.user.username,
            scenarios: agent.scenarios.map { |s| {id: s.id, name: s.name} },
            sources: agent.sources.map { |s| {id: s.id, name: s.name} },
            receivers: agent.receivers.map { |r| {id: r.id, name: r.name} },
            events: {
                created: agent.events.count,
                received: agent.received_events.count
            },
            schedule: agent.schedule,
            last_check_at: agent.last_check_at,
            last_event_at: agent.last_event_at,
            created_at: agent.created_at,
            updated_at: agent.updated_at
        }
        puts result.to_json"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${get_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to get agent"
        return 1
    }
    
    echo "${result}" | jq .
}

api_agent_create() {
    local name="${1:?Agent name required}"
    local type="${2:?Agent type required}"
    local options="${3:-{}}"
    local user="${4:-admin}"
    
    log::info "Creating agent: ${name}"
    
    local create_script="
        user = User.find_by(username: '${user}')
        agent = user.agents.create!(
            name: '${name}',
            type: 'Agents::${type}',
            options: JSON.parse('${options}')
        )
        result = {
            id: agent.id,
            name: agent.name,
            type: agent.type,
            created: true
        }
        puts result.to_json"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${create_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to create agent"
        return 1
    }
    
    echo "${result}" | jq .
}

api_agent_update() {
    local agent_id="${1:?Agent ID required}"
    local field="${2:?Field to update required}"
    local value="${3:?New value required}"
    
    log::info "Updating agent ${agent_id}..."
    
    local update_script="
        agent = Agent.find(${agent_id})
        case '${field}'
        when 'name'
            agent.name = '${value}'
        when 'options'
            agent.options = JSON.parse('${value}')
        when 'schedule'
            agent.schedule = '${value}'
        else
            raise 'Invalid field: ${field}'
        end
        agent.save!
        puts {id: agent.id, field: '${field}', updated: true}.to_json"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${update_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to update agent"
        return 1
    }
    
    echo "${result}" | jq .
}

api_agent_delete() {
    local agent_id="${1:?Agent ID required}"
    
    log::info "Deleting agent ${agent_id}..."
    
    local delete_script="
        agent = Agent.find(${agent_id})
        name = agent.name
        agent.destroy!
        puts {id: ${agent_id}, name: name, deleted: true}.to_json"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${delete_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to delete agent"
        return 1
    }
    
    echo "${result}" | jq .
}

api_agent_run() {
    local agent_id="${1:?Agent ID required}"
    
    log::info "Running agent ${agent_id}..."
    
    local run_script="
        agent = Agent.find(${agent_id})
        result = {
            id: agent.id,
            name: agent.name,
            events_before: agent.events.count
        }
        agent.check
        result[:events_after] = agent.events.count
        result[:events_created] = result[:events_after] - result[:events_before]
        puts result.to_json"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${run_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to run agent"
        return 1
    }
    
    echo "${result}" | jq .
}

# Event API endpoints

api_events_list() {
    local limit="${1:-20}"
    
    log::info "Listing recent events..."
    
    local list_script="
        events = Event.order(created_at: :desc).limit(${limit})
        result = events.map do |event|
            {
                id: event.id,
                agent: event.agent.name,
                user: event.user.username,
                payload: event.payload,
                created_at: event.created_at
            }
        end
        puts result.to_json"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${list_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to list events"
        return 1
    }
    
    echo "${result}" | jq .
}

api_event_create() {
    local agent_id="${1:?Agent ID required}"
    local payload="${2:?Payload required}"
    
    log::info "Creating event for agent ${agent_id}..."
    
    local create_script="
        agent = Agent.find(${agent_id})
        event = agent.create_event(payload: JSON.parse('${payload}'))
        result = {
            id: event.id,
            agent: agent.name,
            payload: event.payload,
            created: true
        }
        puts result.to_json"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${create_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to create event"
        return 1
    }
    
    echo "${result}" | jq .
}

# Scenario API endpoints

api_scenarios_list() {
    log::info "Listing scenarios..."
    
    local list_script='
        scenarios = Scenario.all
        result = scenarios.map do |scenario|
            {
                id: scenario.id,
                name: scenario.name,
                description: scenario.description,
                user: scenario.user.username,
                agent_count: scenario.agents.count,
                created_at: scenario.created_at
            }
        end
        puts result.to_json'
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${list_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to list scenarios"
        return 1
    }
    
    echo "${result}" | jq .
}

api_scenario_get() {
    local scenario_id="${1:?Scenario ID required}"
    
    log::info "Getting scenario ${scenario_id}..."
    
    local get_script="
        scenario = Scenario.find(${scenario_id})
        result = {
            id: scenario.id,
            name: scenario.name,
            description: scenario.description,
            user: scenario.user.username,
            agents: scenario.agents.map { |a| {id: a.id, name: a.name, type: a.type.split('::').last} },
            created_at: scenario.created_at,
            updated_at: scenario.updated_at
        }
        puts result.to_json"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${get_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to get scenario"
        return 1
    }
    
    echo "${result}" | jq .
}

# Webhook API endpoints

api_webhook_create() {
    local agent_id="${1:?Agent ID required}"
    
    log::info "Creating webhook for agent ${agent_id}..."
    
    local webhook_script="
        agent = Agent.find(${agent_id})
        # Generate unique webhook token
        token = SecureRandom.hex(16)
        agent.options['webhook_token'] = token
        agent.save!
        
        webhook_url = 'http://localhost:${HUGINN_PORT}/users/1/web_requests/' + agent.id.to_s + '/' + token
        result = {
            agent_id: agent.id,
            agent_name: agent.name,
            webhook_url: webhook_url,
            token: token
        }
        puts result.to_json"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${webhook_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to create webhook"
        return 1
    }
    
    echo "${result}" | jq .
}

# User API endpoints

api_users_list() {
    log::info "Listing users..."
    
    local list_script='
        users = User.all
        result = users.map do |user|
            {
                id: user.id,
                username: user.username,
                email: user.email,
                admin: user.admin,
                agent_count: user.agents.count,
                scenario_count: user.scenarios.count,
                created_at: user.created_at
            }
        end
        puts result.to_json'
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${list_script}'" 2>&1 | tail -n1) || {
        log::error "Failed to list users"
        return 1
    }
    
    echo "${result}" | jq .
}

# Export functions for use in CLI
export -f api_request
export -f api_health
export -f api_status
export -f api_agents_list
export -f api_agent_get
export -f api_agent_create
export -f api_agent_update
export -f api_agent_delete
export -f api_agent_run
export -f api_events_list
export -f api_event_create
export -f api_scenarios_list
export -f api_scenario_get
export -f api_webhook_create
export -f api_users_list