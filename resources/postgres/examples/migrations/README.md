# PostgreSQL Migration Examples

This directory contains example migration files demonstrating best practices for database schema management in Vrooli PostgreSQL instances.

## Migration Naming Convention

Use a consistent naming pattern for migrations:
- `NNN_descriptive_name.sql` - Forward migrations (NNN = 001, 002, 003...)
- `rollback_NNN_descriptive_name.sql` - Rollback scripts

## Migration Best Practices

### 1. Always Use IF EXISTS/IF NOT EXISTS
```sql
CREATE TABLE IF NOT EXISTS table_name (...);
ALTER TABLE users ADD COLUMN IF NOT EXISTS column_name;
```

### 2. Include Migration Metadata
```sql
-- Migration: 001_create_users_table.sql
-- Description: Create users table with authentication fields
-- Author: Your Name
-- Date: YYYY-MM-DD
```

### 3. Make Migrations Idempotent
Migrations should be safe to run multiple times:
```sql
INSERT INTO roles (name, description) VALUES ('admin', 'Administrator')
ON CONFLICT (name) DO NOTHING;
```

### 4. Consider Rollback Strategy
- Simple changes: Direct rollback (DROP TABLE, DROP COLUMN)
- Data migrations: May require data backup before rollback
- Complex changes: Might need multi-step rollback process

### 5. Performance Considerations
```sql
-- Create indexes for foreign keys and commonly queried columns
CREATE INDEX idx_users_email ON users(email);

-- Use CONCURRENTLY for large tables in production
CREATE INDEX CONCURRENTLY idx_large_table_column ON large_table(column);
```

## Running Migrations

### Initialize Migration System
```bash
./manage.sh --action migrate-init --instance client-name
```

### Run All Migrations
```bash
./manage.sh --action migrate --instance client-name --migrations-dir ./examples/migrations
```

### Check Migration Status
```bash
./manage.sh --action migrate-status --instance client-name
```

### Rollback Specific Migration
```bash
./manage.sh --action migrate-rollback --instance client-name --migration 003
```

## Example Files

1. **001_create_users_table.sql**
   - Creates base users table with authentication fields
   - Demonstrates indexes, triggers, and comments

2. **002_create_roles_permissions.sql**
   - Implements Role-Based Access Control (RBAC)
   - Shows junction tables and foreign key relationships
   - Includes seed data for default roles

3. **003_add_user_preferences.sql**
   - Adds user preferences with JSONB for flexible storage
   - Shows data migration (creating preferences for existing users)
   - Demonstrates ALTER TABLE operations

4. **rollback_003_add_user_preferences.sql**
   - Example rollback script
   - Shows considerations for data loss during rollback

## Tips for Client-Specific Migrations

When creating migrations for client projects:

1. **Namespace Tables**: Consider prefixing tables with client identifier
   ```sql
   CREATE TABLE realestate_properties (...);
   CREATE TABLE realestate_leads (...);
   ```

2. **Environment-Specific Data**: Use conditional logic
   ```sql
   -- Only insert test data in development
   DO $$
   BEGIN
     IF current_database() LIKE '%dev%' THEN
       INSERT INTO users (email, username) VALUES ('test@example.com', 'testuser');
     END IF;
   END $$;
   ```

3. **Version Your Schema**: Track schema version
   ```sql
   CREATE TABLE IF NOT EXISTS schema_version (
     version INTEGER PRIMARY KEY,
     applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   );
   ```

## Integration with n8n/Node-RED

When migrations create tables for automation workflows:

```sql
-- Table for n8n webhook data
CREATE TABLE webhook_events (
    id SERIAL PRIMARY KEY,
    workflow_id VARCHAR(100),
    event_type VARCHAR(50),
    payload JSONB,
    processed BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for unprocessed events
CREATE INDEX idx_webhook_unprocessed 
ON webhook_events(processed, created_at) 
WHERE processed = false;
```