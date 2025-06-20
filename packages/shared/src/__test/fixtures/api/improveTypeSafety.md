# Type Safety Improvement Guide for API Fixtures

## Pattern for Improving Type Safety

### 1. Import Types Properly
```typescript
import type { ModelCreateInput, ModelUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { modelValidation } from "../../../validation/models/model.js";
```

### 2. Add Type Parameters to ModelTestFixtures
```typescript
// Change from:
export const modelFixtures: ModelTestFixtures = {

// To:
export const modelFixtures: ModelTestFixtures<ModelCreateInput, ModelUpdateInput> = {
```

### 3. Handle Invalid Types with @ts-expect-error
```typescript
// For invalid types in test scenarios:
invalidTypes: {
    create: {
        // @ts-expect-error Testing invalid type - field should be string
        field: 123,
        // @ts-expect-error Testing invalid type - isActive should be boolean
        isActive: "yes",
    } as unknown as ModelCreateInput,
}
```

### 4. Type Missing Required Fields
```typescript
missingRequired: {
    create: {
        // Missing required fields: id, name, etc.
        someOptionalField: "value",
    } as ModelCreateInput,
}
```

### 5. Type the Customizers
```typescript
// Change from:
const customizers = {
    create: (base: any): any => ({

// To:
const customizers = {
    create: (base: ModelCreateInput): ModelCreateInput => ({
        ...base,
        id: base.id || testValues.snowflakeId(),
        // other defaults
    }),
    update: (base: ModelUpdateInput): ModelUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};
```

### 6. Export Typed Factories
```typescript
export const modelTestDataFactory = new TypedTestDataFactory(modelFixtures, modelValidation, customizers);
export const typedModelFixtures = createTypedFixtures(modelFixtures, modelValidation);
```

## Files to Update (Priority Order)

### High Priority (Most Used)
1. ✅ userFixtures.ts - DONE
2. ✅ teamFixtures.ts - DONE
3. botFixtures.ts
4. chatFixtures.ts
5. runFixtures.ts

### Medium Priority
6. bookmarkFixtures.ts
7. commentFixtures.ts
8. resourceFixtures.ts
9. meetingFixtures.ts
10. scheduleFixtures.ts

### Lower Priority (Rest of the 39 files)
- apiKeyFixtures.ts
- apiKeyExternalFixtures.ts
- bookmarkListFixtures.ts
- chatInviteFixtures.ts
- chatMessageFixtures.ts
- chatParticipantFixtures.ts
- emailFixtures.ts
- issueFixtures.ts
- meetingInviteFixtures.ts
- memberFixtures.ts
- memberInviteFixtures.ts
- notificationSubscriptionFixtures.ts
- phoneFixtures.ts
- pullRequestFixtures.ts
- pushDeviceFixtures.ts
- reminderFixtures.ts
- reminderItemFixtures.ts
- reminderListFixtures.ts
- reportFixtures.ts
- reportResponseFixtures.ts
- resourceVersionFixtures.ts
- resourceVersionRelationFixtures.ts
- runIOFixtures.ts
- runStepFixtures.ts
- scheduleExceptionFixtures.ts
- scheduleRecurrenceFixtures.ts
- tagFixtures.ts
- transferFixtures.ts
- walletFixtures.ts

## Common Patterns to Fix

1. **Generic ModelTestFixtures without type params**
   - Add `<CreateInput, UpdateInput>` type parameters

2. **`any` type in customizers**
   - Replace with proper input types

3. **Missing type assertions for invalid scenarios**
   - Add `as ModelCreateInput` or `as unknown as ModelCreateInput`

4. **@ts-expect-error for intentionally wrong types**
   - Add before each invalid type assignment with clear comment

5. **Database fixtures using `prisma: any`**
   - These are in `/packages/server` - different pattern, not covered here

## Testing Type Safety

After updating each file:
```bash
cd /root/Vrooli/packages/shared
tsc --noEmit src/__test/fixtures/api/[filename].ts
```

The only errors should be from external dependencies, not from the fixture code itself.