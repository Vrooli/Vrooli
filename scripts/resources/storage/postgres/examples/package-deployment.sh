#!/usr/bin/env bash
set -euo pipefail

# Source var.sh first to get proper directory variables
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true

# Source trash system for safe removal using var_ variables
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# PostgreSQL Client Deployment Packaging Example
# Demonstrates packaging PostgreSQL instances for client deployment

# This script shows how to:
# - Export PostgreSQL instance configurations
# - Create deployment packages for different environments
# - Generate client-specific configuration files
# - Create deployment scripts and documentation
# - Prepare instances for Docker, Kubernetes, or native deployment
# - Package schemas, migrations, and seed data
# - Generate deployment verification scripts

#######################################
# Configuration
#######################################

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
POSTGRES_DIR="${SCRIPT_DIR}/.."

# Source the PostgreSQL management script
# shellcheck disable=SC1091
source "${POSTGRES_DIR}/manage.sh"

#######################################
# Deployment configuration
#######################################
PACKAGE_DIR="/tmp/vrooli_packages"
TEMPLATE_DIR="${SCRIPT_DIR}/../templates"

#######################################
# Utility functions
#######################################
log_info() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] INFO: $1"
}

log_success() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] SUCCESS: $1"
}

log_error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" >&2
}

log_warn() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] WARN: $1"
}

log_step() {
    echo ""
    echo "================================================"
    echo "STEP: $1"
    echo "================================================"
}

#######################################
# Parse command line arguments
#######################################
usage() {
    cat << EOF
PostgreSQL Client Deployment Packaging

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --instance <name>       PostgreSQL instance name (required)
    --client-name <name>    Client name for package (defaults to instance name)
    --deployment-type <type> Deployment type (docker|kubernetes|native) (default: docker)
    --environment <env>     Target environment (development|staging|production) (default: production)
    --include-data          Include current data in package (default: schema only)
    --include-backups       Include recent backups in package
    --output-dir <dir>      Output directory for package (default: /tmp/vrooli_packages)
    --version <version>     Package version (default: 1.0.0)
    --compress              Compress the final package
    --help                  Show this help message

EXAMPLES:
    # Basic Docker deployment package
    $0 --instance real-estate --deployment-type docker

    # Full Kubernetes package with data
    $0 --instance ecommerce --deployment-type kubernetes \\
       --environment production --include-data --include-backups

    # Development native deployment
    $0 --instance testing --deployment-type native \\
       --environment development --client-name test-client

    # Compressed production package
    $0 --instance fintech --deployment-type docker \\
       --environment production --version 2.1.0 --compress

EOF
}

# Parse command line arguments
INSTANCE_NAME=""
CLIENT_NAME=""
DEPLOYMENT_TYPE="docker"
ENVIRONMENT="production"
INCLUDE_DATA=false
INCLUDE_BACKUPS=false
OUTPUT_DIR="$PACKAGE_DIR"
VERSION="1.0.0"
COMPRESS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --instance)
            INSTANCE_NAME="$2"
            shift 2
            ;;
        --client-name)
            CLIENT_NAME="$2"
            shift 2
            ;;
        --deployment-type)
            DEPLOYMENT_TYPE="$2"
            shift 2
            ;;
        --environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --include-data)
            INCLUDE_DATA=true
            shift
            ;;
        --include-backups)
            INCLUDE_BACKUPS=true
            shift
            ;;
        --output-dir)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --compress)
            COMPRESS=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate required arguments
if [[ -z "$INSTANCE_NAME" ]]; then
    log_error "Instance name is required"
    usage
    exit 1
fi

# Set default client name
if [[ -z "$CLIENT_NAME" ]]; then
    CLIENT_NAME="$INSTANCE_NAME"
fi

# Validate deployment type
case "$DEPLOYMENT_TYPE" in
    "docker"|"kubernetes"|"native") ;;
    *)
        log_error "Invalid deployment type: $DEPLOYMENT_TYPE"
        exit 1
        ;;
esac

# Validate environment
case "$ENVIRONMENT" in
    "development"|"staging"|"production") ;;
    *)
        log_error "Invalid environment: $ENVIRONMENT"
        exit 1
        ;;
esac

#######################################
# Package creation functions
#######################################

create_package_structure() {
    local package_name="$1"
    local package_path="$2"
    
    log_step "Creating package structure"
    
    # Create base package directory
    mkdir -p "$package_path"
    
    # Create standard directories
    mkdir -p "$package_path/config"
    mkdir -p "$package_path/schema"
    mkdir -p "$package_path/migrations"
    mkdir -p "$package_path/seeds"
    mkdir -p "$package_path/backups"
    mkdir -p "$package_path/scripts"
    mkdir -p "$package_path/docs"
    mkdir -p "$package_path/templates"
    
    # Create deployment-specific directories
    case "$DEPLOYMENT_TYPE" in
        "docker")
            mkdir -p "$package_path/docker"
            ;;
        "kubernetes")
            mkdir -p "$package_path/kubernetes"
            mkdir -p "$package_path/kubernetes/manifests"
            mkdir -p "$package_path/kubernetes/configmaps"
            mkdir -p "$package_path/kubernetes/secrets"
            ;;
        "native")
            mkdir -p "$package_path/systemd"
            mkdir -p "$package_path/init"
            ;;
    esac
    
    log_success "Package structure created: $package_path"
}

