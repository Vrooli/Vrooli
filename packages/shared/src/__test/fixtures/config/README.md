# Config Fixtures - Foundational Configuration Layer

Config fixtures provide the **configuration definition layer** for the Vrooli testing ecosystem. They serve as the single source of truth for JSON configuration objects used throughout the platform.

## 🎯 Purpose & Importance

Config fixtures are a **foundational layer** that other fixture types depend on. When defining JSON fields in API fixtures, database fixtures, or UI components, config fixtures provide the validated, type-safe configuration objects to use.

**Key Responsibilities:**
- Provide realistic, validated configuration objects for testing
- Ensure type safety with TypeScript interfaces
- Support configuration variations for different test scenarios
- Enable composition and reuse across fixture layers
- Validate configuration integrity before use

## 📁 Current State (FIFTH PASS - REFINEMENT COMPLETED ✅)

### ✅ **Mapping Status**: Perfect 1:1 Mapping Confirmed
### ✅ **Type Safety**: All Critical Issues Fixed

**Source of Truth Analysis:**
- **Config shape files in `packages/shared/src/shape/configs/`**: 14 files (13 configs + 1 utils)
- **Configs needing fixtures**: 13 files (excludes utils.ts - utility functions only)
- **Current config fixture files**: 13 files
- **Infrastructure files**: 4 files (configFactory.ts, configUtils.ts, index.ts, README.md)
- **Total files**: 17 files
- **Mapping Status**: ✅ Perfect 1:1 mapping confirmed
- **Type Safety Status**: ✅ All critical type errors resolved

### **Source Files Analysis (14 total):**
```
✅ api.ts → apiConfigFixtures.ts (✅ verified mapping)
✅ base.ts → baseConfigFixtures.ts (✅ verified mapping)
✅ bot.ts → botConfigFixtures.ts (✅ verified mapping)
✅ chat.ts → chatConfigFixtures.ts (✅ verified mapping)
✅ code.ts → codeConfigFixtures.ts (✅ verified mapping)
✅ credit.ts → creditConfigFixtures.ts (✅ verified mapping)
✅ message.ts → messageConfigFixtures.ts (✅ verified mapping)
✅ note.ts → noteConfigFixtures.ts (✅ verified mapping)
✅ project.ts → projectConfigFixtures.ts (✅ verified mapping)
✅ routine.ts → routineConfigFixtures.ts (✅ all type errors fixed)
✅ run.ts → runConfigFixtures.ts (✅ verified mapping)
✅ standard.ts → standardConfigFixtures.ts (✅ verified mapping)
✅ team.ts → teamConfigFixtures.ts (✅ verified mapping)
✅ utils.ts → NO FIXTURE NEEDED (utility functions only)
```

### **Infrastructure Files (Correct - 4):**
```
✅ configFactory.ts - Factory infrastructure
✅ configUtils.ts - Utility functions
✅ index.ts - Export management
✅ README.md - Documentation
```

### **Verification Summary:**
```
✅ All 13 source files have corresponding fixtures
✅ All 13 fixture files have corresponding sources  
✅ No extra fixtures without source files
✅ No missing fixtures for existing sources
✅ Infrastructure files are appropriate and minimal
✅ All type safety issues resolved
```

**Previous Issues Resolved**: 26 extra fixtures were deleted that had no corresponding source files.

## 🔧 Fifth Pass Refinement Status - CURRENT ANALYSIS

### **Phase 1: Mapping Analysis ✅ COMPLETED**
Perfect 1:1 mapping confirmed - no structural changes needed.

### **Phase 2: Type Safety Analysis ✅ IDENTIFIED CRITICAL ISSUES**

**Critical Type Safety Issues Requiring IMMEDIATE FIX:**

#### **routineConfigFixtures.ts - Multiple Critical Errors:**

1. **Missing BpmnSchema properties** (line 38):
   ```typescript
   // ERROR: Missing required properties
   bpmnSchema: {
     // Missing: activityMap: ActivityMap
     // Missing: rootContext: RootContext  
   }
   ```

2. **Invalid FormElement structures** (lines 196, 202):
   ```typescript
   // ERROR: Missing required 'props' property
   {
     fieldName: "textField",
     type: FormElementType.TextInput,
     // Missing: props: TextFormInputProps
   }
   ```

3. **Unused @ts-expect-error directives** (multiple lines):
   - Several @ts-expect-error comments that are no longer needed
   - May indicate previous type issues that were incorrectly "fixed"

