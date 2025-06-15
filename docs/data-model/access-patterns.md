# Data Access Patterns

Comprehensive guide to data access patterns, repository implementations, transaction management, and best practices for working with Vrooli's database.

## 🏗️ Repository Pattern

### **Base Repository Implementation**

```typescript
// Base repository with common operations
export abstract class BaseRepository<T, CreateInput, UpdateInput> {
  protected abstract model: any;
  protected abstract selectFields: any;
  
  async findById(id: bigint): Promise<T | null> {
    return this.model.findUnique({
      where: { id },
      select: this.selectFields
    });
  }
  
  async findByPublicId(publicId: string): Promise<T | null> {
    return this.model.findUnique({
      where: { publicId },
      select: this.selectFields
    });
  }
  
  async create(data: CreateInput): Promise<T> {
    return this.model.create({
      data,
      select: this.selectFields
    });
  }
  
  async update(id: bigint, data: UpdateInput): Promise<T | null> {
    return this.model.update({
      where: { id },
      data,
      select: this.selectFields
    });
  }
  
  async delete(id: bigint): Promise<boolean> {
    try {
      await this.model.delete({
        where: { id }
      });
      return true;
    } catch (error) {
      return false;
    }
  }
  
  async findMany(options: FindManyOptions<T>): Promise<PaginatedResult<T>> {
    const { where, orderBy, cursor, limit = 20 } = options;
    
    const items = await this.model.findMany({
      where,
      orderBy,
      cursor: cursor ? { id: BigInt(cursor) } : undefined,
      take: limit + 1,
      select: this.selectFields
    });
    
    const hasNextPage = items.length > limit;
    const data = hasNextPage ? items.slice(0, -1) : items;
    const nextCursor = hasNextPage ? data[data.length - 1].id.toString() : null;
    
    return { data, hasNextPage, nextCursor };
  }
}
```

### **Entity-Specific Repositories**

#### **User Repository**
```typescript
export class UserRepository extends BaseRepository<User, CreateUserInput, UpdateUserInput> {
  protected model = prisma.user;
  protected selectFields = {
    id: true,
    publicId: true,
    email: true,
    name: true,
    handle: true,
    isBot: true,
    isPrivate: true,
    languages: true,
    createdAt: true,
    updatedAt: true
  };
  
  async findByEmail(email: string): Promise<User | null> {
    return this.model.findUnique({
      where: { email: email.toLowerCase() },
      select: this.selectFields
    });
  }
  
  async findByHandle(handle: string): Promise<User | null> {
    return this.model.findUnique({
      where: { handle: handle.toLowerCase() },
      select: this.selectFields
    });
  }
  
  async findWithTeams(publicId: string): Promise<UserWithTeams | null> {
    return this.model.findUnique({
      where: { publicId },
      select: {
        ...this.selectFields,
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
                _count: { select: { members: true } }
              }
            }
          },
          orderBy: { createdAt: 'desc' }
        }
      }
    });
  }
  
  async searchUsers(query: string, limit: number = 10): Promise<User[]> {
    return this.model.findMany({
      where: {
        OR: [
          { name: { contains: query, mode: 'insensitive' } },
          { handle: { contains: query, mode: 'insensitive' } },
          { email: { contains: query, mode: 'insensitive' } }
        ],
        isPrivate: false
      },
      select: this.selectFields,
      take: limit,
      orderBy: { createdAt: 'desc' }
    });
  }
  
  async updateLastActive(publicId: string): Promise<void> {
    await this.model.update({
      where: { publicId },
      data: { lastActiveAt: new Date() }
    });
  }
}
```

