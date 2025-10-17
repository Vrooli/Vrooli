#!/usr/bin/env bash
# Windmill User Messages and Help Text
# All user-facing messages, prompts, and documentation

#######################################
# Display usage information
#######################################
windmill::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  resource-windmill manage install                 # Install Windmill with default settings"
    echo "  resource-windmill manage install --workers 5    # Install with 5 worker containers"
    echo "  resource-windmill manage install --external-db yes  # Use external PostgreSQL database"
    echo "  resource-windmill manage install --no-lsp       # Install without Language Server Protocol"
    echo "  resource-windmill status                         # Check Windmill service status"
    echo "  resource-windmill logs                           # View service logs"
    echo "  resource-windmill manage scale-workers 10       # Scale to 10 worker containers"
    echo "  resource-windmill manage restart-workers        # Restart all worker containers"
    echo "  resource-windmill content list                   # List available app examples"
    echo "  resource-windmill content add --name admin-dashboard  # Prepare app for import"
    echo "  resource-windmill manage uninstall              # Remove Windmill completely"
}

#######################################
# Installation success message
#######################################
windmill::show_success_message() {
    local superadmin_email="$1"
    local superadmin_password="$2"
    
    echo "✓ Windmill has been successfully installed!"
    echo
    echo "🌐 Access Windmill at: $WINDMILL_BASE_URL"
    echo
    echo "👤 Super Admin Credentials:"
    echo "  Email: $superadmin_email"
    echo "  Password: $superadmin_password"
    echo
    echo "IMPORTANT: Change the default password after first login!"
    echo
    echo "🔧 Service Information:"
    echo "  Server Container: $WINDMILL_SERVER_CONTAINER"
    echo "  Worker Containers: $WINDMILL_WORKER_REPLICAS replicas"
    echo "  Database: ${WINDMILL_DB_TYPE} (${WINDMILL_DB_EXTERNAL:-internal})"
    echo
    echo "📋 Next Steps:"
    echo "  1. Access the web interface and change the default password"
    echo "  2. Create your first workspace"
    echo "  3. Write your first script (TypeScript, Python, Go, Bash supported)"
    echo "  4. Deploy scripts as workflows with triggers"
    echo
    echo "🛠️  Management Commands:"
    echo "  Status: resource-windmill status"
    echo "  Logs: resource-windmill logs"
    echo "  Scale workers: resource-windmill manage scale-workers <count>"
}

#######################################
# API and workspace setup instructions
#######################################
windmill::show_api_setup_instructions() {
    cat << EOF
=== Windmill API Setup Instructions ===

Windmill provides a comprehensive REST API for automation and integration.

1. Access Windmill Web Interface:
   URL: $WINDMILL_BASE_URL
   Email: $WINDMILL_SUPERADMIN_EMAIL
   Password: $WINDMILL_SUPERADMIN_PASSWORD

2. Create API Token:
   - Go to User Settings → Tokens
   - Click "New Token"
   - Set label (e.g., "CLI Access")
   - Set expiration (optional)
   - Copy the token (shown only once!)

3. Create Workspace:
   - Go to Workspaces
   - Click "Create Workspace"
   - Set workspace ID and name
   - Invite users if needed

4. API Usage Examples:

   # List workspaces
   curl -H "Authorization: Bearer YOUR_TOKEN" \\
        $WINDMILL_BASE_URL/api/w/list

   # Create script
   curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \\
        -H "Content-Type: application/json" \\
        -d '{"path":"f/examples/hello","content":"export function main(name: string) { return \\"Hello \\" + name; }","language":"typescript"}' \\
        $WINDMILL_BASE_URL/api/w/WORKSPACE/scripts/create

   # Execute script
   curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \\
        -H "Content-Type: application/json" \\
        -d '{"args":{"name":"World"}}' \\
        $WINDMILL_BASE_URL/api/w/WORKSPACE/jobs/run/f/examples/hello

   # List jobs
   curl -H "Authorization: Bearer YOUR_TOKEN" \\
        $WINDMILL_BASE_URL/api/w/WORKSPACE/jobs/list

5. Webhook Triggers:
   Windmill supports webhook triggers for external integrations:
   
   # Create webhook endpoint
   URL: $WINDMILL_BASE_URL/api/w/WORKSPACE/jobs/run/f/your/script

Documentation: https://docs.windmill.dev/docs/core_concepts/webhooks
EOF
}

