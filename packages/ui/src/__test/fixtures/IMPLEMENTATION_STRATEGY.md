# UI Fixtures Implementation Strategy

## Overview
This document outlines the strategy for implementing type-safe fixtures for all 41+ Vrooli object types in the UI package.

## Object Categorization by Complexity

### Tier 1: Simple Objects (No relationships)
These objects have minimal dependencies and can be implemented first:
1. **Tag** - Simple label object
2. **ScheduleException** - Date/time data only
3. **ScheduleRecurrence** - Recurrence pattern data
4. **ResourceVersionRelation** - Simple relation data
5. **RunStep** - Execution step data
6. **RunIO** - Input/output data
7. **ReportResponse** - Response to a report
8. **ReminderItem** - Individual reminder task

### Tier 2: Objects with Single Relationships
These objects reference one other object type:
1. **Bookmark** (references any bookmarkable object + optional BookmarkList)
2. **Comment** (references commentable parent)
3. **Member** (references User + Team)
4. **ChatParticipant** (references User + Chat)
5. **ChatMessage** (references User + Chat)
6. **Report** (references reportable object)
7. **Transfer** (references from/to Team/User)
8. **Wallet** (references User or Team)

### Tier 3: Complex Objects with Multiple Relationships
These objects have multiple dependencies and complex data:
1. **User** (has bots, teams, resources, etc.)
2. **Team** (has members, resources, projects)
3. **Project** (has team, versions, resources)
4. **Routine** (has versions, inputs, outputs)
5. **Chat** (has participants, messages, invites)
6. **Resource** (has versions, tags)
7. **Meeting** (has team, invites, attendees)
8. **Schedule** (has exceptions, recurrences)

### Tier 4: Version-based Objects
These follow the versioning pattern:
1. **ResourceVersion**
2. **ProjectVersion**
3. **RoutineVersion**
4. **ApiVersion**
5. **DataStructureVersion**
6. **SmartContractVersion**

## Implementation Order

### Phase 1: Core Infrastructure (Completed ✓)
- [x] Base types and interfaces
- [x] BaseFormFixtureFactory
- [x] BaseRoundTripOrchestrator
- [x] BaseMSWHandlerFactory
- [x] Integration utilities
- [x] Factory registry system

### Phase 2: Foundation Objects (Priority: High)
Start with the most referenced objects:
1. **User** - Most fundamental object (example completed ✓)
2. **Team** - Second most referenced
3. **Tag** - Used everywhere for categorization
4. **Bot** - Special type of User

### Phase 3: Communication Objects (Priority: High)
Essential for collaboration features:
1. **Chat**
2. **ChatMessage**
3. **ChatParticipant**
4. **ChatInvite**
5. **Comment**

### Phase 4: Resource Management (Priority: Medium)
Core content objects:
1. **Resource**
2. **ResourceVersion**
3. **Bookmark**
4. **BookmarkList**
5. **Project**
6. **ProjectVersion**

### Phase 5: Workflow Objects (Priority: Medium)
For automation features:
1. **Routine**
2. **RoutineVersion**
3. **Run**
4. **RunStep**
5. **RunIO**

### Phase 6: Remaining Objects (Priority: Low)
Complete coverage:
- All remaining objects in alphabetical order

## Shared Patterns

### 1. Form Data Pattern
```typescript
interface ObjectFormData {
    // UI-specific fields (e.g., confirmPassword)
    // Form state fields (e.g., isDirty)
    // All API fields from shared types
}
```

### 2. Validation Pattern
- Always use shared validation as base
- Add UI-specific validation on top
- Use createValidationAdapter utility

### 3. Shape Integration
- Import shape functions from @vrooli/shared
- Never duplicate transformation logic
- Use shapeObject.create/update functions

### 4. Config Field Pattern
For JSON config fields, always use config fixtures:
```typescript
import { botConfigFixtures, chatConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

botSettings: botConfigFixtures.complete,
chatSettings: chatConfigFixtures.variants.privateTeamChat,
```

### 5. Relationship Pattern
```typescript
// For single relationships
team: { id: "team_123" } // Connect pattern

// For multiple relationships
members: [
    { id: "user_123" },
    { id: "user_456" }
]

// For create + connect
listCreate: {
    id: generatePK(),
    label: "New List"
}
```

## Subagent Task Template

When spawning subagents, use this template:

```
Task: Implement [ObjectName] fixtures for UI testing

Location: packages/ui/src/__test/fixtures/factories/[objectName]FixtureFactory.ts

Requirements:
1. Follow the UserFixtureFactory pattern exactly
2. Import types from: @vrooli/shared
3. Use shape[ObjectName] and [objectName]Validation from shared
4. Define form data interface with UI-specific fields
5. Implement all factory methods with ZERO any types
6. Use config fixtures for JSON fields
7. Test with: cd packages/ui && tsc --noEmit src/__test/fixtures/factories/[objectName]FixtureFactory.ts

Structure:
- [ObjectName]FormData interface
- [ObjectName]UIState interface  
- [ObjectName]FormFixtureFactory extends BaseFormFixtureFactory
- [ObjectName]MSWHandlerFactory extends BaseMSWHandlerFactory
- [ObjectName]UIStateFixtureFactory implements UIStateFixtureFactory
- [ObjectName]FixtureFactory implements UIFixtureFactory

Scenarios to include:
- minimal: Required fields only
- complete: All fields populated
- invalid: Various validation failures
- [custom]: Object-specific scenarios
```

## Quality Checklist

For each implementation:
- [ ] Zero `any` types
- [ ] All imports use .js extension
- [ ] Uses real shape functions
- [ ] Uses real validation schemas
- [ ] Uses config fixtures for JSON
- [ ] Type checks successfully
- [ ] Follows exact factory pattern
- [ ] Includes all standard scenarios
- [ ] Has proper JSDoc comments
- [ ] Handles relationships correctly

## Success Metrics

1. **Type Safety**: 100% - No `any` types anywhere
2. **Coverage**: All 41+ objects have fixtures
3. **Integration**: All fixtures use shared package functions
4. **Consistency**: Same patterns across all fixtures
5. **Usability**: Easy to use in tests with IntelliSense

## Next Steps

1. Implement 2-3 more examples (Team, Tag, Chat)
2. Validate the pattern works for different object types
3. Create subagent prompts for parallel implementation
4. Monitor progress and adjust strategy as needed