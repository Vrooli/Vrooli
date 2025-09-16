#!/bin/bash

# Multi-tenant support for Huginn
# Provides isolated workflow spaces for different users/teams

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../config/defaults.sh"
source "${SCRIPT_DIR}/../config/messages.sh"
source "${SCRIPT_DIR}/docker.sh"
source "${SCRIPT_DIR}/api.sh"

# Ensure container name is set
HUGINN_CONTAINER_NAME="${HUGINN_CONTAINER_NAME:-huginn}"

# Tenant management functions

create_tenant() {
    local tenant_name="${1:?Tenant name required}"
    local tenant_email="${2:?Tenant email required}"
    local tenant_password="${3:-$(generate_password)}"
    
    log::info "Creating tenant: ${tenant_name}"
    
    # Create user account in Huginn
    local user_creation_script="
        # Skip invitation code requirement
        ENV['SKIP_INVITATION_CODE'] = 'true'
        
        user = User.new(
            username: '${tenant_name}',
            email: '${tenant_email}',
            password: '${tenant_password}',
            password_confirmation: '${tenant_password}',
            admin: false
        )
        
        # Skip invitation validation
        user.skip_invitation_code = true if user.respond_to?(:skip_invitation_code=)
        user.invitation_code = 'admin' # Default code if needed
        user.save!
        
        # Create default scenario for tenant
        scenario = user.scenarios.create!(
            name: '${tenant_name} Workspace',
            description: 'Private workspace for ${tenant_name}'
        )
        
        # Create welcome agent
        agent = user.agents.create!(
            name: 'Welcome Agent',
            type: 'Agents::ManualEventAgent',
            scenario_ids: [scenario.id],
            options: {
                'no_bulk_receive' => 'true'
            }
        )
        
        puts \"Tenant created: #{user.username} (ID: #{user.id})\""
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner \"${user_creation_script}\"" 2>&1) || {
        log::error "Failed to create tenant: ${result}"
        return 1
    }
    
    log::success "Tenant created successfully"
    echo "Username: ${tenant_name}"
    echo "Email: ${tenant_email}"
    echo "Password: ${tenant_password}"
    echo "${result}"
}

list_tenants() {
    log::info "Listing all tenants..."
    
    local list_script='
        User.where(admin: false).each do |user|
            agent_count = user.agents.count
            scenario_count = user.scenarios.count
            event_count = user.events.count
            
            puts "#{user.username}|#{user.email}|#{agent_count}|#{scenario_count}|#{event_count}|#{user.created_at}"
        end'
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner '${list_script}'" 2>&1) || {
        log::error "Failed to list tenants: ${result}"
        return 1
    }
    
    if [[ -z "${result}" ]]; then
        log::warn "No tenants found"
        return 0
    fi
    
    # Format output as table
    echo -e "\n📊 TENANT LIST"
    echo "────────────────────────────────────────────────────────────"
    printf "%-20s %-30s %-8s %-10s %-8s %-20s\n" "USERNAME" "EMAIL" "AGENTS" "SCENARIOS" "EVENTS" "CREATED"
    echo "────────────────────────────────────────────────────────────"
    
    while IFS='|' read -r username email agents scenarios events created; do
        printf "%-20s %-30s %-8s %-10s %-8s %-20s\n" \
            "${username}" "${email}" "${agents}" "${scenarios}" "${events}" "${created}"
    done <<< "${result}"
}

get_tenant() {
    local tenant_name="${1:?Tenant name required}"
    
    log::info "Getting tenant details: ${tenant_name}"
    
    local get_script="
        user = User.find_by(username: '${tenant_name}')
        if user.nil?
            puts 'ERROR: Tenant not found'
            exit 1
        end
        
        puts \"Username: #{user.username}\"
        puts \"Email: #{user.email}\"
        puts \"Admin: #{user.admin}\"
        puts \"Created: #{user.created_at}\"
        puts \"\"
        puts \"Statistics:\"
        puts \"  Agents: #{user.agents.count}\"
        puts \"  Scenarios: #{user.scenarios.count}\"
        puts \"  Events: #{user.events.count}\"
        puts \"\"
        puts \"Recent Activity:\"
        user.agents.order(last_event_at: :desc).limit(5).each do |agent|
            puts \"  - #{agent.name} (#{agent.type.split('::').last}): Last event #{agent.last_event_at || 'Never'}\"
        end"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner \"${get_script}\"" 2>&1) || {
        log::error "Failed to get tenant: ${result}"
        return 1
    }
    
    echo "${result}"
}

