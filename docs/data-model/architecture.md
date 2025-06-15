# Database Architecture

Comprehensive overview of Vrooli's database infrastructure, technology stack, and architectural patterns.

## 🏗️ Architecture Overview

```mermaid
graph TB
    subgraph "Application Layer"
        API[REST API]
        WS[WebSocket API]
        MCP[MCP Server]
    end
    
    subgraph "Data Access Layer"
        Prisma[Prisma ORM]
        Cache[Redis Cache]
        Queue[BullMQ Queues]
    end
    
    subgraph "Storage Layer"
        PG[(PostgreSQL<br/>Primary Database)]
        Vector[(pgvector<br/>Embeddings)]
        S3[(S3<br/>File Storage)]
        Backup[(Backup Storage)]
    end
    
    API --> Prisma
    WS --> Prisma
    MCP --> Prisma
    
    Prisma --> Cache
    Prisma --> PG
    Prisma --> Vector
    
    Cache --> Queue
    PG --> S3
    PG --> Backup
    
    classDef app fill:#e3f2fd,stroke:#1565c0
    classDef access fill:#e8f5e8,stroke:#388e3c
    classDef storage fill:#fff3e0,stroke:#f57c00
    
    class API,WS,MCP app
    class Prisma,Cache,Queue access
    class PG,Vector,S3,Backup storage
```

## 🛠️ Technology Stack

### **Primary Database**
- **PostgreSQL 14+** with extensions:
  - `pgvector` - Vector similarity search for AI embeddings
  - `citext` - Case-insensitive text for emails and handles
  - Full-text search capabilities
  - JSON/JSONB support for flexible schemas

### **ORM & Data Access**
- **Prisma** - Type-safe database client and ORM
  - Automatic TypeScript type generation
  - Database migrations management
  - Query optimization and batching
  - Connection pooling and prepared statements

### **Caching Layer**
- **Redis** - In-memory data structure store
  - Session storage and management
  - Real-time data caching
  - Message queuing with BullMQ
  - WebSocket state synchronization

### **File Storage**
- **S3-Compatible Storage** - Object storage for media
  - User avatars and profile images
  - Resource attachments and files
  - Export data and backups
  - CDN integration for global delivery

### **Monitoring & Observability**
- **Prometheus** - Metrics collection
- **Grafana** - Visualization dashboards
- **Structured logging** - JSON-formatted application logs
- **Query performance monitoring** - Slow query detection

## 🔄 Data Flow Architecture

### **Read Operations**
```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Cache
    participant Prisma
    participant PostgreSQL
    
    Client->>API: Request Data
    API->>Cache: Check Cache
    alt Cache Hit
        Cache-->>API: Return Cached Data
    else Cache Miss
        API->>Prisma: Query Database
        Prisma->>PostgreSQL: Execute SQL
        PostgreSQL-->>Prisma: Return Results
        Prisma-->>API: Return Typed Data
        API->>Cache: Store in Cache
    end
    API-->>Client: Return Response
```

### **Write Operations**
```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Prisma
    participant PostgreSQL
    participant Cache
    participant Queue
    
    Client->>API: Write Request
    API->>Prisma: Begin Transaction
    Prisma->>PostgreSQL: Execute Writes
    PostgreSQL-->>Prisma: Confirm Success
    Prisma-->>API: Transaction Complete
    API->>Cache: Invalidate Cache
    API->>Queue: Queue Background Jobs
    API-->>Client: Success Response
```

## 📊 Storage Systems

### **PostgreSQL Configuration**

#### **Production Settings**
```sql
-- Memory configuration
shared_buffers = '2GB'
effective_cache_size = '6GB'
work_mem = '16MB'
maintenance_work_mem = '512MB'

-- Write performance
wal_buffers = '16MB'
checkpoint_completion_target = 0.9
max_wal_size = '2GB'

-- Connection management
max_connections = 200
shared_preload_libraries = 'pg_stat_statements,pgvector'

-- Extensions
CREATE EXTENSION IF NOT EXISTS pgvector;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

> **Note**: For detailed performance tuning and optimization strategies, see [Performance Guide](performance.md#database-scaling).

#### **Replication Setup**
```yaml
# Primary database
primary:
  host: db-primary.vrooli.com
  port: 5432
  databases: [vrooli_production]
  
