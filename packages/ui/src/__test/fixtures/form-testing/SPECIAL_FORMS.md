# Special Form Handling Documentation

This document outlines forms that require special handling beyond the standard `useStandardUpsertForm` hook pattern.

## Forms Requiring Extended Hook Support

### 1. Forms with `isMutate` Pattern

These forms support both saving to database and returning data without persisting:

#### **Currently Unsupported Forms:**
- `ResourceUpsert` - Allows creating resources that may not be saved
- `ScheduleUpsert` - Can create schedules without immediate persistence  
- `MeetingUpsert` - Meeting scheduling without saving
- `RunUpsert` - Run configuration without immediate execution

#### **Required Hook Extension:**
```typescript
interface StandardUpsertFormConfig {
    // ... existing config
    isMutate?: boolean; // Add optional isMutate support
}
```

The hook would need to:
1. Accept `isMutate` as a prop
2. Conditionally call API or return values directly
3. Handle non-persisted form completion

### 2. Batch/Array Forms

These forms handle arrays of objects instead of single entities:

#### **Batch Forms:**
- `ChatInvitesUpsert` - Creates multiple chat invites
- `MemberInvitesUpsert` - Creates multiple member invites

#### **Characteristics:**
- Initial values are arrays: `MemberInviteShape[]`
- Validation runs on each array item
- Transform functions map over arrays
- API calls handle arrays

#### **Potential Solutions:**
1. Create `useStandardBatchUpsertForm` hook
2. Extend existing hook with array support
3. Keep as exceptions with manual implementation

### 3. Complex State Management Forms

Forms with significant UI-specific state that doesn't fit standard patterns:

#### **Complex Forms:**
- `BookmarkListUpsert` - List management with selection/bulk operations
- `BotUpsert` - AI model configuration with sliders and complex UI
- `PromptUpsert` - Schema editor with dynamic form generation
- `ScheduleUpsert` - Calendar integration with date/time pickers

#### **Recommendation:**
These forms may be better left with custom implementations due to their unique requirements.

## Migration Strategy

### Phase 1: Extend Hook for `isMutate` (Priority)
Many forms use this pattern and could benefit from standardization:

```typescript
// Extended hook usage
const { ... } = useStandardUpsertForm({
    // ... config
}, {
    // ... props
    isMutate: props.isMutate, // New prop
});
```

### Phase 2: Create Batch Form Hook
For array-based forms:

```typescript
const { ... } = useStandardBatchUpsertForm<MemberInviteShape[]>({
    // ... config
    validateBatch: validateMemberInviteValues,
    transformBatch: transformMemberInviteValues,
});
```

### Phase 3: Document Complex Forms
Create detailed documentation for forms that should remain custom due to complexity.

## Current Migration Status

### ✅ Successfully Migrated (5 forms)
1. `TeamUpsert` - Standard pattern with translations
2. `ReportUpsert` - Simple form with conditional validation
3. `ApiUpsert` - Complex with code editor (needs isMutate fix)
4. `CommentUpsert` - Simple with translations
5. `DataStructureUpsert` - Standard pattern

### 🔄 Pending `isMutate` Support (7 forms)
1. `ResourceUpsert` - Started migration, needs completion
2. `DataConverterUpsert` - Similar to ApiUpsert
3. `SmartContractUpsert` - Similar to ApiUpsert
4. `PromptUpsert` - Complex with schema editor
5. `BotUpsert` - Complex with AI configuration
6. `RoutineSingleStepUpsert` - Multi-type form
7. `BookmarkListUpsert` - List management

### 🔄 Pending Batch Support (2 forms)
1. `ChatInvitesUpsert` - Batch invites
2. `MemberInvitesUpsert` - Batch invites

### ❓ Complex/Incomplete (3 forms)
1. `ScheduleUpsert` - Calendar integration
2. `MeetingUpsert` - Mostly TODO placeholders
3. `RunUpsert` - Schedule integration

## Recommendations

1. **Immediate Action**: Extend `useStandardUpsertForm` to support `isMutate`
2. **Next Phase**: Create batch form hook or variant
3. **Long Term**: Evaluate complex forms individually for migration feasibility
4. **Documentation**: Maintain list of forms that intentionally don't use standard hooks

## Code Reduction Potential

If we successfully extend the hooks:
- **isMutate forms**: ~20-25% reduction (similar to current migrations)
- **Batch forms**: ~30-40% reduction (significant boilerplate in validation/transform)
- **Total potential**: 9 additional forms could be migrated
- **Estimated savings**: ~2,000-3,000 lines of code