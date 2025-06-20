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

## 📁 Current Architecture

### File Structure
```
packages/shared/src/__test/fixtures/config/
├── index.ts                    # Central exports and namespace
├── baseConfigFixtures.ts       # Base configuration pattern
├── apiConfigFixtures.ts        # API endpoint configurations
├── botConfigFixtures.ts        # AI bot persona & settings
├── chatConfigFixtures.ts       # Chat room configurations
├── codeConfigFixtures.ts       # Code execution configs
├── creditConfigFixtures.ts     # Credit system configs
├── messageConfigFixtures.ts    # Message formatting configs
├── noteConfigFixtures.ts       # Note/document configs
├── projectConfigFixtures.ts    # Project settings
├── routineConfigFixtures.ts    # Routine execution configs
├── runConfigFixtures.ts        # Run environment configs
├── standardConfigFixtures.ts   # Standard/template configs
├── teamConfigFixtures.ts       # Team structure configs
└── utilsConfigFixtures.ts      # Utility configurations
```

### Current Pattern

Each config fixture follows the `ConfigTestFixtures` interface:

```typescript
interface ConfigTestFixtures<T extends BaseConfigObject> {
    minimal: T;              // Minimal valid config
    complete: T;             // Fully populated config
    withDefaults: T;         // Config with defaults applied
    invalid: {               // Invalid configurations for testing
        missingVersion?: Partial<T>;
        invalidVersion?: T;
        malformedStructure?: any;
        invalidTypes?: Partial<T>;
    };
    variants: Record<string, T>;  // Different valid configurations
}
```

### Strengths of Current Architecture

1. **Consistent Structure**: All configs follow the same pattern
2. **Type Safety**: Full TypeScript support with shape types
3. **Validation Testing**: Invalid fixtures for testing error cases
4. **Scenario Coverage**: Variants provide realistic use cases
5. **Factory Functions**: Helper functions for common patterns

### Areas for Improvement

1. **Validation Integration**: No built-in validation methods
2. **Composition Helpers**: Limited support for merging configs
3. **Documentation**: Missing usage examples in each file
4. **Cross-References**: No clear links to consuming fixtures
5. **Version Management**: Manual version tracking

## 🏗️ Ideal Architecture

### Enhanced Factory Pattern

```typescript
interface ConfigFixtureFactory<TConfig extends BaseConfigObject> {
    // Core fixtures
    minimal: TConfig;
    complete: TConfig;
    withDefaults: TConfig;
    
    // Variant collections
    variants: {
        [key: string]: TConfig;
    };
    
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
    validate: (config: TConfig) => ValidationResult;
    isValid: (config: unknown) => config is TConfig;
    
    // Composition helpers
    merge: (base: TConfig, override: Partial<TConfig>) => TConfig;
    withDefaults: (partialConfig: Partial<TConfig>) => TConfig;
    
    // Integration helpers
    toApiFormat: () => ApiConfigFormat;
    toDbFormat: () => DbConfigFormat;
    fromJson: (json: string) => TConfig;
    toJson: (config: TConfig) => string;
}
```

### Enhanced Implementation Example

```typescript
// Enhanced bot config fixtures with ideal pattern
export const botConfigFixtures: ConfigFixtureFactory<BotConfigObject> = {
    // Core fixtures (existing)
    minimal: { __version: LATEST_CONFIG_VERSION },
    complete: { /* ... */ },
    withDefaults: { /* ... */ },
    variants: { /* ... */ },
    invalid: { /* ... */ },
    
    // Factory methods
    create: (overrides = {}) => {
        return mergeWithValidation(botConfigFixtures.minimal, overrides);
    },
    
    createVariant: (variant, overrides = {}) => {
        const base = botConfigFixtures.variants[variant];
        if (!base) throw new Error(`Unknown variant: ${variant}`);
        return mergeWithValidation(base, overrides);
    },
    
    // Validation methods
    validate: (config) => {
        return botConfigSchema.validate(config);
    },
    
    isValid: (config): config is BotConfigObject => {
        return botConfigSchema.isValid(config);
    },
    
    // Composition helpers
    merge: (base, override) => {
        return deepMerge(base, override, { 
            arrayMerge: 'replace',
            preserveVersion: true 
        });
    },
    
    withDefaults: (partial) => {
        return { ...DEFAULT_BOT_CONFIG, ...partial };
    },
    
    // Integration helpers
    toApiFormat: () => {
        // Transform for API consumption
    },
    
    toDbFormat: () => {
        // Transform for database storage
    },
    
    fromJson: (json) => {
        const parsed = JSON.parse(json);
        return botConfigFixtures.validate(parsed).data;
    },
    
    toJson: (config) => {
        return JSON.stringify(config, null, 2);
    }
};
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

export const chatApiFixtures = {
    variants: {
        supportChat: {
            id: "chat_456",
            name: "Customer Support",
            // Use config fixture variant
            chatSettings: chatConfigFixtures.variants.supportChat,
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

### UI Component Integration

```typescript
// In UI tests, use configs for component props
import { configFixtures } from "@vrooli/shared/__test/fixtures";