#### **Resource Repository**
```typescript
export class ResourceRepository extends BaseRepository<Resource, CreateResourceInput, UpdateResourceInput> {
  protected model = prisma.resource;
  protected selectFields = {
    id: true,
    publicId: true,
    resourceType: true,
    isPrivate: true,
    hasCompleteVersion: true,
    score: true,
    bookmarks: true,
    views: true,
    createdAt: true,
    updatedAt: true
  };
  
  async findWithLatestVersion(publicId: string): Promise<ResourceWithVersion | null> {
    return this.model.findUnique({
      where: { publicId },
      select: {
        ...this.selectFields,
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
            id: true,
            publicId: true,
            versionLabel: true,
            config: true,
            isComplete: true,
            createdAt: true
          },
          take: 1
        }
      }
    });
  }
  
  async findByType(
    resourceType: ResourceType,
    options: FindResourceOptions = {}
  ): Promise<PaginatedResult<Resource>> {
    const { isPrivate = false, limit = 20, cursor, orderBy = 'createdAt' } = options;
    
    return this.findMany({
      where: {
        resourceType,
        isPrivate,
        isDeleted: false,
        hasCompleteVersion: true
      },
      orderBy: { [orderBy]: 'desc' },
      cursor,
      limit
    });
  }
  
  async findTrending(days: number = 7, limit: number = 50): Promise<Resource[]> {
    const since = new Date(Date.now() - days * 24 * 60 * 60 * 1000);
    
    return this.model.findMany({
      where: {
        isPrivate: false,
        hasCompleteVersion: true,
        isDeleted: false,
        createdAt: { gte: since }
      },
      select: this.selectFields,
      orderBy: [
        { score: 'desc' },
        { bookmarks: 'desc' },
        { views: 'desc' }
      ],
      take: limit
    });
  }
  
  async incrementViews(publicId: string): Promise<void> {
    await this.model.update({
      where: { publicId },
      data: { views: { increment: 1 } }
    });
  }
  
  async incrementBookmarks(publicId: string): Promise<void> {
    await this.model.update({
      where: { publicId },
      data: { bookmarks: { increment: 1 } }
    });
  }
  
  async searchResources(
    query: string,
    filters: ResourceSearchFilters = {}
  ): Promise<Resource[]> {
    const { resourceType, limit = 20 } = filters;
    
    return this.model.findMany({
      where: {
        AND: [
          {
            OR: [
              { 
                translations: {
                  some: {
                    name: { contains: query, mode: 'insensitive' }
                  }
                }
              },
              {
                translations: {
                  some: {
                    description: { contains: query, mode: 'insensitive' }
                  }
                }
              }
            ]
          },
          resourceType ? { resourceType } : {},
          { isPrivate: false },
          { isDeleted: false },
          { hasCompleteVersion: true }
        ]
      },
      select: this.selectFields,
      take: limit,
      orderBy: { score: 'desc' }
    });
  }
}
```

