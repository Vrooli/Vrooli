# API Fixtures Implementation Strategy

This document outlines the comprehensive strategy for implementing type-safe API fixtures for all 41+ object types in the Vrooli system.

## 🎯 Goals

1. **Zero `any` Types**: Achieve complete type safety across all fixtures
2. **Validation Integration**: Connect with real Yup validation schemas
3. **Shape Function Integration**: Use actual transformation functions
4. **Comprehensive Coverage**: All 41+ objects with minimal, complete, invalid, and edge cases
5. **Gradual Migration**: Replace legacy fixtures without breaking existing code

## 📊 Current State Analysis

### ✅ What's Working
- **39 fixture files exist** covering most object types
- **Type imports available** from `api/types.ts`
- **Some fixtures are well-typed** (e.g., `userFixtures`)
- **Config fixtures exist** for JSON field population
- **Validation schemas available** for most objects
- **Shape functions available** in `shape/models/models.ts`

### ❌ Critical Issues
- **Extensive `any` usage** in most fixtures
- **Missing type parameters** (e.g., `teamFixtures: ModelTestFixtures` instead of `ModelTestFixtures<TeamCreateInput, TeamUpdateInput>`)
- **No validation integration** in most fixtures
- **No shape function integration** anywhere
- **Inconsistent patterns** across different fixture files

## 🏗️ Architecture Built

### Core Infrastructure ✅ COMPLETED
- **`types.ts`**: Core factory interfaces with full type safety
- **`BaseAPIFixtureFactory.ts`**: Base implementation with validation/shape integration
- **`integrationUtils.ts`**: Utilities for connecting validation and shape functions
- **Updated `index.ts`**: Organized exports with backward compatibility

### Integration Points ✅ AVAILABLE
- **Validation Schemas**: Located in `src/validation/models/`
- **Shape Functions**: Located in `src/shape/models/models.ts`
- **Config Fixtures**: Located in `src/__test/fixtures/config/`
- **Type Definitions**: Located in `src/api/types.ts`

## 🎯 Implementation Strategy

### Phase 1: Foundation Objects (Week 1)
**Priority: HIGH** - Start with most commonly used objects

**Objects to implement:**
1. **User** - Already well-typed, needs validation integration
2. **Team** - Needs type parameters and validation
3. **Project** - Core object, high usage
4. **Routine** - Core AI system object
5. **Run** - Execution tracking

**Pattern:**
```typescript
// 1. Fix type parameters
export const teamFixtures: ModelTestFixtures<TeamCreateInput, TeamUpdateInput> = {

// 2. Add validation integration  
const teamIntegration = new FullIntegration(
    { create: teamValidation.create, update: teamValidation.update },
    shapeTeam
);

// 3. Create factory
export const teamAPIFixtures = new BaseAPIFixtureFactory(config, customizers);
```

### Phase 2: Content Objects (Week 2)
**Priority: HIGH** - Content creation and management

**Objects to implement:**
1. **Comment** - User-generated content
2. **Resource** - File management
3. **Note** - Documentation
4. **Issue** - Problem tracking
5. **PullRequest** - Collaboration

### Phase 3: Communication Objects (Week 3)
**Priority: MEDIUM** - Real-time and messaging

**Objects to implement:**
1. **Chat** - Real-time messaging
2. **ChatMessage** - Individual messages
3. **Meeting** - Scheduled interactions
4. **Notification** - System alerts

### Phase 4: Organization Objects (Week 4)
**Priority: MEDIUM** - Structure and relationships

**Objects to implement:**
1. **Bookmark** - User organization
2. **Tag** - Categorization
3. **Member** - Team relationships
4. **Schedule** - Time management

### Phase 5: System Objects (Week 5)
**Priority: LOW** - Infrastructure and admin

**Objects to implement:**
1. **ApiKey** - Authentication
2. **Transfer** - Data movement
3. **Report** - Moderation
4. **Wallet** - Payments

## 🤖 Subagent Distribution Strategy

### Parallel Implementation Approach

Each subagent will be given:

1. **Specific object(s)** to implement (1-3 related objects)
2. **Factory interface** to follow
3. **Integration utilities** for validation/shape
4. **Example implementation** (bookmark factory)
5. **Type checking requirements**

### Subagent Task Template