export_instance_config() {
    local instance_name="$1"
    local package_path="$2"
    
    log_step "Exporting instance configuration"
    
    # Check if instance exists
    if ! "${POSTGRES_DIR}/manage.sh" --action status --instance "$instance_name" >/dev/null 2>&1; then
        log_error "Instance not found: $instance_name"
        return 1
    fi
    
    # Get connection information
    local conn_info=$("${POSTGRES_DIR}/manage.sh" --action connect --instance "$instance_name" 2>/dev/null)
    
    # Extract connection details
    local host="localhost"
    local port=$(echo "$conn_info" | grep "Port:" | awk '{print $2}')
    local username=$(echo "$conn_info" | grep "Username:" | awk '{print $2}')
    local password=$(echo "$conn_info" | grep "Password:" | awk '{print $2}')
    local database=$(echo "$conn_info" | grep "Database:" | awk '{print $2}')
    
    # Get instance configuration
    local template=""
    local created=""
    if [[ -f "${POSTGRES_INSTANCES_DIR}/${instance_name}/config.json" ]]; then
        template=$(grep '"template"' "${POSTGRES_INSTANCES_DIR}/${instance_name}/config.json" | cut -d'"' -f4 2>/dev/null || echo "unknown")
        created=$(grep '"created"' "${POSTGRES_INSTANCES_DIR}/${instance_name}/config.json" | cut -d'"' -f4 2>/dev/null || echo "unknown")
    fi
    
    # Create configuration file
    cat > "$package_path/config/instance.json" << EOF
{
  "instance": {
    "name": "$instance_name",
    "client": "$CLIENT_NAME",
    "template": "$template",
    "created": "$created",
    "exported": "$(date -Iseconds)",
    "version": "$VERSION"
  },
  "database": {
    "host": "$host",
    "port": $port,
    "database": "$database",
    "username": "$username",
    "password": "$password"
  },
  "deployment": {
    "type": "$DEPLOYMENT_TYPE",
    "environment": "$ENVIRONMENT",
    "target_port": 5432
  },
  "features": {
    "include_data": $INCLUDE_DATA,
    "include_backups": $INCLUDE_BACKUPS,
    "compression": $COMPRESS
  }
}
EOF
    
    # Create environment file
    cat > "$package_path/config/database.env" << EOF
# Database Configuration for $CLIENT_NAME
# Generated: $(date)
# Environment: $ENVIRONMENT

DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=$database
DATABASE_USER=$username
DATABASE_PASSWORD=$password
DATABASE_URL=postgresql://$username:$password@localhost:5432/$database

# Instance Information
POSTGRES_INSTANCE=$instance_name
POSTGRES_CLIENT=$CLIENT_NAME
POSTGRES_TEMPLATE=$template
POSTGRES_VERSION=$VERSION

# Deployment Configuration
DEPLOYMENT_TYPE=$DEPLOYMENT_TYPE
DEPLOYMENT_ENVIRONMENT=$ENVIRONMENT
EOF
    
    # Copy PostgreSQL configuration template
    if [[ -f "${TEMPLATE_DIR}/${template}.conf" ]]; then
        cp "${TEMPLATE_DIR}/${template}.conf" "$package_path/config/postgresql.conf"
        log_success "PostgreSQL configuration template copied: $template"
    else
        log_warn "Template configuration not found: $template"
    fi
    
    log_success "Instance configuration exported"
}

export_schema() {
    local instance_name="$1"
    local package_path="$2"
    
    log_step "Exporting database schema"
    
    # Export schema
    if "${POSTGRES_DIR}/manage.sh" --action db-stats --instance "$instance_name" >/dev/null 2>&1; then
        # Create schema dump
        local schema_file="$package_path/schema/schema.sql"
        
        # Use database dump functionality to get schema
        if postgres::database::dump_schema "$instance_name" "$schema_file"; then
            log_success "Schema exported to: schema.sql"
        else
            log_warn "Failed to export schema, creating placeholder"
            echo "-- Schema export failed - manual export required" > "$schema_file"
        fi
        
        # Export database statistics
        "${POSTGRES_DIR}/manage.sh" --action db-stats --instance "$instance_name" > "$package_path/schema/stats.txt" 2>/dev/null || echo "Stats not available" > "$package_path/schema/stats.txt"
        
        # List databases
        "${POSTGRES_DIR}/manage.sh" --action list --instance "$instance_name" > "$package_path/schema/databases.txt" 2>/dev/null || echo "Database list not available" > "$package_path/schema/databases.txt"
    else
        log_warn "Instance not running, creating placeholder schema files"
        echo "-- Instance not running during export" > "$package_path/schema/schema.sql"
    fi
}

