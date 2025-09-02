#\!/bin/bash
# Simulate the lifecycle.sh postgres export logic

declare -A RESOURCE_PORTS
RESOURCE_PORTS[postgres]=5433

postgres_port="${RESOURCE_PORTS[postgres]:-5433}"
if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' | grep -q "vrooli-postgres-main"; then
    container_env=$(docker inspect vrooli-postgres-main --format '{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null)
    
    db_user=$(echo "$container_env" | grep '^POSTGRES_USER=' | cut -d'=' -f2)
    db_pass=$(echo "$container_env" | grep '^POSTGRES_PASSWORD=' | cut -d'=' -f2)
    db_name=$(echo "$container_env" | grep '^POSTGRES_DB=' | cut -d'=' -f2)
    
    if [[ -n "$db_user" && -n "$db_pass" && -n "$db_name" ]]; then
        export POSTGRES_URL="postgres://${db_user}:${db_pass}@localhost:${postgres_port}/${db_name}?sslmode=disable"
        export DATABASE_URL="$POSTGRES_URL"
        export POSTGRES_USER="$db_user"
        export POSTGRES_PASSWORD="$db_pass"
        export POSTGRES_DB="$db_name"
        echo "Exported:"
        echo "  POSTGRES_URL=$POSTGRES_URL"
        echo "  DATABASE_URL=$DATABASE_URL"
        echo "  POSTGRES_DB=$POSTGRES_DB"
    else
        echo "Failed to get credentials"
    fi
else
    echo "Container not found"
fi
