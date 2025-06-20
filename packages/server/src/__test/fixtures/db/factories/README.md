# Database Fixture Factories - Core Business Objects Part 2

This directory contains database fixture factories for the core business objects in Vrooli, specifically focusing on **Routine**, **RoutineVersion**, **Resource**, **ResourceVersion**, and **ResourceVersionRelation** models.

## Overview

These factories implement the enhanced database fixture factory pattern, providing comprehensive test data creation capabilities including:

- **Relationship management**: Automatic handling of complex object relationships
- **Bulk operations**: Efficient creation of multiple related objects
- **State verification**: Built-in constraint validation and state checking
- **Cleanup utilities**: Automatic cleanup of created test data
- **Transaction support**: Safe, atomic operations

## Factory Classes

### RoutineDbFactory

Handles creation of **Routine** objects with complex workflow definitions and JSON configurations.

**Features:**
- Complex workflow definitions with BPMN configurations
- Version management with multiple versions per routine
- Owner relationships (user or team ownership)
- Tag associations for categorization
- Permission management

**Key Methods:**
```typescript
// Create different types of routines
createSimpleTask(ownerId: string): Promise<Prisma.Routine>
createComplexWorkflow(ownerId: string): Promise<Prisma.Routine>
createAutomatedRoutine(teamId: string): Promise<Prisma.Routine>
createWithVersionHistory(ownerId: string): Promise<Prisma.Routine>
```

**Required Scenarios:**
- Simple task routines
- Complex workflows with branching logic
- Automated routines with AI integration

### RoutineVersionDbFactory

Manages **RoutineVersion** objects with versioned configurations and complex graph structures.

**Features:**
- Version management with `isLatest` flag handling
- Complex JSON configurations for different routine types
- Resource list attachments for documentation
- Multi-language translation support
- Graph structure management for multi-step workflows

**Key Methods:**
```typescript
// Create different version types
createSimpleAction(routineId: string): Promise<Prisma.RoutineVersion>
createTextGeneration(routineId: string): Promise<Prisma.RoutineVersion>
createMultiStepWorkflow(routineId: string): Promise<Prisma.RoutineVersion>
createDraftVersion(routineId: string): Promise<Prisma.RoutineVersion>
```

**Configuration Types:**
- Action routines (tool execution)
- Generate routines (AI text generation)
- Multi-step workflows (BPMN-based)

### ResourceDbFactory

Creates **Resource** objects for external links, documentation, and tutorials.

**Features:**
- Multiple resource types (Documentation, Tutorial, Code, etc.)
- External link management
- Version tracking with semantic versioning
- Permission and privacy controls
- Tag-based categorization

**Key Methods:**
```typescript
// Create specific resource types
createDocumentation(ownerId: string): Promise<Prisma.Resource>
createVideoTutorial(ownerId: string): Promise<Prisma.Resource>
createExternalTool(teamId: string): Promise<Prisma.Resource>
createWithVersionHistory(ownerId: string): Promise<Prisma.Resource>
createPrivateInternal(teamId: string): Promise<Prisma.Resource>
```

**Resource Types:**
- Documentation
- Video tutorials
- External tools
- Code repositories

### ResourceVersionDbFactory

Handles **ResourceVersion** objects with versioned content and metadata.

**Features:**
- Semantic versioning support
- Rich metadata (instructions, details, descriptions)
- Multi-language translations
- Link validation and management
- Complexity scoring

**Key Methods:**
```typescript
// Create different version types
createDocumentationVersion(resourceId: string): Promise<Prisma.ResourceVersion>
createTutorialVersionWithRelations(resourceId: string, relations: Array<...>): Promise<Prisma.ResourceVersion>
createCodeRepositoryVersion(resourceId: string): Promise<Prisma.ResourceVersion>
createDraftVersion(resourceId: string): Promise<Prisma.ResourceVersion>
createWithFullMetadata(resourceId: string): Promise<Prisma.ResourceVersion>
```

### ResourceVersionRelationDbFactory

Manages the **ResourceVersionRelation** junction table connecting resources to other versioned objects.

**Features:**
- Junction table pattern implementation
- Ordered relationship management
- Multiple target type support
- Bulk relation operations
- Index-based ordering

**Key Methods:**
```typescript
// Create relations to different target types
createResourceToProject(resourceVersionId: string, projectVersionId: string, index?: number): Promise<Prisma.ResourceVersionRelation>
createResourceToRoutine(resourceVersionId: string, routineVersionId: string, index?: number): Promise<Prisma.ResourceVersionRelation>
createResourceToSmartContract(resourceVersionId: string, smartContractVersionId: string, index?: number): Promise<Prisma.ResourceVersionRelation>
createResourceToDataStructure(resourceVersionId: string, dataStructureVersionId: string, index?: number): Promise<Prisma.ResourceVersionRelation>

// Bulk operations
createMultipleRelations(resourceVersionId: string, targets: Array<...>): Promise<Prisma.ResourceVersionRelation[]>
createOrderedRelations(resourceVersionId: string, targetIds: string[], targetType: string): Promise<Prisma.ResourceVersionRelation[]>
```