### **Phase 3: Correction Implementation ✅ COMPLETED**

**Issues Fixed:**
1. ✅ **routineConfigFixtures.ts**: Fixed BpmnSchema structure (added missing activityMap and rootContext)
2. ✅ **routineConfigFixtures.ts**: Added missing props property to form elements
3. ✅ **routineConfigFixtures.ts**: Cleaned up unused @ts-expect-error directives
4. ✅ **All config fixtures**: Verified as correctly structured and type-safe

**Structural Status:**
- **NO FILES DELETED**: Perfect 1:1 mapping confirmed  
- **NO FILES CREATED**: All required fixtures exist  
- **NO FILES RENAMED**: All names match source correctly

## 🏗️ Ideal Architecture (CORRECTED)

### **Final Structure (17 files total):**
```
packages/shared/src/__test/fixtures/config/
├── README.md                     # This documentation
├── index.ts                      # Exports (17 items total)
├── configFactory.ts              # Infrastructure: Factory pattern
├── configUtils.ts                # Infrastructure: Utilities
├── apiConfigFixtures.ts          # Config: API settings
├── baseConfigFixtures.ts         # Config: Base configuration
├── botConfigFixtures.ts          # Config: Bot personality
├── chatConfigFixtures.ts         # Config: Chat settings  
├── codeConfigFixtures.ts         # Config: Code execution
├── creditConfigFixtures.ts       # Config: Credit system
├── messageConfigFixtures.ts      # Config: Message formatting
├── noteConfigFixtures.ts         # Config: Note/document settings
├── projectConfigFixtures.ts      # Config: Project settings
├── routineConfigFixtures.ts      # Config: Routine execution
├── runConfigFixtures.ts          # Config: Run environment
├── standardConfigFixtures.ts     # Config: Standard templates
└── teamConfigFixtures.ts         # Config: Team structure
```

### **Enhanced Factory Pattern**

Each config fixture implements the `ConfigFixtureFactory<T>` interface:

```typescript
interface ConfigFixtureFactory<TConfig extends BaseConfigObject> {
    // Core fixtures
    minimal: TConfig;
    complete: TConfig;
    withDefaults: TConfig;
    
    // Variant collections
    variants: { [key: string]: TConfig };
    
    // Invalid configurations for validation testing
    invalid: {
        missingVersion?: Partial<TConfig>;
        invalidVersion?: TConfig;
        malformedStructure?: any;
        invalidTypes?: Partial<TConfig>;
    };
    
    // Factory methods
    create: (overrides?: Partial<TConfig>) => TConfig;
    createVariant: (variant: keyof this['variants'], overrides?: Partial<TConfig>) => TConfig;
    
    // Validation methods
    validate: (config: any) => ValidationResult;
    isValid: (config: unknown) => config is TConfig;
    
    // Composition helpers
    merge: (base: TConfig, override: Partial<TConfig>) => TConfig;
    applyDefaults: (partialConfig: Partial<TConfig>) => TConfig;
    
    // Integration helpers
    toApiFormat: () => ApiConfigFormat;
    toDbFormat: () => DbConfigFormat;
    fromJson: (json: string) => TConfig;
    toJson: (config: TConfig) => string;
}
```

## 🔄 Integration with Other Fixtures

### API Fixtures Integration

```typescript
// In API fixtures, use config fixtures for JSON fields
import { botConfigFixtures, chatConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

export const botApiFixtures = {
    complete: {
        create: {
            id: "bot_123",
            name: "Assistant Bot",
            // Use config fixture for botSettings
            botSettings: botConfigFixtures.complete,
        }
    }
};
```

### Database Fixtures Integration

```typescript
// In database fixtures, use configs for JSON columns
import { routineConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

export const RoutineDbFactory = {
    createWithConfig: async (config?: Partial<RoutineVersionConfigObject>) => {
        const routineConfig = routineConfigFixtures.create(config);
        
        return await prisma.routineVersion.create({
            data: {
                configCallbacks: routineConfig,
                // ... other fields
            }
        });
    }
};
```

## 📋 Usage Guidelines

### 1. Always Use Config Fixtures for JSON Fields

```typescript
// ✅ GOOD: Use config fixtures for JSON configuration fields
const resource = {
    id: "resource_123",
    versionId: "version_456",
    config: configFixtures.apiConfigFixtures.complete,
};

// ❌ BAD: Hardcode JSON config
const resource = {
    id: "resource_123", 
    versionId: "version_456",
    config: { version: "1.0", settings: {} }, // Don't do this!
};
```