delete_tenant() {
    local tenant_name="${1:?Tenant name required}"
    local force="${2:-false}"
    
    # Confirm deletion unless forced
    if [[ "${force}" != "true" ]]; then
        log::warn "⚠️  This will permanently delete tenant '${tenant_name}' and all associated data!"
        read -p "Are you sure? (yes/no): " confirmation
        if [[ "${confirmation}" != "yes" ]]; then
            log::info "Deletion cancelled"
            return 0
        fi
    fi
    
    log::info "Deleting tenant: ${tenant_name}"
    
    local delete_script="
        user = User.find_by(username: '${tenant_name}')
        if user.nil?
            puts 'ERROR: Tenant not found'
            exit 1
        end
        
        if user.admin?
            puts 'ERROR: Cannot delete admin users'
            exit 1
        end
        
        # Count items being deleted
        agent_count = user.agents.count
        scenario_count = user.scenarios.count
        event_count = user.events.count
        
        # Delete user and cascade to all owned resources
        user.destroy!
        
        puts \"Tenant deleted: ${tenant_name}\"
        puts \"Removed: #{agent_count} agents, #{scenario_count} scenarios, #{event_count} events\""
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner \"${delete_script}\"" 2>&1) || {
        log::error "Failed to delete tenant: ${result}"
        return 1
    }
    
    log::success "${result}"
}

update_tenant() {
    local tenant_name="${1:?Tenant name required}"
    local field="${2:?Field to update required (email/password)}"
    local value="${3:?New value required}"
    
    log::info "Updating tenant ${tenant_name}: ${field}"
    
    case "${field}" in
        email)
            local update_script="
                user = User.find_by(username: '${tenant_name}')
                if user.nil?
                    puts 'ERROR: Tenant not found'
                    exit 1
                end
                
                user.email = '${value}'
                user.save!
                puts \"Email updated to: #{user.email}\""
            ;;
        password)
            local update_script="
                user = User.find_by(username: '${tenant_name}')
                if user.nil?
                    puts 'ERROR: Tenant not found'
                    exit 1
                end
                
                user.password = '${value}'
                user.password_confirmation = '${value}'
                user.save!
                puts \"Password updated successfully\""
            ;;
        *)
            log::error "Invalid field: ${field}. Use 'email' or 'password'"
            return 1
            ;;
    esac
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner \"${update_script}\"" 2>&1) || {
        log::error "Failed to update tenant: ${result}"
        return 1
    }
    
    log::success "${result}"
}

export_tenant() {
    local tenant_name="${1:?Tenant name required}"
    local output_file="${2:-${tenant_name}_export.json}"
    
    log::info "Exporting tenant data: ${tenant_name}"
    
    local export_script="
        user = User.find_by(username: '${tenant_name}')
        if user.nil?
            puts 'ERROR: Tenant not found'
            exit 1
        end
        
        export_data = {
            user: {
                username: user.username,
                email: user.email
            },
            scenarios: user.scenarios.map do |s|
                {
                    name: s.name,
                    description: s.description,
                    guid: s.guid
                }
            end,
            agents: user.agents.map do |a|
                {
                    name: a.name,
                    type: a.type,
                    options: a.options,
                    scenario_guids: a.scenarios.map(&:guid),
                    guid: a.guid
                }
            end
        }
        
        puts export_data.to_json"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner \"${export_script}\"" 2>&1) || {
        log::error "Failed to export tenant: ${result}"
        return 1
    }
    
    echo "${result}" > "${output_file}"
    log::success "Tenant data exported to: ${output_file}"
}

import_tenant() {
    local import_file="${1:?Import file required}"
    local new_username="${2:-}"
    
    if [[ ! -f "${import_file}" ]]; then
        log::error "Import file not found: ${import_file}"
        return 1
    fi
    
    log::info "Importing tenant data from: ${import_file}"
    
    # Read and prepare import data
    local import_data
    import_data=$(<"${import_file}")
    
    # If new username provided, update it in the JSON
    if [[ -n "${new_username}" ]]; then
        import_data=$(echo "${import_data}" | jq --arg name "${new_username}" '.user.username = $name')
    fi
    
    # Create import script
    local import_script="
        require 'json'
        
        data = JSON.parse('${import_data}')
        
        # Create or find user
        user = User.find_or_create_by(username: data['user']['username']) do |u|
            u.email = data['user']['email']
            u.password = SecureRandom.hex(16)
            u.password_confirmation = u.password
        end
        
        # Import scenarios
        scenario_map = {}
        data['scenarios'].each do |scenario_data|
            scenario = user.scenarios.find_or_create_by(guid: scenario_data['guid']) do |s|
                s.name = scenario_data['name']
                s.description = scenario_data['description']
            end
            scenario_map[scenario_data['guid']] = scenario
        end
        
        # Import agents
        data['agents'].each do |agent_data|
            scenarios = agent_data['scenario_guids'].map { |guid| scenario_map[guid] }.compact
            
            agent = user.agents.find_or_create_by(guid: agent_data['guid']) do |a|
                a.name = agent_data['name']
                a.type = agent_data['type']
                a.options = agent_data['options']
                a.scenario_ids = scenarios.map(&:id)
            end
        end
        
        puts \"Import complete: #{user.username}\"
        puts \"  Scenarios: #{user.scenarios.count}\"
        puts \"  Agents: #{user.agents.count}\""
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner \"${import_script}\"" 2>&1) || {
        log::error "Failed to import tenant: ${result}"
        return 1
    }
    
    log::success "${result}"
}