export_migrations() {
    local instance_name="$1"
    local package_path="$2"
    
    log_step "Exporting migration history"
    
    # Export migration status if available
    if "${POSTGRES_DIR}/manage.sh" --action migrate-status --instance "$instance_name" >/dev/null 2>&1; then
        "${POSTGRES_DIR}/manage.sh" --action migrate-status --instance "$instance_name" > "$package_path/migrations/migration_status.txt"
        log_success "Migration status exported"
    else
        log_info "No migration system found, creating placeholder"
        cat > "$package_path/migrations/migration_status.txt" << EOF
Migration system not initialized for this instance.
To use migrations in the deployed environment:
1. Initialize: ./scripts/manage.sh --action migrate-init --instance <name>
2. Run migrations: ./scripts/manage.sh --action migrate --instance <name> --migrations-dir ./migrations
EOF
    fi
    
    # Create sample migration
    cat > "$package_path/migrations/001_initial_setup.sql" << EOF
-- Description: Initial database setup for $CLIENT_NAME
-- ROLLBACK: DROP TABLE IF EXISTS example_table;

-- Create example table
CREATE TABLE IF NOT EXISTS example_table (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_example_table_name ON example_table(name);
CREATE INDEX IF NOT EXISTS idx_example_table_created_at ON example_table(created_at);

-- Insert sample data
INSERT INTO example_table (name) VALUES 
    ('Sample Entry 1'),
    ('Sample Entry 2'),
    ('Sample Entry 3')
ON CONFLICT DO NOTHING;
EOF
    
    log_success "Migration template created"
}

export_data() {
    local instance_name="$1"
    local package_path="$2"
    
    if [[ "$INCLUDE_DATA" != "true" ]]; then
        log_info "Data export skipped (not requested)"
        return 0
    fi
    
    log_step "Exporting database data"
    
    # Export data
    local data_file="$package_path/seeds/current_data.sql"
    
    if postgres::database::dump_data "$instance_name" "$data_file"; then
        log_success "Data exported to: seeds/current_data.sql"
    else
        log_warn "Failed to export data, creating placeholder"
        cat > "$package_path/seeds/sample_data.sql" << EOF
-- Sample data for $CLIENT_NAME
-- Replace with actual seed data

INSERT INTO example_table (name) VALUES 
    ('Client Specific Data 1'),
    ('Client Specific Data 2'),
    ('Client Specific Data 3');
EOF
    fi
}

export_backups() {
    local instance_name="$1"
    local package_path="$2"
    
    if [[ "$INCLUDE_BACKUPS" != "true" ]]; then
        log_info "Backup export skipped (not requested)"
        return 0
    fi
    
    log_step "Exporting recent backups"
    
    # Create fresh backup for deployment
    local deployment_backup="deployment_$(date +%Y%m%d_%H%M%S)"
    
    if "${POSTGRES_DIR}/manage.sh" --action backup --instance "$instance_name" --backup-name "$deployment_backup" --backup-type "full"; then
        # Copy backup to package
        # Use POSTGRES_BACKUP_DIR from defaults to get proper backup path
        local backup_source="${POSTGRES_BACKUP_DIR}/${instance_name}/${deployment_backup}"
        if [[ -d "$backup_source" ]]; then
            cp -r "$backup_source" "$package_path/backups/"
            log_success "Deployment backup included: $deployment_backup"
        fi
    else
        log_warn "Failed to create deployment backup"
    fi
    
    # List available backups
    "${POSTGRES_DIR}/manage.sh" --action list-backups --instance "$instance_name" > "$package_path/backups/backup_list.txt" 2>/dev/null || echo "No backups available" > "$package_path/backups/backup_list.txt"
}

create_deployment_scripts() {
    local instance_name="$1"
    local package_path="$2"
    
    log_step "Creating deployment scripts"
    
    case "$DEPLOYMENT_TYPE" in
        "docker")
            create_docker_deployment "$instance_name" "$package_path"
            ;;
        "kubernetes")
            create_kubernetes_deployment "$instance_name" "$package_path"
            ;;
        "native")
            create_native_deployment "$instance_name" "$package_path"
            ;;
    esac
}

