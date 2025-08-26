#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL Client Setup Automation Script
# Quickly create complete client database environments for Vrooli meta-automation projects

DESCRIPTION="Automated client database environment setup for Vrooli projects"

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
POSTGRES_LIB_DIR="${APP_ROOT}/resources/postgres/lib"
source "${APP_ROOT}/resources/postgres/config/defaults.sh"

# Default values
CLIENT_NAME=""
PROJECT_TYPE=""
DATABASES=""
USERS=""
GENERATE_PACKAGE="yes"
SKIP_CONFIRMATION="no"
TEMPLATE=""

# Available project types and their configurations
declare -A PROJECT_CONFIGS=(
    ["ecommerce"]="template:ecommerce databases:store_app,products,orders,customers users:app_user,readonly_user,admin_user"
    ["real-estate"]="template:real-estate databases:leads_app,properties,contacts,analytics users:crm_user,readonly_user"
    ["saas"]="template:saas databases:app_data,tenant_configs,usage_metrics users:app_user,metrics_user,admin_user"
    ["analytics"]="template:production databases:raw_data,processed_data,reports users:analyst_user,readonly_user"
    ["custom"]="template:development databases:app_db users:app_user"
)

#######################################
# Show usage information
#######################################
client_setup::show_usage() {
    cat << EOF
PostgreSQL Client Setup Automation

DESCRIPTION:
    $DESCRIPTION

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -h, --help                          Show this help message
    -c, --client-name <name>            Client/project name (required)
    -t, --project-type <type>           Project type: ecommerce, real-estate, saas, analytics, custom
    -d, --databases <list>              Comma-separated list of databases to create
    -u, --users <list>                  Comma-separated list of users to create
    --template <template>               PostgreSQL template to use
    --no-package                        Skip generating client delivery package
    -y, --yes                           Skip confirmation prompts

PROJECT TYPES:
    ecommerce       E-commerce platform with products, orders, customers
    real-estate     Real estate CRM with leads, properties, contacts
    saas            Multi-tenant SaaS application with usage tracking
    analytics       Data analytics platform with ETL pipeline support
    custom          Custom setup (specify databases and users manually)

EXAMPLES:
    # Quick e-commerce setup
    $0 --client-name "acme-store" --project-type ecommerce

    # Real estate CRM setup
    $0 --client-name "realty-pro" --project-type real-estate

    # Custom setup with specific databases
    $0 --client-name "my-client" --project-type custom \\
       --databases "app_db,logs_db" --users "app_user,readonly_user"

    # Automated setup without prompts
    $0 -c "client-saas" -t saas --yes

NOTES:
    - Client names should be lowercase with hyphens (e.g., 'my-client')
    - Passwords are automatically generated using secure random methods
    - A complete delivery package is created by default
    - All operations can be reviewed before execution

EOF
}

#######################################
# Parse command line arguments
#######################################
client_setup::parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                client_setup::show_usage
                exit 0
                ;;
            -c|--client-name)
                CLIENT_NAME="$2"
                shift 2
                ;;
            -t|--project-type)
                PROJECT_TYPE="$2"
                shift 2
                ;;
            -d|--databases)
                DATABASES="$2"
                shift 2
                ;;
            -u|--users)
                USERS="$2"
                shift 2
                ;;
            --template)
                TEMPLATE="$2"
                shift 2
                ;;
            --no-package)
                GENERATE_PACKAGE="no"
                shift
                ;;
            -y|--yes)
                SKIP_CONFIRMATION="yes"
                shift
                ;;
            *)
                echo "Unknown option: $1" >&2
                client_setup::show_usage >&2
                exit 1
                ;;
        esac
    done
}

#######################################
# Validate inputs
#######################################
client_setup::validate_inputs() {
    if [[ -z "$CLIENT_NAME" ]]; then
        echo "❌ Error: Client name is required" >&2
        echo "Use: $0 --client-name <name> --project-type <type>" >&2
        exit 1
    fi
    
    # Validate client name format
    if [[ ! "$CLIENT_NAME" =~ ^[a-z0-9-]+$ ]]; then
        echo "❌ Error: Client name must be lowercase letters, numbers, and hyphens only" >&2
        echo "Example: 'my-client' or 'acme-store-v2'" >&2
        exit 1
    fi
    
    if [[ -z "$PROJECT_TYPE" ]]; then
        echo "❌ Error: Project type is required" >&2
        echo "Available types: ${!PROJECT_CONFIGS[*]}" >&2
        exit 1
    fi
    
    if [[ ! "${PROJECT_CONFIGS[$PROJECT_TYPE]+exists}" ]]; then
        echo "❌ Error: Unknown project type: $PROJECT_TYPE" >&2
        echo "Available types: ${!PROJECT_CONFIGS[*]}" >&2
        exit 1
    fi
}

