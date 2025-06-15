# Performance Guide

Comprehensive guide to database performance optimization, indexing strategies, query patterns, and scaling approaches for Vrooli's data model.

## 📊 Performance Overview

### **Performance Targets**

| Operation Type | Target Response Time | Throughput Target |
|----------------|---------------------|-------------------|
| **Read Queries** | < 100ms (p95) | 1000+ QPS |
| **Write Operations** | < 200ms (p95) | 500+ QPS |
| **Complex Aggregations** | < 500ms (p95) | 100+ QPS |
| **Bulk Operations** | < 2s (p95) | 50+ QPS |
| **Real-time Updates** | < 50ms (p95) | 2000+ QPS |

### **Current Performance Metrics**

```typescript
interface PerformanceMetrics {
  avgQueryTime: number;          // Average query execution time
  p95QueryTime: number;          // 95th percentile query time
  slowQueries: number;           // Queries > 1s per hour
  cacheHitRate: number;          // Redis cache hit rate (%)
  connectionPoolUtilization: number; // Active connections (%)
  indexUsageRate: number;        // Queries using indexes (%)
}
```

## 🔍 Indexing Strategy

### **Primary Indexes**

#### **Core Entity Lookups**
```sql
-- Primary key indexes (automatic)
CREATE UNIQUE INDEX pk_user ON user(id);
CREATE UNIQUE INDEX pk_resource ON resource(id);
CREATE UNIQUE INDEX pk_run ON run(id);

-- Public ID lookups (very frequent)
CREATE UNIQUE INDEX idx_user_public_id ON user(publicId);
CREATE UNIQUE INDEX idx_resource_public_id ON resource(publicId);
CREATE UNIQUE INDEX idx_run_public_id ON run(publicId);

-- Email and handle lookups (authentication)
CREATE UNIQUE INDEX idx_user_email ON user(email);
CREATE UNIQUE INDEX idx_user_handle ON user(handle);
CREATE UNIQUE INDEX idx_team_handle ON team(handle);
```

#### **Foreign Key Indexes**
```sql
-- User relationships
CREATE INDEX idx_resource_created_by ON resource(createdById);
CREATE INDEX idx_run_user ON run(userId);
CREATE INDEX idx_member_user ON member(userId);
CREATE INDEX idx_chat_participants_user ON chat_participants(userId);

-- Team relationships  
CREATE INDEX idx_resource_team ON resource(teamId);
CREATE INDEX idx_chat_team ON chat(teamId);
CREATE INDEX idx_member_team ON member(teamId);

-- Resource relationships
CREATE INDEX idx_resource_version_root ON resource_version(rootId);
CREATE INDEX idx_run_resource_version ON run(resourceVersionId);
CREATE INDEX idx_resource_tag_resource ON resource_tag(resourceId);

-- Chat relationships
CREATE INDEX idx_chat_message_chat ON chat_message(chatId);
CREATE INDEX idx_chat_message_parent ON chat_message(parentId);
```

### **Composite Indexes**

#### **Time-Based Queries**
```sql
-- Recent resources by user
CREATE INDEX idx_resource_user_created 
ON resource(createdById, createdAt DESC) 
WHERE isDeleted = FALSE;

-- Recent messages in chat
CREATE INDEX idx_chat_message_chat_time 
ON chat_message(chatId, createdAt DESC);

-- User team memberships with timestamps
CREATE INDEX idx_member_user_created 
ON member(userId, createdAt DESC);

-- Run executions by resource version
CREATE INDEX idx_run_version_started 
ON run(resourceVersionId, startedAt DESC) 
WHERE startedAt IS NOT NULL;
```

#### **Status and State Queries**
```sql
-- Active runs by status
CREATE INDEX idx_run_status_started 
ON run(status, startedAt) 
WHERE status IN ('Scheduled', 'InProgress');

-- Latest resource versions
CREATE INDEX idx_resource_version_latest 
ON resource_version(rootId, isLatest) 
WHERE isLatest = TRUE;

-- Public resources with complete versions
CREATE INDEX idx_resource_public_complete 
ON resource(isPrivate, hasCompleteVersion, score DESC) 
WHERE isPrivate = FALSE AND hasCompleteVersion = TRUE;

-- Unread chat participants
CREATE INDEX idx_chat_participants_unread 
ON chat_participants(userId, hasUnread) 
WHERE hasUnread = TRUE;
```

### **Partial Indexes**

