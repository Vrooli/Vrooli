# Database Fixtures Infrastructure Repair Plan

## Executive Summary

The database fixtures system requires comprehensive infrastructure repair before new fixtures can be safely added. This plan addresses foundational issues in a systematic, risk-managed approach.

## Root Cause Analysis

### Issue 1: Import Chain Failures
**Symptom**: `generatePK` import from `@vrooli/shared` failing across multiple files
**Root Causes**:
- Potential circular dependency issues
- Module resolution problems in test environment
- TypeScript compilation target mismatches between packages
- Shared package build state inconsistencies

### Issue 2: Prisma Type System Misalignment
**Symptom**: Incorrect Prisma type names (PascalCase vs snake_case)
**Root Causes**:
- Schema evolution without fixture updates
- Manual type naming instead of generated types
- Inconsistent prisma generate execution
- Mixed conventions from different development periods

### Issue 3: BigInt/ES Target Incompatibility
**Symptom**: BigInt literals not supported in current TypeScript target
**Root Causes**:
- TypeScript target below ES2020
- Mixed ES targets across packages
- Legacy configuration not updated for modern Prisma

### Issue 4: Factory Interface Architecture Drift
**Symptom**: Factory classes not properly implementing base interfaces
**Root Causes**:
- Evolution of EnhancedDatabaseFactory without migration
- Missing abstract method implementations
- Type parameter mismatches
- Interface changes without fixture updates

---

## Strategic Approach

### Phase-Gate Methodology
Each phase has:
- **Entry Criteria**: Prerequisites that must be met
- **Success Criteria**: Measurable outcomes
- **Rollback Plan**: How to safely revert if issues arise
- **Verification Steps**: How to confirm success

### Risk Mitigation
- **Incremental Changes**: Small, testable steps
- **Backup Strategy**: Git branching + backup copies of critical files
- **Parallel Development**: Keep existing system functional during repairs
- **Validation Gates**: Type checking + runtime testing at each step

---

## PHASE 1: Environment Stabilization (Foundation)

### 1.1 Package Build Verification
**Objective**: Ensure all packages build correctly in isolation

**Entry Criteria**: 
- Clean git working directory
- All containers stopped

**Steps**:
```bash
# 1. Clean all build artifacts
cd /root/Vrooli
pnpm clean
rm -rf node_modules
rm -rf packages/*/node_modules
rm -rf packages/*/dist

# 2. Fresh install with locked versions
pnpm install --frozen-lockfile

# 3. Build packages in dependency order
cd packages/shared && pnpm build
cd ../server && pnpm build
cd ../ui && pnpm build
cd ../jobs && pnpm build
```

**Verification**:
```bash
# Verify shared package exports
node -e "console.log(Object.keys(require('./packages/shared/dist/index.js')))"
# Should include: generatePK, generatePublicId, etc.

# Check Prisma client generation
cd packages/server && npx prisma generate
ls node_modules/.prisma/client/index.d.ts
```

**Success Criteria**:
- [ ] All packages build without errors
- [ ] Shared package exports include `generatePK`
- [ ] Prisma client is properly generated
- [ ] No circular dependency warnings

**Rollback Plan**: 
- Restore from git clean state
- Use `pnpm install` to restore previous state

### 1.2 TypeScript Configuration Audit
**Objective**: Align TypeScript targets across all packages

**Investigation Steps**:
```bash
# Check current targets
grep -r "target" packages/*/tsconfig*.json
grep -r "module" packages/*/tsconfig*.json
grep -r "lib" packages/*/tsconfig*.json
```

**Analysis Required**:
- Are all packages using compatible targets?
- Does server package support ES2020+ for BigInt?
- Are module resolution settings consistent?
- Do test configurations inherit properly?

**Expected Findings**: Document in `TSCONFIG_AUDIT.md`

### 1.3 Import Path Verification
**Objective**: Verify import resolution works correctly