```markdown
## Task: Implement {ObjectName} API Fixtures

### Objectives
- Create type-safe fixture factory for {ObjectName}
- Integrate with validation schema: `{objectName}Validation`
- Integrate with shape function: `shape{ObjectName}`
- Include comprehensive test scenarios

### Requirements
1. **Zero `any` types** - Full TypeScript safety
2. **Use existing types** from `@vrooli/shared/api/types`
3. **Integration**: Connect validation and shape functions
4. **Scenarios**: minimal, complete, invalid (3+ cases), edge cases (3+ cases)
5. **Factory methods**: createFactory, updateFactory, findFactory
6. **Config usage**: Use config fixtures for JSON fields

### Implementation Steps
1. **Import types**: Import {ObjectName}CreateInput, {ObjectName}UpdateInput, {ObjectName}
2. **Create fixture data**: Define all scenarios with proper types
3. **Add validation**: Import and integrate {objectName}Validation
4. **Add shape integration**: Import and integrate shape{ObjectName}
5. **Create factory**: Extend BaseAPIFixtureFactory
6. **Type check**: Run `tsc --noEmit` on your file
7. **Export**: Add to main index.ts

### Dependencies
- `@vrooli/shared/api/types`
- `@vrooli/shared/validation/models/{objectName}`
- `@vrooli/shared/shape/models/models` (shape{ObjectName})
- `@vrooli/shared/__test/fixtures/config` (for JSON fields)

### Example Pattern
See: `/src/__test/fixtures/api/factories/bookmarkAPIFixtures.ts`
```

## 📋 Object Inventory

### ✅ Existing Fixtures (39 objects)
- apiKeyExternal, apiKey, bookmark, bookmarkList, bot, chat, chatInvite
- chatMessage, chatParticipant, comment, email, issue, meeting, meetingInvite
- member, memberInvite, notificationSubscription, phone, pullRequest
- pushDevice, reminder, reminderItem, reminderList, report, reportResponse
- resource, resourceVersion, resourceVersionRelation, run, runIO, runStep
- scheduleException, schedule, scheduleRecurrence, tag, team, transfer
- user, wallet

### ❓ Potentially Missing Objects
- Project (major object, likely needs creation)
- Routine variations (single-step, multi-step)
- AI-related objects (agents, swarms)
- Analytics/stats objects
- Notification types
- Permission/role objects

## 🔄 Migration Strategy

### Backward Compatibility
1. **Keep existing exports** - No breaking changes
2. **Add new factories** alongside legacy fixtures
3. **Gradual replacement** - Teams can migrate when ready
4. **Namespace organization** - `apiFixtures.{objectName}Fixtures`

### Legacy → New Mapping
```typescript
// OLD (still works)
import { userFixtures } from "@vrooli/shared/__test/fixtures/api";
const user = userFixtures.minimal.create;

// NEW (recommended)
import { userAPIFixtures } from "@vrooli/shared/__test/fixtures/api";
const user = userAPIFixtures.createFactory({ name: "Test User" });
const validation = await userAPIFixtures.validateCreate(user);
```

## 🧪 Testing Strategy

### Factory Validation
```typescript
describe("User API Fixtures", () => {
    it("should create valid minimal user", async () => {
        const user = userAPIFixtures.minimal.create;
        const result = await userAPIFixtures.validateCreate(user);
        expect(result.isValid).toBe(true);
    });
    
    it("should integrate with shape functions", () => {
        const shapeData: UserShape = { /* ... */ };
        const apiInput = userAPIFixtures.transformToAPI(shapeData);
        expect(apiInput).toBeDefined();
    });
});
```

### Round-Trip Testing
```typescript
it("should handle complete user lifecycle", async () => {
    const formData = { userName: "test", email: "test@example.com" };
    const apiInput = userAPIFixtures.transformToAPI(formData);
    const validation = await userAPIFixtures.validateCreate(apiInput);
    // ... continue with full round-trip
});
```

## 📈 Success Metrics

### Type Safety
- [ ] 0 `any` types in all fixture files
- [ ] 100% TypeScript compilation success
- [ ] All factories properly typed

### Integration
- [ ] All validation schemas connected
- [ ] All shape functions connected
- [ ] Config fixtures used for JSON fields

### Coverage
- [ ] All 41+ objects have fixtures
- [ ] All objects have minimal/complete/invalid/edge cases
- [ ] All factories export createFactory, validateCreate, etc.

### Performance
- [ ] Type checking passes in <30 seconds
- [ ] No circular dependencies
- [ ] Memory usage reasonable

## 🚀 Getting Started

### For Core Team
1. **Review this strategy** - Ensure alignment with project goals
2. **Approve subagent tasks** - Ready to distribute work
3. **Set up coordination** - How to track progress across subagents

### For Subagents
1. **Read documentation** - Understand fixture architecture
2. **Review example** - Study bookmark factory implementation  
3. **Pick object(s)** - Choose from priority list
4. **Follow template** - Use provided task structure
5. **Type check early** - Validate as you build

## 🔮 Future Enhancements

### Round-Trip Testing
- Full cycle testing: Form → API → DB → Response → UI
- Error scenario testing at each layer
- Performance benchmarking

### Advanced Features
- Relationship builders for complex object graphs
- Permission scenario testing
- Real-time event simulation integration
- Automated fixture generation from schemas

### Documentation
- Auto-generated fixture reference
- Usage examples for each object
- Migration guides for teams
- Performance optimization tips

---

**Next Action**: Begin Phase 1 implementation with subagent distribution for User, Team, Project, Routine, and Run objects.