#### **Active Records Only**
```sql
-- Non-deleted resources only
CREATE INDEX idx_resource_active_type 
ON resource(resourceType, createdAt DESC) 
WHERE isDeleted = FALSE;

-- Complete resource versions only
CREATE INDEX idx_resource_version_complete 
ON resource_version(rootId, versionIndex DESC) 
WHERE isComplete = TRUE;

-- Active user sessions only
CREATE INDEX idx_session_active_user 
ON session(userId, expiresAt) 
WHERE isActive = TRUE;
```

#### **Performance-Critical Queries**
```sql
-- Team admin lookups
CREATE INDEX idx_member_team_admin 
ON member(teamId, userId) 
WHERE isAdmin = TRUE;

-- Recent failed runs
CREATE INDEX idx_run_failed_recent 
ON run(status, createdAt DESC) 
WHERE status = 'Failed' AND createdAt > NOW() - INTERVAL '7 days';
```

### **JSON Indexes**

#### **Configuration Searches**
```sql
-- User preferences
CREATE INDEX idx_user_preferences_gin ON user USING GIN (preferences);

-- Resource configuration
CREATE INDEX idx_resource_config_gin ON resource USING GIN (config);

-- Team permissions
CREATE INDEX idx_team_permissions_gin ON team USING GIN (permissions);
```

#### **Specific JSON Paths**
```sql
-- User theme preferences
CREATE INDEX idx_user_theme 
ON user((preferences->>'theme')) 
WHERE preferences ? 'theme';

-- Resource type-specific configs
CREATE INDEX idx_resource_routine_config 
ON resource((config->>'executionTimeout')) 
WHERE resourceType = 'Routine' AND config ? 'executionTimeout';
```

## 🚀 Query Optimization Patterns

### **Efficient Pagination**

#### **Cursor-Based Pagination**
```typescript
// Efficient pagination using cursor-based approach
export async function getPaginatedResources(
  cursor?: string,
  limit: number = 20,
  filters?: ResourceFilters
): Promise<PaginatedResult<Resource>> {
  const where: Prisma.ResourceWhereInput = {
    isDeleted: false,
    isPrivate: false,
    hasCompleteVersion: true,
    ...(cursor && { 
      createdAt: { lt: new Date(cursor) } 
    }),
    ...(filters?.resourceType && { 
      resourceType: filters.resourceType 
    }),
    ...(filters?.teamId && { 
      teamId: BigInt(filters.teamId) 
    })
  };

  const resources = await prisma.resource.findMany({
    where,
    orderBy: { createdAt: 'desc' },
    take: limit + 1, // Take one extra to check if there's a next page
    select: {
      id: true,
      publicId: true,
      resourceType: true,
      score: true,
      bookmarks: true,
      views: true,
      createdAt: true,
      creator: {
        select: {
          publicId: true,
          name: true,
          handle: true
        }
      },
      versions: {
        where: { isLatest: true },
        select: {
          publicId: true,
          versionLabel: true,
          isComplete: true
        },
        take: 1
      }
    }
  });

  const hasNextPage = resources.length > limit;
  const items = hasNextPage ? resources.slice(0, -1) : resources;
  const nextCursor = hasNextPage ? items[items.length - 1].createdAt.toISOString() : null;

  return { items, hasNextPage, nextCursor };
}
```

#### **Offset-Based Pagination (When Needed)**
```typescript
// Use offset pagination only when cursor-based isn't suitable
export async function getResourcesWithOffset(
  page: number,
  limit: number
): Promise<PaginatedResult<Resource>> {
  const offset = (page - 1) * limit;
  
  // Use efficient count query
  const [resources, total] = await Promise.all([
    prisma.resource.findMany({
      skip: offset,
      take: limit,
      where: {
        isDeleted: false,
        isPrivate: false
      },
      orderBy: { createdAt: 'desc' },
      // Use minimal select to reduce data transfer
      select: {
        id: true,
        publicId: true,
        resourceType: true,
        createdAt: true
      }
    }),
    // Optimize count query with index
    prisma.resource.count({
      where: {
        isDeleted: false,
        isPrivate: false
      }
    })
  ]);

  return {
    items: resources,
    hasNextPage: offset + limit < total,
    total,
    page,
    totalPages: Math.ceil(total / limit)
  };
}
```

### **Selective Field Loading**