isolate_tenant() {
    local tenant_name="${1:?Tenant name required}"
    
    log::info "Isolating tenant workspace: ${tenant_name}"
    
    local isolate_script="
        user = User.find_by(username: '${tenant_name}')
        if user.nil?
            puts 'ERROR: Tenant not found'
            exit 1
        end
        
        # Ensure agents only connect within tenant's scenarios
        user.agents.each do |agent|
            if agent.sources.any? { |s| s.user_id != user.id }
                agent.sources = agent.sources.where(user_id: user.id)
                agent.save!
            end
            
            if agent.receivers.any? { |r| r.user_id != user.id }
                agent.receivers = agent.receivers.where(user_id: user.id)
                agent.save!
            end
        end
        
        # Ensure no cross-tenant scenario sharing
        user.scenarios.each do |scenario|
            scenario.agents = scenario.agents.where(user_id: user.id)
            scenario.save!
        end
        
        puts \"Tenant isolated: #{user.username}\"
        puts \"  Scenarios: #{user.scenarios.count}\"
        puts \"  Agents: #{user.agents.count}\""
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner \"${isolate_script}\"" 2>&1) || {
        log::error "Failed to isolate tenant: ${result}"
        return 1
    }
    
    log::success "${result}"
}

quota_status() {
    local tenant_name="${1:?Tenant name required}"
    
    log::info "Checking quota for tenant: ${tenant_name}"
    
    # Default quotas (could be made configurable)
    local max_agents=50
    local max_scenarios=10
    local max_events=10000
    
    local quota_script="
        user = User.find_by(username: '${tenant_name}')
        if user.nil?
            puts 'ERROR: Tenant not found'
            exit 1
        end
        
        agent_count = user.agents.count
        scenario_count = user.scenarios.count
        event_count = user.events.count
        
        puts \"QUOTA STATUS FOR: #{user.username}\"
        puts \"\"
        puts \"Agents:    #{agent_count} / ${max_agents} (#{(agent_count * 100.0 / ${max_agents}).round(1)}%)\"
        puts \"Scenarios: #{scenario_count} / ${max_scenarios} (#{(scenario_count * 100.0 / ${max_scenarios}).round(1)}%)\"
        puts \"Events:    #{event_count} / ${max_events} (#{(event_count * 100.0 / ${max_events}).round(1)}%)\"
        puts \"\"
        
        if agent_count >= ${max_agents}
            puts \"⚠️  Agent quota exceeded!\"
        end
        
        if scenario_count >= ${max_scenarios}
            puts \"⚠️  Scenario quota exceeded!\"
        end
        
        if event_count >= ${max_events}
            puts \"⚠️  Event quota exceeded!\"
        end"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner \"${quota_script}\"" 2>&1) || {
        log::error "Failed to check quota: ${result}"
        return 1
    }
    
    echo "${result}"
}

tenant_stats() {
    log::info "Generating tenant statistics..."
    
    local stats_script="
        total_users = User.where(admin: false).count
        total_agents = Agent.joins(:user).where(users: {admin: false}).count
        total_scenarios = Scenario.joins(:user).where(users: {admin: false}).count
        total_events = Event.joins(:user).where(users: {admin: false}).count
        
        puts \"═══════════════════════════════════════════════════\"
        puts \"           MULTI-TENANT STATISTICS\"
        puts \"═══════════════════════════════════════════════════\"
        puts \"\"
        puts \"📊 OVERALL\"
        puts \"  Total Tenants: #{total_users}\"
        puts \"  Total Agents: #{total_agents}\"
        puts \"  Total Scenarios: #{total_scenarios}\"
        puts \"  Total Events: #{total_events}\"
        puts \"\"
        puts \"📈 TOP TENANTS BY ACTIVITY\"
        
        User.where(admin: false)
            .joins(:events)
            .group('users.id')
            .order('COUNT(events.id) DESC')
            .limit(5)
            .pluck('users.username', 'COUNT(events.id)')
            .each do |username, count|
                puts \"  - #{username}: #{count} events\"
            end
        
        puts \"\"
        puts \"⚡ RECENT ACTIVITY\"
        
        Event.joins(:user)
            .where(users: {admin: false})
            .where('events.created_at > ?', 1.hour.ago)
            .group('users.username')
            .count
            .each do |username, count|
                puts \"  - #{username}: #{count} events in last hour\"
            end"
    
    local result
    result=$(docker exec -i "${HUGINN_CONTAINER_NAME}" \
        bash -c "cd /app && RAILS_ENV=production bundle exec rails runner \"${stats_script}\"" 2>&1) || {
        log::error "Failed to get stats: ${result}"
        return 1
    }
    
    echo "${result}"
}

generate_password() {
    # Generate secure random password
    openssl rand -base64 12 | tr -d "=+/" | cut -c1-16
}

# Export functions for use in CLI
export -f create_tenant
export -f list_tenants
export -f get_tenant
export -f delete_tenant
export -f update_tenant
export -f export_tenant
export -f import_tenant
export -f isolate_tenant
export -f quota_status
export -f tenant_stats
export -f generate_password