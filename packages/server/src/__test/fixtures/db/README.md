# Database Fixtures Architecture

## Overview

Database fixtures represent the **persistence and seeding layer** in the Vrooli testing ecosystem. They handle:
- Prisma database operations and test data seeding
- Complex relationship setup between entities
- Integration with testcontainers (PostgreSQL, Redis)
- Cleanup and verification utilities
- Support for bulk operations and factory patterns

## Current Architecture

### Structure
The database fixtures currently follow a hybrid pattern:
- Individual fixture files for each model (e.g., `userFixtures.ts`, `chatFixtures.ts`)
- An `EnhancedDbFactory` base class providing common functionality
- Type definitions in `types.ts` for consistency
- A mix of static data exports and factory methods

### Strengths
1. **Type Safety**: Uses Prisma's generated types (`Prisma.UserCreateInput`, etc.)
2. **ID Generation**: Consistent use of `generatePK()` and `generatePublicId()`
3. **Factory Pattern**: `EnhancedDbFactory` provides a solid foundation
4. **Relationship Support**: Some fixtures handle relationships (e.g., user with auth)

### Areas for Improvement
1. **Inconsistent Patterns**: Mix of static exports and factory classes
2. **Limited Relationship Handling**: Complex relationships require manual setup
3. **No Bulk Operations**: Missing standardized bulk seeding utilities
4. **Verification Gaps**: No built-in relationship verification
5. **Cleanup Strategy**: No centralized cleanup utilities
6. **Transaction Support**: Missing transaction-based test isolation

## Ideal Architecture

### Core Principles
1. **Factory-First**: All fixtures should use the factory pattern for consistency
2. **Relationship-Aware**: Built-in support for complex object graphs
3. **Testcontainer Integration**: Seamless work with isolated database instances
4. **Verification Built-In**: Every factory includes state verification utilities
5. **Transaction Isolated**: Support for transaction-based test isolation

### Proposed Structure

```typescript
// Base factory interface that all database fixtures implement
interface DatabaseFixtureFactory<TPrismaCreateInput, TPrismaInclude> {
  // Core factory methods
  createMinimal: (overrides?: Partial<TPrismaCreateInput>) => Promise<DbResult>
  createComplete: (overrides?: Partial<TPrismaCreateInput>) => Promise<DbResult>
  createWithRelations: (config: RelationConfig) => Promise<DbResult>
  
  // Bulk operations
  seedMultiple: (count: number, template?: Partial<TPrismaCreateInput>) => Promise<DbResult[]>
  seedScenario: (scenario: TestScenario) => Promise<ScenarioResult>
  
  // Relationship management
  setupRelationships: (parentId: string, config: RelationConfig) => Promise<void>
  connectExisting: (id: string, relations: RelationConnections) => Promise<void>
  
  // Verification utilities
  verifyState: (id: string, expected: Partial<DbResult>) => Promise<void>
  verifyRelationships: (id: string, expectedCounts: RelationCounts) => Promise<void>
  verifyConstraints: (id: string) => Promise<ConstraintValidation>
  
  // Cleanup utilities
  cleanup: (ids: string[]) => Promise<void>
  cleanupAll: () => Promise<void>
  cleanupRelated: (id: string, depth?: number) => Promise<void>
}
```

### Example Implementation

```typescript
// UserDbFactory implementing the ideal pattern
export class UserDbFactory extends EnhancedDatabaseFactory<Prisma.UserCreateInput> {
  constructor(private prisma: PrismaClient) {
    super('User', prisma)
  }

  async createMinimal(overrides?: Partial<Prisma.UserCreateInput>) {
    const data = this.generateMinimalData(overrides)
    return await this.prisma.user.create({ data })
  }

  async createWithRelations(config: UserRelationConfig) {
    return await this.prisma.$transaction(async (tx) => {
      // Create base user
      const user = await tx.user.create({
        data: this.generateCompleteData(config.userOverrides)
      })

      // Setup authentication if requested
      if (config.withAuth) {
        await tx.auth.create({
          data: {
            userId: user.id,
            provider: 'Password',
            hashed_password: await hashPassword(config.password || 'test123')
          }
        })
      }

      // Setup team memberships
      if (config.teams?.length) {
        await tx.member.createMany({
          data: config.teams.map(team => ({
            id: generatePK(),
            userId: user.id,
            teamId: team.id,
            role: team.role
          }))
        })
      }

      return user
    })
  }

  async verifyRelationships(userId: string, expected: UserRelationCounts) {
    const user = await this.prisma.user.findUnique({
      where: { id: userId },
      include: {
        _count: {
          select: {
            emails: true,
            auths: true,
            teams: true,
            projects: true
          }
        }
      }
    })

    expect(user?._count).toMatchObject(expected)
  }
}
```

