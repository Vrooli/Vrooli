# Schema Evolution

Comprehensive guide to database schema evolution, migration strategies, versioning, and backwards compatibility for Vrooli's data model.

## 🔄 Migration Strategy

### **Migration Philosophy**

Vrooli follows a **zero-downtime migration** approach with these core principles:

1. **Backwards Compatibility**: New schema versions support old application code
2. **Forward Compatibility**: Old schema versions handle new data gracefully  
3. **Incremental Changes**: Small, atomic changes over large migrations
4. **Rollback Safety**: All migrations can be safely reversed
5. **Data Preservation**: Never lose user data during migrations

### **Migration Types**

#### **Safe Migrations (No Downtime)**
```sql
-- Adding nullable columns
ALTER TABLE user ADD COLUMN bio VARCHAR(2048);

-- Adding new tables
CREATE TABLE user_preferences (
    id BIGINT PRIMARY KEY,
    userId BIGINT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    preferences JSON DEFAULT '{}',
    createdAt TIMESTAMPTZ DEFAULT NOW(),
    updatedAt TIMESTAMPTZ DEFAULT NOW()
);

-- Adding indexes
CREATE INDEX CONCURRENTLY idx_resource_score ON resource(score DESC);

-- Creating new enums
CREATE TYPE "UserTheme" AS ENUM ('light', 'dark', 'auto');
```

#### **Careful Migrations (Minimal Downtime)**
```sql
-- Adding non-nullable columns with defaults
ALTER TABLE resource ADD COLUMN priority INTEGER DEFAULT 0 NOT NULL;

-- Renaming columns (requires application coordination)
ALTER TABLE resource RENAME COLUMN "name" TO "title";

-- Adding constraints
ALTER TABLE user ADD CONSTRAINT check_email_format 
CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');
```

#### **Complex Migrations (Planned Downtime)**
```sql
-- Changing column types
ALTER TABLE resource ALTER COLUMN score TYPE DECIMAL(10,2);

-- Dropping columns (after ensuring no application usage)
ALTER TABLE user DROP COLUMN deprecated_field;

-- Restructuring relationships
-- (Requires multi-step migration process)
```

## 📋 Migration Procedures

### **Pre-Migration Checklist**

```bash
#!/bin/bash
# pre-migration-checklist.sh

echo "🔍 Pre-Migration Checklist"

# 1. Backup verification
echo "1. Verifying recent backup..."
LATEST_BACKUP=$(ls -t /backup/postgresql/ | head -1)
if [ -z "$LATEST_BACKUP" ]; then
    echo "❌ No recent backup found!"
    exit 1
fi
echo "✅ Latest backup: $LATEST_BACKUP"

# 2. Test migration on staging
echo "2. Testing migration on staging..."
cd packages/server
pnpm prisma migrate deploy --preview-feature
if [ $? -ne 0 ]; then
    echo "❌ Migration failed on staging!"
    exit 1
fi
echo "✅ Migration tested successfully"

# 3. Check application compatibility
echo "3. Checking application compatibility..."
pnpm run type-check
if [ $? -ne 0 ]; then
    echo "❌ Type checking failed!"
    exit 1
fi
echo "✅ Application compatibility confirmed"

# 4. Verify rollback procedure
echo "4. Verifying rollback procedure..."
if [ ! -f "migrations/rollback-$(date +%Y%m%d).sql" ]; then
    echo "❌ Rollback script not prepared!"
    exit 1
fi
echo "✅ Rollback procedure prepared"

echo "🎉 Pre-migration checklist complete!"
```

### **Migration Execution Process**

#### **Step 1: Schema Migration**
```typescript
// Migration execution with monitoring
export async function executeMigrationWithMonitoring(
  migrationName: string
): Promise<MigrationResult> {
  const startTime = Date.now();
  const migrationId = generateMigrationId();
  
  try {
    // Log migration start
    await logMigrationEvent({
      migrationId,
      name: migrationName,
      status: 'STARTED',
      timestamp: new Date()
    });
    
    // Execute Prisma migration
    const result = await execShellCommand('pnpm prisma migrate deploy');
    
    // Verify migration success
    const verification = await verifyMigrationIntegrity();
    if (!verification.success) {
      throw new Error(`Migration verification failed: ${verification.errors.join(', ')}`);
    }
    
    // Log migration completion
    await logMigrationEvent({
      migrationId,
      name: migrationName,
      status: 'COMPLETED',
      duration: Date.now() - startTime,
      timestamp: new Date()
    });
    
    return {
      success: true,
      migrationId,
      duration: Date.now() - startTime
    };
    
  } catch (error) {
    // Log migration failure
    await logMigrationEvent({
      migrationId,
      name: migrationName,
      status: 'FAILED',
      error: error.message,
      duration: Date.now() - startTime,
      timestamp: new Date()
    });
    
    throw error;
  }
}
```

