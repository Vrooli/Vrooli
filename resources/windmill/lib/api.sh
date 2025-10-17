#!/usr/bin/env bash
# Windmill API Functions
# Functions for interacting with Windmill REST API

# This script expects var.sh to be already sourced by the parent script
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

#######################################
# Test API connectivity and health
# Returns: 0 if healthy, 1 otherwise
#######################################
windmill::test_api() {
    log::info "Testing Windmill API connectivity..."
    
    if ! windmill::is_running; then
        log::error "Windmill is not running"
        return 1
    fi
    
    # Test version endpoint
    local version_response
    if version_response=$(curl -s -f "$WINDMILL_BASE_URL/api/version" 2>/dev/null); then
        log::success "✅ API is responding"
        log::info "Version: $version_response"
    else
        log::error "❌ API is not responding"
        return 1
    fi
    
    # Test health endpoint
    if curl -s -f "$WINDMILL_BASE_URL/api/health" >/dev/null 2>&1; then
        log::success "✅ Health check passed"
    else
        log::warn "⚠️  Health endpoint not available"
    fi
    
    return 0
}

#######################################
# Get API information and endpoints
#######################################
windmill::show_api_info() {
    cat << EOF
=== Windmill API Information ===

Base URL: $WINDMILL_BASE_URL
API Version: v1

Authentication:
  Method: Bearer Token
  Header: Authorization: Bearer <token>

Key Endpoints:
  Version:        GET  /api/version
  Health:         GET  /api/health
  Workspaces:     GET  /api/w/list
  User Info:      GET  /api/users/whoami
  
Workspace Endpoints (replace {w} with workspace ID):
  Scripts:        GET  /api/w/{w}/scripts/list
  Create Script:  POST /api/w/{w}/scripts/create
  Jobs:           GET  /api/w/{w}/jobs/list
  Run Script:     POST /api/w/{w}/jobs/run/{path}
  Resources:      GET  /api/w/{w}/resources/list
  Variables:      GET  /api/w/{w}/variables/list
  Schedules:      GET  /api/w/{w}/schedules/list

Admin Endpoints:
  Workers:        GET  /api/workers/list
  Global Settings: GET  /api/settings/global

Documentation:
  OpenAPI Spec:   GET  /api/openapi.yaml
  Swagger UI:     GET  /openapi.html

Example Usage:

# Get API version
curl $WINDMILL_BASE_URL/api/version

# List workspaces (requires authentication)
curl -H "Authorization: Bearer \$TOKEN" \\
     $WINDMILL_BASE_URL/api/w/list

# Run a script (requires authentication)
curl -X POST \\
     -H "Authorization: Bearer \$TOKEN" \\
     -H "Content-Type: application/json" \\
     -d '{"args":{"name":"World"}}' \\
     $WINDMILL_BASE_URL/api/w/WORKSPACE/jobs/run/f/examples/hello

Authentication Setup:
1. Login to Windmill web interface: $WINDMILL_BASE_URL
2. Go to User Settings → Tokens
3. Create a new API token
4. Use the token in Authorization header

For more details, visit: https://docs.windmill.dev/docs/core_concepts/api
EOF
}

#######################################
# Create an API token (requires manual steps)
#######################################
windmill::create_api_token_instructions() {
    cat << EOF
=== Creating Windmill API Token ===

Windmill API tokens must be created through the web interface.
Follow these steps:

1. Open Windmill in your browser:
   $WINDMILL_BASE_URL

2. Login with your credentials:
   Email: $SUPERADMIN_EMAIL
   Password: $SUPERADMIN_PASSWORD

3. Navigate to User Settings:
   • Click your profile icon (top right)
   • Select "User Settings"

4. Go to the Tokens tab:
   • Click on "Tokens" in the left sidebar

5. Create a new token:
   • Click "New Token"
   • Enter a label (e.g., "CLI Access")
   • Set expiration (optional, leave blank for no expiration)
   • Click "Create"

6. Copy the token:
   • The token will be displayed once
   • Copy it immediately - it won't be shown again

7. Test the token:
   export WINDMILL_TOKEN="your-token-here"
   curl -H "Authorization: Bearer \$WINDMILL_TOKEN" \\
        $WINDMILL_BASE_URL/api/w/list

8. Store securely:
   • Add to your environment variables
   • Or save in a secure configuration file

Security Notes:
• Tokens have the same permissions as your user account
• Store tokens securely and never commit them to version control
• Rotate tokens regularly for security
• Delete unused tokens from the web interface

EOF
}

