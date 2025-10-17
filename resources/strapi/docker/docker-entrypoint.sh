#!/bin/sh
set -e

# Initialize Strapi if not already initialized
if [ ! -f "/app/.strapi-initialized" ]; then
    echo "Initializing Strapi for the first time..."
    
    # Create new Strapi project if needed
    if [ ! -f "/app/package.json" ] || [ ! -f "/app/config/database.js" ]; then
        echo "Creating new Strapi project..."
        npx create-strapi-app@latest . --no-run --dbclient=postgres \
            --dbhost="${POSTGRES_HOST:-postgres}" \
            --dbport="${POSTGRES_PORT:-5432}" \
            --dbname="${STRAPI_DATABASE_NAME:-strapi}" \
            --dbusername="${POSTGRES_USER:-postgres}" \
            --dbpassword="${POSTGRES_PASSWORD:-postgres}" \
            --dbssl=false
    fi
    
    # Build Strapi
    echo "Building Strapi..."
    npm run build
    
    # Mark as initialized
    touch /app/.strapi-initialized
    
    # Create admin user if credentials provided
    if [ -n "${STRAPI_ADMIN_EMAIL}" ] && [ -n "${STRAPI_ADMIN_PASSWORD}" ]; then
        echo "Creating admin user..."
        npm run strapi admin:create-user -- \
            --firstname="Admin" \
            --lastname="User" \
            --email="${STRAPI_ADMIN_EMAIL}" \
            --password="${STRAPI_ADMIN_PASSWORD}" || true
    fi
else
    echo "Strapi already initialized, starting..."
fi

# Execute the main command
exec "$@"