#######################################
# Worker scaling information
#######################################
windmill::show_worker_info() {
    local current_workers="$1"
    local target_workers="$2"
    
    cat << EOF
=== Windmill Worker Information ===

Current Configuration:
  Active Workers: $current_workers
  Memory Limit: $WINDMILL_WORKER_MEMORY_LIMIT per worker
  Worker Group: $WINDMILL_WORKER_GROUP

Worker Scaling:
  Target Workers: $target_workers
  Recommended: 1 worker per CPU core
  Memory Required: ~2GB per worker
  
Worker Types:
  • Default Workers: Run TypeScript, Python, Go scripts
  • Native Workers: Run Bash scripts and system commands
  • Specialized Workers: Can be configured for specific workloads

Performance Guidelines:
  • 1-3 workers: Development/testing
  • 4-10 workers: Small production workloads
  • 10+ workers: High-throughput production

Monitor worker performance in the Windmill UI under Admin → Workers.
EOF
}

#######################################
# Error messages
#######################################
windmill::show_docker_error() {
    echo "Error: Docker and Docker Compose are required but not available."
    echo
    echo "Please install Docker and Docker Compose:"
    echo "  - Ubuntu/Debian: sudo apt-get install docker.io docker-compose-v2"
    echo "  - macOS: Install Docker Desktop from docker.com"
    echo "  - Windows: Install Docker Desktop from docker.com"
    echo
    echo "Ensure Docker daemon is running and try again."
}

windmill::show_permission_error() {
    echo "Error: Cannot access Docker daemon."
    echo
    echo "Fix permissions:"
    echo "  1. Add user to docker group: sudo usermod -aG docker \$USER"
    echo "  2. Log out and back in"
    echo "  3. Verify: docker ps"
}

windmill::show_port_error() {
    local port="$1"
    echo "Error: Port $port is already in use."
    echo
    echo "Solutions:"
    echo "  1. Stop the service using port $port"
    echo "  2. Use custom port: WINDMILL_CUSTOM_PORT=5682 resource-windmill manage install"
    echo "  3. Check port usage: sudo lsof -i :$port"
}

windmill::show_memory_error() {
    local required_gb="$1"
    local available_gb="$2"
    echo "Error: Insufficient memory for Windmill."
    echo
    echo "Required: ${required_gb}GB (recommended)"
    echo "Available: ${available_gb}GB"
    echo
    echo "Windmill requires significant memory for:"
    echo "  • PostgreSQL database"
    echo "  • Windmill server"
    echo "  • Multiple worker containers"
    echo
    echo "Consider:"
    echo "  1. Reducing worker count: --workers 1"
    echo "  2. Using external database: --external-db yes"
    echo "  3. Adding more RAM to your system"
}

windmill::show_database_error() {
    echo "Error: Database connection failed."
    echo
    echo "Troubleshooting:"
    echo "  1. Check database logs: resource-windmill logs --filter db"
    echo "  2. Verify database URL: echo \$WINDMILL_DB_URL"
    echo "  3. Test connection: docker exec windmill-db pg_isready"
    echo "  4. Check network: docker network ls"
}

windmill::show_health_check_error() {
    echo "Error: Windmill services failed to start properly."
    echo
    echo "Debugging steps:"
    echo "  1. Check all logs: resource-windmill logs"
    echo "  2. Verify containers: docker ps -a"
    echo "  3. Check resources: docker stats"
    echo "  4. Inspect networks: docker network inspect windmill-network"
    echo "  5. Review configuration: cat $WINDMILL_ENV_FILE"
}

