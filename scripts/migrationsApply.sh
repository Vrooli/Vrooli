#!/bin/bash
# Locates all Prisma migrations up to the one you specify and applies them.
# Usage: docker exec -i server sh < $( ./scripts/migrationsApply.sh "<migration_name>" )
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"

# Load variables from .env file
if [ -f "${HERE}/../.env-dev" ]; then
    . "${HERE}/../.env-dev"
else
    error "Could not find .env-dev file. This may break the script."
fi

# Read argument
if [ $# -ne 1 ]; then
    error "Usage: $0 <migration_name>"
    exit 1
fi

# Get migration name
MIGRATION_NAME=$1
# Set DB_URL environment variable
DB_URL="postgresql://${DB_USER}:${DB_PASSWORD}@localhost:${PORT_DB}"

# Navigate to correct directory
cd ${HERE}/../packages/server

# Make copy of schema.prisma file
cp src/db/schema.prisma src/db/schema.prisma.bak
# Set trap to restore schema.prisma file on exit
trap "mv src/db/schema.prisma.bak src/db/schema.prisma && rm src/db/schema.prisma.bak" EXIT
# Replace DB_URL in schema.prisma file
sed -i "s|env(\"DB_URL\")|\"${DB_URL}\"|g" src/db/schema.prisma
head -n 20 src/db/schema.prisma

# Loop through migrations folder. Every subfolder is a migration.
for migration in $(ls -d ./src/db/migrations/*/); do
    # Get migration name
    migration_name=$(basename ${migration})
    # Apply migration
    info "Applying migration ${migration_name}"
    prisma migrate resolve --applied "${migration_name}"
    # If migration name is equal to the argument passed into the script, stop looping
    if [ "${migration_name}" = "${MIGRATION_NAME}" ]; then
        success "Migration applies complete!"
        break
    fi
done
