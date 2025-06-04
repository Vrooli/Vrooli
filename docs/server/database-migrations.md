# Database Migrations & Management

This comprehensive guide covers database migrations, schema management, and data operations for the Vrooli server using Prisma ORM with PostgreSQL.

## Table of Contents

- [Overview](#overview)
- [Database Architecture](#database-architecture)
- [Prisma Setup & Configuration](#prisma-setup--configuration)
- [Migration Workflow](#migration-workflow)
- [Development Environment Setup](#development-environment-setup)
- [Migration Operations](#migration-operations)
- [Seeding System](#seeding-system)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)
- [Production Considerations](#production-considerations)

## Overview

Vrooli uses **Prisma ORM** with **PostgreSQL** for robust, type-safe database operations. The migration system handles schema versioning, data seeding, and ensures consistent database states across all environments.

### Key Features

- **Type-safe Database Access**: Full TypeScript integration with generated types
- **Automated Schema Management**: Version-controlled schema changes
- **Intelligent Seeding**: Consistent data initialization with retry logic
- **Migration Rollbacks**: Safe schema reversions when needed
- **Connection Pooling**: Efficient database connection management
- **Health Monitoring**: Real-time database performance tracking

## Database Architecture

```mermaid
graph TB
    A[Vrooli Application] --> B[Prisma Client]
    B --> C[Connection Pool]
    C --> D[PostgreSQL Database]
    
    subgraph "Schema Management"
        E[schema.prisma] --> F[Prisma Generate]
        F --> G[TypeScript Types]
        E --> H[Prisma Migrate]
        H --> I[SQL Migrations]
    end
    
    subgraph "Data Operations"
        J[Seeding Scripts] --> K[Initial Data]
        L[Migration Scripts] --> M[Schema Changes]
        N[Backup Scripts] --> O[Data Protection]
    end
    
    I --> D
    K --> D
    M --> D
```

### Database Provider Architecture

The `DbProvider` class manages the database connection lifecycle:

```typescript
export class DbProvider {
    private static client: PrismaClient | null = null;
    private static isInitialized = false;
    private static seedingSuccessful = false;
    
    static async init(): Promise<void> {
        // Connection setup, seeding, health checks
    }
    
    static get(): PrismaClient {
        // Returns the Prisma client instance
    }
}
```

## Prisma Setup & Configuration

### Schema Definition (`packages/server/src/db/schema.prisma`)

The Prisma schema defines the database structure, relationships, and constraints:

```prisma
generator client {
  provider = "prisma-client-js"
  output   = "../node_modules/@prisma/client"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id        String   @id @default(cuid())
  email     String   @unique
  name      String?
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
  
  // Relations
  sessions  Session[]
  @@map("users")
}
```

### Environment Configuration

Database connection is configured through environment variables:

```bash
# Development
DATABASE_URL="postgresql://username:password@localhost:5432/vrooli_dev"

# Production
DATABASE_URL="postgresql://username:password@host:5432/vrooli_prod"

# Test
DATABASE_URL="postgresql://username:password@localhost:5432/vrooli_test"
```

## Migration Workflow

```mermaid
graph TD
    A[Schema Changes] --> B[Create Migration]
    B --> C[Review Migration SQL]
    C --> D{Approved?}
    D -->|No| E[Modify Schema]
    E --> B
    D -->|Yes| F[Apply Migration]
    F --> G[Generate Types]
    G --> H[Test Changes]
    H --> I{Tests Pass?}
    I -->|No| J[Fix Issues]
    J --> A
    I -->|Yes| K[Commit Changes]
    K --> L[Deploy to Staging]
    L --> M[Deploy to Production]
```

## Development Environment Setup

### Prerequisites

1. **Docker & Docker Compose**: For containerized development
2. **Node.js**: Version specified in `.nvmrc`
3. **Database Access**: PostgreSQL instance running

### Initial Setup

1. **Start the Development Environment**:
   ```bash
   docker-compose up -d
   ```

2. **Wait for Services**: Ensure all containers are healthy:
   ```bash
   docker ps -a  # Check container status
   ```

3. **Access the Server Container**:
   ```bash
   docker exec -it server sh
   ```

4. **Set Database URL** (if needed manually):
   ```bash
   export DATABASE_URL=postgresql://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}
   
   # For default development setup:
   export DATABASE_URL=postgresql://site:databasepassword@postgres:5432/vrooli
   ```

### Database Initialization

The database is automatically initialized during server startup, but for manual operations:

```bash
cd packages/server
pnpm prisma generate    # Generate Prisma client
pnpm prisma db push     # Push schema to database (development)
```

## Migration Operations

### Creating Migrations

1. **Modify the Schema**: Edit `packages/server/src/db/schema.prisma`

2. **Generate Migration**:
   ```bash
   cd packages/server
   pnpm prisma migrate dev --name descriptive_migration_name
   ```

3. **Review Generated SQL**: Check the migration file in `src/db/migrations/`

### Migration Commands

```bash
# Check migration status
pnpm prisma migrate status

# Apply pending migrations
pnpm prisma migrate deploy

# Reset database (development only)
pnpm prisma migrate reset

# Create migration without applying
pnpm prisma migrate dev --create-only --name migration_name

# Generate Prisma client after schema changes
pnpm prisma generate
```

### Migration File Structure

```
packages/server/src/db/migrations/
├── 20240101120000_initial_schema/
│   └── migration.sql
├── 20240102130000_add_user_profiles/
│   └── migration.sql
└── migration_lock.toml
```

### Working with Existing Data

When migrations affect existing data, create data migration scripts:

```sql
-- migration.sql
-- Step 1: Add new column
ALTER TABLE "users" ADD COLUMN "profile_id" TEXT;

-- Step 2: Populate data
UPDATE "users" SET "profile_id" = gen_random_uuid() WHERE "profile_id" IS NULL;

-- Step 3: Add constraints
ALTER TABLE "users" ALTER COLUMN "profile_id" SET NOT NULL;
```

## Seeding System

### Automatic Seeding

The server automatically seeds the database during initialization:

```typescript
// In DbProvider.init()
private static async seedDatabase(): Promise<void> {
    const seedingSuccessful = await this.performSeeding();
    if (!seedingSuccessful && this.shouldRetry()) {
        await this.scheduleRetry();
    }
}
```

### Manual Seeding

```bash
cd packages/server
pnpm prisma db seed
```

### Seed Script Structure

```typescript
// packages/server/src/db/seeds/index.ts
import { PrismaClient } from '@prisma/client';

const prisma = new PrismaClient();

async function main() {
    // Create default roles
    await prisma.role.createMany({
        data: [
            { name: 'USER', description: 'Standard user' },
            { name: 'ADMIN', description: 'Administrator' },
        ],
        skipDuplicates: true,
    });
    
    // Create initial users
    await prisma.user.upsert({
        where: { email: 'admin@vrooli.com' },
        update: {},
        create: {
            email: 'admin@vrooli.com',
            name: 'Admin User',
            role: { connect: { name: 'ADMIN' } },
        },
    });
}

main()
    .catch((e) => {
        console.error(e);
        process.exit(1);
    })
    .finally(async () => {
        await prisma.$disconnect();
    });
```

### Seed Data Management

```mermaid
graph TD
    A[Seed Execution] --> B{Database Empty?}
    B -->|Yes| C[Create Initial Data]
    B -->|No| D[Check Required Data]
    D --> E{Missing Required?}
    E -->|Yes| F[Add Missing Data]
    E -->|No| G[Skip Seeding]
    C --> H[Verify Data Integrity]
    F --> H
    G --> I[Complete]
    H --> J{Integrity OK?}
    J -->|Yes| I
    J -->|No| K[Log Error & Retry]
    K --> L{Max Retries?}
    L -->|No| A
    L -->|Yes| M[Fail with Error]
```

## Troubleshooting

### Common Issues

#### 1. Migration Status Issues

**Problem**: Migrations show as pending or failed
```bash
pnpm prisma migrate status
```

**Solution**: Check and resolve migration conflicts:
```bash
# Reset to clean state (development only)
pnpm prisma migrate reset

# Or mark specific migrations as applied
pnpm prisma migrate resolve --applied 20240101120000_migration_name
```

#### 2. Connection Issues

**Problem**: Database connection failures

**Diagnosis**:
```bash
# Test database connectivity
pnpm prisma db pull

# Check database status in Docker
docker logs postgres
```

**Solutions**:
- Verify `DATABASE_URL` environment variable
- Ensure PostgreSQL container is running and healthy
- Check network connectivity between containers

#### 3. Schema Drift

**Problem**: Database schema doesn't match Prisma schema

**Detection**:
```bash
pnpm prisma db pull  # Pull current database schema
pnpm prisma validate # Validate schema consistency
```

**Resolution**:
```bash
# Option 1: Push schema to database (development)
pnpm prisma db push

# Option 2: Create migration for differences
pnpm prisma migrate dev --name fix_schema_drift
```

#### 4. Seeding Failures

**Problem**: Database seeding fails or times out

**Diagnosis**: Check server logs for seeding errors and retry attempts

**Solutions**:
- Verify database connectivity
- Check for data conflicts or constraint violations
- Review seeding logic for idempotency
- Increase retry limits if needed

### Advanced Troubleshooting

#### Database Backup & Restore

1. **Create Backup**:
   ```bash
   # From host machine
   docker exec postgres pg_dump -U ${DB_USER} -d ${DB_NAME} > backup_$(date +%Y%m%d_%H%M%S).sql
   
   # From inside container
   pg_dump -U site vrooli > /tmp/backup.sql
   ```

2. **Restore from Backup**:
   ```bash
   # Stop application
   docker-compose stop server
   
   # Restore database
   docker exec -i postgres psql -U ${DB_USER} -d ${DB_NAME} < backup.sql
   
   # Restart application
   docker-compose start server
   ```

#### Migration Rollback

1. **Identify Target Migration**:
   ```bash
   pnpm prisma migrate status
   ```

2. **Rollback to Specific Migration**:
   ```bash
   # This requires manual intervention
   # 1. Restore database from backup taken before problematic migration
   # 2. Mark subsequent migrations as not applied
   pnpm prisma migrate resolve --rolled-back 20240102130000_problematic_migration
   ```

## Best Practices

### Schema Design

1. **Use Descriptive Names**: Clear table and column names
2. **Follow Naming Conventions**: Consistent casing and structure
3. **Add Constraints**: Proper foreign keys, indexes, and validations
4. **Version Control**: All schema changes through migrations
5. **Documentation**: Comment complex relationships and constraints

### Migration Safety

1. **Backward Compatibility**: Ensure new migrations don't break existing code
2. **Data Preservation**: Never lose data in migrations
3. **Testing**: Test migrations on staging before production
4. **Rollback Plans**: Always have a rollback strategy
5. **Incremental Changes**: Keep migrations small and focused

### Performance Considerations

1. **Index Strategy**: Add indexes for frequently queried columns
2. **Connection Pooling**: Configure appropriate pool sizes
3. **Query Optimization**: Use Prisma's query analysis tools
4. **Monitoring**: Track slow queries and database performance

### Security

1. **Access Control**: Limit database user permissions
2. **Environment Isolation**: Separate development, staging, and production
3. **Backup Strategy**: Regular automated backups
4. **Audit Logging**: Track schema and data changes

## Production Considerations

### Deployment Pipeline

```mermaid
graph TD
    A[Code Changes] --> B[Local Testing]
    B --> C[CI/CD Pipeline]
    C --> D[Staging Deployment]
    D --> E[Migration Testing]
    E --> F{Tests Pass?}
    F -->|No| G[Fix Issues]
    G --> A
    F -->|Yes| H[Production Deployment]
    H --> I[Health Checks]
    I --> J[Monitoring]
```

### Production Migration Strategy

1. **Pre-deployment Backup**: Always backup before migrations
2. **Blue-Green Deployment**: Use deployment strategies that minimize downtime
3. **Migration Validation**: Verify migrations in staging environment
4. **Rollback Preparation**: Have rollback scripts ready
5. **Monitoring**: Watch for performance impacts post-migration

### Monitoring & Alerting

- **Connection Pool Status**: Monitor active/idle connections
- **Query Performance**: Track slow queries and execution times
- **Migration Status**: Alert on migration failures
- **Database Health**: Monitor CPU, memory, and disk usage
- **Backup Status**: Ensure backup completion and validation

### Scaling Considerations

- **Read Replicas**: For read-heavy workloads
- **Connection Pooling**: Optimize for high concurrency
- **Index Optimization**: Regular index maintenance
- **Query Analysis**: Identify and optimize expensive queries
- **Database Partitioning**: For very large datasets

---

This documentation provides comprehensive guidance for database operations in Vrooli. For specific implementation details, refer to the Prisma documentation and the database provider source code.
