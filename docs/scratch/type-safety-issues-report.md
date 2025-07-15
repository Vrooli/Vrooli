# Type Safety Issues Report - Server Package

This report summarizes type safety issues found in the server package (excluding services/ folder), organized by type of issue and difficulty level.

## Summary Statistics

### By Issue Type:
1. **Usage of 'any' type**: ~580+ occurrences
2. **Unsafe type assertions (as any)**: ~180+ occurrences  
3. **@ts-ignore/@ts-expect-error**: 5 occurrences
4. **Missing return type annotations**: ~50+ functions
5. **Potential null/undefined issues**: Numerous instances

### Most Problematic Files:
1. `src/builders/types.ts` - 52 'any' occurrences
2. `src/builders/infoConverter.ts` - 44 'any' occurrences
3. `src/models/types.ts` - 19 'any' occurrences
4. `src/builders/combineQueries.ts` - 18 'any' occurrences
5. `src/builders/shapeHelper.ts` - 17 'any' occurrences

## Issues by Difficulty Level

### Easy Fixes (Quick Wins)

#### 1. @ts-ignore/@ts-expect-error Comments (5 total)
These are the easiest to fix as they're limited in number and clearly marked:

- `src/__tools/api/partial/resourceVersion.ts` - 1 instance (JSONB field)
- `src/__tools/api/partial/chatMessage.ts` - 2 instances (JSONB fields)
- `src/__tools/api/partial/team.ts` - 1 instance (JSONB field)
- `src/__tools/api/partial/user.ts` - 1 instance (JSONB field)

**Fix**: Replace with proper Prisma JSONB type definitions

#### 2. Simple Function Return Types
Functions with missing return types that have obvious returns:

```typescript
// src/auth/wallet.ts
export function serializedAddressToBech32(address: string) // Missing: : string
export function verifySignedMessage(address: string, payload: string, coseSign1Hex: string) // Missing: : boolean
function verifyPayload(payload: string, payloadCose: Uint8Array) // Missing: : boolean
function verifyAddress(address: string, addressCose: Serialization.Address, publicKeyCose: Serialization.PublicKey) // Missing: : boolean

// src/utils/singletons.ts
export async function initSingletons() // Missing: : Promise<void>

// src/utils/fileStorage.ts
async function getHeicConvert() // Missing return type
async function getFileType(buffer: Uint8Array | ArrayBuffer) // Missing return type
```

#### 3. Recently Fixed Files (AI_CHECK comments)
These files have already been partially fixed and may have remaining issues:

- `src/utils/calendar.ts` - Fixed RRule options type
- `src/utils/getAuthenticatedData.ts` - Fixed any types to Record<string, unknown>
- `src/utils/toYaml.ts` - Replaced any with unknown
- `src/utils/searchMap.ts` - Fixed searchInput type
- `src/endpoints/logic/auth.ts` - Fixed OAuth config casting

### Medium Difficulty Fixes

#### 1. PrismaDelegate and Related Types
`src/builders/types.ts` has extensive use of 'any' in Prisma-related types:

```typescript
export type PrismaSearch = {
    where: any;      // Should be specific where clause type
    select?: any;    // Should be specific select type
    orderBy?: any;   // Should be specific orderBy type
    cursor?: any;    // Should be specific cursor type
    take?: any;      // Should be number
    skip?: any;      // Should be number
    distinct?: any;  // Should be specific field union
}
```

#### 2. Generic Model Logic Types
`src/models/base/index.ts` uses generic any types:

```typescript
export type GenericModelLogic = ModelLogic<any, any, any>;
display: "display" extends Logic ? Displayer<any> : never;
duplicate: "duplicate" extends Logic ? Duplicator<any, any> : never;
// ... more similar lines
```

#### 3. Unsafe Type Assertions in Actions
`src/endpoints/logic/actions.ts`:

```typescript
const passwordHash = PasswordAuthService.getAuthPassword(user as any);
await PasswordAuthService.setupPasswordReset(user as any);
const session = await PasswordAuthService.logIn(input?.password as string, user as any, req);
```

### Hard Fixes (Require Significant Refactoring)

#### 1. InfoConverter Complex Types
`src/builders/infoConverter.ts` has deeply nested any types in union handling:

```typescript
data: { [x: string]: any },
partialInfo: { [x: string]: any },
// Complex union field handling with any[]
```

#### 2. Dynamic Query Building
Files like `src/builders/combineQueries.ts` and `src/builders/shapeHelper.ts` use 'any' for dynamic query construction, which would require significant type system improvements.

#### 3. Test Mock Types
Many test files use 'as any' for mocking, which while not production code, still affects type safety during testing.

## Recommendations

### Phase 1: Quick Wins (1-2 days)
1. Fix all @ts-ignore/@ts-expect-error comments
2. Add missing return types to simple functions
3. Replace obvious 'any' with 'unknown' where appropriate

### Phase 2: Prisma Types (3-5 days)
1. Create proper typed interfaces for PrismaSearch, PrismaSelect, etc.
2. Replace generic 'any' in database query types
3. Fix unsafe type assertions in authentication logic

### Phase 3: Model System (1-2 weeks)
1. Refactor ModelLogic to use proper generics instead of 'any'
2. Type the union handling in infoConverter
3. Improve type safety in dynamic query builders

### Phase 4: Test Infrastructure (Ongoing)
1. Replace test mocks using 'as any' with proper type-safe mocks
2. Use type-safe test utilities instead of type assertions

## Priority Areas

Based on impact and ease of fixing:

1. **auth/** directory - Security-critical code with some easy fixes
2. **utils/** directory - Many utility functions missing return types
3. **endpoints/logic/** - API endpoints with unsafe assertions
4. **builders/types.ts** - Core types file with extensive 'any' usage
5. **models/base/** - Model system foundation needs type improvements

## Next Steps

1. Start with the @ts-expect-error comments in __tools/api/partial/
2. Add return types to functions in auth/ and utils/
3. Create typed interfaces for Prisma operations
4. Gradually replace 'any' with proper types or 'unknown'