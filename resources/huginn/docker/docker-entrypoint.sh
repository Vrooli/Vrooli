#!/bin/bash
set -e
# Enhanced entrypoint for Huginn with Vrooli integration

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "/host/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

echo "Starting Huginn with enhanced entrypoint..."

# Set up PATH to include host directories
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/host/usr/bin:/host/bin:$PATH"

# Set up environment for Huginn
export RAILS_ENV="${RAILS_ENV:-production}"
export DATABASE_ADAPTER="${DATABASE_ADAPTER:-postgresql}"
export DATABASE_ENCODING="${DATABASE_ENCODING:-utf8}"
export DATABASE_RECONNECT="${DATABASE_RECONNECT:-true}"
export DATABASE_POOL="${DATABASE_POOL:-20}"

# Ensure required environment variables are set
: "${DATABASE_USERNAME:?DATABASE_USERNAME must be set}"
: "${DATABASE_PASSWORD:?DATABASE_PASSWORD must be set}"
: "${DATABASE_NAME:?DATABASE_NAME must be set}"
: "${DATABASE_HOST:?DATABASE_HOST must be set}"
: "${DATABASE_PORT:?DATABASE_PORT must be set}"

# Function to wait for database
wait_for_database() {
    echo "Waiting for PostgreSQL to be ready..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if pg_isready -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USERNAME" 2>/dev/null; then
            echo "PostgreSQL is ready!"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo "Waiting for PostgreSQL... (attempt $attempt/$max_attempts)"
        sleep 2
    done
    
    echo "ERROR: PostgreSQL did not become ready in time"
    return 1
}

# Handle running as root
if [[ "$(id -u)" = "0" ]]; then
    echo "Running as root, setting up permissions..."
    
    # Fix ownership of app directory
    chown -R huginn:huginn /app 2>/dev/null || true
    
    # Fix Docker socket access if socket exists
    if [[ -S "/var/run/docker.sock" ]]; then
        # Get the group ID of the docker socket
        DOCKER_GID=$(stat -c '%g' /var/run/docker.sock)
        
        # Create/update docker group with correct GID
        if ! getent group docker >/dev/null 2>&1; then
            groupadd -g "$DOCKER_GID" docker
        else
            groupmod -g "$DOCKER_GID" docker 2>/dev/null || true
        fi
        
        # Add huginn user to docker group
        usermod -a -G docker huginn
        
        echo "Added huginn user to docker group (GID: $DOCKER_GID)"
    fi
    
    # Create required directories
    mkdir -p /app/log /app/tmp/pids /app/tmp/cache /app/uploads
    chown -R huginn:huginn /app/log /app/tmp /app/uploads
    
    # Drop to huginn user and re-run this script
    echo "Dropping privileges to huginn user..."
    exec su -s /bin/bash huginn -c "$0 $*"
fi

# Running as huginn user from here
echo "Running as user: $(whoami)"

# Wait for database to be ready
wait_for_database || exit 1

# Check if database needs initialization
echo "Checking database status..."
if ! bundle exec rails db:version >/dev/null 2>&1; then
    echo "Database needs initialization, running setup..."
    bundle exec rails db:create db:schema:load db:seed
else
    echo "Database exists, running migrations..."
    bundle exec rails db:migrate
fi

# Create default admin user if it doesn't exist
echo "Ensuring default admin user exists..."
bundle exec rails runner "
  admin = User.find_by(email: 'admin@localhost')
  unless admin
    admin = User.create!(
      email: 'admin@localhost',
      password: ENV['SEED_PASSWORD'] || 'vrooli_huginn_secure_2025',
      password_confirmation: ENV['SEED_PASSWORD'] || 'vrooli_huginn_secure_2025',
      username: 'admin',
      admin: true
    )
    puts 'Created default admin user'
  else
    puts 'Admin user already exists'
  end
" || echo "Warning: Could not ensure admin user"

# Precompile assets if needed
if [[ "$RAILS_ENV" == "production" ]] && [[ ! -d "public/assets" ]]; then
    echo "Precompiling assets..."
    bundle exec rails assets:precompile
fi

# Clean up old PID files
trash::safe_remove /app/tmp/pids/server.pid --temp

# Export host directories in PATH for child processes
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/host/usr/bin:/host/bin:$PATH"

# Start Huginn with the provided command or default
if [ $# -eq 0 ]; then
    echo "Starting Huginn web server..."
    exec bundle exec rails server -b 0.0.0.0 -p 3000
else
    echo "Running command: $@"
    exec "$@"
fi