**Supported Target Types:**
- ProjectVersion
- RoutineVersion  
- SmartContractVersion
- DataStructureVersion

## Usage Examples

### Basic Usage

```typescript
import { createRoutineDbFactory, createResourceDbFactory } from "./factories";

const routineFactory = createRoutineDbFactory(prisma);
const resourceFactory = createResourceDbFactory(prisma);

// Create a complex workflow routine
const routine = await routineFactory.createComplexWorkflow(userId);

// Create documentation for the routine
const documentation = await resourceFactory.createDocumentation(userId);

// Link them together
const relation = await resourceVersionRelationFactory.createResourceToRoutine(
    documentation.versions[0].id,
    routine.versions[0].id
);
```

### Complex Scenario Creation

```typescript
// Create a complete workflow with all related objects
const workflow = async () => {
    // 1. Create routine with multiple versions
    const routine = await routineFactory.createWithVersionHistory(userId);
    
    // 2. Create supporting resources
    const docs = await resourceFactory.createDocumentation(userId);
    const tutorial = await resourceFactory.createVideoTutorial(userId);
    
    // 3. Link resources to routine
    await resourceVersionRelationFactory.createMultipleRelations(
        docs.versions[0].id,
        [
            { id: routine.versions[0].id, type: "RoutineVersion" },
            { id: routine.versions[1].id, type: "RoutineVersion" },
        ]
    );
    
    return { routine, docs, tutorial };
};
```

### Test Scenarios

```typescript
describe("Routine Integration", () => {
    test("should create routine with attached resources", async () => {
        const routine = await routineFactory.createComplexWorkflow(userId);
        const resource = await resourceFactory.createDocumentation(userId);
        
        const relation = await resourceVersionRelationFactory.createResourceToRoutine(
            resource.versions[0].id,
            routine.versions[0].id
        );
        
        expect(relation.resourceVersionId).toBe(resource.versions[0].id);
        expect(relation.routineVersionId).toBe(routine.versions[0].id);
    });
});
```

## Configuration Examples

### Routine Configurations

The factories use predefined configurations from `@vrooli/shared/__test/fixtures/config`:

```typescript
// Simple action routine
config: routineConfigFixtures.action.simple

// Complex multi-step workflow
config: routineConfigFixtures.multiStep.complexWorkflow

// AI text generation
config: routineConfigFixtures.generate.withComplexPrompt
```

### Resource Types

Resources support various types for different use cases:

```typescript
// Documentation
resourceType: ResourceType.Documentation

// Video tutorial
resourceType: ResourceType.Tutorial

// External tool
resourceType: ResourceType.ExternalTool

// Code repository
resourceType: ResourceType.Code
```

## Constraint Validation

All factories include built-in constraint validation:

### Routine Constraints
- Must have at least one version
- Must have exactly one latest version
- Must have an owner (user or team, not both)

### RoutineVersion Constraints
- Version label must follow semantic versioning
- Complexity must be between 1-10
- Only one version can be marked as latest per routine
- Config must have `__version` field

### Resource Constraints
- Must have at least one version
- Must have exactly one latest version
- Private resources must have an owner
- Internal resources should be team-owned

### ResourceVersion Constraints
- Version label must follow semantic versioning
- Complexity must be between 1-10
- Links must be valid URLs
- Only one version can be marked as latest per resource

### ResourceVersionRelation Constraints
- Must connect to exactly one target type
- Index must be non-negative
- Index must be unique within the same resource version

## Cleanup and Testing

All factories support automatic cleanup:

```typescript
afterEach(async () => {
    // Clean up all created records
    await routineFactory.cleanupAll();
    await resourceFactory.cleanupAll();
    await resourceVersionRelationFactory.cleanupAll();
});
```

## Advanced Features

### Cascade Delete Support
Factories handle cascade deletes properly, removing related records in the correct order.

### Transaction Support
All operations are transaction-safe for data integrity.

### Bulk Operations
Support for creating multiple related objects efficiently.

### State Verification
Built-in methods to verify object state and relationships.

## Dependencies

These factories require:
- `@vrooli/shared` for types and utilities
- `@prisma/client` for database operations
- `@vrooli/shared/__test/fixtures/config` for routine configurations

## Testing

Run the comprehensive test suite:

```bash
cd packages/server
pnpm test CoreBusinessObjectsPart2.test.ts
```

The test file demonstrates all factory capabilities and provides examples for integration testing.