#### **Minimal Data Fetching**
```typescript
// Only fetch required fields
export async function getResourceSummary(id: string) {
  return prisma.resource.findUnique({
    where: { publicId: id },
    select: {
      publicId: true,
      resourceType: true,
      score: true,
      bookmarks: true,
      views: true,
      createdAt: true,
      // Only fetch latest version
      versions: {
        where: { isLatest: true },
        select: {
          publicId: true,
          versionLabel: true,
          isComplete: true
        },
        take: 1
      },
      // Minimal creator info
      creator: {
        select: {
          publicId: true,
          name: true,
          handle: true
        }
      }
    }
  });
}
```

#### **Progressive Data Loading**
```typescript
// Load data progressively based on need
export async function getResourceDetails(id: string, includeSteps = false) {
  const baseResource = await prisma.resource.findUnique({
    where: { publicId: id },
    select: {
      id: true,
      publicId: true,
      resourceType: true,
      isPrivate: true,
      score: true,
      createdAt: true,
      versions: {
        where: { isLatest: true },
        select: {
          id: true,
          publicId: true,
          versionLabel: true,
          config: true,
          isComplete: true
        },
        take: 1
      }
    }
  });

  if (!baseResource) return null;

  // Only fetch additional data if needed
  const additionalData = await Promise.all([
    // Fetch creator info
    prisma.user.findUnique({
      where: { id: baseResource.createdById },
      select: { publicId: true, name: true, handle: true }
    }),
    
    // Fetch recent runs if requested
    includeSteps ? prisma.run.findMany({
      where: { 
        resourceVersionId: baseResource.versions[0]?.id
      },
      orderBy: { createdAt: 'desc' },
      take: 5,
      select: {
        publicId: true,
        name: true,
        status: true,
        startedAt: true,
        completedAt: true
      }
    }) : Promise.resolve([])
  ]);

  return {
    ...baseResource,
    creator: additionalData[0],
    recentRuns: additionalData[1]
  };
}
```

### **Efficient Joins**

#### **Strategic Include Usage**
```typescript
// Efficient join patterns
export async function getUserWithTeams(userId: string) {
  return prisma.user.findUnique({
    where: { publicId: userId },
    select: {
      publicId: true,
      name: true,
      handle: true,
      // Use select in includes to minimize data
      memberships: {
        select: {
          isAdmin: true,
          createdAt: true,
          team: {
            select: {
              publicId: true,
              name: true,
              handle: true,
              isPrivate: true,
              // Count members without fetching them
              _count: {
                select: { members: true }
              }
            }
          }
        },
        orderBy: { createdAt: 'desc' }
      }
    }
  });
}
```

#### **Batched Queries**
```typescript
// Batch related queries to reduce N+1 problems
export async function getResourcesWithCreators(resourceIds: string[]) {
  // First, get all resources
  const resources = await prisma.resource.findMany({
    where: {
      publicId: { in: resourceIds }
    },
    select: {
      publicId: true,
      resourceType: true,
      createdById: true,
      score: true,
      createdAt: true
    }
  });

  // Extract unique creator IDs
  const creatorIds = [...new Set(
    resources
      .map(r => r.createdById)
      .filter(id => id !== null)
  )];

  // Batch fetch creators
  const creators = await prisma.user.findMany({
    where: { id: { in: creatorIds } },
    select: {
      id: true,
      publicId: true,
      name: true,
      handle: true
    }
  });

  // Map creators by ID for O(1) lookup
  const creatorsMap = new Map(creators.map(c => [c.id.toString(), c]));

  // Combine data
  return resources.map(resource => ({
    ...resource,
    creator: resource.createdById ? creatorsMap.get(resource.createdById.toString()) : null
  }));
}
```

## 💾 Caching Strategy

### **Redis Caching Patterns**

#### **Query Result Caching**
```typescript
// Cache expensive query results
export async function getCachedResourceList(
  key: string,
  queryFn: () => Promise<any[]>,
  ttl: number = 300 // 5 minutes
) {
  const cached = await redis.get(key);
  if (cached) {
    return JSON.parse(cached);
  }

  const result = await queryFn();
  await redis.setex(key, ttl, JSON.stringify(result));
  return result;
}

// Usage example
export async function getTrendingResources() {
  return getCachedResourceList(
    'trending:resources:weekly',
    async () => {
      return prisma.resource.findMany({
        where: {
          isPrivate: false,
          hasCompleteVersion: true,
          createdAt: {
            gte: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000)
          }
        },
        orderBy: { score: 'desc' },
        take: 50,
        select: {
          publicId: true,
          resourceType: true,
          score: true,
          createdAt: true
        }
      });
    },
    3600 // Cache for 1 hour
  );
}
```