**Create Test File**:
```typescript
// packages/server/test-imports.ts
import { generatePK, generatePublicId } from "@vrooli/shared";
import { PrismaClient } from "@prisma/client";

console.log("generatePK:", typeof generatePK);
console.log("generatePublicId:", typeof generatePublicId);
console.log("PrismaClient:", typeof PrismaClient);

const pk = generatePK();
console.log("Generated PK:", pk, typeof pk);
```

**Test Execution**:
```bash
cd packages/server
npx ts-node test-imports.ts
```

**Success Criteria**:
- [ ] All imports resolve successfully
- [ ] generatePK returns bigint type
- [ ] No module resolution errors

---

## PHASE 2: TypeScript Configuration Repair

### 2.1 Server Package TypeScript Update
**Objective**: Enable ES2020+ support for BigInt and modern features

**Backup**:
```bash
cp packages/server/tsconfig.json packages/server/tsconfig.json.backup
cp packages/server/tsconfig.test.json packages/server/tsconfig.test.json.backup
```

**Configuration Changes**:
```json
// packages/server/tsconfig.json
{
  "extends": "../../tsconfig.json",
  "compilerOptions": {
    "target": "ES2020",           // Enable BigInt support
    "module": "ESNext",           // Modern module system
    "lib": ["ES2020", "DOM"],     // Include BigInt in lib
    "moduleResolution": "node",   // Ensure proper resolution
    "allowSyntheticDefaultImports": true,
    "esModuleInterop": true,
    "resolveJsonModule": true     // For Prisma JSON imports
  },
  "include": [
    "src/**/*",
    "src/**/*.json"
  ],
  "exclude": [
    "dist",
    "node_modules"
  ]
}
```

**Verification**:
```bash
cd packages/server
npx tsc --noEmit test-imports.ts
# Should compile without BigInt errors
```

### 2.2 Shared Package Compatibility Check
**Objective**: Ensure shared package exports are compatible with server target

**Test**:
```bash
cd packages/shared
npx tsc --noEmit src/id/snowflake.ts
# Check for BigInt literal errors
```

**If Issues Found**:
- Update shared package tsconfig similarly
- Ensure build artifacts are regenerated
- Verify export compatibility

**Cross-Package Test**:
```typescript
// packages/server/cross-package-test.ts
import { generatePK } from "@vrooli/shared";

const id: bigint = generatePK();
console.log("ID type check passed:", typeof id === 'bigint');
```

---

## PHASE 3: Prisma Type System Repair

### 3.1 Prisma Client Regeneration
**Objective**: Ensure latest Prisma types are available

**Steps**:
```bash
cd packages/server
# Force clean regeneration
rm -rf node_modules/.prisma
npx prisma generate
```

**Type Export Verification**:
```typescript
// packages/server/verify-prisma-types.ts
import { Prisma } from '@prisma/client';

// Test all model types we use in fixtures
type UserType = Prisma.user;
type TeamType = Prisma.team;
type BookmarkType = Prisma.bookmark;
type CreditAccountType = Prisma.credit_account;

console.log("Prisma types available:", {
  user: 'user' in Prisma,
  team: 'team' in Prisma,
  bookmark: 'bookmark' in Prisma,
  credit_account: 'credit_account' in Prisma,
});
```

### 3.2 Prisma Type Naming Convention Audit
**Objective**: Document correct type names for all 66 models

**Create Reference Document**:
```bash
# Generate complete type mapping
cd packages/server
node -e "
const { Prisma } = require('@prisma/client');
const fs = require('fs');
const types = Object.keys(Prisma).filter(key => 
  key.endsWith('CreateInput') || 
  key.endsWith('UpdateInput') || 
  key.endsWith('Include')
);
fs.writeFileSync('PRISMA_TYPE_REFERENCE.md', 
  '# Prisma Type Reference\\n\\n' + 
  types.map(t => \`- \${t}\`).join('\\n')
);
"
```

**Expected Output**: Complete mapping of:
- Model names (snake_case in schema)
- CreateInput types (camelCase + CreateInput)
- UpdateInput types (camelCase + UpdateInput)
- Include types (camelCase + Include)

### 3.3 Model-to-Type Mapping Creation
**Objective**: Create authoritative mapping for fixture development