#### **Step 2: Data Migration**
```typescript
// Data migration for complex schema changes
export async function migrateUserPreferences(): Promise<void> {
  const batchSize = 1000;
  let offset = 0;
  let processedCount = 0;
  
  while (true) {
    const users = await prisma.user.findMany({
      select: { id: true, preferences: true },
      skip: offset,
      take: batchSize,
      where: {
        // Only migrate users who haven't been migrated yet
        userPreferences: null
      }
    });
    
    if (users.length === 0) break;
    
    // Process batch in transaction
    await prisma.$transaction(async (tx) => {
      for (const user of users) {
        // Extract preferences from old JSON field
        const oldPreferences = user.preferences as any;
        
        // Create new preferences record
        await tx.userPreferences.create({
          data: {
            userId: user.id,
            preferences: {
              theme: oldPreferences?.theme || 'light',
              language: oldPreferences?.language || 'en',
              notifications: oldPreferences?.notifications || {}
            }
          }
        });
      }
    });
    
    processedCount += users.length;
    offset += batchSize;
    
    // Log progress
    console.log(`Migrated ${processedCount} users...`);
    
    // Add small delay to avoid overwhelming the database
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  
  console.log(`Migration complete. Processed ${processedCount} users.`);
}
```

### **Rollback Procedures**

#### **Automatic Rollback**
```typescript
// Automatic rollback on migration failure
export async function executeWithAutoRollback(
  migrationFn: () => Promise<void>,
  rollbackFn: () => Promise<void>
): Promise<void> {
  const checkpointId = await createMigrationCheckpoint();
  
  try {
    await migrationFn();
    await validateMigrationSuccess();
  } catch (error) {
    console.error('Migration failed, initiating rollback...', error);
    
    try {
      await rollbackFn();
      await restoreFromCheckpoint(checkpointId);
      console.log('Rollback completed successfully');
    } catch (rollbackError) {
      console.error('Rollback failed!', rollbackError);
      throw new Error(`Migration failed and rollback failed: ${rollbackError.message}`);
    }
    
    throw error;
  }
}
```

#### **Manual Rollback Scripts**
```sql
-- rollback-20240115-user-preferences.sql
-- Rollback user preferences migration

BEGIN;

-- Step 1: Copy data back to old structure
UPDATE user 
SET preferences = up.preferences
FROM user_preferences up
WHERE user.id = up.userId;

-- Step 2: Drop new table
DROP TABLE user_preferences;

-- Step 3: Drop new constraints if any
ALTER TABLE user DROP CONSTRAINT IF EXISTS check_preferences_format;

-- Step 4: Update migration history
DELETE FROM _prisma_migrations 
WHERE migration_name = '20240115000000_add_user_preferences';

COMMIT;
```

## 📊 Versioning Strategy

### **Schema Versioning**

#### **Version Numbering**
```
Format: YYYY.MM.PATCH
Example: 2024.01.001

Components:
- YYYY: Year
- MM: Month  
- PATCH: Sequential patch number within the month
```

#### **Version Tracking**
```typescript
// Schema version tracking
export interface SchemaVersion {
  version: string;
  timestamp: Date;
  description: string;
  breaking: boolean;
  rollbackAvailable: boolean;
  migrations: string[];
}

// Version registry
export const SCHEMA_VERSIONS: SchemaVersion[] = [
  {
    version: '2024.01.001',
    timestamp: new Date('2024-01-15'),
    description: 'Add user preferences table',
    breaking: false,
    rollbackAvailable: true,
    migrations: ['20240115000000_add_user_preferences']
  },
  {
    version: '2024.01.002', 
    timestamp: new Date('2024-01-20'),
    description: 'Update resource scoring algorithm',
    breaking: false,
    rollbackAvailable: true,
    migrations: ['20240120000000_update_resource_scoring']
  }
];

// Get current schema version
export async function getCurrentSchemaVersion(): Promise<string> {
  const result = await prisma.$queryRaw<{version: string}[]>`
    SELECT MAX(migration_name) as version 
    FROM _prisma_migrations 
    WHERE finished_at IS NOT NULL
  `;
  
  return result[0]?.version || 'unknown';
}
```

