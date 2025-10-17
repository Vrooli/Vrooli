#!/bin/bash
# Get PostgreSQL connection URL from resource credentials

set -e

# Get postgres credentials from resource
CREDS=$(resource-postgres credentials 2>/dev/null)

# Extract connection details (handle missing user/password by defaulting)
HOST=$(echo "$CREDS" | jq -r '.connections[0].connection.host // "localhost"')
PORT=$(echo "$CREDS" | jq -r '.connections[0].connection.port // 5433')
DB=$(echo "$CREDS" | jq -r '.connections[0].connection.database // "vrooli"')

# Get user and password from docker container env if available
if docker ps --filter "name=vrooli-postgres" --format "{{.Names}}" | grep -q "vrooli-postgres"; then
    CONTAINER=$(docker ps --filter "name=vrooli-postgres" --format "{{.Names}}" | head -1)
    USER=$(docker exec "$CONTAINER" env 2>/dev/null | grep POSTGRES_USER | cut -d= -f2 || echo "postgres")
    PASS=$(docker exec "$CONTAINER" env 2>/dev/null | grep POSTGRES_PASSWORD | cut -d= -f2 || echo "postgres")
else
    # Fallback defaults
    USER="postgres"
    PASS="postgres"
fi

# Build and output connection string
echo "postgres://${USER}:${PASS}@${HOST}:${PORT}/${DB}?sslmode=disable"