#######################################
# Parse project configuration
#######################################
client_setup::parse_project_config() {
    local config="${PROJECT_CONFIGS[$PROJECT_TYPE]}"
    
    # Extract template if not specified
    if [[ -z "$TEMPLATE" ]]; then
        TEMPLATE=$(echo "$config" | grep -o 'template:[^[:space:]]*' | cut -d: -f2)
    fi
    
    # Extract databases if not specified
    if [[ -z "$DATABASES" ]]; then
        DATABASES=$(echo "$config" | grep -o 'databases:[^[:space:]]*' | cut -d: -f2)
    fi
    
    # Extract users if not specified
    if [[ -z "$USERS" ]]; then
        USERS=$(echo "$config" | grep -o 'users:[^[:space:]]*' | cut -d: -f2)
    fi
}

#######################################
# Show setup plan
#######################################
client_setup::show_setup_plan() {
    echo "🚀 PostgreSQL Client Setup Plan"
    echo "================================"
    echo "Client Name:     $CLIENT_NAME"
    echo "Project Type:    $PROJECT_TYPE"
    echo "Template:        $TEMPLATE"
    echo "Databases:       $DATABASES"
    echo "Users:           $USERS"
    echo "Generate Package: $GENERATE_PACKAGE"
    echo ""
    
    if [[ "$SKIP_CONFIRMATION" != "yes" ]]; then
        read -p "🤔 Proceed with this setup? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "❌ Setup cancelled"
            exit 0
        fi
    fi
}

#######################################
# Generate secure password
#######################################
client_setup::generate_password() {
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -base64 32
    else
        # Fallback to /dev/urandom
        head -c 32 /dev/urandom | base64 | tr -d '=\n'
    fi
}

#######################################
# Create PostgreSQL instance
#######################################
client_setup::create_instance() {
    echo "🗄️  Creating PostgreSQL instance..."
    
    if ! "${SCRIPT_DIR}/manage.sh" --action create \
        --instance "$CLIENT_NAME" \
        --template "$TEMPLATE" \
        --yes yes; then
        echo "❌ Failed to create PostgreSQL instance" >&2
        exit 1
    fi
    
    echo "✅ PostgreSQL instance created successfully"
}

#######################################
# Create databases
#######################################
client_setup::create_databases() {
    if [[ -z "$DATABASES" ]]; then
        echo "⚠️  No databases specified, skipping database creation"
        return 0
    fi
    
    echo "💾 Creating databases..."
    
    IFS=',' read -ra DB_ARRAY <<< "$DATABASES"
    for db in "${DB_ARRAY[@]}"; do
        db=$(echo "$db" | xargs)  # Trim whitespace
        echo "  Creating database: $db"
        
        if ! "${SCRIPT_DIR}/manage.sh" --action create-db \
            --instance "$CLIENT_NAME" \
            --database "$db" \
            --yes yes; then
            echo "❌ Failed to create database: $db" >&2
            exit 1
        fi
    done
    
    echo "✅ All databases created successfully"
}

#######################################
# Create users
#######################################
client_setup::create_users() {
    if [[ -z "$USERS" ]]; then
        echo "⚠️  No users specified, skipping user creation"
        return 0
    fi
    
    echo "👥 Creating users..."
    
    # Create a password file for the client
    local password_file="/tmp/${CLIENT_NAME}-passwords.txt"
    echo "# Database Passwords for $CLIENT_NAME" > "$password_file"
    echo "# Generated on $(date)" >> "$password_file"
    echo "" >> "$password_file"
    
    IFS=',' read -ra USER_ARRAY <<< "$USERS"
    for user in "${USER_ARRAY[@]}"; do
        user=$(echo "$user" | xargs)  # Trim whitespace
        local password=$(client_setup::generate_password)
        
        echo "  Creating user: $user"
        
        if ! "${SCRIPT_DIR}/manage.sh" --action create-user \
            --instance "$CLIENT_NAME" \
            --username "$user" \
            --password "$password" \
            --yes yes; then
            echo "❌ Failed to create user: $user" >&2
            exit 1
        fi
        
        # Store password securely
        echo "$user: $password" >> "$password_file"
    done
    
    echo "✅ All users created successfully"
    echo "🔐 Passwords stored in: $password_file"
}