#### **Run Repository**
```typescript
export class RunRepository extends BaseRepository<Run, CreateRunInput, UpdateRunInput> {
  protected model = prisma.run;
  protected selectFields = {
    id: true,
    name: true,
    status: true,
    isPrivate: true,
    startedAt: true,
    completedAt: true,
    timeElapsed: true,
    completedComplexity: true,
    contextSwitches: true,
    createdAt: true,
    updatedAt: true
  };
  
  async findWithDetails(id: string): Promise<RunWithDetails | null> {
    return this.model.findUnique({
      where: { id: BigInt(id) },
      select: {
        ...this.selectFields,
        data: true,
        resourceVersion: {
          select: {
            publicId: true,
            versionLabel: true,
            root: {
              select: {
                publicId: true,
                resourceType: true
              }
            }
          }
        },
        user: {
          select: {
            publicId: true,
            name: true,
            handle: true
          }
        },
        steps: {
          select: {
            id: true,
            name: true,
            status: true,
            step: true,
            timeElapsed: true,
            createdAt: true
          },
          orderBy: { step: 'asc' }
        },
        inputs: {
          select: {
            key: true,
            value: true
          }
        },
        outputs: {
          select: {
            key: true,
            value: true
          }
        }
      }
    });
  }
  
  async findByStatus(
    status: RunStatus[],
    options: FindRunOptions = {}
  ): Promise<PaginatedResult<Run>> {
    const { userId, limit = 20, cursor } = options;
    
    return this.findMany({
      where: {
        status: { in: status },
        ...(userId && { userId: BigInt(userId) })
      },
      orderBy: { createdAt: 'desc' },
      cursor,
      limit
    });
  }
  
  async findActiveRuns(userId?: string): Promise<Run[]> {
    return this.model.findMany({
      where: {
        status: { in: ['Scheduled', 'InProgress'] },
        ...(userId && { userId: BigInt(userId) })
      },
      select: this.selectFields,
      orderBy: { createdAt: 'desc' }
    });
  }
  
  async updateStatus(id: string, status: RunStatus, metadata?: any): Promise<Run | null> {
    const updateData: any = { status };
    
    if (status === 'InProgress') {
      updateData.startedAt = new Date();
    } else if (status === 'Completed' || status === 'Failed') {
      updateData.completedAt = new Date();
      if (metadata?.timeElapsed) {
        updateData.timeElapsed = metadata.timeElapsed;
      }
    }
    
    return this.model.update({
      where: { id: BigInt(id) },
      data: updateData,
      select: this.selectFields
    });
  }
  
  async getExecutionStats(userId?: string): Promise<ExecutionStats> {
    const where = userId ? { userId: BigInt(userId) } : {};
    
    const stats = await this.model.groupBy({
      by: ['status'],
      where,
      _count: true,
      _avg: { timeElapsed: true }
    });
    
    return {
      total: stats.reduce((sum, stat) => sum + stat._count, 0),
      byStatus: Object.fromEntries(
        stats.map(stat => [stat.status, stat._count])
      ),
      averageExecutionTime: stats
        .filter(stat => stat.status === 'Completed')
        .reduce((sum, stat) => sum + (stat._avg.timeElapsed || 0), 0)
    };
  }
}
```

## 💾 Transaction Patterns

### **Simple Transactions**

#### **Basic Transaction Wrapper**
```typescript
export async function withTransaction<T>(
  operation: (tx: PrismaTransaction) => Promise<T>
): Promise<T> {
  return prisma.$transaction(async (tx) => {
    try {
      return await operation(tx);
    } catch (error) {
      // Transaction will automatically rollback
      throw error;
    }
  });
}

// Usage example
export async function createResourceWithVersion(
  resourceData: CreateResourceInput,
  versionData: CreateResourceVersionInput
): Promise<{ resource: Resource; version: ResourceVersion }> {
  return withTransaction(async (tx) => {
    // Create resource
    const resource = await tx.resource.create({
      data: resourceData
    });
    
    // Create initial version
    const version = await tx.resourceVersion.create({
      data: {
        ...versionData,
        rootId: resource.id,
        versionIndex: 0,
        isLatest: true
      }
    });
    
    // Update resource completion status
    await tx.resource.update({
      where: { id: resource.id },
      data: { hasCompleteVersion: version.isComplete }
    });
    
    return { resource, version };
  });
}
```

### **Complex Multi-Entity Transactions**

#### **Team Creation with Initial Setup**
```typescript
export async function createTeamWithInitialSetup(
  teamData: CreateTeamInput,
  creatorId: string,
  initialMembers: string[] = []
): Promise<TeamCreationResult> {
  return withTransaction(async (tx) => {
    // 1. Create team
    const team = await tx.team.create({
      data: teamData
    });
    
    // 2. Add creator as admin member
    await tx.member.create({
      data: {
        userId: BigInt(creatorId),
        teamId: team.id,
        isAdmin: true,
        permissions: DEFAULT_ADMIN_PERMISSIONS
      }
    });
    
    // 3. Add initial members
    if (initialMembers.length > 0) {
      await tx.member.createMany({
        data: initialMembers.map(userId => ({
          userId: BigInt(userId),
          teamId: team.id,
          isAdmin: false,
          permissions: DEFAULT_MEMBER_PERMISSIONS
        }))
      });
    }
    
    // 4. Create default team chat
    const chat = await tx.chat.create({
      data: {
        isPrivate: teamData.isPrivate,
        openToAnyoneWithInvite: false,
        config: DEFAULT_TEAM_CHAT_CONFIG,
        creatorId: BigInt(creatorId),
        teamId: team.id
      }
    });
    
    // 5. Add all members to chat
    const allMemberIds = [creatorId, ...initialMembers];
    await tx.chatParticipants.createMany({
      data: allMemberIds.map(userId => ({
        userId: BigInt(userId),
        chatId: chat.id,
        hasUnread: false
      }))
    });
    
    // 6. Create team statistics record
    await tx.statsTeam.create({
      data: {
        teamId: team.id,
        periodType: 'Daily',
        periodStart: new Date(),
        periodEnd: new Date(),
        membersCreated: allMemberIds.length
      }
    });
    
    return {
      team,
      chat,
      memberCount: allMemberIds.length
    };
  });
}
```