#### **Entity Caching**
```typescript
// Cache individual entities
export async function getCachedUser(publicId: string) {
  const cacheKey = `user:${publicId}`;
  const cached = await redis.get(cacheKey);
  
  if (cached) {
    return JSON.parse(cached);
  }

  const user = await prisma.user.findUnique({
    where: { publicId },
    select: {
      publicId: true,
      name: true,
      handle: true,
      isPrivate: true,
      languages: true,
      createdAt: true
    }
  });

  if (user) {
    await redis.setex(cacheKey, 1800, JSON.stringify(user)); // 30 minutes
  }

  return user;
}

// Cache invalidation
export async function invalidateUserCache(publicId: string) {
  await redis.del(`user:${publicId}`);
  // Invalidate related caches
  await redis.del(`user:${publicId}:teams`);
  await redis.del(`user:${publicId}:resources`);
}
```

#### **Count Caching**
```typescript
// Cache expensive count queries
export async function getCachedCounts() {
  const cacheKey = 'counts:global';
  const cached = await redis.get(cacheKey);
  
  if (cached) {
    return JSON.parse(cached);
  }

  const counts = await Promise.all([
    prisma.user.count({ where: { isBot: false } }),
    prisma.resource.count({ where: { isPrivate: false, hasCompleteVersion: true } }),
    prisma.run.count({ where: { status: 'Completed' } }),
    prisma.team.count({ where: { isPrivate: false } })
  ]);

  const result = {
    users: counts[0],
    resources: counts[1],
    completedRuns: counts[2],
    teams: counts[3],
    updatedAt: new Date()
  };

  await redis.setex(cacheKey, 7200, JSON.stringify(result)); // 2 hours
  return result;
}
```

### **Application-Level Caching**

#### **In-Memory Caching**
```typescript
// LRU cache for frequently accessed data
import LRU from 'lru-cache';

const resourceCache = new LRU<string, any>({
  max: 1000,
  maxAge: 1000 * 60 * 5 // 5 minutes
});

export async function getResourceWithCache(publicId: string) {
  const cached = resourceCache.get(publicId);
  if (cached) {
    return cached;
  }

  const resource = await prisma.resource.findUnique({
    where: { publicId },
    include: {
      versions: {
        where: { isLatest: true },
        take: 1
      }
    }
  });

  if (resource) {
    resourceCache.set(publicId, resource);
  }

  return resource;
}
```

## 📈 Database Scaling

### **Read Replicas**

#### **Read/Write Splitting**
```typescript
// Database connection management
class DatabaseManager {
  private primaryDB: PrismaClient;
  private replicaDBs: PrismaClient[];
  private currentReplicaIndex = 0;

  constructor() {
    this.primaryDB = new PrismaClient({
      datasources: { db: { url: process.env.PRIMARY_DB_URL } }
    });

    this.replicaDBs = [
      new PrismaClient({
        datasources: { db: { url: process.env.REPLICA_1_DB_URL } }
      }),
      new PrismaClient({
        datasources: { db: { url: process.env.REPLICA_2_DB_URL } }
      })
    ];
  }

  // Route reads to replicas
  getReadClient(): PrismaClient {
    const client = this.replicaDBs[this.currentReplicaIndex];
    this.currentReplicaIndex = (this.currentReplicaIndex + 1) % this.replicaDBs.length;
    return client;
  }

  // Route writes to primary
  getWriteClient(): PrismaClient {
    return this.primaryDB;
  }
}

// Usage
export async function getResource(id: string) {
  return db.getReadClient().resource.findUnique({
    where: { publicId: id }
  });
}

export async function createResource(data: any) {
  return db.getWriteClient().resource.create({
    data
  });
}
```

### **Connection Pooling**

#### **Optimized Pool Configuration**
```typescript
// Production connection pool settings
const prisma = new PrismaClient({
  datasources: {
    db: {
      url: `${process.env.DATABASE_URL}?connection_limit=20&pool_timeout=10&socket_timeout=30`
    }
  },
  log: process.env.NODE_ENV === 'development' ? ['query', 'info', 'warn', 'error'] : ['error']
});

// Pool monitoring
export async function getConnectionPoolStats() {
  const result = await prisma.$queryRaw`
    SELECT 
      count(*) as total_connections,
      count(*) filter (where state = 'active') as active_connections,
      count(*) filter (where state = 'idle') as idle_connections
    FROM pg_stat_activity 
    WHERE datname = current_database()
  `;
  
  return result[0];
}
```