#######################################
# Create backup
#######################################
client_setup::create_backup() {
    echo "💾 Creating initial backup..."
    
    local backup_name="initial-setup-$(date +%Y%m%d-%H%M%S)"
    
    if ! "${SCRIPT_DIR}/manage.sh" --action backup \
        --instance "$CLIENT_NAME" \
        --backup-name "$backup_name" \
        --yes yes; then
        echo "❌ Failed to create backup" >&2
        exit 1
    fi
    
    echo "✅ Initial backup created: $backup_name"
}

#######################################
# Generate client package
#######################################
client_setup::generate_client_package() {
    if [[ "$GENERATE_PACKAGE" != "yes" ]]; then
        echo "⚠️  Skipping client package generation"
        return 0
    fi
    
    echo "📦 Generating client delivery package..."
    
    local package_dir="client-${CLIENT_NAME}-delivery-$(date +%Y%m%d)"
    mkdir -p "$package_dir"/{config,backup,docs,docker}
    
    # Generate connection configurations
    echo "  Generating connection configurations..."
    "${SCRIPT_DIR}/manage.sh" --action connect --instance "$CLIENT_NAME" --format json > "$package_dir/config/connection.json"
    "${SCRIPT_DIR}/manage.sh" --action connect --instance "$CLIENT_NAME" --format n8n > "$package_dir/config/n8n-credentials.json"
    "${SCRIPT_DIR}/manage.sh" --action connect --instance "$CLIENT_NAME" --format default > "$package_dir/config/connection.txt"
    
    # Copy backup files
    echo "  Copying backup files..."
    local latest_backup
    latest_backup=$("${SCRIPT_DIR}/manage.sh" --action list-backups --instance "$CLIENT_NAME" 2>/dev/null | grep -v "INFO" | grep -v "====" | head -1 | awk '{print $1}' || echo "")
    if [[ -n "$latest_backup" && -d ~/.vrooli/backups/postgres/"$CLIENT_NAME"/"$latest_backup" ]]; then
        cp -r ~/.vrooli/backups/postgres/"$CLIENT_NAME"/"$latest_backup"/* "$package_dir/backup/"
    fi
    
    # Copy password file if it exists
    if [[ -f "/tmp/${CLIENT_NAME}-passwords.txt" ]]; then
        cp "/tmp/${CLIENT_NAME}-passwords.txt" "$package_dir/config/passwords.txt"
        rm "/tmp/${CLIENT_NAME}-passwords.txt"  # Clean up temp file
    fi
    
    # Generate Docker Compose file
    echo "  Generating Docker Compose configuration..."
    local port external_password
    port=$(jq -r '.port_external' "$package_dir/config/connection.json" 2>/dev/null || echo "5432")
    external_password=$(jq -r '.password' "$package_dir/config/connection.json" 2>/dev/null || echo "changeme")
    
    cat > "$package_dir/docker/docker-compose.yml" << EOF
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    container_name: ${CLIENT_NAME}-postgres
    environment:
      POSTGRES_DB: vrooli_client
      POSTGRES_USER: vrooli
      POSTGRES_PASSWORD: ${external_password}
    ports:
      - "${port}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ../backup:/backup:ro
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U vrooli -d vrooli_client"]
      interval: 30s
      timeout: 10s
      retries: 5

volumes:
  postgres_data:
    driver: local
EOF
    
    # Generate deployment script
    echo "  Generating deployment script..."
    cat > "$package_dir/deploy.sh" << 'EOF'
#!/bin/bash
set -euo pipefail

echo "🚀 Deploying PostgreSQL database..."

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

# Start database
echo "📊 Starting database container..."
cd docker
docker-compose up -d

# Wait for database to be ready
echo "⏳ Waiting for database to be ready..."
timeout=60
while [ $timeout -gt 0 ]; do
    if docker-compose exec -T postgres pg_isready -U vrooli -d vrooli_client >/dev/null 2>&1; then
        echo "✅ Database is ready!"
        break
    fi
    sleep 2
    ((timeout-=2))
done

if [ $timeout -le 0 ]; then
    echo "❌ Database failed to start within 60 seconds"
    exit 1
fi

# Restore backup if available
if [ -f "../backup/full.sql" ]; then
    echo "🔄 Restoring database from backup..."
    docker-compose exec -T postgres psql -U vrooli -d vrooli_client -f /backup/full.sql
    echo "✅ Database restored successfully!"
else
    echo "⚠️  No backup file found, starting with empty database"
fi

echo ""
echo "🎉 Deployment complete!"
echo "📊 Database is running on port: $(grep -o '[0-9]*:5432' docker-compose.yml | cut -d: -f1)"
echo "📖 Check ../config/connection.txt for connection details"
echo ""
echo "To stop the database: docker-compose down"
echo "To view logs: docker-compose logs -f"
EOF
    chmod +x "$package_dir/deploy.sh"
    
    # Generate README
    echo "  Generating documentation..."
    cat > "$package_dir/README.md" << EOF
# $CLIENT_NAME Database Package

This package contains everything needed to deploy and run the PostgreSQL database for the **$CLIENT_NAME** project.

## Quick Start

1. **Deploy the database**:
   \`\`\`bash
   ./deploy.sh
   \`\`\`

2. **Verify deployment**:
   \`\`\`bash
   cd docker
   docker-compose ps
   docker-compose logs postgres
   \`\`\`

## Package Contents

- \`config/\` - Connection details and credentials
- \`backup/\` - Database backup files
- \`docker/\` - Docker Compose configuration
- \`docs/\` - Additional documentation
- \`deploy.sh\` - One-click deployment script

## Project Configuration

- **Project Type**: $PROJECT_TYPE
- **Template**: $TEMPLATE
- **Databases**: $DATABASES
- **Users**: $USERS

## Connection Information

See \`config/connection.txt\` for complete connection details.

For automation tools (n8n, Node-RED), use \`config/n8n-credentials.json\`.

## Maintenance

### Backup Database
\`\`\`bash
cd docker
docker-compose exec postgres pg_dump -U vrooli vrooli_client > backup-\$(date +%Y%m%d).sql
\`\`\`

### View Logs
\`\`\`bash
cd docker
docker-compose logs -f postgres
\`\`\`

### Stop Database
\`\`\`bash
cd docker
docker-compose down
\`\`\`

## Support

For technical support or questions about this database setup, please contact your Vrooli support team.

---
*Generated by Vrooli PostgreSQL Client Setup - $(date)*
EOF
    
    echo "✅ Client package generated: $package_dir"
    echo "📋 Package contents:"
    find "$package_dir" -type f | sort | sed 's/^/  /'
}

#######################################
# Show completion summary
#######################################
client_setup::show_completion_summary() {
    echo ""
    echo "🎉 Client Setup Complete!"
    echo "========================"
    echo "Client:          $CLIENT_NAME"
    echo "Project Type:    $PROJECT_TYPE"
    echo "Status:          Ready for delivery"
    echo ""
    
    # Show connection info
    echo "🔗 Quick Connection Test:"
    "${SCRIPT_DIR}/manage.sh" --action connect --instance "$CLIENT_NAME"
    
    echo ""
    echo "📋 Next Steps:"
    echo "1. Review the generated client package"
    echo "2. Test the deployment with ./deploy.sh"
    echo "3. Customize documentation as needed"
    echo "4. Deliver to client with handoff documentation"
    echo ""
    echo "💡 Pro Tip: Use the generated client package for easy deployment!"
}

#######################################
# Main execution
#######################################
client_setup::main() {
    echo "🚀 PostgreSQL Client Setup Automation"
    echo "====================================="
    
    client_setup::parse_args "$@"
    client_setup::validate_inputs
    client_setup::parse_project_config
    client_setup::show_setup_plan
    
    # Execute setup steps
    client_setup::create_instance
    client_setup::create_databases
    client_setup::create_users
    client_setup::create_backup
    client_setup::generate_client_package
    
    client_setup::show_completion_summary
}

# Run main function with all arguments
client_setup::main "$@"