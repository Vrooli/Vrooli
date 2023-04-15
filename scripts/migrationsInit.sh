#!/bin/sh
# Creates initial migration for Prisma.
# NOTE 1: Before running this script, make sure that:
# 1. schema.prisma file is up to date
# 2. database is empty
# NOTE 2: This script should be run from the server's docker container.
HERE=$(dirname $0)
source "${HERE}/prettify.sh"

PRISMA_SCHEMA_FILE="src/db/schema.prisma"

cd ${PROJECT_DIR}/packages/server

# Prompt user to confirm
read -p "Are you sure you want to create an initial migration? THIS WILL ERASE ALL PREVIOUS MIGRATIONS! (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
fi

# Delete all migrations in src/db/migrations folder by moving them to /tmp
info "Deleting existing migrations..."
mv src/db/migrations/* /tmp

# Migrate without applying schema
info "Creating initial migration..."
prisma migrate dev --name init --create-only --schema ${PRISMA_SCHEMA_FILE}

# Check if migration was created
if [ $? -ne 0 ]; then
    error "Failed to create initial migration"
    exit 1
fi

# Find the migration directory that was just created.
# It will be the only directory in the src/db/migrations folder.
MIGRATION_DIR=$(ls -d ./src/db/migrations/*/)

# Now we must add the line "CREATE EXTENSION IF NOT EXISTS citext;"
# to the beginning of the migration file. This is required for case-insensitive
# email addresses, and maybe other things in the future.
sed -i "1i CREATE EXTENSION IF NOT EXISTS citext;" ${MIGRATION_DIR}/migration.sql

# Now we can apply the migration
info "Applying initial migration..."
prisma migrate dev --name init --schema ${PRISMA_SCHEMA_FILE}

# Check if migration was applied
if [ $? -ne 0 ]; then
    error "Failed to apply initial migration"
    exit 1
else
    success "Initial migration applied successfully!"
fi