## Integration with Testcontainers

Database fixtures should work seamlessly with testcontainers for isolated testing:

```typescript
// Test setup with testcontainers
beforeAll(async () => {
  // Start PostgreSQL container
  const postgres = await new PostgreSQLContainer()
    .withDatabase('test_db')
    .start()

  // Start Redis container  
  const redis = await new RedisContainer().start()

  // Initialize Prisma with test database
  process.env.DATABASE_URL = postgres.getConnectionUri()
  prisma = new PrismaClient()
  await prisma.$connect()
})

beforeEach(async () => {
  // Start transaction for test isolation
  await prisma.$transaction(async (tx) => {
    // Run test with transaction-isolated prisma client
    global.testPrisma = tx
  })
})

afterEach(async () => {
  // Rollback transaction to clean state
  await prisma.$executeRaw`ROLLBACK`
})
```

## Relationship Patterns

### One-to-Many Relationships
```typescript
// Create user with multiple projects
const user = await userDbFactory.createWithRelations({
  projects: [
    { name: 'Project 1', isPrivate: false },
    { name: 'Project 2', isPrivate: true }
  ]
})
```

### Many-to-Many Relationships
```typescript
// Create team with members
const team = await teamDbFactory.createWithRelations({
  members: [
    { userId: user1.id, role: 'Owner' },
    { userId: user2.id, role: 'Member' }
  ]
})
```

### Complex Object Graphs
```typescript
// Create complete project structure
const project = await projectDbFactory.createScenario('fullProject', {
  team: { memberCount: 3 },
  versions: { count: 2, withRoutines: true },
  directory: { depth: 3, filesPerLevel: 5 }
})
```

## Cleanup Strategies

### Transaction-Based Cleanup
- Preferred method using database transactions
- Automatic rollback after each test
- No manual cleanup required

### ID-Based Cleanup
```typescript
// Track created IDs for cleanup
const createdIds = {
  users: [],
  projects: [],
  teams: []
}

afterEach(async () => {
  await userDbFactory.cleanup(createdIds.users)
  await projectDbFactory.cleanup(createdIds.projects)
  await teamDbFactory.cleanup(createdIds.teams)
})
```

### Cascade Cleanup
```typescript
// Clean up entity and all related data
await userDbFactory.cleanupRelated(userId, { 
  depth: 2, // How many relationship levels to traverse
  include: ['projects', 'teams', 'comments']
})
```

## Factory Pattern Implementation

All database fixtures should follow this pattern:

```typescript
export class ModelDbFactory extends EnhancedDatabaseFactory<Prisma.ModelCreateInput> {
  // Singleton instance per test context
  private static instances = new Map<PrismaClient, ModelDbFactory>()
  
  static getInstance(prisma: PrismaClient): ModelDbFactory {
    if (!this.instances.has(prisma)) {
      this.instances.set(prisma, new ModelDbFactory(prisma))
    }
    return this.instances.get(prisma)!
  }

  // Define test scenarios
  protected scenarios = {
    minimal: () => this.generateMinimalData(),
    complete: () => this.generateCompleteData(),
    withOwner: (userId: string) => ({
      ...this.generateMinimalData(),
      owner: { connect: { id: userId } }
    })
  }

  // Relationship configuration
  protected relationshipHandlers = {
    children: async (parentId: string, children: any[]) => {
      // Handle child creation
    },
    members: async (parentId: string, members: any[]) => {
      // Handle member assignment
    }
  }
}
```

