#!/bin/bash
# Huginn Authentication Helper Functions

#######################################
# Create a new user programmatically
#######################################
huginn::create_user() {
    local email="${USER_EMAIL:-}"
    local password="${USER_PASSWORD:-}"
    local admin="${USER_ADMIN:-false}"
    
    if [[ -z "$email" ]]; then
        log::error "Email is required. Use --user-email <email>"
        return 1
    fi
    
    if [[ -z "$password" ]]; then
        log::error "Password is required. Use --user-password <password>"
        return 1
    fi
    
    if ! huginn::is_healthy; then
        log::error "Huginn is not running or not healthy"
        return 1
    fi
    
    log::info "Creating user: $email"
    
    if docker::run exec "$HUGINN_CONTAINER_NAME" bundle exec rails runner "
        begin
          user = User.find_or_create_by(email: '$email')
          user.password = '$password'
          user.password_confirmation = '$password'
          user.admin = $admin
          if user.save
            puts 'User created/updated successfully: $email'
            exit 0
          else
            puts 'Failed to create user: ' + user.errors.full_messages.join(', ')
            exit 1
          end
        rescue => e
          puts 'Error: ' + e.message
          exit 1
        end
    "; then
        log::success "User created successfully"
        return 0
    else
        log::error "Failed to create user"
        return 1
    fi
}

#######################################
# List all users
#######################################
huginn::list_users() {
    if ! huginn::is_healthy; then
        log::error "Huginn is not running or not healthy"
        return 1
    fi
    
    log::info "Listing users..."
    
    docker::run exec "$HUGINN_CONTAINER_NAME" bundle exec rails runner "
        users = User.all
        if users.empty?
            puts 'No users found'
        else
            puts 'Email | Admin | Created At'
            puts '------|-------|------------'
            users.each do |u|
                puts \"#{u.email} | #{u.admin? ? 'Yes' : 'No'} | #{u.created_at.strftime('%Y-%m-%d')}}\"
            end
        end
    "
}

#######################################
# Generate API token for user
# Note: Huginn doesn't have native API tokens,
# but we can create a custom authentication token
#######################################
huginn::generate_api_token() {
    local email="${USER_EMAIL:-admin@localhost}"
    
    if ! huginn::is_healthy; then
        log::error "Huginn is not running or not healthy"
        return 1
    fi
    
    log::info "Generating authentication token for: $email"
    
    if docker::run exec "$HUGINN_CONTAINER_NAME" bundle exec rails runner "
        begin
          user = User.find_by(email: '$email')
          if user.nil?
            puts 'User not found: $email'
            exit 1
          end
          
          # Generate a secure token
          token = SecureRandom.hex(32)
          
          # Store token in user's options (custom field)
          user.options ||= {}
          user.options['api_token'] = token
          user.options['api_token_created_at'] = Time.now.iso8601
          
          if user.save
            puts 'API Token generated successfully'
            puts '=================================='
            puts 'Email: $email'
            puts 'Token: ' + token
            puts '=================================='
            puts ''
            puts 'Note: Huginn does not have native API authentication.'
            puts 'This token is stored but requires custom middleware to use.'
            puts 'For automation, use Rails runner commands instead.'
            exit 0
          else
            puts 'Failed to save token'
            exit 1
          end
        rescue => e
          puts 'Error: ' + e.message
          exit 1
        end
    "; then
        return 0
    else
        return 1
    fi
}