const mockBot = {
    id: "bot_789",
    name: "Test Bot",
    settings: configFixtures.botConfigFixtures.variants.customerServiceBot,
};

render(<BotConfigEditor bot={mockBot} />);
```

## 📋 Usage Guidelines

### 1. Always Use Config Fixtures for JSON Fields

```typescript
// ✅ GOOD: Use config fixtures
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

### 3. Validate Configurations

```typescript
// Always validate when creating custom configs
const customConfig = botConfigFixtures.create({
    model: "custom-model",
    maxTokens: 4096,
});

// Validate external configs
const isValid = botConfigFixtures.isValid(externalConfig);
if (!isValid) {
    throw new Error("Invalid bot configuration");
}
```

### 4. Use Factory Methods for Variations

```typescript
// Create variations using factory methods
const technicalBot = createBotConfigWithPersona("Software Engineer", {
    tone: "technical and precise",
    creativity: 0.2,
    verbosity: 0.8,
});

// Create team structures
const startupTeam = createMoiseTeamStructure("Startup", [
    { name: "founder", min: 1, max: 2 },
    { name: "developer", min: 1, max: 5 },
]);
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
        const config = botConfigFixtures.withDefaults({ model: "gpt-4" });
        expect(config.persona).toEqual(DEFAULT_PERSONA);
    });
});
```

### Testing Configuration Usage

```typescript
describe("Resource with Config", () => {
    it("should create resource with API config", async () => {
        const resource = await ResourceDbFactory.create({
            config: apiConfigFixtures.variants.rateLimited,
        });
        
        expect(resource.config.rateLimit).toBeDefined();
        expect(resource.config.rateLimit.maxRequests).toBe(100);
    });
});
```

## 🚀 Best Practices

1. **Version All Configs**: Always include `__version` field
2. **Provide Variants**: Cover common use cases with named variants
3. **Document Variants**: Add comments explaining each variant's purpose
4. **Type Everything**: Use TypeScript interfaces for all configs
5. **Validate Early**: Validate configs at creation time
6. **Test Invalid States**: Include invalid fixtures for error testing
7. **Use Composition**: Build complex configs from simpler ones
8. **Keep Realistic**: Ensure configs mirror production usage

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
| `utilsConfig` | Utility settings | `features`, `defaults`, `overrides` | `basic`, `advanced`, `custom` |

## 🔗 Related Documentation

- [Fixtures Overview](/docs/testing/fixtures-overview.md) - General fixture system documentation
- [API Fixtures](/packages/shared/src/__test/fixtures/api/README.md) - API fixture layer that consumes configs
- [Database Fixtures](/packages/server/src/__test/fixtures/db/README.md) - DB fixtures that persist configs
- [Shape Configs](/packages/shared/src/shape/configs/README.md) - Source configuration type definitions

## 📝 Implementation Checklist

When adding new config fixtures:

- [ ] Create TypeScript interface extending `BaseConfigObject`
- [ ] Implement `ConfigTestFixtures` structure
- [ ] Include `minimal`, `complete`, and `withDefaults` fixtures
- [ ] Add at least 3 meaningful variants
- [ ] Include invalid fixtures for validation testing
- [ ] Create factory functions for common patterns
- [ ] Add to index.ts exports
- [ ] Document in this README
- [ ] Update consuming fixtures to use new configs
- [ ] Add validation tests

## 🤝 Contributing

When contributing to config fixtures:

1. Follow the established `ConfigTestFixtures` pattern
2. Ensure all configs have proper TypeScript types
3. Include realistic data that mirrors production usage
4. Document the purpose of each variant
5. Test both valid and invalid configurations
6. Update related fixtures that should use the new configs

Config fixtures are the foundation of our testing strategy - they ensure consistency, type safety, and realistic test scenarios across the entire Vrooli platform.