#######################################
# List available workspaces (if token provided)
# Arguments:
#   $1 - API token (optional)
#######################################
windmill::list_workspaces() {
    local token="${1:-}"
    
    # Load token from config if not provided
    if [[ -z "$token" ]]; then
        token=$(windmill::load_api_key)
    fi
    
    if [[ -z "$token" ]]; then
        log::error "API token is required"
        log::info "Get a token from: $WINDMILL_BASE_URL (User Settings → Tokens)"
        log::info "Or save it with: resource-windmill content add --type api-key --value YOUR_TOKEN"
        return 1
    fi
    
    log::info "Listing Windmill workspaces..."
    
    local response
    if response=$(curl -s -H "Authorization: Bearer $token" "$WINDMILL_BASE_URL/api/w/list" 2>/dev/null); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            log::success "✅ Workspaces retrieved successfully"
            echo "$response" | jq -r '.[] | "• \(.id) - \(.name)"'
        else
            log::error "❌ Invalid response from API"
            echo "Response: $response"
            return 1
        fi
    else
        log::error "❌ Failed to connect to API"
        return 1
    fi
}

#######################################
# Get worker status via API
# Arguments:
#   $1 - API token (optional)
#######################################
windmill::get_workers_api() {
    local token="${1:-}"
    
    # Load token from config if not provided
    if [[ -z "$token" ]]; then
        token=$(windmill::load_api_key)
    fi
    
    if [[ -z "$token" ]]; then
        log::warn "API token not provided, skipping worker API check"
        log::info "Save token with: resource-windmill content add --type api-key --value YOUR_TOKEN"
        return 1
    fi
    
    log::info "Getting worker status via API..."
    
    local response
    if response=$(curl -s -H "Authorization: Bearer $token" "$WINDMILL_BASE_URL/api/workers/list" 2>/dev/null); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local worker_count
            worker_count=$(echo "$response" | jq '. | length')
            log::success "✅ API reports $worker_count workers"
            
            # Show worker details
            echo "$response" | jq -r '.[] | "• \(.worker_instance) - \(.worker_group) - \(.last_ping)"'
        else
            log::error "❌ Invalid worker response from API"
            return 1
        fi
    else
        log::error "❌ Failed to get worker status from API"
        return 1
    fi
}