## Integration with API Fixtures

Database fixtures support API fixtures for round-trip testing:

```typescript
// 1. Seed database with DB fixtures
const dbUser = await userDbFactory.createComplete()

// 2. Fetch via API using shape transformation
const apiResponse = await api.get(`/user/${dbUser.id}`)
const shapedUser = shapeUser.get(apiResponse.data)

// 3. Verify round-trip integrity
expect(shapedUser).toMatchObject({
  id: dbUser.id,
  name: dbUser.name,
  // ... other fields transformed by shape function
})
```

## Best Practices

### 1. Always Use Factories
```typescript
// ❌ Bad - Manual object creation
const user = await prisma.user.create({
  data: {
    id: generatePK(),
    name: 'Test User',
    // ... manual setup
  }
})

// ✅ Good - Factory usage
const user = await userDbFactory.createMinimal({ name: 'Test User' })
```

### 2. Leverage Transactions
```typescript
// ❌ Bad - Multiple separate operations
const user = await prisma.user.create({ data: userData })
const team = await prisma.team.create({ data: teamData })
await prisma.member.create({ data: { userId: user.id, teamId: team.id } })

// ✅ Good - Transaction-wrapped operations
const result = await userDbFactory.createWithRelations({
  teams: [{ teamId: team.id, role: 'Member' }]
})
```

### 3. Verify Relationships
```typescript
// Always verify complex relationships after creation
const project = await projectDbFactory.createWithRelations({
  team: true,
  versions: 3,
  directory: true
})

await projectDbFactory.verifyRelationships(project.id, {
  team: 1,
  versions: 3,
  directories: 1
})
```

### 4. Use Appropriate Scenarios
```typescript
// Choose the right level of complexity for your test
const simple = await userDbFactory.createMinimal() // Basic tests
const full = await userDbFactory.createComplete() // Integration tests
const complex = await userDbFactory.createScenario('enterpriseUser') // E2E tests
```

## Migration Guide

To migrate existing fixtures to the ideal architecture:

1. **Extend EnhancedDatabaseFactory**
   ```typescript
   export class YourModelDbFactory extends EnhancedDatabaseFactory<Prisma.YourModelCreateInput> {
     // Implementation
   }
   ```

2. **Move Static Data to Factory Methods**
   ```typescript
   // Old
   export const minimalUser = { ... }
   
   // New
   protected generateMinimalData() {
     return { ... }
   }
   ```

3. **Add Relationship Handlers**
   ```typescript
   protected relationshipHandlers = {
     relatedModel: async (parentId, config) => {
       // Handle relationship creation
     }
   }
   ```

4. **Implement Verification Methods**
   ```typescript
   async verifyState(id: string, expected: Partial<Model>) {
     const actual = await this.prisma.model.findUnique({ where: { id } })
     expect(actual).toMatchObject(expected)
   }
   ```

5. **Add Cleanup Utilities**
   ```typescript
   async cleanup(ids: string[]) {
     await this.prisma.model.deleteMany({
       where: { id: { in: ids } }
     })
   }
   ```

## Future Enhancements

1. **Snapshot Testing**: Save and restore complete database states
2. **Performance Fixtures**: Large datasets for performance testing
3. **GraphQL Integration**: Support for GraphQL testing scenarios
4. **AI Context Fixtures**: Specialized fixtures for AI tier testing
5. **Event Sourcing**: Fixtures that include event history

## Cross-Reference with Unified Testing System

This database fixture architecture integrates with the broader testing ecosystem:

- **API Fixtures**: Database fixtures provide the persistence layer for API testing
- **Config Fixtures**: Database fixtures can use config fixtures for consistent settings
- **UI Testing**: Database fixtures seed the backend for E2E UI tests
- **Round-Trip Testing**: Database → API → UI → Database verification

See [Fixtures Overview](/docs/testing/fixtures-overview.md#database-fixtures) for more details on how database fixtures fit into the unified testing architecture.