# Type Safety Migration Guide

## Overview
This guide helps developers address type safety issues revealed by the recent improvements to core types in the server package.

## Common Patterns and Solutions

### 1. ModelMap Access

**Problem:**
```typescript
const validator = ModelMap.get(authData.__typename).validate();
```

**Solution:**
```typescript
import { hasTypename } from "../utils/typeGuards.js";

if (!hasTypename(authData)) {
    throw new CustomError("XXXX", "InternalError", { authData });
}
const modelLogic = ModelMap.get(authData.__typename);
if (!modelLogic || !modelLogic.validate) {
    throw new CustomError("XXXX", "InternalError", { type: authData.__typename });
}
const validator = modelLogic.validate();
```

### 2. Task Data Access

**Problem:**
```typescript
const ownerId = data.userId ?? data.startedById ?? data.userData?.id;
```

**Solution:**
```typescript
import { extractOwnerId } from "../utils/typeGuards.js";

const ownerId = extractOwnerId(data);
```

### 3. Type Assertions

**Problem:**
```typescript
const formatted = someFunction() as any;
const result = processData(formatted) as SpecificType;
```

**Solution:**
```typescript
const formatted = someFunction();
if (!isExpectedType(formatted)) {
    throw new CustomError("XXXX", "InternalError", { formatted });
}
const result = processData(formatted);
```

### 4. Optional Property Access

**Problem:**
```typescript
if (data.status) {
    // use data.status
}
```

**Solution:**
```typescript
import { hasStatus } from "../utils/typeGuards.js";

if (hasStatus(data)) {
    // TypeScript now knows data.status exists and is a string
}
```

## Type Guard Utilities

The following type guards are available in `src/utils/typeGuards.ts`:

- `hasProperty(obj, key)` - Check if object has a property
- `hasStringProperty(obj, key)` - Check if object has a string property
- `hasId(obj)` - Check if object has an id
- `hasTypename(obj)` - Check if object has a __typename
- `hasUserId(obj)` - Check if object has a userId
- `hasStatus(obj)` - Check if object has a status
- `isAuthData(value)` - Check if value is valid auth data
- `isObject(value)` - Check if value is a non-null object
- `extractOwnerId(data)` - Safely extract owner ID from various formats

## Migration Steps

### Phase 1: Add Type Guards (Current)
1. Import type guards where needed
2. Replace unsafe property access with type guards
3. Add null checks before type assertions

### Phase 2: Remove Type Assertions
1. Replace `as any` with proper types
2. Replace `as unknown as X` with type guards
3. Add explicit return types to functions

### Phase 3: Enable Stricter Rules
1. Enable `noImplicitAny` in tsconfig
2. Enable `strictNullChecks`
3. Enable `noUncheckedIndexedAccess`

## Testing Type Safety

Run type checking on specific files:
```bash
cd packages/server && tsc --noEmit src/path/to/file.ts
```

Check for remaining `any` usage:
```bash
grep -r ":\s*any\|as\s*any" src/ --include="*.ts" | grep -v test
```

## Gradual Migration

For files that can't be immediately fixed:

1. Use the compatibility layer:
```typescript
import { asCompatModelLogic } from "../models/compatibility.js";
const logic = asCompatModelLogic(ModelMap.get(type));
```

2. Add AI_CHECK marker:
```typescript
// AI_CHECK: TYPE_SAFETY=needs-migration | LAST: 2025-07-04 - Partial migration, needs completion
```

3. Document specific issues in comments for future resolution

## Error Codes

New error codes for type safety issues:
- 0033: Missing __typename on auth data
- 0034: Model logic missing required method
- 0035: Invalid auth data structure
- 0454: Validator missing permissions select
- 0531: Create node missing __typename
- 0532: Update node missing __typename

## Best Practices

1. **Never use `any`** - Use `unknown` and type guards instead
2. **Check before access** - Use type guards before accessing optional properties
3. **Explicit is better** - Add explicit return types to all functions
4. **Document assumptions** - If you must use a type assertion, document why
5. **Test edge cases** - Ensure your code handles null/undefined gracefully

## Resources

- TypeScript Handbook: https://www.typescriptlang.org/docs/handbook/
- Type Guards: https://www.typescriptlang.org/docs/handbook/2/narrowing.html
- Strict Mode: https://www.typescriptlang.org/tsconfig#strict