#### **Run Execution Transaction**
```typescript
export async function executeRunWithSteps(
  runId: string,
  steps: ExecutionStep[]
): Promise<RunExecutionResult> {
  return withTransaction(async (tx) => {
    // 1. Update run status to InProgress
    const run = await tx.run.update({
      where: { id: BigInt(runId) },
      data: {
        status: 'InProgress',
        startedAt: new Date()
      }
    });
    
    // 2. Create run steps
    const createdSteps = await Promise.all(
      steps.map((step, index) =>
        tx.runStep.create({
          data: {
            runId: run.id,
            name: step.name,
            status: 'Scheduled',
            step: index + 1
          }
        })
      )
    );
    
    // 3. Create input records
    if (steps.some(s => s.inputs)) {
      const inputRecords = steps.flatMap(step =>
        step.inputs?.map(input => ({
          runId: run.id,
          side: 'input' as const,
          key: input.key,
          value: JSON.stringify(input.value)
        })) || []
      );
      
      if (inputRecords.length > 0) {
        await tx.runIO.createMany({
          data: inputRecords
        });
      }
    }
    
    // 4. Update user statistics
    await tx.statsUser.upsert({
      where: {
        userId_periodType_periodStart: {
          userId: run.userId!,
          periodType: 'Daily',
          periodStart: new Date(new Date().setHours(0, 0, 0, 0))
        }
      },
      create: {
        userId: run.userId!,
        periodType: 'Daily',
        periodStart: new Date(new Date().setHours(0, 0, 0, 0)),
        periodEnd: new Date(new Date().setHours(23, 59, 59, 999)),
        runsStarted: 1
      },
      update: {
        runsStarted: { increment: 1 }
      }
    });
    
    return {
      run,
      steps: createdSteps,
      inputCount: inputRecords?.length || 0
    };
  });
}
```

### **Transaction Error Handling**

#### **Retry Logic for Transactions**
```typescript
export async function withRetryTransaction<T>(
  operation: (tx: PrismaTransaction) => Promise<T>,
  maxRetries: number = 3,
  retryDelay: number = 1000
): Promise<T> {
  let lastError: Error;
  
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      return await withTransaction(operation);
    } catch (error) {
      lastError = error as Error;
      
      // Check if error is retryable
      if (!isRetryableError(error) || attempt === maxRetries) {
        throw error;
      }
      
      // Wait before retry with exponential backoff
      await new Promise(resolve => 
        setTimeout(resolve, retryDelay * Math.pow(2, attempt - 1))
      );
    }
  }
  
  throw lastError!;
}

function isRetryableError(error: any): boolean {
  // PostgreSQL serialization failure
  if (error.code === 'P2034') return true;
  
  // Connection errors
  if (error.code === 'P1001' || error.code === 'P1002') return true;
  
  // Deadlock
  if (error.code === 'P2034') return true;
  
  return false;
}
```