**Create Mapping File**:
```typescript
// packages/server/src/__test/fixtures/db/PRISMA_TYPE_MAP.ts
export const PRISMA_TYPE_MAP = {
  // Model name -> Prisma types
  'user': {
    model: 'user',
    createInput: 'userCreateInput',
    updateInput: 'userUpdateInput', 
    include: 'userInclude'
  },
  'credit_account': {
    model: 'credit_account',
    createInput: 'credit_accountCreateInput',
    updateInput: 'credit_accountUpdateInput',
    include: 'credit_accountInclude'
  },
  // ... complete mapping for all 66 models
} as const;
```

---

## PHASE 4: Base Factory Interface Repair

### 4.1 EnhancedDatabaseFactory Analysis
**Objective**: Understand current interface and identify breaking changes

**Investigation**:
```bash
cd packages/server/src/__test/fixtures/db
# Analyze the base factory
head -100 EnhancedDatabaseFactory.ts
# Check what methods are required vs optional
grep -n "abstract" EnhancedDatabaseFactory.ts
```

**Create Interface Documentation**:
```typescript
// Document current interface requirements
interface RequiredMethods {
  // What MUST be implemented?
  // What parameters are expected?
  // What return types are required?
}
```

### 4.2 Factory Implementation Pattern Analysis
**Objective**: Analyze existing working factories to understand the pattern

**Study Successful Implementations**:
```bash
# Find factories that compile without errors
# (if any exist)
cd packages/server
for factory in src/__test/fixtures/db/*DbFactory.ts; do
  echo "Checking $factory..."
  npx tsc --noEmit "$factory" 2>/dev/null && echo "✅ $factory compiles"
done
```

**Document Working Patterns**:
- How do successful factories implement required methods?
- What's the correct constructor signature?
- How are Prisma delegates accessed?
- What's the proper generic type parameter pattern?

### 4.3 Interface Standardization
**Objective**: Create a standardized factory template

**Create Template**:
```typescript
// packages/server/src/__test/fixtures/db/FACTORY_TEMPLATE.ts
import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { DbTestFixtures } from "./types.js";

/**
 * Template for creating model-specific database factories
 * Replace MODEL_NAME with actual model name
 */
export class ModelNameDbFactory extends EnhancedDatabaseFactory<
    Prisma.model_name,              // Actual model type
    Prisma.model_nameCreateInput,   // Create input type
    Prisma.model_nameInclude,       // Include type
    Prisma.model_nameUpdateInput    // Update input type
> {
    constructor(prisma: PrismaClient) {
        super('model_name', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.model_name;
    }

    protected getFixtures(): DbTestFixtures<
        Prisma.model_nameCreateInput, 
        Prisma.model_nameUpdateInput
    > {
        return {
            minimal: {
                // Minimal required fields
            },
            complete: {
                // All commonly used fields
            },
            variants: {
                // Named variations for specific test scenarios
            },
            invalid: {
                // Invalid data for error testing
                missingRequired: {
                    // Missing required fields
                },
                invalidTypes: {
                    // Invalid field types
                }
            },
            edgeCase: {
                // Edge cases and boundary conditions
            },
            update: {
                // Update operations
            }
        };
    }
}

export function createModelNameDbFactory(prisma: PrismaClient): ModelNameDbFactory {
    return new ModelNameDbFactory(prisma);
}
```

---

## PHASE 5: Incremental Factory Repair

### 5.1 Pilot Factory Repair
**Objective**: Fix one factory completely to validate the approach

**Choose Pilot**: Start with simplest model (e.g., `tag` or `view`)

**Repair Process**:
1. **Backup Original**:
   ```bash
   cp src/__test/fixtures/db/TagDbFactory.ts src/__test/fixtures/db/TagDbFactory.ts.backup
   ```

2. **Apply Template**:
   - Use correct Prisma types from mapping
   - Implement all required methods
   - Follow established patterns

3. **Incremental Testing**:
   ```bash
   npx tsc --noEmit src/__test/fixtures/db/TagDbFactory.ts
   # Fix compilation errors one by one
   ```