## 🔧 Performance Monitoring

### **Query Performance Tracking**

#### **Slow Query Detection**
```sql
-- Enable pg_stat_statements
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Find slow queries
SELECT 
  query,
  calls,
  total_time,
  mean_time,
  rows,
  100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
FROM pg_stat_statements 
WHERE mean_time > 100 -- queries slower than 100ms
ORDER BY mean_time DESC 
LIMIT 20;
```

#### **Index Usage Analysis**
```sql
-- Check index usage
SELECT 
  schemaname,
  tablename,
  indexname,
  idx_tup_read,
  idx_tup_fetch,
  idx_scan
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;

-- Find unused indexes
SELECT 
  schemaname,
  tablename,
  indexname,
  pg_size_pretty(pg_relation_size(indexrelid)) as size
FROM pg_stat_user_indexes
WHERE idx_scan = 0 
  AND schemaname = 'public'
ORDER BY pg_relation_size(indexrelid) DESC;
```

### **Application Monitoring**

#### **Performance Metrics Collection**
```typescript
// Query performance middleware
export function performanceMiddleware(req: Request, res: Response, next: NextFunction) {
  const startTime = Date.now();
  
  res.on('finish', () => {
    const duration = Date.now() - startTime;
    const route = req.route?.path || req.path;
    
    // Log slow requests
    if (duration > 1000) {
      logger.warn('Slow request detected', {
        route,
        method: req.method,
        duration,
        statusCode: res.statusCode
      });
    }
    
    // Collect metrics
    performanceMetrics.recordRequest(route, req.method, duration, res.statusCode);
  });
  
  next();
}

// Database query timing
export async function timedQuery<T>(
  operation: string,
  queryFn: () => Promise<T>
): Promise<T> {
  const startTime = Date.now();
  
  try {
    const result = await queryFn();
    const duration = Date.now() - startTime;
    
    // Log slow queries
    if (duration > 500) {
      logger.warn('Slow query detected', { operation, duration });
    }
    
    // Record metrics
    performanceMetrics.recordQuery(operation, duration, 'success');
    
    return result;
  } catch (error) {
    const duration = Date.now() - startTime;
    performanceMetrics.recordQuery(operation, duration, 'error');
    throw error;
  }
}
```

### **Automated Performance Alerts**

#### **Alert Configuration**
```typescript
// Performance alert thresholds
const PERFORMANCE_THRESHOLDS = {
  avgQueryTime: 200, // ms
  p95QueryTime: 500, // ms
  slowQueriesPerHour: 10,
  cacheHitRate: 85, // %
  connectionPoolUtilization: 80, // %
  errorRate: 1 // %
};

export async function checkPerformanceAlerts() {
  const metrics = await getPerformanceMetrics();
  const alerts = [];

  if (metrics.avgQueryTime > PERFORMANCE_THRESHOLDS.avgQueryTime) {
    alerts.push({
      type: 'HIGH_AVG_QUERY_TIME',
      message: `Average query time ${metrics.avgQueryTime}ms exceeds threshold ${PERFORMANCE_THRESHOLDS.avgQueryTime}ms`,
      severity: 'warning'
    });
  }

  if (metrics.p95QueryTime > PERFORMANCE_THRESHOLDS.p95QueryTime) {
    alerts.push({
      type: 'HIGH_P95_QUERY_TIME',
      message: `P95 query time ${metrics.p95QueryTime}ms exceeds threshold ${PERFORMANCE_THRESHOLDS.p95QueryTime}ms`,
      severity: 'critical'
    });
  }

  if (metrics.cacheHitRate < PERFORMANCE_THRESHOLDS.cacheHitRate) {
    alerts.push({
      type: 'LOW_CACHE_HIT_RATE',
      message: `Cache hit rate ${metrics.cacheHitRate}% below threshold ${PERFORMANCE_THRESHOLDS.cacheHitRate}%`,
      severity: 'warning'
    });
  }

  // Send alerts if any
  if (alerts.length > 0) {
    await sendPerformanceAlerts(alerts);
  }

  return alerts;
}
```

---

**Related Documentation:**
- [Database Architecture](architecture.md) - Infrastructure configuration
- [Access Patterns](access-patterns.md) - Query implementation patterns  
- [Entity Model](entities.md) - Schema structure and relationships