#### **Nested Transaction Handling**
```typescript
export async function handleNestedOperations(
  parentData: ParentCreateInput,
  childrenData: ChildCreateInput[]
): Promise<ParentWithChildren> {
  return withTransaction(async (tx) => {
    // Create parent
    const parent = await tx.parent.create({
      data: parentData
    });
    
    // Handle potential errors in child creation
    const children = [];
    const errors = [];
    
    for (const childData of childrenData) {
      try {
        const child = await tx.child.create({
          data: {
            ...childData,
            parentId: parent.id
          }
        });
        children.push(child);
      } catch (error) {
        errors.push({
          childData,
          error: error.message
        });
      }
    }
    
    // If too many children failed, rollback entire transaction
    if (errors.length > childrenData.length * 0.5) {
      throw new Error(`Too many child creation failures: ${errors.length}/${childrenData.length}`);
    }
    
    // Log non-critical errors but continue
    if (errors.length > 0) {
      logger.warn('Some child entities failed to create', { errors });
    }
    
    return {
      parent,
      children,
      errors
    };
  });
}
```

## 📦 Bulk Operations

### **Efficient Bulk Inserts**

#### **Batch Processing**
```typescript
export async function bulkCreateResources(
  resources: CreateResourceInput[],
  batchSize: number = 100
): Promise<BulkOperationResult> {
  const results = [];
  const errors = [];
  
  for (let i = 0; i < resources.length; i += batchSize) {
    const batch = resources.slice(i, i + batchSize);
    
    try {
      // Use createMany for better performance
      const result = await prisma.resource.createMany({
        data: batch,
        skipDuplicates: true
      });
      
      results.push({
        batchIndex: Math.floor(i / batchSize),
        count: result.count,
        success: true
      });
    } catch (error) {
      // Handle batch failure by processing individually
      for (const item of batch) {
        try {
          await prisma.resource.create({ data: item });
          results.push({ item, success: true });
        } catch (itemError) {
          errors.push({ item, error: itemError.message });
        }
      }
    }
  }
  
  return {
    totalProcessed: resources.length,
    successful: results.length,
    failed: errors.length,
    errors
  };
}
```

#### **Bulk Updates with Optimistic Locking**
```typescript
export async function bulkUpdateResources(
  updates: { id: string; data: UpdateResourceInput; version: number }[]
): Promise<BulkUpdateResult> {
  const results = [];
  const conflicts = [];
  
  await withTransaction(async (tx) => {
    for (const update of updates) {
      try {
        // Check current version for optimistic locking
        const current = await tx.resource.findUnique({
          where: { id: BigInt(update.id) },
          select: { id: true, updatedAt: true }
        });
        
        if (!current) {
          conflicts.push({ id: update.id, reason: 'Resource not found' });
          continue;
        }
        
        // Simple version check using updatedAt timestamp
        const currentVersion = current.updatedAt.getTime();
        if (currentVersion !== update.version) {
          conflicts.push({ 
            id: update.id, 
            reason: 'Version conflict',
            expected: update.version,
            actual: currentVersion
          });
          continue;
        }
        
        // Perform update
        const updated = await tx.resource.update({
          where: { id: BigInt(update.id) },
          data: update.data
        });
        
        results.push({ id: update.id, success: true, newVersion: updated.updatedAt.getTime() });
      } catch (error) {
        conflicts.push({ id: update.id, reason: error.message });
      }
    }
    
    // Rollback if too many conflicts
    if (conflicts.length > updates.length * 0.3) {
      throw new Error(`Too many update conflicts: ${conflicts.length}/${updates.length}`);
    }
  });
  
  return {
    updated: results.length,
    conflicts: conflicts.length,
    conflictDetails: conflicts
  };
}
```

### **Bulk Delete Operations**