### **Application Compatibility**

#### **Version Compatibility Matrix**
```typescript
// Application version compatibility
export const COMPATIBILITY_MATRIX = {
  '1.0.0': {
    minSchemaVersion: '2024.01.001',
    maxSchemaVersion: '2024.03.999',
    breaking: false
  },
  '1.1.0': {
    minSchemaVersion: '2024.02.001',
    maxSchemaVersion: '2024.06.999',
    breaking: false
  },
  '2.0.0': {
    minSchemaVersion: '2024.06.001',
    maxSchemaVersion: '2024.12.999',
    breaking: true
  }
};

// Check version compatibility
export async function checkVersionCompatibility(
  appVersion: string,
  schemaVersion: string
): Promise<CompatibilityResult> {
  const compatibility = COMPATIBILITY_MATRIX[appVersion];
  
  if (!compatibility) {
    return {
      compatible: false,
      reason: `Unknown application version: ${appVersion}`
    };
  }
  
  const isCompatible = 
    schemaVersion >= compatibility.minSchemaVersion &&
    schemaVersion <= compatibility.maxSchemaVersion;
  
  return {
    compatible: isCompatible,
    breaking: compatibility.breaking,
    reason: isCompatible ? null : 'Schema version outside compatible range'
  };
}
```

## 🔀 Backwards Compatibility

### **Backwards Compatible Changes**

#### **Adding Optional Fields**
```sql
-- Safe: Adding nullable columns
ALTER TABLE user ADD COLUMN avatar_url VARCHAR(512);
ALTER TABLE resource ADD COLUMN tags TEXT[] DEFAULT '{}';

-- Safe: Adding tables that don't affect existing queries
CREATE TABLE user_activity_log (
    id BIGINT PRIMARY KEY,
    userId BIGINT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    activity VARCHAR(128) NOT NULL,
    metadata JSON DEFAULT '{}',
    createdAt TIMESTAMPTZ DEFAULT NOW()
);
```

#### **Expanding Enums**
```sql
-- Safe: Adding new enum values at the end
ALTER TYPE "ResourceType" ADD VALUE 'Template';
ALTER TYPE "RunStatus" ADD VALUE 'Paused';

-- Update enum usage
INSERT INTO resource (resourceType, ...) VALUES ('Template', ...);
```

#### **Adding Indexes**
```sql
-- Safe: Adding performance indexes
CREATE INDEX CONCURRENTLY idx_user_activity_log_user_time 
ON user_activity_log(userId, createdAt DESC);

CREATE INDEX CONCURRENTLY idx_resource_tags_gin 
ON resource USING GIN(tags);
```

### **Handling Breaking Changes**

#### **Multi-Phase Migration**
```typescript
// Phase 1: Add new structure alongside old
export async function addNewStructurePhase1(): Promise<void> {
  await prisma.$executeRaw`
    -- Add new table
    CREATE TABLE user_settings (
        id BIGINT PRIMARY KEY,
        userId BIGINT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
        settings JSON NOT NULL DEFAULT '{}',
        createdAt TIMESTAMPTZ DEFAULT NOW(),
        updatedAt TIMESTAMPTZ DEFAULT NOW()
    );
    
    -- Add trigger to sync data
    CREATE OR REPLACE FUNCTION sync_user_settings()
    RETURNS TRIGGER AS $$
    BEGIN
        IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
            INSERT INTO user_settings (userId, settings)
            VALUES (NEW.id, NEW.preferences)
            ON CONFLICT (userId) DO UPDATE SET
                settings = EXCLUDED.settings,
                updatedAt = NOW();
        END IF;
        RETURN NEW;
    END;
    $$ LANGUAGE plpgsql;
    
    CREATE TRIGGER user_settings_sync
        AFTER INSERT OR UPDATE ON user
        FOR EACH ROW EXECUTE FUNCTION sync_user_settings();
  `;
}

// Phase 2: Migrate application to use new structure
// (Deploy application changes)

// Phase 3: Remove old structure
export async function removeOldStructurePhase3(): Promise<void> {
  await prisma.$executeRaw`
    -- Drop sync trigger
    DROP TRIGGER IF EXISTS user_settings_sync ON user;
    DROP FUNCTION IF EXISTS sync_user_settings();
    
    -- Remove old column
    ALTER TABLE user DROP COLUMN preferences;
  `;
}
```

