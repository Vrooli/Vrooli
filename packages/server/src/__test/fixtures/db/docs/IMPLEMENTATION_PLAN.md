# Database Fixtures Implementation Plan

## Overview
This document outlines the implementation plan for converting all database fixtures to the new factory pattern following the ideal architecture.

## Core Infrastructure (✅ Completed)
- `DatabaseFixtureFactory.ts` - Base factory class with all required methods
- `FixtureFactoryRegistry.ts` - Registry for managing all factories
- `FactoryMigrationHelper.ts` - Helper for converting existing fixtures
- Example implementations: `UserDbFactory.ts`, `TeamDbFactory.ts`

## Implementation Groups

### Group 1: Authentication & Session Objects (3 objects)
**Objects**: Auth, Email, Session
**Dependencies**: User
**Special Considerations**: 
- Auth needs password hashing utilities
- Email needs verification timestamps
- Session needs expiry handling

### Group 2: Core Business Objects - Part 1 (5 objects)
**Objects**: Project, ProjectVersion, Api, ApiVersion, DataStructure
**Dependencies**: User, Team
**Special Considerations**:
- Version handling for Project and Api
- Complex relationships between versions
- Need to create Project fixtures (currently missing)

### Group 3: Core Business Objects - Part 2 (5 objects)
**Objects**: Routine, RoutineVersion, Resource, ResourceVersion, ResourceVersionRelation
**Dependencies**: User, Team, Project
**Special Considerations**:
- Complex version relationships
- ResourceVersionRelation is a junction table
- Need to create Routine fixtures (currently missing)

### Group 4: Chat System (4 objects)
**Objects**: Chat, ChatMessage, ChatParticipant, ChatInvite
**Dependencies**: User, Team
**Special Considerations**:
- Real-time event considerations
- Message threading
- Participant roles

### Group 5: Content & Interaction (5 objects)
**Objects**: Comment, Tag, Bookmark, BookmarkList, View
**Dependencies**: User, various parent objects
**Special Considerations**:
- Polymorphic relationships (Comment, Bookmark can attach to many types)
- Tag reuse across objects
- View tracking

### Group 6: Notifications & Communications (4 objects)
**Objects**: Notification, NotificationSubscription, PushDevice, Reaction
**Dependencies**: User
**Special Considerations**:
- Push notification tokens
- Subscription preferences (use config fixtures)
- Reaction aggregation

### Group 7: Scheduling System (3 objects)
**Objects**: Schedule, ScheduleException, ScheduleRecurrence
**Dependencies**: User
**Special Considerations**:
- Complex recurrence rules
- Exception handling
- Timezone considerations

### Group 8: Execution System (6 objects)
**Objects**: Run, RunIO, RunStep, RunProject, RunRoutine, Report
**Dependencies**: User, Routine, Project
**Special Considerations**:
- Complex state machine
- IO mapping
- Execution metrics
- Report generation

### Group 9: Collaboration Objects (6 objects)
**Objects**: Meeting, MeetingInvite, Member, MemberInvite, Issue, PullRequest
**Dependencies**: User, Team
**Special Considerations**:
- Invite workflows
- Permission levels
- Issue/PR relationships

### Group 10: Financial Objects (5 objects)
**Objects**: Payment, Plan, Premium, CreditAccount, CreditLedgerEntry
**Dependencies**: User, Team
**Special Considerations**:
- Transaction safety
- Balance calculations
- Subscription management

### Group 11: Statistics & Reputation (6 objects)
**Objects**: StatsResource, StatsSite, StatsTeam, StatsUser, Award, ReputationHistory
**Dependencies**: User, Team, Resource
**Special Considerations**:
- Aggregated data
- Time-series data
- Award criteria

### Group 12: Remaining Objects (5 objects)
**Objects**: Transfer, Wallet, Reminder, ReminderItem, ReminderList
**Dependencies**: User
**Special Considerations**:
- Wallet security
- Transfer validation
- Reminder scheduling

## Implementation Requirements for Each Object

### Required Methods
1. `getMinimalData()` - Minimal valid data
2. `getCompleteData()` - All fields populated
3. `getDefaultInclude()` - Default query includes
4. `applyRelationships()` - Handle relationship setup
5. `checkModelConstraints()` - Validate business rules
6. `getInvalidScenarios()` - Invalid test data
7. `getEdgeCaseScenarios()` - Edge case test data

### Required Features
1. **Type Safety**: ZERO `any` types except where absolutely necessary
2. **ID Generation**: Use `generatePK()` and `generatePublicId()`
3. **Unique Fields**: Use `nanoid()` for handles, emails, etc.
4. **Config Usage**: Use config fixtures for all JSON fields
5. **Relationship Support**: Proper handling of all relationships
6. **Cleanup**: Implement cascade delete logic
7. **Verification**: State and constraint verification

### Factory Structure Template
```typescript
import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";
// Import config fixtures if needed
import { [objectName]ConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

interface [ObjectName]RelationConfig extends RelationConfig {
    // Add specific relationship configs
}

export class [ObjectName]DbFactory extends DatabaseFixtureFactory<
    Prisma.[ObjectName],
    Prisma.[ObjectName]CreateInput,
    Prisma.[ObjectName]Include,
    Prisma.[ObjectName]UpdateInput
> {
    // Implementation following UserDbFactory and TeamDbFactory examples
}

export const create[ObjectName]DbFactory = (prisma: PrismaClient) => new [ObjectName]DbFactory(prisma);
```

## Testing Requirements
Each factory must support:
1. Creation of minimal and complete fixtures
2. Relationship setup and verification
3. Bulk seeding operations
4. Invalid data scenarios
5. Edge case handling
6. Proper cleanup

## Migration Process
1. Check if fixture file already exists
2. If exists, extract fixture data and convert using `FactoryMigrationHelper`
3. If not, create new factory from scratch
4. Implement all required methods
5. Add specific scenario methods
6. Test with `tsc --noEmit`
7. Update exports in index.ts

## Success Criteria
- ✅ All 50+ objects have factory implementations
- ✅ Zero TypeScript errors
- ✅ All factories follow the same pattern
- ✅ Comprehensive relationship support
- ✅ Integration with config fixtures
- ✅ Proper cleanup utilities
- ✅ Scenario-based test data