### 2. Choose Appropriate Complexity

```typescript
// For simple tests, use minimal configs
const basicBot = {
    botSettings: botConfigFixtures.minimal,
};

// For integration tests, use complete configs
const fullBot = {
    botSettings: botConfigFixtures.complete,
};

// For specific scenarios, use variants
const supportBot = {
    botSettings: botConfigFixtures.variants.customerServiceBot,
};
```

## 🧪 Testing Patterns

### Testing Configuration Validation

```typescript
describe("Bot Configuration", () => {
    it("should accept valid minimal config", () => {
        const result = botConfigFixtures.validate(botConfigFixtures.minimal);
        expect(result.success).toBe(true);
    });
    
    it("should reject missing version", () => {
        const result = botConfigFixtures.validate(botConfigFixtures.invalid.missingVersion);
        expect(result.success).toBe(false);
        expect(result.error).toContain("version");
    });
    
    it("should provide defaults", () => {
        const config = botConfigFixtures.applyDefaults({ model: "gpt-4" });
        expect(config.persona).toEqual(DEFAULT_PERSONA);
    });
});
```

## 🚀 Best Practices

1. **1:1 Mapping**: One fixture file per config shape file, nothing more
2. **Version All Configs**: Always include `__version` field
3. **Provide Variants**: Cover common use cases with named variants
4. **Type Everything**: Use TypeScript interfaces for all configs
5. **Validate Early**: Validate configs at creation time
6. **Test Invalid States**: Include invalid fixtures for error testing
7. **Keep Realistic**: Ensure configs mirror production usage

## 📚 Config Types Reference

| Config Type | Purpose | Key Fields | Common Variants |
|------------|---------|------------|-----------------|
| `apiConfig` | API endpoint settings | `rateLimit`, `auth`, `cors` | `public`, `authenticated`, `rateLimited` |
| `botConfig` | AI bot personality | `model`, `persona`, `maxTokens` | `customerService`, `technical`, `creative` |
| `chatConfig` | Chat room settings | `privacy`, `moderation`, `features` | `privateTeam`, `publicSupport`, `aiAssisted` |
| `codeConfig` | Code execution | `language`, `runtime`, `limits` | `javascript`, `python`, `sandboxed` |
| `creditConfig` | Credit system | `costs`, `limits`, `rollover` | `free`, `premium`, `enterprise` |
| `messageConfig` | Message formatting | `format`, `attachments`, `limits` | `plainText`, `markdown`, `rich` |
| `noteConfig` | Document settings | `format`, `versioning`, `sharing` | `private`, `collaborative`, `published` |
| `projectConfig` | Project settings | `visibility`, `permissions`, `workflow` | `personal`, `team`, `openSource` |
| `routineConfig` | Routine execution | `type`, `steps`, `errorHandling` | `simple`, `multiStep`, `conditional` |
| `runConfig` | Run environment | `environment`, `resources`, `timeout` | `development`, `production`, `testing` |
| `standardConfig` | Template settings | `category`, `tags`, `licensing` | `official`, `community`, `premium` |
| `teamConfig` | Team structure | `structure`, `roles`, `permissions` | `startup`, `enterprise`, `flat` |

## 🔗 Related Documentation

- [Fixtures Overview](/docs/testing/fixtures-overview.md) - General fixture system documentation
- [API Fixtures](/packages/shared/src/__test/fixtures/api/README.md) - API fixture layer that consumes configs
- [Database Fixtures](/packages/server/src/__test/fixtures/db/README.md) - DB fixtures that persist configs
- [Shape Configs](/packages/shared/src/shape/configs/README.md) - Source configuration type definitions

## 📝 Implementation Checklist

When adding new config fixtures **ONLY IF** a corresponding config shape file exists:

- [ ] Verify corresponding file exists in `packages/shared/src/shape/configs/`
- [ ] Create TypeScript interface extending `BaseConfigObject`
- [ ] Implement `ConfigFixtureFactory` pattern
- [ ] Include `minimal`, `complete`, and `withDefaults` fixtures
- [ ] Add at least 3 meaningful variants
- [ ] Include invalid fixtures for validation testing
- [ ] Create factory functions for common patterns
- [ ] Add to index.ts exports
- [ ] Update this README
- [ ] Add validation tests

Config fixtures are the foundation of our testing strategy - they ensure consistency, type safety, and realistic test scenarios across the entire Vrooli platform.