#### **Feature Flags for Migration**
```typescript
// Use feature flags during migration
export class MigrationFeatureFlags {
  private static flags = new Map<string, boolean>();
  
  static async initialize(): Promise<void> {
    // Load flags from database or config
    const flags = await getFeatureFlags();
    this.flags = new Map(Object.entries(flags));
  }
  
  static isEnabled(flag: string): boolean {
    return this.flags.get(flag) ?? false;
  }
  
  // Migration-specific flags
  static useNewUserSettings(): boolean {
    return this.isEnabled('use_new_user_settings');
  }
  
  static useNewResourceStructure(): boolean {
    return this.isEnabled('use_new_resource_structure');
  }
}

// Application code adapts based on flags
export async function getUserSettings(userId: string): Promise<UserSettings> {
  if (MigrationFeatureFlags.useNewUserSettings()) {
    // Use new table
    return getSettingsFromNewTable(userId);
  } else {
    // Use old column
    return getSettingsFromOldColumn(userId);
  }
}
```

## 🛠️ Migration Best Practices

### **Testing Strategy**

#### **Migration Testing Pipeline**
```typescript
// Comprehensive migration testing
export class MigrationTester {
  async testMigration(migrationPath: string): Promise<TestResult> {
    const testDb = await this.createTestDatabase();
    
    try {
      // 1. Load production data sample
      await this.loadDataSample(testDb);
      
      // 2. Run migration
      await this.executeMigration(testDb, migrationPath);
      
      // 3. Verify data integrity
      const integrityCheck = await this.verifyDataIntegrity(testDb);
      
      // 4. Test application compatibility
      const compatibilityCheck = await this.testApplicationCompatibility(testDb);
      
      // 5. Test rollback
      const rollbackCheck = await this.testRollback(testDb, migrationPath);
      
      return {
        success: integrityCheck.success && compatibilityCheck.success && rollbackCheck.success,
        results: {
          integrity: integrityCheck,
          compatibility: compatibilityCheck,
          rollback: rollbackCheck
        }
      };
      
    } finally {
      await this.cleanupTestDatabase(testDb);
    }
  }
  
  private async verifyDataIntegrity(db: Database): Promise<IntegrityResult> {
    const checks = await Promise.all([
      this.checkForeignKeyConstraints(db),
      this.checkDataConsistency(db),
      this.checkIndexIntegrity(db),
      this.validateBusinessRules(db)
    ]);
    
    return {
      success: checks.every(check => check.success),
      checks
    };
  }
}
```

### **Performance Considerations**

#### **Large Table Migrations**
```sql
-- For very large tables, use incremental approach
CREATE TABLE user_new (LIKE user INCLUDING ALL);

-- Add new columns to new table
ALTER TABLE user_new ADD COLUMN new_field VARCHAR(255);

-- Migrate data in batches
DO $$
DECLARE
    batch_size INTEGER := 10000;
    offset_val INTEGER := 0;
    row_count INTEGER;
BEGIN
    LOOP
        -- Copy batch
        INSERT INTO user_new
        SELECT *, NULL as new_field
        FROM user
        ORDER BY id
        LIMIT batch_size OFFSET offset_val;
        
        GET DIAGNOSTICS row_count = ROW_COUNT;
        EXIT WHEN row_count = 0;
        
        offset_val := offset_val + batch_size;
        
        -- Log progress
        RAISE NOTICE 'Migrated % rows', offset_val;
        
        -- Small delay to reduce load
        PERFORM pg_sleep(0.1);
    END LOOP;
END $$;

-- Swap tables atomically
BEGIN;
ALTER TABLE user RENAME TO user_old;
ALTER TABLE user_new RENAME TO user;
COMMIT;

-- Verify and cleanup
-- (After verification)
DROP TABLE user_old;
```

#### **Zero-Downtime Index Creation**
```sql
-- Create indexes concurrently to avoid blocking
CREATE INDEX CONCURRENTLY idx_resource_created_at_score 
ON resource(createdAt, score) 
WHERE isDeleted = false;

-- Verify index creation
SELECT schemaname, tablename, indexname, indexdef
FROM pg_indexes 
WHERE indexname = 'idx_resource_created_at_score';
```

