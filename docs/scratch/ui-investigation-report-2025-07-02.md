# UI Package Investigation Report - Critical Issues & Resolution Plan

## Executive Summary
The UI package has significant TypeScript compilation errors (218 errors) but builds successfully. The most critical issues are TypeScript type mismatches and MSW handler incompatibilities that could prevent proper form handling and API communication.

## Investigation Results

### 1. Build & Compilation Status ✓
- **Build Process**: Completes successfully (vite build)
- **TypeScript Compilation**: 218 errors found
- **Critical Finding**: Build succeeds despite TS errors (likely due to swc transpilation)

### 2. Critical Issues Found

#### Issue #1: TypeScript Type Errors (218 total)
**Severity: HIGH**
**Confidence: 95%**
**Time to Fix: 4-6 hours**

Most common patterns:
- Yup validation schema errors (`.validate` property missing)
- MSW handler type mismatches (new v2 API incompatibility)
- Translation key type errors
- Component prop type mismatches

**Example Errors:**
```typescript
// Yup validation
error TS2339: Property 'validate' does not exist on type '(params: YupMutateParams) => AnyObjectSchema'

// MSW handlers
error TS2345: Argument of type '(req: any, res: any, ctx: any) => Promise<any>' is not assignable to parameter of type 'HttpResponseResolver<PathParams, DefaultBodyType, undefined>'

// Translation keys
error TS2322: Type '"DonationPercentageMustBeBetween0And100"' is not assignable to parameter
```

#### Issue #2: MSW (Mock Service Worker) Version Incompatibility
**Severity: HIGH**
**Confidence: 90%**
**Time to Fix: 2-3 hours**

- Test fixtures using MSW v1 syntax with v2 library
- All API response handlers need updating
- Affects all integration tests

#### Issue #3: Form Component Prop Mismatches
**Severity: MEDIUM**
**Confidence: 85%**
**Time to Fix: 3-4 hours**

- Formik integration issues in multiple components
- Missing required props in form components
- Type incompatibilities between expected and actual props

#### Issue #4: Translation System Type Safety
**Severity: MEDIUM**
**Confidence: 80%**
**Time to Fix: 2-3 hours**

- Missing translation keys causing type errors
- Incorrect type definitions for i18n integration
- Affects all components using translations

#### Issue #5: React Hook Violations
**Severity: MEDIUM**
**Confidence: 75%**
**Time to Fix: 4-5 hours**

- SSR unsafe window access in hooks
- Potential conditional hook calls
- Race conditions in store initialization

### 3. Dependencies Status ✓
- All dependencies properly installed
- No version conflicts detected
- Using latest TypeScript (5.8.3)

### 4. Routing & Navigation ✓
- Custom router implementation working
- Lazy loading properly configured
- No critical routing issues found

### 5. State Management ✓
- Zustand stores properly structured
- Some race condition risks in async initialization
- Overall functional but needs optimization

## Resolution Plan (Ranked by Confidence)

### Phase 1: High Confidence Fixes (1-2 days)

#### 1. Fix MSW Handler Syntax (90% confidence, 2-3 hours)
```typescript
// OLD (v1)
rest.post('/api/auth', (req, res, ctx) => {
  return res(ctx.json({ data: {} }))
})

// NEW (v2)
http.post('/api/auth', () => {
  return HttpResponse.json({ data: {} })
})
```

#### 2. Fix Yup Validation Imports (95% confidence, 1-2 hours)
```typescript
// Change from function imports to schema imports
import { apiKeyValidation } from "@vrooli/shared";
// TO
import { apiKeySchema } from "@vrooli/shared";
// Use schema.validateSync() instead of validate()
```

#### 3. Add Missing Translation Keys (80% confidence, 2-3 hours)
- Audit all translation usage
- Add missing keys to en/common.json
- Update type definitions

### Phase 2: Medium Confidence Fixes (2-3 days)

#### 4. Fix Form Component Props (85% confidence, 3-4 hours)
- Update ProfileUpdateInput interface
- Add missing required props
- Remove type assertions

#### 5. Fix SSR Window Access (75% confidence, 4-5 hours)
```typescript
// Add guards
const [width, setWidth] = useState(() => 
  typeof window !== 'undefined' ? window.innerWidth : 1024
);
```

### Phase 3: Lower Confidence Fixes (3-5 days)

#### 6. Remove Type Assertions (70% confidence, 6-8 hours)
- Replace all `as any` with proper types
- Update type definitions
- May require shared package updates

#### 7. Optimize State Management (65% confidence, 8-10 hours)
- Fix async initialization patterns
- Add proper loading states
- Implement error boundaries

## Implementation Strategy

### Immediate Actions (Day 1)
1. Fix all MSW handlers in test files
2. Update Yup validation usage
3. Create missing translation keys

### Short Term (Days 2-3)
4. Fix form component prop types
5. Add SSR guards to all hooks
6. Run full test suite after each fix

### Medium Term (Days 4-7)
7. Remove type assertions systematically
8. Optimize state management
9. Add comprehensive error boundaries

## Success Metrics
- [ ] TypeScript compilation passes with 0 errors
- [ ] All tests pass (currently some failing)
- [ ] No console errors in development
- [ ] Forms validate and submit correctly
- [ ] API calls work with proper error handling

## Risk Assessment
- **Low Risk**: MSW updates, translation fixes
- **Medium Risk**: Form prop updates, SSR fixes
- **High Risk**: Type system overhaul, state management changes

## Recommended Approach
1. Start with MSW handler fixes (blocks all tests)
2. Fix validation and translation issues
3. Address form component props
4. Tackle deeper architectural issues last

## Notes
- Build succeeds despite TS errors due to SWC transpilation
- Most issues are type-related, not runtime breaking
- Prioritize fixes that unblock testing first