# Read replicas
replicas:
  - host: db-replica-1.vrooli.com
    port: 5432
    lag_threshold: 100ms
  - host: db-replica-2.vrooli.com
    port: 5432
    lag_threshold: 100ms

# Automatic failover
failover:
  enabled: true
  timeout: 30s
  max_lag: 1s
```

### **Redis Configuration**

#### **Cache Configuration**
```redis
# Memory management
maxmemory 4gb
maxmemory-policy allkeys-lru

# Persistence
save 900 1
save 300 10
save 60 10000

# Performance
tcp-keepalive 300
timeout 0
databases 16

# Specific database usage
# DB 0: Session storage
# DB 1: Application cache
# DB 2: Queue data
# DB 3: WebSocket state
```

#### **Cache Patterns**

> **Note**: For comprehensive caching strategies and implementation patterns, see [Performance Guide](../data-model/performance.md#caching-strategy).

Basic cache usage:
```typescript
// Session storage
await redis.setex(`session:${sessionId}`, 86400, JSON.stringify(userData));

// Query result caching  
await redis.setex(`user:${userId}:teams`, 300, JSON.stringify(teams));
```

## 🔧 Connection Management

### **Connection Pooling**
```typescript
// Prisma connection pool configuration
const prisma = new PrismaClient({
  datasources: {
    db: {
      url: process.env.DATABASE_URL,
    },
  },
  log: ['query', 'info', 'warn', 'error'],
});

// Connection pool settings (in DATABASE_URL)
// postgresql://user:pass@host:5432/db?connection_limit=20&pool_timeout=10
```

### **Health Checks**
```typescript
// Database health check
export async function checkDatabaseHealth(): Promise<HealthStatus> {
  try {
    const start = Date.now();
    await prisma.$queryRaw`SELECT 1`;
    const responseTime = Date.now() - start;
    
    return {
      status: 'healthy',
      responseTime,
      timestamp: new Date()
    };
  } catch (error) {
    return {
      status: 'unhealthy',
      error: error.message,
      timestamp: new Date()
    };
  }
}

// Redis health check
export async function checkRedisHealth(): Promise<HealthStatus> {
  try {
    const start = Date.now();
    await redis.ping();
    const responseTime = Date.now() - start;
    
    return {
      status: 'healthy',
      responseTime,
      timestamp: new Date()
    };
  } catch (error) {
    return {
      status: 'unhealthy',
      error: error.message,
      timestamp: new Date()
    };
  }
}
```

## 🔐 Security Architecture

### **Database Security**

#### **Authentication & Authorization**
```sql
-- Create dedicated application user
CREATE USER vrooli_app WITH PASSWORD 'secure_random_password';

-- Grant minimal required permissions
GRANT CONNECT ON DATABASE vrooli TO vrooli_app;
GRANT USAGE ON SCHEMA public TO vrooli_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO vrooli_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO vrooli_app;

-- Row-level security (if needed)
ALTER TABLE user ENABLE ROW LEVEL SECURITY;
CREATE POLICY user_own_data ON user FOR ALL TO vrooli_app USING (id = current_user_id());
```

#### **SSL/TLS Configuration**
```sql
-- Enable SSL
ALTER SYSTEM SET ssl = on;
ALTER SYSTEM SET ssl_cert_file = '/etc/ssl/certs/server.crt';
ALTER SYSTEM SET ssl_key_file = '/etc/ssl/private/server.key';

-- Require SSL connections
ALTER SYSTEM SET ssl_require = on;
```

### **Application-Level Security**
```typescript
// Sensitive data encryption
import crypto from 'crypto';

const ENCRYPTION_KEY = process.env.ENCRYPTION_KEY;

export function encryptSensitiveData(data: string): string {
  const iv = crypto.randomBytes(16);
  const cipher = crypto.createCipher('aes-256-gcm', ENCRYPTION_KEY, iv);
  
  let encrypted = cipher.update(data, 'utf8', 'hex');
  encrypted += cipher.final('hex');
  
  const authTag = cipher.getAuthTag();
  return `${iv.toString('hex')}:${authTag.toString('hex')}:${encrypted}`;
}