4. **Runtime Testing**:
   ```typescript
   // Create test script
   import { TagDbFactory } from './TagDbFactory.js';
   import { PrismaClient } from '@prisma/client';
   
   const prisma = new PrismaClient();
   const factory = new TagDbFactory(prisma);
   
   // Test basic functionality
   const minimal = factory.getFixtures().minimal;
   console.log("Minimal fixture:", minimal);
   ```

### 5.2 Factory Repair Automation
**Objective**: Create tools to systematically repair all factories

**Create Repair Script**:
```typescript
// packages/server/src/__test/fixtures/db/repair-factories.ts
import fs from 'fs';
import path from 'path';
import { PRISMA_TYPE_MAP } from './PRISMA_TYPE_MAP.js';

interface FactoryRepairConfig {
    modelName: string;
    factoryFile: string;
    hasSimpleFixtures: boolean;
}

class FactoryRepairer {
    async repairFactory(config: FactoryRepairConfig) {
        // 1. Load current factory
        // 2. Apply type corrections
        // 3. Fix interface implementations
        // 4. Validate against template
        // 5. Test compilation
    }
    
    async validateRepair(factoryFile: string): Promise<boolean> {
        // Run type checking
        // Verify runtime instantiation
        // Check method implementations
    }
}
```

### 5.3 Batch Repair Execution
**Objective**: Repair all existing factories systematically

**Execution Strategy**:
1. **Group by Complexity**:
   - Simple models (no relationships)
   - Models with basic relationships
   - Complex models with multiple relationships

2. **Repair in Batches**:
   ```bash
   # Batch 1: Simple models
   node repair-factories.ts --batch simple
   
   # Batch 2: Basic relationships
   node repair-factories.ts --batch basic
   
   # Batch 3: Complex models
   node repair-factories.ts --batch complex
   ```

3. **Validation After Each Batch**:
   ```bash
   # Test compilation of repaired batch
   npx tsc --noEmit src/__test/fixtures/db/*DbFactory.ts
   
   # Run basic runtime tests
   npm run test:fixtures:quick
   ```

---

## PHASE 6: Missing Factory Creation

### 6.1 Missing Factory Identification
**Objective**: Create systematic list of missing factories

**Gap Analysis**:
```bash
# Generate complete missing factory list
cd packages/server/src/__test/fixtures/db
node -e "
const fs = require('fs');
const path = require('path');

// Get all Prisma models
const schema = fs.readFileSync('../../../db/schema.prisma', 'utf8');
const models = [...schema.matchAll(/^model (\\w+)/gm)].map(m => m[1]);

// Get existing factories
const existing = fs.readdirSync('.')
  .filter(f => f.endsWith('DbFactory.ts'))
  .map(f => f.replace('DbFactory.ts', '').toLowerCase());

const missing = models.filter(model => {
  const expectedName = model.replace(/_([a-z])/g, (_, char) => char.toUpperCase());
  return !existing.includes(expectedName.toLowerCase());
});

console.log('Missing Factories:');
missing.forEach(model => console.log(\`- \${model}DbFactory.ts\`));
"
```

### 6.2 Priority-Based Factory Creation
**Objective**: Create missing factories in order of importance

**Priority Classification**:
1. **Critical**: Core business models (user, team, project, routine)
2. **High**: Frequently used models (payment, notification, resource)
3. **Medium**: Supporting models (translation, stats)
4. **Low**: Rarely used models (specialized features)

**Creation Process for Each Factory**:
1. **Generate from Template**:
   ```bash
   cd packages/server/src/__test/fixtures/db
   node create-factory.ts --model credit_account --priority high
   ```

2. **Implement Model-Specific Logic**:
   - Research model relationships in schema
   - Define appropriate test scenarios
   - Create meaningful edge cases
   - Add model-specific helper methods

3. **Validate Implementation**:
   ```bash
   npx tsc --noEmit CreditAccountDbFactory.ts
   node test-factory.ts --factory CreditAccountDbFactory
   ```

### 6.3 Relationship Testing
**Objective**: Ensure factories properly handle relationships