create_docker_deployment() {
    local instance_name="$1"
    local package_path="$2"
    
    log_info "Creating Docker deployment files"
    
    # Create Dockerfile
    cat > "$package_path/docker/Dockerfile" << EOF
FROM postgres:16-alpine

# Set environment variables
ENV POSTGRES_DB=\${DATABASE_NAME:-${instance_name}_app}
ENV POSTGRES_USER=\${DATABASE_USER:-vrooli}
ENV POSTGRES_PASSWORD=\${DATABASE_PASSWORD}

# Copy configuration
COPY config/postgresql.conf /etc/postgresql/postgresql.conf

# Copy initialization scripts
COPY schema/schema.sql /docker-entrypoint-initdb.d/01-schema.sql
COPY migrations/*.sql /docker-entrypoint-initdb.d/migrations/
COPY seeds/*.sql /docker-entrypoint-initdb.d/seeds/

# Set configuration file
CMD ["postgres", "-c", "config_file=/etc/postgresql/postgresql.conf"]
EOF
    
    # Create docker-compose.yml
    cat > "$package_path/docker/docker-compose.yml" << EOF
version: '3.8'

services:
  postgres:
    build: .
    container_name: ${CLIENT_NAME}-postgres
    environment:
      - POSTGRES_DB=${instance_name}_app
      - POSTGRES_USER=vrooli
      - POSTGRES_PASSWORD=\${DATABASE_PASSWORD}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backups:/backups
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U vrooli -d ${instance_name}_app"]
      interval: 30s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
EOF
    
    # Create deployment script
    cat > "$package_path/scripts/deploy.sh" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

# Docker Deployment Script
echo "Deploying PostgreSQL with Docker..."

# Check if Docker is available
if ! command -v docker >/dev/null 2>&1; then
    echo "Error: Docker is not installed"
    exit 1
fi

if ! command -v docker-compose >/dev/null 2>&1; then
    echo "Error: Docker Compose is not installed"
    exit 1
fi

# Load configuration
if [[ -f "../config/database.env" ]]; then
    set -a
    source "../config/database.env"
    set +a
    echo "Configuration loaded from database.env"
else
    echo "Warning: database.env not found, using defaults"
fi

# Check required environment variables
if [[ -z "${DATABASE_PASSWORD:-}" ]]; then
    echo "Error: DATABASE_PASSWORD environment variable is required"
    echo "Set it in database.env or export it before running this script"
    exit 1
fi

# Change to docker directory
cd "$(dirname "$0")/../docker"

# Build and start the container
echo "Building PostgreSQL image..."
docker-compose build

echo "Starting PostgreSQL container..."
docker-compose up -d

# Wait for database to be ready
echo "Waiting for database to be ready..."
sleep 10

# Test connection
echo "Testing database connection..."
if docker-compose exec postgres pg_isready -U vrooli; then
    echo "✅ PostgreSQL is running and ready"
    echo ""
    echo "Connection details:"
    echo "  Host: localhost"
    echo "  Port: 5432"
    echo "  Database: ${DATABASE_NAME:-app}"
    echo "  Username: ${DATABASE_USER:-vrooli}"
    echo "  Password: [from DATABASE_PASSWORD]"
    echo ""
    echo "Container name: ${CLIENT_NAME:-client}-postgres"
    echo ""
    echo "To connect: psql -h localhost -p 5432 -U ${DATABASE_USER:-vrooli} -d ${DATABASE_NAME:-app}"
    echo "To stop: docker-compose down"
    echo "To view logs: docker-compose logs postgres"
else
    echo "❌ PostgreSQL deployment failed"
    docker-compose logs postgres
    exit 1
fi
EOF
    
    chmod +x "$package_path/scripts/deploy.sh"
    
    log_success "Docker deployment files created"
}

create_kubernetes_deployment() {
    local instance_name="$1"
    local package_path="$2"
    
    log_info "Creating Kubernetes deployment files"
    
    # Create namespace
    cat > "$package_path/kubernetes/manifests/namespace.yaml" << EOF
apiVersion: v1
kind: Namespace
metadata:
  name: ${CLIENT_NAME}
  labels:
    app: ${CLIENT_NAME}
    component: postgres
EOF
    
    # Create secret
    cat > "$package_path/kubernetes/secrets/postgres-secret.yaml" << EOF
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: ${CLIENT_NAME}
type: Opaque
stringData:
  POSTGRES_PASSWORD: "CHANGE_ME"
  POSTGRES_USER: "vrooli"
  POSTGRES_DB: "${instance_name}_app"
EOF
    
    # Create configmap
    cat > "$package_path/kubernetes/configmaps/postgres-config.yaml" << EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: postgres-config
  namespace: ${CLIENT_NAME}
data:
  postgresql.conf: |
$(sed 's/^/    /' "$package_path/config/postgresql.conf")
EOF
    
    # Create PVC
    cat > "$package_path/kubernetes/manifests/pvc.yaml" << EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
  namespace: ${CLIENT_NAME}
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: standard
EOF
    
    # Create deployment
    cat > "$package_path/kubernetes/manifests/deployment.yaml" << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: ${CLIENT_NAME}
  labels:
    app: postgres
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:16-alpine
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_DB
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: POSTGRES_DB
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: POSTGRES_USER
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: POSTGRES_PASSWORD
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        - name: postgres-config
          mountPath: /etc/postgresql/postgresql.conf
          subPath: postgresql.conf
        livenessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - vrooli
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          exec:
            command:
            - pg_isready
            - -U
            - vrooli
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
      - name: postgres-config
        configMap:
          name: postgres-config
EOF
    
    # Create service
    cat > "$package_path/kubernetes/manifests/service.yaml" << EOF
apiVersion: v1
kind: Service
metadata:
  name: postgres-service
  namespace: ${CLIENT_NAME}
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
  type: ClusterIP
EOF
    
    # Create deployment script
    cat > "$package_path/scripts/deploy.sh" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

# Kubernetes Deployment Script
echo "Deploying PostgreSQL to Kubernetes..."

# Check if kubectl is available
if ! command -v kubectl >/dev/null 2>&1; then
    echo "Error: kubectl is not installed"
    exit 1
fi

# Check if we can connect to cluster  
if ! kubectl cluster-info >/dev/null 2>&1; then
    echo "Error: Cannot connect to Kubernetes cluster"
    exit 1
fi

cd "$(dirname "$0")/../kubernetes"

# Apply manifests
echo "Creating namespace..."
kubectl apply -f manifests/namespace.yaml

echo "Creating secret (update password first!)..."
kubectl apply -f secrets/postgres-secret.yaml

echo "Creating configmap..."
kubectl apply -f configmaps/postgres-config.yaml

echo "Creating PVC..."
kubectl apply -f manifests/pvc.yaml

echo "Creating deployment..."
kubectl apply -f manifests/deployment.yaml

echo "Creating service..."
kubectl apply -f manifests/service.yaml

# Wait for deployment
echo "Waiting for deployment to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/postgres -n ${CLIENT_NAME}

echo "✅ PostgreSQL deployed successfully"
echo ""
echo "Connection details:"
echo "  Service: postgres-service.${CLIENT_NAME}.svc.cluster.local"
echo "  Port: 5432"
echo "  Database: ${instance_name}_app"
echo "  Username: vrooli"
echo "  Password: [from secret]"
echo ""
echo "To connect from within cluster:"
echo "  psql -h postgres-service.${CLIENT_NAME}.svc.cluster.local -p 5432 -U vrooli -d ${instance_name}_app"
echo ""
echo "To port-forward for external access:"
echo "  kubectl port-forward -n ${CLIENT_NAME} service/postgres-service 5432:5432"
EOF
    
    chmod +x "$package_path/scripts/deploy.sh"
    
    log_success "Kubernetes deployment files created"
}

create_native_deployment() {
    local instance_name="$1"
    local package_path="$2"
    
    log_info "Creating native deployment files"
    
    # Create systemd service
    cat > "$package_path/systemd/postgresql-${CLIENT_NAME}.service" << EOF
[Unit]
Description=PostgreSQL database server for ${CLIENT_NAME}
Documentation=man:postgres(1)
After=network.target

[Service]
Type=notify
User=postgres
Group=postgres
ExecStart=/usr/bin/postgres -D /var/lib/postgresql/${CLIENT_NAME}/data -c config_file=/etc/postgresql/${CLIENT_NAME}/postgresql.conf
ExecReload=/bin/kill -HUP \$MAINPID
TimeoutSec=300
KillMode=mixed
KillSignal=SIGINT

# Auto-restart on failure
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF
    
    # Create init script
    cat > "$package_path/init/postgresql-${CLIENT_NAME}" << EOF
#!/bin/bash
# PostgreSQL init script for ${CLIENT_NAME}
# chkconfig: 35 99 99
# description: PostgreSQL database server for ${CLIENT_NAME}

. /etc/rc.d/init.d/functions

USER="postgres"
DAEMON="postgres"
ROOT_DIR="/var/lib/postgresql/${CLIENT_NAME}"
DATA_DIR="\$ROOT_DIR/data"
CONFIG_FILE="/etc/postgresql/${CLIENT_NAME}/postgresql.conf"

LOCK_FILE="/var/lock/subsys/postgresql-${CLIENT_NAME}"

start() {
    if [ -f \$LOCK_FILE ]; then
        echo "PostgreSQL for ${CLIENT_NAME} is already running."
        return 1
    fi
    
    echo -n "Starting PostgreSQL for ${CLIENT_NAME}: "
    runuser -l "\$USER" -c "\$DAEMON -D \$DATA_DIR -c config_file=\$CONFIG_FILE" && echo_success || echo_failure
    RETVAL=\$?
    echo
    [ \$RETVAL -eq 0 ] && touch \$LOCK_FILE
    return \$RETVAL
}

stop() {
    echo -n "Shutting down PostgreSQL for ${CLIENT_NAME}: "
    pid=\$(ps -aefw | grep "\$DAEMON" | grep -v " grep " | awk '{print \$2}')
    kill -9 \$pid > /dev/null 2>&1
    [ \$? -eq 0 ] && echo_success || echo_failure
    RETVAL=\$?
    echo
    [ \$RETVAL -eq 0 ] && rm -f \$LOCK_FILE
    return \$RETVAL
}

restart() {
    stop
    start
}

status() {
    if [ -f \$LOCK_FILE ]; then
        echo "PostgreSQL for ${CLIENT_NAME} is running."
    else
        echo "PostgreSQL for ${CLIENT_NAME} is stopped."
    fi
}

case "\$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    status)
        status
        ;;
    restart)
        restart
        ;;
    *)
        echo "Usage: {start|stop|status|restart}"
        exit 1
        ;;
esac

exit \$?
EOF
    
    chmod +x "$package_path/init/postgresql-${CLIENT_NAME}"
    
    # Create deployment script
    cat > "$package_path/scripts/deploy.sh" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

# Native Deployment Script
echo "Deploying PostgreSQL natively..."

# Check if PostgreSQL is installed
if ! command -v postgres >/dev/null 2>&1; then
    echo "Error: PostgreSQL is not installed"
    echo "Install PostgreSQL first: sudo apt-get install postgresql postgresql-contrib"
    exit 1
fi

# Check if running as root
if [[ $EUID -eq 0 ]]; then
    echo "Error: Do not run this script as root"
    echo "Run as a user with sudo privileges"
    exit 1
fi

# Load configuration
if [[ -f "../config/database.env" ]]; then
    set -a
    source "../config/database.env"
    set +a
    echo "Configuration loaded from database.env"
fi

CLIENT_NAME="${CLIENT_NAME:-client}"
DATA_DIR="/var/lib/postgresql/${CLIENT_NAME}/data"
CONFIG_DIR="/etc/postgresql/${CLIENT_NAME}"

echo "Setting up PostgreSQL for client: $CLIENT_NAME"

# Create directories
sudo mkdir -p "$DATA_DIR"
sudo mkdir -p "$CONFIG_DIR"
sudo chown postgres:postgres "$DATA_DIR"
sudo chown postgres:postgres "$CONFIG_DIR"

# Copy configuration
sudo cp "../config/postgresql.conf" "$CONFIG_DIR/"
sudo chown postgres:postgres "$CONFIG_DIR/postgresql.conf"

# Initialize database
if [[ ! -f "$DATA_DIR/PG_VERSION" ]]; then
    echo "Initializing database..."
    sudo -u postgres initdb -D "$DATA_DIR" --encoding=UTF8 --locale=en_US.UTF-8
fi

# Install systemd service
if [[ -f "/etc/systemd/system/postgresql-${CLIENT_NAME}.service" ]]; then
    echo "Systemd service already exists"
else
    echo "Installing systemd service..."
    sudo cp "../systemd/postgresql-${CLIENT_NAME}.service" "/etc/systemd/system/"
    sudo systemctl daemon-reload
    sudo systemctl enable "postgresql-${CLIENT_NAME}"
fi

# Start service
echo "Starting PostgreSQL service..."
sudo systemctl start "postgresql-${CLIENT_NAME}"

# Wait for service to be ready
sleep 5

# Test connection
if sudo -u postgres pg_isready -p 5432; then
    echo "✅ PostgreSQL is running and ready"
    echo ""
    echo "Service name: postgresql-${CLIENT_NAME}"
    echo "Data directory: $DATA_DIR"
    echo "Config directory: $CONFIG_DIR"
    echo ""
    echo "To connect: sudo -u postgres psql"
    echo "To stop: sudo systemctl stop postgresql-${CLIENT_NAME}"
    echo "To check status: sudo systemctl status postgresql-${CLIENT_NAME}"
else
    echo "❌ PostgreSQL deployment failed"
    sudo systemctl status "postgresql-${CLIENT_NAME}"
    exit 1
fi
EOF
    
    chmod +x "$package_path/scripts/deploy.sh"
    
    log_success "Native deployment files created"
}

create_documentation() {
    local instance_name="$1"
    local package_path="$2"
    
    log_step "Creating deployment documentation"
    
    # Create main README
    cat > "$package_path/README.md" << EOF
# PostgreSQL Deployment Package for $CLIENT_NAME

This package contains everything needed to deploy the PostgreSQL database for $CLIENT_NAME.

## Package Information

- **Client**: $CLIENT_NAME
- **Instance**: $instance_name
- **Version**: $VERSION
- **Environment**: $ENVIRONMENT
- **Deployment Type**: $DEPLOYMENT_TYPE
- **Created**: $(date)

## Contents

- \`config/\` - Configuration files and environment variables
- \`schema/\` - Database schema and structure
- \`migrations/\` - Database migration files
- \`seeds/\` - Initial data and sample data
- \`scripts/\` - Deployment and management scripts
- \`docs/\` - Documentation and guides
EOF

    case "$DEPLOYMENT_TYPE" in
        "docker")
            cat >> "$package_path/README.md" << EOF
- \`docker/\` - Docker deployment files (Dockerfile, docker-compose.yml)
EOF
            ;;
        "kubernetes")
            cat >> "$package_path/README.md" << EOF
- \`kubernetes/\` - Kubernetes deployment manifests and configurations
EOF
            ;;
        "native")
            cat >> "$package_path/README.md" << EOF
- \`systemd/\` - Systemd service files
- \`init/\` - Init scripts for older systems
EOF
            ;;
    esac

    if [[ "$INCLUDE_BACKUPS" == "true" ]]; then
        cat >> "$package_path/README.md" << EOF
- \`backups/\` - Database backup files
EOF
    fi

    cat >> "$package_path/README.md" << EOF

## Quick Start

1. **Review Configuration**
   \`\`\`bash
   # Edit database password and other settings
   nano config/database.env
   \`\`\`

2. **Deploy**
   \`\`\`bash
   cd scripts
   ./deploy.sh
   \`\`\`

3. **Verify Deployment**
   \`\`\`bash
   # Test database connection
   psql -h localhost -p 5432 -U vrooli -d ${instance_name}_app
   \`\`\`

## Detailed Instructions

### Prerequisites

EOF

    case "$DEPLOYMENT_TYPE" in
        "docker")
            cat >> "$package_path/README.md" << EOF
- Docker Engine 20.10+
- Docker Compose 2.0+
EOF
            ;;
        "kubernetes")
            cat >> "$package_path/README.md" << EOF
- Kubernetes cluster 1.20+
- kubectl configured with cluster access
- Storage class for persistent volumes
EOF
            ;;
        "native")
            cat >> "$package_path/README.md" << EOF
- PostgreSQL 12+ installed
- systemd (for service management)
- sudo privileges
EOF
            ;;
    esac

    cat >> "$package_path/README.md" << EOF

### Configuration

1. **Database Settings**
   - Edit \`config/database.env\` with your desired settings
   - **Important**: Change the default password!

2. **PostgreSQL Configuration**
   - Review \`config/postgresql.conf\` for performance settings
   - Adjust memory and connection settings as needed

### Deployment

Run the deployment script:
\`\`\`bash
cd scripts
./deploy.sh
\`\`\`

The script will:
- Validate prerequisites
- Set up the database
- Apply schema and migrations
- Start the PostgreSQL service
- Perform health checks

### Post-Deployment

1. **Test Connection**
   \`\`\`bash
   psql -h localhost -p 5432 -U vrooli -d ${instance_name}_app
   \`\`\`

2. **Apply Migrations** (if needed)
   \`\`\`bash
   # Copy migration files to server
   # Run migrations using your application's migration tool
   \`\`\`

3. **Load Initial Data** (if needed)
   \`\`\`bash
   psql -h localhost -p 5432 -U vrooli -d ${instance_name}_app < seeds/sample_data.sql
   \`\`\`

### Backup and Restore

EOF

    if [[ "$INCLUDE_BACKUPS" == "true" ]]; then
        cat >> "$package_path/README.md" << EOF
This package includes backup files in the \`backups/\` directory.

To restore from backup:
\`\`\`bash
psql -h localhost -p 5432 -U vrooli -d ${instance_name}_app < backups/deployment_*/full.sql
\`\`\`
EOF
    fi

    cat >> "$package_path/README.md" << EOF

To create new backups:
\`\`\`bash
pg_dump -h localhost -p 5432 -U vrooli -d ${instance_name}_app > backup_\$(date +%Y%m%d).sql
\`\`\`

### Troubleshooting

Common issues and solutions:

- **Connection refused**: Check if PostgreSQL is running and listening on the correct port
- **Authentication failed**: Verify username/password in configuration
- **Permission denied**: Ensure proper file permissions and ownership
EOF

    case "$DEPLOYMENT_TYPE" in
        "docker")
            cat >> "$package_path/README.md" << EOF
- **Container won't start**: Check Docker logs with \`docker-compose logs postgres\`
EOF
            ;;
        "kubernetes")
            cat >> "$package_path/README.md" << EOF
- **Pod not starting**: Check pod logs with \`kubectl logs -n $CLIENT_NAME <pod-name>\`
- **Storage issues**: Verify persistent volume claims and storage class
EOF
            ;;
        "native")
            cat >> "$package_path/README.md" << EOF
- **Service won't start**: Check service status with \`systemctl status postgresql-$CLIENT_NAME\`
- **Permission issues**: Ensure postgres user owns data directory
EOF
            ;;
    esac

    cat >> "$package_path/README.md" << EOF

### Support

For support with this deployment:
1. Check the troubleshooting section above
2. Review the configuration files
3. Check logs for error messages
4. Contact your Vrooli administrator

---
*This deployment package was generated by Vrooli PostgreSQL Resource v$VERSION*
EOF
    
    # Create deployment guide
    cat > "$package_path/docs/deployment-guide.md" << EOF
# Deployment Guide for $CLIENT_NAME PostgreSQL

## Overview

This guide provides detailed instructions for deploying the PostgreSQL database for $CLIENT_NAME using $DEPLOYMENT_TYPE deployment.

## Architecture

- **Database Engine**: PostgreSQL 16
- **Template**: $ENVIRONMENT
- **Port**: 5432 (internal)
- **Data Persistence**: $(if [[ "$DEPLOYMENT_TYPE" == "kubernetes" ]]; then echo "Persistent Volumes"; elif [[ "$DEPLOYMENT_TYPE" == "docker" ]]; then echo "Docker Volumes"; else echo "Local Filesystem"; fi)

## Security Considerations

1. **Change Default Passwords**: Update all default passwords before deployment
2. **Network Security**: Configure firewall rules to restrict database access
3. **SSL/TLS**: Consider enabling SSL for production deployments
4. **User Permissions**: Follow principle of least privilege for database users
5. **Backup Encryption**: Encrypt backups for sensitive data

## Performance Tuning

The included PostgreSQL configuration is optimized for $ENVIRONMENT environments.

Key settings:
- **shared_buffers**: Optimized for available memory
- **work_mem**: Tuned for query performance
- **maintenance_work_mem**: Set for maintenance operations
- **checkpoint_completion_target**: Balanced for performance and reliability

## Monitoring

Recommended monitoring practices:
- Monitor database connections
- Track query performance
- Monitor disk usage
- Set up log rotation
- Monitor backup success

## Maintenance

Regular maintenance tasks:
- Update PostgreSQL security patches
- Monitor and clean up old log files
- Verify backup integrity
- Update statistics with ANALYZE
- Monitor and maintain indexes

EOF
    
    log_success "Documentation created"
}

create_verification_script() {
    local instance_name="$1"
    local package_path="$2"
    
    log_step "Creating verification script"
    
    cat > "$package_path/scripts/verify.sh" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail

# Deployment Verification Script
echo "Verifying PostgreSQL deployment..."

# Load configuration
if [[ -f "../config/database.env" ]]; then
    set -a
    source "../config/database.env"
    set +a
fi

HOST="${DATABASE_HOST:-localhost}"
PORT="${DATABASE_PORT:-5432}"
USER="${DATABASE_USER:-vrooli}"
DATABASE="${DATABASE_NAME:-app}"

echo "Testing connection to $HOST:$PORT..."

# Test 1: Basic connectivity
if pg_isready -h "$HOST" -p "$PORT" -U "$USER"; then
    echo "✅ PostgreSQL is accepting connections"
else
    echo "❌ Cannot connect to PostgreSQL"
    exit 1
fi

# Test 2: Authentication
if PGPASSWORD="$DATABASE_PASSWORD" psql -h "$HOST" -p "$PORT" -U "$USER" -d "$DATABASE" -c "SELECT version();" >/dev/null 2>&1; then
    echo "✅ Authentication successful"
else
    echo "❌ Authentication failed"
    exit 1
fi

# Test 3: Database operations
if PGPASSWORD="$DATABASE_PASSWORD" psql -h "$HOST" -p "$PORT" -U "$USER" -d "$DATABASE" -c "CREATE TABLE IF NOT EXISTS test_table (id SERIAL, test TEXT); DROP TABLE test_table;" >/dev/null 2>&1; then
    echo "✅ Database operations working"
else
    echo "❌ Database operations failed"
    exit 1
fi

# Test 4: Check configuration
CONFIG_CHECK=$(PGPASSWORD="$DATABASE_PASSWORD" psql -h "$HOST" -p "$PORT" -U "$USER" -d "$DATABASE" -t -c "SHOW shared_buffers;" 2>/dev/null | tr -d ' ')
if [[ -n "$CONFIG_CHECK" ]]; then
    echo "✅ Configuration loaded (shared_buffers: $CONFIG_CHECK)"
else
    echo "⚠️  Could not verify configuration"
fi

# Test 5: Performance test
echo "Running simple performance test..."
START_TIME=$(date +%s)
PGPASSWORD="$DATABASE_PASSWORD" psql -h "$HOST" -p "$PORT" -U "$USER" -d "$DATABASE" -c "SELECT generate_series(1,1000);" >/dev/null 2>&1
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

if [[ $DURATION -lt 5 ]]; then
    echo "✅ Performance test passed (${DURATION}s)"
else
    echo "⚠️  Performance test slow (${DURATION}s)"
fi

echo ""
echo "🎉 PostgreSQL deployment verification completed successfully!"
echo ""
echo "Connection details:"
echo "  Host: $HOST"
echo "  Port: $PORT"
echo "  Database: $DATABASE"
echo "  User: $USER"
echo ""
echo "Next steps:"
echo "1. Run your application's database migrations"
echo "2. Load initial data if needed"
echo "3. Configure monitoring and backups"
echo "4. Update firewall rules as needed"
EOF
    
    chmod +x "$package_path/scripts/verify.sh"
    
    log_success "Verification script created"
}

create_package_manifest() {
    local package_name="$1"
    local package_path="$2"
    
    log_step "Creating package manifest"
    
    cat > "$package_path/MANIFEST.json" << EOF
{
  "package": {
    "name": "$package_name",
    "version": "$VERSION",
    "created": "$(date -Iseconds)",
    "type": "vrooli-postgres-deployment"
  },
  "client": {
    "name": "$CLIENT_NAME",
    "instance": "$INSTANCE_NAME"
  },
  "deployment": {
    "type": "$DEPLOYMENT_TYPE",
    "environment": "$ENVIRONMENT"
  },
  "contents": {
    "config": true,
    "schema": true,
    "migrations": true,
    "seeds": true,
    "scripts": true,
    "documentation": true,
    "data": $INCLUDE_DATA,
    "backups": $INCLUDE_BACKUPS
  },
  "files": {
EOF
    
    # Generate file list
    find "$package_path" -type f -exec basename {} \; | sort | while read -r file; do
        echo "    \"$file\": \"$(date -r "$package_path/$file" -Iseconds 2>/dev/null || echo "unknown")\"," >> "$package_path/MANIFEST.json"
    done
    
    # Remove last comma and close JSON
    sed -i '$ s/,$//' "$package_path/MANIFEST.json"
    cat >> "$package_path/MANIFEST.json" << EOF
  },
  "checksum": "$(find "$package_path" -type f -exec sha256sum {} \; | sha256sum | cut -d' ' -f1)"
}
EOF
    
    log_success "Package manifest created"
}

compress_package() {
    local package_path="$1"
    local package_name="$2"
    
    if [[ "$COMPRESS" != "true" ]]; then
        log_info "Package compression skipped"
        return 0
    fi
    
    log_step "Compressing package"
    
    local archive_name="${package_name}-${VERSION}.tar.gz"
    local archive_path="${OUTPUT_DIR}/${archive_name}"
    
    cd "$(dirname "$package_path")"
    if tar -czf "$archive_path" "$(basename "$package_path")"; then
        log_success "Package compressed: $archive_path"
        
        # Generate checksum
        local checksum=$(sha256sum "$archive_path" | cut -d' ' -f1)
        echo "$checksum  $archive_name" > "${archive_path}.sha256"
        
        log_info "Archive size: $(ls -lh "$archive_path" | awk '{print $5}')"
        log_info "SHA256: $checksum"
    else
        log_error "Failed to compress package"
        return 1
    fi
}

#######################################
# Main packaging function
#######################################
main() {
    echo "PostgreSQL Client Deployment Packaging"
    echo "======================================="
    echo "Instance: $INSTANCE_NAME"
    echo "Client: $CLIENT_NAME"
    echo "Deployment Type: $DEPLOYMENT_TYPE"
    echo "Environment: $ENVIRONMENT"
    echo "Version: $VERSION"
    echo "Include Data: $INCLUDE_DATA"
    echo "Include Backups: $INCLUDE_BACKUPS"
    echo "Compress: $COMPRESS"
    echo ""
    
    # Create package name and path
    local package_name="${CLIENT_NAME}-postgres-${DEPLOYMENT_TYPE}-${ENVIRONMENT}"
    local package_path="${OUTPUT_DIR}/${package_name}"
    
    # Create output directory
    mkdir -p "$OUTPUT_DIR"
    
    # Remove existing package if it exists
    if [[ -d "$package_path" ]]; then
        log_info "Removing existing package: $package_path"
        trash::safe_remove "$package_path" --temp
    fi
    
    # Execute packaging steps
    create_package_structure "$package_name" "$package_path" || exit 1
    export_instance_config "$INSTANCE_NAME" "$package_path" || exit 1
    export_schema "$INSTANCE_NAME" "$package_path" || exit 1
    export_migrations "$INSTANCE_NAME" "$package_path" || exit 1
    export_data "$INSTANCE_NAME" "$package_path" || exit 1
    export_backups "$INSTANCE_NAME" "$package_path" || exit 1
    create_deployment_scripts "$INSTANCE_NAME" "$package_path" || exit 1
    create_documentation "$INSTANCE_NAME" "$package_path" || exit 1
    create_verification_script "$INSTANCE_NAME" "$package_path" || exit 1
    create_package_manifest "$package_name" "$package_path" || exit 1
    compress_package "$package_path" "$package_name" || exit 1
    
    # Final summary
    echo ""
    echo "🎉 Package Creation Complete!"
    echo "============================="
    echo "Package: $package_name"
    echo "Location: $package_path"
    echo "Size: $(du -sh "$package_path" | cut -f1)"
    
    if [[ "$COMPRESS" == "true" ]]; then
        echo "Archive: ${OUTPUT_DIR}/${package_name}-${VERSION}.tar.gz"
    fi
    
    echo ""
    echo "Package Contents:"
    echo "- Configuration files and environment settings"
    echo "- Database schema and structure"  
    echo "- Migration files and history"
    echo "- Deployment scripts for $DEPLOYMENT_TYPE"
    echo "- Comprehensive documentation"
    echo "- Verification and testing scripts"
    
    if [[ "$INCLUDE_DATA" == "true" ]]; then
        echo "- Current database data"
    fi
    
    if [[ "$INCLUDE_BACKUPS" == "true" ]]; then
        echo "- Recent backup files"
    fi
    
    echo ""
    echo "Next Steps:"
    echo "1. Review the package contents"
    echo "2. Test deployment in a staging environment"
    echo "3. Transfer package to target deployment server"
    echo "4. Follow deployment guide in README.md"
    echo "5. Run verification script after deployment"
    
    log_success "PostgreSQL deployment package created successfully!"
}

# Execute main function
main "$@"