#######################################
# Interactive prompts
#######################################
windmill::prompt_worker_count() {
    local default_workers="$WINDMILL_WORKER_REPLICAS"
    local cpu_cores
    cpu_cores=$(nproc 2>/dev/null || echo "unknown")
    
    echo
    echo "Worker Configuration:"
    echo "  Current setting: $default_workers workers"
    echo "  System CPU cores: $cpu_cores"
    echo "  Recommended: 1 worker per CPU core"
    echo
    read -p "Number of worker containers [$default_workers]: " -r
    if [[ -n "$REPLY" && "$REPLY" =~ ^[1-9][0-9]*$ ]]; then
        echo "$REPLY"
    else
        echo "$default_workers"
    fi
}

windmill::prompt_external_database() {
    echo
    echo "Database Configuration:"
    echo "  Internal: Uses Docker PostgreSQL container (easier setup)"
    echo "  External: Connect to existing PostgreSQL database (production recommended)"
    echo
    read -p "Use external PostgreSQL database? [y/N]: " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "yes"
        echo
        echo "You'll need to set these environment variables:"
        echo "  WINDMILL_DB_URL=postgresql://user:pass@host:5432/windmill"
        echo "  WINDMILL_DB_EXTERNAL=yes"
    else
        echo "no"
    fi
}

windmill::prompt_lsp_support() {
    echo
    echo "Language Server Protocol (LSP) Support:"
    echo "  Enables: Code completion, syntax highlighting, error detection"
    echo "  Resource usage: Additional container (~500MB RAM)"
    echo
    read -p "Enable LSP support for better development experience? [Y/n]: " -r
    if [[ $REPLY =~ ^[Nn]$ ]]; then
        echo "no"
    else
        echo "yes"
    fi
}

windmill::prompt_uninstall_confirmation() {
    echo
    echo "⚠️  WARNING: This will permanently remove Windmill and all data!"
    echo
    echo "The following will be deleted:"
    echo "  • All Windmill containers and images"
    echo "  • All scripts, workflows, and executions"
    echo "  • All user accounts and workspaces"
    echo "  • Database data (if using internal database)"
    echo
    echo "This action cannot be undone!"
    echo
    read -p "Type 'DELETE' to confirm uninstallation: " -r
    if [[ "$REPLY" == "DELETE" ]]; then
        return 0
    else
        echo "Uninstallation cancelled."
        return 1
    fi
}

windmill::prompt_password_change() {
    echo
    echo "🔒 Security Notice:"
    echo "You're using the default password. This is a security risk!"
    echo
    echo "After installation:"
    echo "1. Login to Windmill web interface"
    echo "2. Go to User Settings"
    echo "3. Change your password immediately"
    echo
    read -p "Continue with installation? [Y/n]: " -r
    if [[ $REPLY =~ ^[Nn]$ ]]; then
        return 1
    fi
    return 0
}

#######################################
# Development and debugging help
#######################################
windmill::show_development_help() {
    cat << EOF
=== Windmill Development Guide ===

Supported Languages:
  • TypeScript/JavaScript (recommended)
  • Python 3
  • Go
  • Bash/Shell
  • SQL
  • REST API calls

Script Development:
  1. Use the built-in web IDE at $WINDMILL_BASE_URL
  2. Create scripts in workspaces
  3. Test scripts with the "Test" button
  4. Deploy as workflows with triggers

Local Development:
  • Scripts are stored in the database
  • Use version control via Git sync (Enterprise)
  • Export/import scripts via API
  • Local CLI for bulk operations

Workflow Features:
  • Triggers: Webhook, Schedule, Manual
  • Flow control: Conditional, loops, parallel execution
  • Error handling: Try/catch, retries
  • Secrets management: Encrypted variables
  • Resource sharing: Connect to databases, APIs

Debugging:
  • View execution logs in web UI
  • Real-time job monitoring
  • Step-by-step workflow debugging
  • Performance metrics and traces

Best Practices:
  1. Use TypeScript for better IDE support
  2. Define clear input/output types
  3. Handle errors gracefully
  4. Use secrets for sensitive data
  5. Test workflows thoroughly

Learn more: https://docs.windmill.dev
EOF
}