**Create Relationship Test Suite**:
```typescript
// packages/server/src/__test/fixtures/db/relationship-tests.ts
describe('Factory Relationship Tests', () => {
  test('User -> CreditAccount relationship', async () => {
    const userFactory = new UserDbFactory(prisma);
    const creditFactory = new CreditAccountDbFactory(prisma);
    
    const user = await userFactory.createMinimal();
    const account = await creditFactory.createForUser(user.id);
    
    expect(account.userId).toBe(user.id);
  });
  
  // Test all critical relationships
});
```

---

## PHASE 7: Integration and Validation

### 7.1 Complete System Testing
**Objective**: Verify the entire fixture system works correctly

**Test Categories**:
1. **Compilation Tests**:
   ```bash
   cd packages/server
   npx tsc --noEmit src/__test/fixtures/db/*.ts
   ```

2. **Factory Instantiation Tests**:
   ```typescript
   // Test all factories can be instantiated
   for (const Factory of allFactories) {
     const instance = new Factory(prisma);
     expect(instance).toBeInstanceOf(Factory);
   }
   ```

3. **Basic CRUD Tests**:
   ```typescript
   // Test basic operations for each factory
   for (const factory of allFactories) {
     const minimal = await factory.createMinimal();
     expect(minimal.id).toBeDefined();
     
     const complete = await factory.createComplete();
     expect(complete.id).toBeDefined();
   }
   ```

4. **Relationship Tests**:
   ```typescript
   // Test cross-factory relationships
   const user = await userFactory.createMinimal();
   const team = await teamFactory.createWithOwner(user.id);
   const member = await memberFactory.createMember(user.id, team.id);
   
   expect(member.userId).toBe(user.id);
   expect(member.teamId).toBe(team.id);
   ```

### 7.2 Performance Testing
**Objective**: Ensure fixtures don't have performance issues

**Performance Benchmarks**:
```typescript
describe('Fixture Performance', () => {
  test('Factory creation time', async () => {
    const start = Date.now();
    const factory = new UserDbFactory(prisma);
    const duration = Date.now() - start;
    expect(duration).toBeLessThan(10); // < 10ms
  });
  
  test('Bulk creation performance', async () => {
    const start = Date.now();
    const users = await userFactory.seedMultiple(100);
    const duration = Date.now() - start;
    expect(duration).toBeLessThan(5000); // < 5s for 100 records
  });
});
```

### 7.3 Documentation Generation
**Objective**: Generate comprehensive documentation

**Auto-Generated Documentation**:
```typescript
// packages/server/src/__test/fixtures/db/generate-docs.ts
class DocumentationGenerator {
  generateFactoryDocs() {
    // For each factory:
    // - List available methods
    // - Show example usage
    // - Document relationships
    // - List test scenarios
  }
  
  generateAPIReference() {
    // Complete API reference for all factories
  }
  
  generateMigrationGuide() {
    // Guide for updating fixtures when schema changes
  }
}
```

---

## PHASE 8: Maintenance and Governance

### 8.1 Automated Quality Gates
**Objective**: Prevent future fixture system degradation

**CI/CD Integration**:
```yaml
# .github/workflows/fixture-quality.yml
name: Fixture Quality Check
on: [push, pull_request]

jobs:
  fixture-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Install dependencies
        run: pnpm install
      
      - name: Type check fixtures
        run: |
          cd packages/server
          npx tsc --noEmit src/__test/fixtures/db/*.ts
      
      - name: Run fixture tests
        run: |
          cd packages/server
          npm run test:fixtures
      
      - name: Validate fixture coverage
        run: |
          cd packages/server
          node src/__test/fixtures/db/validate-coverage.ts
```

### 8.2 Schema Change Management
**Objective**: Handle Prisma schema changes gracefully

**Schema Change Detection**:
```typescript
// packages/server/src/__test/fixtures/db/schema-monitor.ts
class SchemaChangeDetector {
  detectChanges(oldSchema: string, newSchema: string): ChangeReport {
    // Detect:
    // - New models (need new fixtures)
    // - Deleted models (remove fixtures)
    // - Field changes (update fixtures)
    // - Relationship changes (update relationships)
  }
  
  generateMigrationTasks(changes: ChangeReport): MigrationTask[] {
    // Generate specific tasks for fixture updates
  }
}
```

