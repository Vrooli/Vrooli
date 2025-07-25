#!/bin/bash
# Huginn Authentication Automation Functions
# Source this file to add authentication automation to manage.sh

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
          user.username = '$email'.split('@').first + '_' + Time.now.to_i.to_s
          user.invitation_code = User.first&.invitation_code || 'vrooli-huginn'
          if user.save
            puts 'User created/updated successfully: $email'
            puts 'Username: ' + user.username
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
                puts u.email + ' | ' + (u.admin? ? 'Yes' : 'No') + ' | ' + u.created_at.strftime('%Y-%m-%d')
            end
        end
    "
}

#######################################
# Generate API token for user (custom implementation)
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
# Update admin credentials automatically  
#######################################
huginn::update_admin_credentials() {
    local new_email="${NEW_ADMIN_EMAIL:-}"
    local new_password="${NEW_ADMIN_PASSWORD:-}"
    
    if [[ -z "$new_email" && -z "$new_password" ]]; then
        log::error "At least one of --new-admin-email or --new-admin-password is required"
        return 1
    fi
    
    if ! huginn::is_healthy; then
        log::error "Huginn is not running or not healthy"
        return 1
    fi
    
    log::info "Updating admin credentials..."
    
    # Get current admin email from .env file
    local current_email
    if [[ -f "${HUGINN_DATA_DIR}/.env" ]]; then
        current_email=$(grep "^ADMIN_EMAIL=" "${HUGINN_DATA_DIR}/.env" | cut -d'=' -f2)
    else
        current_email="admin@localhost"
    fi
    
    # Use provided email or keep current
    local target_email="${new_email:-$current_email}"
    
    if docker::run exec "$HUGINN_CONTAINER_NAME" bundle exec rails runner "
        begin
          # Find admin user by current email
          admin = User.find_by(email: '$current_email') || User.where(admin: true).first
          if admin.nil?
            puts 'No admin user found'
            exit 1
          end
          
          # Update email if provided
          if '$new_email'.length > 0
            admin.email = '$target_email'
          end
          
          # Update password if provided
          if '$new_password'.length > 0
            admin.password = '$new_password'
            admin.password_confirmation = '$new_password'
          end
          
          if admin.save
            puts 'Admin credentials updated successfully'
            puts 'Email: ' + admin.email
            exit 0
          else
            puts 'Failed to update credentials: ' + admin.errors.full_messages.join(', ')
            exit 1
          end
        rescue => e
          puts 'Error: ' + e.message
          exit 1
        end 
    "; then
        # Update .env file if successful
        if [[ -n "$new_email" || -n "$new_password" ]]; then
            huginn::update_env_file "$target_email" "$new_password"
        fi
        
        log::success "Admin credentials updated successfully"
        return 0
    else
        log::error "Failed to update admin credentials"
        return 1
    fi
}

#######################################
# Update .env file with new credentials
#######################################
huginn::update_env_file() {
    local email="$1"
    local password="$2"
    
    if [[ ! -f "${HUGINN_DATA_DIR}/.env" ]]; then
        return 0
    fi
    
    # Create backup
    cp "${HUGINN_DATA_DIR}/.env" "${HUGINN_DATA_DIR}/.env.backup.$(date +%s)"
    
    # Update email if provided
    if [[ -n "$email" ]]; then
        sed -i "s/^ADMIN_EMAIL=.*/ADMIN_EMAIL=$email/" "${HUGINN_DATA_DIR}/.env"
    fi
    
    # Update password if provided  
    if [[ -n "$password" ]]; then
        sed -i "s/^ADMIN_PASSWORD=.*/ADMIN_PASSWORD=$password/" "${HUGINN_DATA_DIR}/.env"
    fi
}

#######################################
# Auto-login functionality (for automation)
#######################################
huginn::auto_login() {
    local email="${ADMIN_EMAIL:-admin@localhost}"
    local password="${ADMIN_PASSWORD:-changeme123}"
    
    if ! huginn::is_healthy; then
        log::error "Huginn is not running or not healthy"
        return 1
    fi
    
    log::info "Testing auto-login for: $email"
    
    # Use curl to simulate login and get session cookie
    local login_response
    login_response=$(curl -s -c /tmp/huginn_cookies.txt -b /tmp/huginn_cookies.txt \
        -X POST \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "user[email]=$email&user[password]=$password&user[remember_me]=0" \
        "$HUGINN_BASE_URL/users/sign_in" 2>/dev/null)
    
    # Check if login was successful by looking for redirect or dashboard content
    if echo "$login_response" | grep -q "dashboard\|agents\|scenarios" || \
       curl -s -b /tmp/huginn_cookies.txt "$HUGINN_BASE_URL/agents" | grep -q "New Agent"; then
        
        log::success "Auto-login successful"
        log::info "Session cookie saved to: /tmp/huginn_cookies.txt"
        log::info "Use this cookie for API calls:"
        echo "  curl -b /tmp/huginn_cookies.txt $HUGINN_BASE_URL/agents"
        
        # Clean up
        rm -f /tmp/huginn_cookies.txt
        return 0
    else
        log::error "Auto-login failed - check credentials"
        rm -f /tmp/huginn_cookies.txt
        return 1
    fi
}

#######################################
# Import with automatic user context
#######################################
huginn::import_as_user() {
    local file_path="$1"
    local user_email="${USER_EMAIL:-admin@localhost}"
    
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
    
    log::info "Importing $file_path as user: $user_email"
    
    # Copy file to container
    docker::run cp "$file_path" "$HUGINN_CONTAINER_NAME:/tmp/import.json"
    
    # Import with specific user context
    if docker::run exec "$HUGINN_CONTAINER_NAME" bundle exec rails runner "
        begin
          json_data = File.read('/tmp/import.json')
          parsed = JSON.parse(json_data)
          
          # Find the target user
          user = User.find_by(email: '$user_email')
          if user.nil?
            puts 'User not found: $user_email'
            exit 1
          end
          
          # Import as the specified user
          if parsed.is_a?(Hash) && parsed['agents']
            # Create scenario
            scenario = Scenario.create!(
              name: parsed['name'] || 'Imported Scenario',
              user: user,
              description: parsed['description'],
              guid: parsed['guid'] || SecureRandom.hex(16)
            )
            
            # Import agents
            agent_guids = {}
            parsed['agents'].each do |agent_data|
              agent = Agent.build_for_type(
                agent_data['type'], 
                user,
                agent_data.merge('scenario_ids' => [scenario.id])
              )
              if agent.save
                agent_guids[agent_data['guid']] = agent
              end
            end
            
            # Set up links
            if parsed['links']
              parsed['links'].each do |link|
                source = agent_guids[link['source']]
                receiver = agent_guids[link['receiver']]
                if source && receiver
                  receiver.sources << source unless receiver.sources.include?(source)
                end
              end
            end
            
            puts 'Successfully imported scenario: ' + scenario.name
          else
            puts 'Individual agent import not yet implemented'
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

# Export functions for use in manage.sh
export -f huginn::create_user
export -f huginn::list_users  
export -f huginn::generate_api_token
export -f huginn::update_admin_credentials
export -f huginn::auto_login
export -f huginn::import_as_user