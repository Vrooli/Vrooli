# Detailed Missing Properties Report - UI Package

This report provides a comprehensive list of all missing properties identified during TypeScript compilation, organized by file and issue type.

## Summary
- **Total Files with Missing Properties**: 30+
- **Total Missing Property Issues**: ~50
- **Most Common Missing Properties**: `name`, `type`, `id`, `dispatch`, `value`

## Files and Missing Properties

### 1. Test Fixtures - API Responses

#### `/src/__test/fixtures/api-responses/ChatInviteResponses.ts`
- **Line 331**: Missing property `reactionSummary` in type `UserYou`
- **Line 443**: Missing property `name` in type `Team`

#### `/src/__test/fixtures/api-responses/ChatMessageResponses.ts`
- **Line 168**: Missing property `hasPreviousPage` in type `PageInfo`
- **Line 354**: Missing property `isBlocked` in type `UserYou`
- **Line 394**: Missing property `canRead` in type `ChatYou`
- **Line 440**: Missing property `translations` in type `ChatMessage`

#### `/src/__test/fixtures/api-responses/ChatParticipantResponses.ts`
- **Line 291**: Missing properties in type `User`: `bookmarkedBy`, `bookmarks`, `isBotDepictingPerson`, `isPrivateBookmarks`, and 12 more
- **Line 338**: Missing properties in type `Chat`: Multiple properties missing when converting types

#### `/src/__test/fixtures/api-responses/PremiumResponses.ts`
- **Line 367**: Property `__typename` does not exist in type `ExtendedPremium`
- **Line 430**: Property `id` does not exist in type `Partial<ExtendedPremium>`
- **Line 589**: Property `id` does not exist in type `Partial<ExtendedPremium>`
- **Line 786**: Property `expiresAt` does not exist in type `Partial<ExtendedPremium>`

### 2. Form Testing Files

#### `/src/__test/fixtures/form-testing/CommentFormTest.ts`
- **Line 21**: Missing property `fixtures` in form test config

#### `/src/__test/fixtures/form-testing/ReportFormTest.test.ts`
- **Multiple lines (153, 165, 290, 298, 306, 314, 322)**: Missing property `isCreate` in test options

#### `/src/__test/fixtures/form-testing/TeamFormTest.test.ts`
- **Lines 115, 127, 314**: Missing property `isCreate` in test options

#### `/src/__test/fixtures/form-testing/UIFormTest.test.ts`
- **Line 195**: Missing property `params` in MSWRequest type

### 3. Component Files

#### `/src/components/ChatInterface/ChatInterface.tsx`
- **Line 128**: Missing property `externalApps` in `IntegrationSettings`

#### `/src/components/inputs/AdvancedInput/AdvancedInput.test.tsx`
- **Line 48**: Missing property `name` in `AdvancedInputBaseProps`
- **Line 65**: Missing property `name` in `AdvancedInputBaseProps`
- **Line 360**: Property `value` does not exist in type `AdvancedInputProps`

#### `/src/components/inputs/EditableText.tsx`
- **Line 24**: Missing property `language` in `TranslatedAdvancedInputProps`

#### `/src/components/inputs/form/FormInputRadio.tsx`
- **Line 186**: Missing property `name` in `RadioFormikProps`

#### `/src/components/inputs/CodeInput/CodeInput.tsx`
- **Multiple lines**: Missing properties related to CodeMirror types:
  - Missing property `value` in `SelectionRange` (should be `Range<number>`)
  - Missing property `dispatch` in `EditorView`

#### `/src/components/inputs/TagSelector/TagSelector.tsx`
- **Line 347**: Missing property `align` in `HTMLTextAreaElement` (required in `HTMLDivElement`)

### 4. Navigation Components

#### `/src/components/navigation/BottomNav.test.tsx`
- **Line 32**: Property `setLocation` does not exist in `MockRouterConfig`
- **Line 37**: Missing property `i18n` in translation mock
- **Line 219**: Property `name` does not exist in `MockSession`

#### `/src/components/navigation/TopBar.test.tsx`
- **Lines 184, 210, 238**: Missing property `type` in `TextIconInfo`

### 5. Settings Views

#### `/src/views/settings/SettingsPrivacyView.tsx`
- **Line 144**: Missing property `numVerifiedWallets` in `SettingsPrivacyFormProps`

#### `/src/views/settings/SettingsProfileView.tsx`
- **Lines 89, 90**: Property `updatedAt` does not exist on type `ProfileUpdateInput`
- **Line 225**: Property `numVerifiedWallets` does not exist

### 6. Other Files

#### `/src/utils/display/translationTools.test.ts`
- **Line 882**: Missing property `id` in translation object

#### `/src/hooks/useSocketChat.ts`
- **Line 467**: Property `values` does not exist in type `SnackPub<any>`

#### `/src/components/lists/EventList/EventList.tsx`
- **Line 490**: Property `resources` does not exist in type `CalendarEvent[]`

## Resolution Priority

### High Priority (Easy Fixes)
1. **Add missing `name` properties** in form components
2. **Add missing `type` properties** in icon info objects
3. **Add missing `id` properties** in translation objects
4. **Add missing `isCreate` properties** in test configurations

### Medium Priority (Type Updates)
1. **Update `ProfileUpdateInput` type** to include/exclude `updatedAt`
2. **Update `SettingsPrivacyFormProps`** to include `numVerifiedWallets`
3. **Update `ExtendedPremium` type** to match usage
4. **Fix CodeMirror type incompatibilities**

### Low Priority (Complex Refactors)
1. **Resolve HTMLElement type mismatches** (TextArea vs Div)
2. **Fix MSW request type definitions**
3. **Update mock configurations** for tests

## Quick Fix Examples

### Adding Missing Properties
```typescript
// Before
const iconInfo = { name: 'Home' };

// After
const iconInfo = { name: 'Home', type: 'text' as const };
```

### Updating Form Props
```typescript
// Before
const formProps = { /* other props */ };

// After
const formProps = { 
  /* other props */,
  numVerifiedWallets: 0 
};
```

### Fixing Test Configurations
```typescript
// Before
const testOptions = { user: mockUser };

// After
const testOptions = { 
  user: mockUser,
  isCreate: true 
};
```

## Next Steps
1. Start with high-priority fixes (simple property additions)
2. Update type definitions for medium-priority issues
3. Refactor complex type mismatches last
4. Run type-check after each batch of fixes to verify progress