### 8.3 Fixture Maintenance Tools
**Objective**: Provide tools for ongoing maintenance

**Maintenance Commands**:
```bash
# Check fixture coverage
npm run fixtures:coverage

# Update fixtures after schema changes
npm run fixtures:update

# Validate all fixtures
npm run fixtures:validate

# Generate new factory template
npm run fixtures:generate -- --model new_model

# Repair broken fixtures
npm run fixtures:repair
```

---

## Risk Assessment and Mitigation

### High-Risk Activities
1. **TypeScript Configuration Changes**
   - **Risk**: Break compilation for entire server package
   - **Mitigation**: Test with isolated files first, maintain backups
   - **Rollback**: Restore tsconfig.json from backup

2. **Batch Factory Repairs**
   - **Risk**: Mass introduction of bugs
   - **Mitigation**: Repair in small batches with validation
   - **Rollback**: Git branch for each batch, easy revert

3. **Prisma Type System Changes**
   - **Risk**: Break existing database operations
   - **Mitigation**: Only change fixture code, not schema
   - **Rollback**: No schema changes involved

### Medium-Risk Activities
1. **Import Path Changes**
   - **Risk**: Break imports in other parts of system
   - **Mitigation**: Only change test fixtures, not shared exports
   - **Rollback**: Revert specific import changes

2. **Factory Interface Updates**
   - **Risk**: Break existing test code using factories
   - **Mitigation**: Maintain backward compatibility in public methods
   - **Rollback**: Restore interface definitions

### Low-Risk Activities
1. **Documentation Updates**
2. **Adding new factories**
3. **Tool creation**

---

## Success Metrics

### Phase Completion Metrics
- [ ] **Phase 1**: All packages build successfully
- [ ] **Phase 2**: TypeScript compilation passes
- [ ] **Phase 3**: All Prisma types resolve correctly  
- [ ] **Phase 4**: Base factory interface implemented properly
- [ ] **Phase 5**: All existing factories compile and run
- [ ] **Phase 6**: All 66 models have corresponding factories
- [ ] **Phase 7**: Complete test suite passes
- [ ] **Phase 8**: Maintenance tools operational

### Quality Metrics
- **Type Safety**: 100% TypeScript compilation success
- **Test Coverage**: All factories have basic functionality tests
- **Performance**: Factory operations complete within benchmarks
- **Documentation**: Complete API documentation generated

### Operational Metrics
- **Developer Experience**: Easy to create new fixtures
- **Maintenance**: Automated detection of schema changes
- **Reliability**: Fixtures don't break with normal development

---

## Timeline Estimate

### Detailed Timeline
- **Phase 1 (Environment)**: 1-2 days
- **Phase 2 (TypeScript)**: 1 day  
- **Phase 3 (Prisma Types)**: 1-2 days
- **Phase 4 (Base Interface)**: 2-3 days
- **Phase 5 (Factory Repair)**: 3-5 days
- **Phase 6 (Missing Factories)**: 2-3 days
- **Phase 7 (Integration)**: 2-3 days
- **Phase 8 (Maintenance)**: 1-2 days

**Total Estimate**: 13-21 days (2.5-4 weeks)

### Critical Path
1. Environment stabilization (blocks everything)
2. TypeScript configuration (blocks type-dependent work)
3. Base interface repair (blocks factory development)
4. Factory repair/creation (parallel work possible)

### Resource Requirements
- **Primary Developer**: Full-time focus on fixture system
- **Secondary Developer**: Code review and testing support
- **Database Access**: For integration testing
- **CI/CD Pipeline**: For automated validation

---

## Conclusion

This infrastructure repair plan provides a systematic approach to fixing the database fixtures system. The key is to address foundational issues first, then build incrementally with proper validation at each step.

The plan prioritizes stability and maintainability over speed, ensuring that once repaired, the fixture system will be robust and easy to extend.

**Next Step**: Begin with Phase 1 environment stabilization to establish a solid foundation for all subsequent work.