#######################################
# Import with automatic authentication
#######################################
huginn::import_auto() {
    local file_path="$1"
    
    if [[ -z "$file_path" ]]; then
        log::error "File path is required"
        return 1
    fi
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    if ! huginn::is_healthy; then
        log::error "Huginn is not running or not healthy"
        return 1
    fi
    
    log::info "Importing from: $file_path"
    
    # Copy file to container
    docker::run cp "$file_path" "$HUGINN_CONTAINER_NAME:/tmp/import.json"
    
    # Import automatically using Rails runner
    log::info "Importing agents/scenarios..."
    if docker::run exec "$HUGINN_CONTAINER_NAME" bundle exec rails runner "
        begin
          json_data = File.read('/tmp/import.json')
          parsed = JSON.parse(json_data)
          
          # Get first admin user for ownership
          admin = User.where(admin: true).first
          if admin.nil?
            puts 'No admin user found. Please create one first.'
            exit 1
          end
          
          # Detect if it's a scenario or individual agents
          if parsed.is_a?(Hash) && parsed['agents']
            # Import as scenario
            scenario = Scenario.new(
              name: parsed['name'] || 'Imported Scenario',
              user: admin,
              description: parsed['description'],
              guid: parsed['guid'] || SecureRandom.hex(16),
              tag_fg_color: parsed['tag_fg_color'] || '#FFFFFF',
              tag_bg_color: parsed['tag_bg_color'] || '#5BC0DE'
            )
            
            if scenario.save
              # Import agents
              agent_map = {}
              parsed['agents'].each do |agent_data|
                agent = Agent.build_for_type(
                  agent_data['type'],
                  admin,
                  {
                    name: agent_data['name'],
                    guid: agent_data['guid'],
                    options: agent_data['options'],
                    schedule: agent_data['schedule'],
                    keep_events_for: agent_data['keep_events_for'],
                    scenario_ids: [scenario.id]
                  }
                )
                if agent.save
                  agent_map[agent_data['guid']] = agent
                else
                  puts 'Failed to create agent: ' + agent.name
                end
              end
              
              # Set up links between agents
              if parsed['links']
                parsed['links'].each do |link|
                  source = agent_map[link['source']]
                  receiver = agent_map[link['receiver']]
                  if source && receiver
                    receiver.sources << source unless receiver.sources.include?(source)
                  end
                end
              end
              
              puts 'Successfully imported scenario: ' + scenario.name
            else
              puts 'Failed to create scenario: ' + scenario.errors.full_messages.join(', ')
              exit 1
            end
          else
            # Import individual agents
            agents = parsed.is_a?(Array) ? parsed : [parsed]
            count = 0
            agents.each do |agent_data|
              agent = Agent.build_for_type(
                agent_data['type'],
                admin,
                agent_data.merge('user' => admin)
              )
              if agent.save
                count += 1
              else
                puts 'Failed to create agent: ' + (agent_data['name'] || 'Unnamed')
              end
            end
            puts 'Successfully imported ' + count.to_s + ' agent(s)'
          end
          
          File.delete('/tmp/import.json')
          exit 0
        rescue => e
          puts 'Import failed: ' + e.message
          exit 1
        end
    "; then
        log::success "Import completed successfully"
        return 0
    else
        log::error "Import failed"
        return 1
    fi
}

#######################################
# Export with automatic authentication
#######################################
huginn::export_auto() {
    local file_path="$1"
    local scenario_id="${SCENARIO_ID:-all}"
    
    if [[ -z "$file_path" ]]; then
        log::error "Output file path is required"
        return 1
    fi
    
    if ! huginn::is_healthy; then
        log::error "Huginn is not running or not healthy"
        return 1
    fi
    
    log::info "Exporting to: $file_path"
    
    # Export using Rails runner
    if docker::run exec "$HUGINN_CONTAINER_NAME" bundle exec rails runner "
        begin
          if '$scenario_id' == 'all'
            # Export all scenarios
            scenarios = Scenario.all
            output = scenarios.map do |scenario|
              scenario.export
            end
          else
            # Export specific scenario
            scenario = Scenario.find_by(id: '$scenario_id')
            if scenario.nil?
              puts 'Scenario not found: $scenario_id'
              exit 1
            end
            output = scenario.export
          end
          
          File.write('/tmp/export.json', output.to_json)
          puts 'Export completed successfully'
          exit 0
        rescue => e
          puts 'Export failed: ' + e.message
          exit 1
        end
    "; then
        # Copy file from container
        if docker::run cp "$HUGINN_CONTAINER_NAME:/tmp/export.json" "$file_path"; then
            docker::run exec "$HUGINN_CONTAINER_NAME" rm -f /tmp/export.json
            log::success "Export saved to: $file_path"
            return 0
        else
            log::error "Failed to copy export file"
            return 1
        fi
    else
        log::error "Export failed"
        return 1
    fi
}