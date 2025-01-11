#!/bin/sh
# Creates initial migration for Prisma.
# NOTE 1: Before running this script, make sure that:
# 1. schema.prisma file is up to date
# 2. database is empty
# NOTE 2: This script should be run from the server's docker container.
HERE=$(dirname $0)
. "${HERE}/utils.sh"

PRISMA_SCHEMA_FILE="dist/db/schema.prisma"

cd ${PROJECT_DIR}/packages/server

# Prompt user to confirm
read -p "Are you sure you want to create an initial migration? THIS WILL ERASE ALL PREVIOUS MIGRATIONS! (y/n) " -n 1 -r
echo
if is_yes "$REPLY"; then
    exit 1
fi

# Delete all migrations in src/db/migrations folder by moving them to /tmp
if [ -d "src/db/migrations" ]; then
    info "Migrations folder exists. Moving all migrations to /tmp"
    mv src/db/migrations/* /tmp
fi

# Copy schema.prisma to dist folder
if [ -f "src/db/schema.prisma" ]; then
    info "Copying schema.prisma to dist folder"
    cp src/db/schema.prisma dist/db/schema.prisma
fi

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