#######################################
# Run a simple test script via API
# Arguments:
#   $1 - API token
#   $2 - Workspace ID
#   $3 - Script path (optional)
#######################################
windmill::run_test_script() {
    local token="$1"
    local workspace="$2"
    local script_path="${3:-f/examples/hello}"
    
    if [[ -z "$token" || -z "$workspace" ]]; then
        log::error "API token and workspace ID are required"
        return 1
    fi
    
    log::info "Running test script: $script_path in workspace: $workspace"
    
    local job_data='{"args":{"name":"Windmill Test"}}'
    local response
    
    if response=$(curl -s -X POST \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$job_data" \
        "$WINDMILL_BASE_URL/api/w/$workspace/jobs/run/$script_path" 2>/dev/null); then
        
        if echo "$response" | jq . >/dev/null 2>&1; then
            local job_id
            job_id=$(echo "$response" | jq -r '.job_id // .id // "unknown"')
            log::success "✅ Script executed successfully"
            log::info "Job ID: $job_id"
            
            # Try to get job result
            sleep 2
            windmill::get_job_result "$token" "$workspace" "$job_id"
        else
            log::error "❌ Script execution failed"
            echo "Response: $response"
            return 1
        fi
    else
        log::error "❌ Failed to execute script"
        return 1
    fi
}

#######################################
# Get job result via API
# Arguments:
#   $1 - API token
#   $2 - Workspace ID
#   $3 - Job ID
#######################################
windmill::get_job_result() {
    local token="$1"
    local workspace="$2"
    local job_id="$3"
    
    if [[ -z "$token" || -z "$workspace" || -z "$job_id" ]]; then
        return 1
    fi
    
    local response
    if response=$(curl -s -H "Authorization: Bearer $token" \
        "$WINDMILL_BASE_URL/api/w/$workspace/jobs/get/$job_id" 2>/dev/null); then
        
        if echo "$response" | jq . >/dev/null 2>&1; then
            local status result
            status=$(echo "$response" | jq -r '.type // "unknown"')
            result=$(echo "$response" | jq -r '.result // "no result"')
            
            log::info "Job Status: $status"
            if [[ "$result" != "no result" && "$result" != "null" ]]; then
                log::info "Job Result: $result"
            fi
        fi
    fi
}

#######################################
# Create a simple test script in workspace
# Arguments:
#   $1 - API token
#   $2 - Workspace ID
#######################################
windmill::create_test_script() {
    local token="$1"
    local workspace="$2"
    
    if [[ -z "$token" || -z "$workspace" ]]; then
        log::error "API token and workspace ID are required"
        return 1
    fi
    
    local script_content='export function main(name: string): string {
    return `Hello ${name} from Windmill!`;
}'
    
    local script_data
    script_data=$(jq -n \
        --arg path "f/examples/hello" \
        --arg content "$script_content" \
        --arg language "typescript" \
        --arg description "Simple hello world script" \
        '{
            path: $path,
            content: $content,
            language: $language,
            description: $description,
            schema: {
                $schema: "https://json-schema.org/draft/2020-12/schema",
                type: "object",
                properties: {
                    name: {
                        type: "string",
                        description: "Name to greet"
                    }
                },
                required: ["name"]
            }
        }')
    
    log::info "Creating test script in workspace: $workspace"
    
    local response
    if response=$(curl -s -X POST \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$script_data" \
        "$WINDMILL_BASE_URL/api/w/$workspace/scripts/create" 2>/dev/null); then
        
        if echo "$response" | jq . >/dev/null 2>&1; then
            log::success "✅ Test script created successfully"
            log::info "Script path: f/examples/hello"
        else
            log::error "❌ Failed to create test script"
            echo "Response: $response"
            return 1
        fi
    else
        log::error "❌ Failed to connect to API for script creation"
        return 1
    fi
}

#######################################
# Validate API connectivity and authentication
# Arguments:
#   $1 - API token (optional)
#######################################
windmill::validate_api_access() {
    local token="${1:-}"
    
    # Load token from config if not provided
    if [[ -z "$token" ]]; then
        token=$(windmill::load_api_key)
    fi
    
    log::info "Validating API access..."
    
    # Test basic connectivity
    if ! windmill::test_api; then
        return 1
    fi
    
    # Test authentication if token provided
    if [[ -n "$token" ]]; then
        log::info "Testing API authentication..."
        
        local response
        if response=$(curl -s -H "Authorization: Bearer $token" \
            "$WINDMILL_BASE_URL/api/users/whoami" 2>/dev/null); then
            
            if echo "$response" | jq . >/dev/null 2>&1; then
                local username
                username=$(echo "$response" | jq -r '.username // .email // "unknown"')
                log::success "✅ Authenticated as: $username"
            else
                log::error "❌ Authentication failed"
                log::info "Check your API token: $WINDMILL_BASE_URL (User Settings → Tokens)"
                return 1
            fi
        else
            log::error "❌ Authentication request failed"
            return 1
        fi
    else
        log::info "No API token provided - skipping authentication test"
    fi
    
    return 0
}

#######################################
# Show comprehensive API status
#######################################
windmill::api_status() {
    log::header "🔌 Windmill API Status"
    
    # Basic connectivity
    if windmill::test_api; then
        echo
        
        # Show API information
        log::info "📡 API Endpoints:"
        echo "  Base URL: $WINDMILL_BASE_URL"
        echo "  OpenAPI Spec: $WINDMILL_BASE_URL/api/openapi.yaml"
        echo "  Swagger UI: $WINDMILL_BASE_URL/openapi.html"
        
        # Test authentication if token available
        local api_token
        api_token=$(windmill::load_api_key)
        
        if [[ -n "$api_token" ]]; then
            echo
            windmill::validate_api_access "$api_token"
            
            echo
            windmill::list_workspaces "$api_token"
            
            echo
            windmill::get_workers_api "$api_token"
        else
            echo
            log::info "💡 To test authenticated endpoints:"
            log::info "Save token with: resource-windmill content add --type api-key --value YOUR_TOKEN"
            echo
            windmill::create_api_token_instructions
        fi
    else
        log::error "❌ API is not accessible"
        return 1
    fi
}

#######################################
# Load Windmill API key from configuration
# Returns: API key from config or environment variable
#######################################
windmill::load_api_key() {
    local api_key="${WINDMILL_API_TOKEN:-}"
    
    # Try to load API key using 3-layer resolution if not in env
    if [[ -z "$api_key" ]]; then
        if ! api_key=$(secrets::resolve "WINDMILL_API_KEY"); then
            api_key=""
        fi
    fi
    
    echo "$api_key"
}

#######################################
# Save Windmill API key to local configuration
# Arguments: None (uses --api-key argument)
#######################################
windmill::save_api_key() {
    local api_key
    api_key=$(args::get "api-key")
    
    log::header "💾 Save Windmill API Key"
    
    # Check if API key is provided
    if [[ -z "$api_key" ]]; then
        log::error "API key is required"
        echo ""
        echo "Usage: $0 --action save-api-key --api-key YOUR_API_KEY"
        echo ""
        echo "To create an API key:"
        echo "  1. Access Windmill at $WINDMILL_BASE_URL"
        echo "  2. Go to User Settings → Tokens"
        echo "  3. Click 'New Token'"
        echo "  4. Set label and expiration"
        echo "  5. Copy the token (shown only once!)"
        return 1
    fi
    
    # Save API key to project secrets using shared library
    if secrets::save_key "WINDMILL_API_KEY" "$api_key"; then
        local secrets_file
        secrets_file="$(secrets::get_secrets_file)"
        log::success "✅ API key saved to $secrets_file"
        echo ""
        echo "You can now execute workflows without setting WINDMILL_API_TOKEN:"
        echo "  $0 --action test-api"
        echo ""
        echo "The API key will be loaded automatically using 3-layer resolution:"
        echo "  1. Environment variable WINDMILL_API_KEY"
        echo "  2. Project secrets file: $secrets_file"
        echo "  3. HashiCorp Vault (if configured)"
    else
        log::error "Failed to save API key"
        return 1
    fi
}