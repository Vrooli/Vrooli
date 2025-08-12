#!/usr/bin/env bash
# n8n User Messages and Help Text
# All user-facing messages, prompts, and documentation

#######################################
# Display usage information
#######################################
n8n::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                              # Install n8n with basic auth"
    echo "  $0 --action install --basic-auth no              # Install without authentication"
    echo "  $0 --action install --username admin --password secret  # Custom credentials"
    echo "  $0 --action install --database postgres          # Use PostgreSQL instead of SQLite"
    echo "  $0 --action install --tunnel yes                 # Enable webhook tunnel (dev only)"
    echo "  $0 --action install --build-image yes            # Build custom image with host access"
    echo "  $0 --action status                               # Check n8n status"
    echo "  $0 --action test                                 # Test n8n functionality and API"
    echo "  $0 --action list-workflows                       # List all workflows"
    echo "  $0 --action list-executions                      # List workflow executions"
    echo "  $0 --action execute --workflow-id ID             # Execute specific workflow"
    echo "  $0 --action logs                                 # View n8n logs"
    echo "  $0 --action reset-password                       # Reset admin password"
    echo "  $0 --action uninstall                           # Remove n8n"
}

#######################################
# API setup instructions
#######################################
n8n::show_api_setup_instructions() {
    local auth_user="${N8N_BASIC_AUTH_USER:-admin}"
    local auth_pass
    
    # Try to get password from running container
    if docker ps --format "{{.Names}}" | grep -q "^${N8N_CONTAINER_NAME}$"; then
        auth_pass=$(docker exec "$N8N_CONTAINER_NAME" env | grep N8N_BASIC_AUTH_PASSWORD | cut -d'=' -f2)
    fi
    
    cat << EOF
=== n8n API Setup Instructions ===

The n8n CLI execute command has a known bug in versions 1.93.0+.
You need to use the REST API instead. Here's how:

1. Access n8n Web Interface:
   URL: $N8N_BASE_URL
   Username: $auth_user
   Password: ${auth_pass:-[check container logs or env]}

2. Create API Key:
   - Go to Settings → n8n API
   - Click "Create an API key"
   - Set a label (e.g., "CLI Access")
   - Set expiration (optional)
   - Copy the API key (shown only once!)

3. Save API Key (Choose One):
   Option A - Save to configuration file (recommended):
     $0 --action save-api-key --api-key YOUR_API_KEY
   
   Option B - Set environment variable (temporary):
     export N8N_API_KEY="your-api-key-here"

4. Execute Workflows:
   $0 --action execute --workflow-id YOUR_WORKFLOW_ID

Alternative: Direct API Call:
   curl -X POST -H "X-N8N-API-KEY: \$N8N_API_KEY" \\
        -H "Content-Type: application/json" \\
        ${N8N_BASE_URL}/api/v1/workflows/WORKFLOW_ID/execute

Note: This is a workaround for GitHub issue #15567.
The standard CLI command 'n8n execute --id' is broken in current versions.
EOF
}