#### **Soft Delete with Cleanup**
```typescript
export async function bulkSoftDeleteResources(
  resourceIds: string[],
  deletedBy: string
): Promise<BulkDeleteResult> {
  return withTransaction(async (tx) => {
    const timestamp = new Date();
    
    // 1. Mark resources as deleted
    const deleteResult = await tx.resource.updateMany({
      where: {
        id: { in: resourceIds.map(id => BigInt(id)) },
        isDeleted: false
      },
      data: {
        isDeleted: true,
        deletedAt: timestamp,
        deletedBy: BigInt(deletedBy)
      }
    });
    
    // 2. Cancel any active runs
    await tx.run.updateMany({
      where: {
        resourceVersion: {
          rootId: { in: resourceIds.map(id => BigInt(id)) }
        },
        status: { in: ['Scheduled', 'InProgress'] }
      },
      data: {
        status: 'Cancelled'
      }
    });
    
    // 3. Update statistics
    await tx.statsUser.updateMany({
      where: {
        userId: BigInt(deletedBy),
        periodType: 'Daily',
        periodStart: {
          gte: new Date(new Date().setHours(0, 0, 0, 0))
        }
      },
      data: {
        resourcesDeleted: { increment: deleteResult.count }
      }
    });
    
    return {
      deletedCount: deleteResult.count,
      cancelledRuns: await tx.run.count({
        where: {
          status: 'Cancelled',
          updatedAt: { gte: timestamp }
        }
      })
    };
  });
}
```

## 🔄 Data Synchronization Patterns

### **Event-Driven Updates**

#### **Change Event Publishing**
```typescript
export async function updateResourceWithEvents(
  resourceId: string,
  updateData: UpdateResourceInput,
  userId: string
): Promise<Resource> {
  return withTransaction(async (tx) => {
    // 1. Get current state
    const currentResource = await tx.resource.findUnique({
      where: { id: BigInt(resourceId) }
    });
    
    if (!currentResource) {
      throw new Error('Resource not found');
    }
    
    // 2. Perform update
    const updatedResource = await tx.resource.update({
      where: { id: BigInt(resourceId) },
      data: updateData
    });
    
    // 3. Record change event
    await tx.changeEvent.create({
      data: {
        entityType: 'Resource',
        entityId: resourceId,
        changeType: 'UPDATE',
        userId: BigInt(userId),
        previousData: JSON.stringify(currentResource),
        newData: JSON.stringify(updatedResource),
        timestamp: new Date()
      }
    });
    
    // 4. Publish event for real-time updates
    await publishChangeEvent({
      type: 'resource.updated',
      resourceId,
      changes: getChangedFields(currentResource, updatedResource),
      userId
    });
    
    // 5. Invalidate caches
    await invalidateResourceCache(resourceId);
    
    return updatedResource;
  });
}

function getChangedFields(previous: any, current: any): string[] {
  const changes = [];
  for (const key in current) {
    if (previous[key] !== current[key]) {
      changes.push(key);
    }
  }
  return changes;
}
```

### **Data Consistency Patterns**

#### **Eventually Consistent Counters**
```typescript
// Handle denormalized counter updates
export async function updateResourceBookmarkCount(
  resourceId: string,
  increment: boolean
): Promise<void> {
  // Update the denormalized counter
  await prisma.resource.update({
    where: { id: BigInt(resourceId) },
    data: {
      bookmarks: {
        [increment ? 'increment' : 'decrement']: 1
      }
    }
  });
  
  // Queue a reconciliation job for eventual consistency
  await queueCounterReconciliation('resource', resourceId, 'bookmarks');
}

// Reconciliation job to ensure counter accuracy
export async function reconcileResourceBookmarks(resourceId: string): Promise<void> {
  await withTransaction(async (tx) => {
    // Count actual bookmarks
    const actualCount = await tx.bookmark.count({
      where: { resourceId: BigInt(resourceId) }
    });
    
    // Update resource with correct count
    await tx.resource.update({
      where: { id: BigInt(resourceId) },
      data: { bookmarks: actualCount }
    });
  });
}
```

---

**Related Documentation:**
- [Entity Model](entities.md) - Database schema and relationships
- [Performance Guide](performance.md) - Query optimization strategies
- [Database Architecture](architecture.md) - Infrastructure configuration