### **Monitoring and Alerts**

#### **Migration Monitoring**
```typescript
// Real-time migration monitoring
export class MigrationMonitor {
  private metrics = new Map<string, number>();
  
  async monitorMigration(migrationId: string): Promise<void> {
    const monitor = setInterval(async () => {
      const stats = await this.getMigrationStats(migrationId);
      
      // Check for issues
      if (stats.blockedQueries > 10) {
        await this.sendAlert('HIGH_QUERY_BLOCKING', stats);
      }
      
      if (stats.lockWaitTime > 30000) {
        await this.sendAlert('LONG_LOCK_WAIT', stats);
      }
      
      if (stats.replicationLag > 5000) {
        await this.sendAlert('HIGH_REPLICATION_LAG', stats);
      }
      
      // Log progress
      console.log(`Migration ${migrationId} progress:`, stats);
    }, 5000);
    
    // Stop monitoring when migration completes
    this.onMigrationComplete(migrationId, () => {
      clearInterval(monitor);
    });
  }
  
  private async getMigrationStats(migrationId: string): Promise<MigrationStats> {
    const [blocking, locks, replication] = await Promise.all([
      this.getBlockedQueries(),
      this.getLockWaitTimes(),
      this.getReplicationLag()
    ]);
    
    return {
      migrationId,
      timestamp: new Date(),
      blockedQueries: blocking.length,
      lockWaitTime: Math.max(...locks.map(l => l.waitTime), 0),
      replicationLag: replication.maxLag
    };
  }
}
```

## 📝 Documentation Requirements

### **Migration Documentation**

#### **Migration Record Template**
```markdown
# Migration: Add User Preferences Table

**Migration ID**: 20240115000000_add_user_preferences
**Schema Version**: 2024.01.001
**Date**: 2024-01-15
**Author**: developer@vrooli.com

## Summary
Add dedicated user_preferences table to improve performance and flexibility of user preference management.

## Changes
- Add `user_preferences` table
- Migrate data from `user.preferences` JSON column
- Add indexes for common preference queries
- Update application code to use new table

## Backwards Compatibility
- ✅ Backwards compatible
- Old application versions can continue reading from `user.preferences`
- Sync trigger maintains data consistency during transition

## Rollback Plan
1. Stop application
2. Run rollback script: `rollback-20240115-user-preferences.sql`
3. Restart application with previous version

## Testing
- [x] Tested on staging environment
- [x] Data integrity verified
- [x] Performance benchmarks passed
- [x] Rollback procedure tested

## Performance Impact
- Migration time: ~30 minutes for 1M users
- Temporary disk usage: +20GB during migration
- Expected performance improvement: 40% faster preference queries

## Monitoring
- Watch for replication lag during migration
- Monitor query blocking during index creation
- Verify application error rates post-migration
```

### **Schema Change Log**

#### **Automated Change Tracking**
```typescript
// Track all schema changes
export async function logSchemaChange(change: SchemaChange): Promise<void> {
  await prisma.schemaChangeLog.create({
    data: {
      migrationId: change.migrationId,
      changeType: change.type,
      tableName: change.table,
      columnName: change.column,
      oldDefinition: change.oldDef,
      newDefinition: change.newDef,
      backwards_compatible: change.backwardsCompatible,
      createdAt: new Date()
    }
  });
}

// Generate change documentation
export async function generateChangeReport(
  fromVersion: string,
  toVersion: string
): Promise<ChangeReport> {
  const changes = await prisma.schemaChangeLog.findMany({
    where: {
      createdAt: {
        gte: new Date(fromVersion),
        lte: new Date(toVersion)
      }
    },
    orderBy: { createdAt: 'asc' }
  });
  
  return {
    fromVersion,
    toVersion,
    totalChanges: changes.length,
    breakingChanges: changes.filter(c => !c.backwards_compatible).length,
    changes: changes.map(c => ({
      migration: c.migrationId,
      type: c.changeType,
      table: c.tableName,
      description: generateChangeDescription(c)
    }))
  };
}
```

---

**Related Documentation:**
- [Database Architecture](architecture.md) - Infrastructure setup and configuration
- [Performance Guide](performance.md) - Query optimization and monitoring
- [Entity Models](entities/README.md) - Current schema structure and relationships