// Audit logging
export async function logDataAccess(
  operation: string,
  entityType: string,
  entityId: string,
  userId: string
): Promise<void> {
  await prisma.auditLog.create({
    data: {
      operation,
      entityType,
      entityId,
      userId,
      timestamp: new Date(),
      metadata: { source: 'database_access' }
    }
  });
}
```

## 🚀 Performance Optimization

For comprehensive performance optimization strategies, query patterns, indexing, and caching, see **[Performance Guide](performance.md)**.

## 💾 Backup & Recovery

### **Backup Strategy**
```bash
#!/bin/bash
# Automated backup script

# Configuration
BACKUP_DIR="/backup/postgresql"
RETENTION_DAYS=30
DB_NAME="vrooli"

# Create timestamped backup
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/vrooli_$TIMESTAMP.sql.gz"

# Perform backup
pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME | gzip > $BACKUP_FILE

# Upload to S3
aws s3 cp $BACKUP_FILE s3://vrooli-backups/postgresql/

# Cleanup old backups
find $BACKUP_DIR -name "vrooli_*.sql.gz" -mtime +$RETENTION_DAYS -delete

# Verify backup integrity
gunzip -t $BACKUP_FILE && echo "Backup verification successful"
```

### **Point-in-Time Recovery**
```sql
-- Enable WAL archiving
ALTER SYSTEM SET archive_mode = on;
ALTER SYSTEM SET archive_command = 'cp %p /backup/wal/%f';
ALTER SYSTEM SET wal_level = replica;

-- Recovery configuration
-- In recovery.conf or postgresql.auto.conf:
restore_command = 'cp /backup/wal/%f %p'
recovery_target_time = '2024-01-15 14:30:00'
```

## 📈 Monitoring & Alerting

### **Key Metrics**
```yaml
# Prometheus metrics configuration
metrics:
  database:
    - query_duration_p95
    - connection_pool_usage
    - active_connections
    - deadlocks_total
    - cache_hit_ratio
  
  redis:
    - memory_usage_bytes
    - connected_clients
    - commands_processed_total
    - keyspace_hits_ratio
  
  application:
    - api_request_duration
    - error_rate_5m
    - active_users
    - background_job_queue_size
```

### **Alerting Rules**
```yaml
# Grafana alerting rules
alerts:
  - name: "High Database Response Time"
    condition: "query_duration_p95 > 500ms"
    severity: "warning"
    
  - name: "Database Connection Pool Exhausted"
    condition: "connection_pool_usage > 90%"
    severity: "critical"
    
  - name: "Redis Memory Usage High"
    condition: "redis_memory_usage > 85%"
    severity: "warning"
    
  - name: "Replication Lag High"
    condition: "replication_lag > 1s"
    severity: "critical"
```

## 🔧 Maintenance Procedures

### **Routine Maintenance**
```sql
-- Weekly maintenance tasks
BEGIN;
  -- Update table statistics
  ANALYZE;
  
  -- Vacuum to reclaim space
  VACUUM (ANALYZE, VERBOSE);
  
  -- Reindex if needed
  REINDEX INDEX CONCURRENTLY idx_user_email;
COMMIT;
```

### **Capacity Planning**
```typescript
// Growth monitoring
export async function generateCapacityReport(): Promise<CapacityReport> {
  const [
    tableStats,
    queryStats,
    connectionStats
  ] = await Promise.all([
    getTableSizeStats(),
    getQueryPerformanceStats(),
    getConnectionStats()
  ]);
  
  return {
    timestamp: new Date(),
    storage: {
      currentSize: tableStats.totalSize,
      growthRate: calculateGrowthRate(tableStats),
      projectedCapacity: projectCapacity(tableStats, 12) // 12 months
    },
    performance: {
      averageQueryTime: queryStats.avgDuration,
      slowQueries: queryStats.slowQueries,
      recommendations: generateOptimizationRecommendations(queryStats)
    },
    connections: {
      current: connectionStats.active,
      peak: connectionStats.peak,
      utilization: connectionStats.utilization
    }
  };
}
```

---

**Related Documentation:**
- [Entity Models](entities/README.md) - Database schema and relationships
- [Performance Guide](performance.md) - Optimization strategies
- [Schema Evolution](schema-evolution.md) - Migration procedures