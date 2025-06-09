# 🔧 Vrooli Shared Package

> **Core foundation for the Vrooli platform** - Essential types, validation, execution framework, and utilities shared across all packages.

> 📖 **Quick Links**: [Architecture Overview](../../docs/architecture/execution/README.md) | [Implementation Mapping](../../docs/implementation-mapping.md) | [Validation Guide](#validation-system)

---

## 🎯 Purpose & Architecture Role

The `@vrooli/shared` package serves as the **foundational layer** for Vrooli's architecture, providing:

- **🏗️ Three-Tier Execution Types** - Complete type system for Coordination, Process, and Execution Intelligence
- **🔐 Validation Framework** - Comprehensive Yup-based validation for API and form data
- **🌊 Event System** - Type-safe event definitions for the execution architecture
- **⚙️ Utility Functions** - Core utilities for data transformation, caching, and scheduling
- **🌍 Internationalization** - Translation system and locale management

This package enables **type safety** and **consistency** across server, UI, and background job processing.

---

## 📦 Package Structure

### Core Modules

```
shared/src/
├── 🧠 execution/           # Three-tier execution architecture types
│   ├── types/             # Core execution interfaces
│   ├── events/            # Event bus type definitions
│   └── security/          # Security validation types
├── 🔐 validation/         # Yup validation framework
│   ├── models/           # Entity validation schemas
│   ├── utils/            # Validation builders and utilities
│   └── forms/            # Form-specific validation
├── 🔧 utils/             # Core utility functions
├── 🌍 translations/      # i18n system and locales
├── 📊 api/               # API type definitions
├── 🏃 run/               # Runtime execution utilities
└── 🔗 shape/             # Data transformation utilities
```

### Implementation Files

| Component | Location | Purpose |
|-----------|----------|---------|
| **Execution Types** | `src/execution/types/` | Three-tier architecture interfaces |
| **Event System** | `src/execution/events/` | Event bus definitions and validation |
| **Validation Core** | `src/validation/models/` | Entity validation schemas |
| **Validation Builders** | `src/validation/utils/builders/` | Yup schema composition utilities |
| **Runtime Context** | `src/run/` | Execution context and state management |
| **Data Shaping** | `src/shape/` | Transform data between API and UI formats |

> 📖 **Related Documentation**: 
> - [Three-Tier Architecture](../../docs/architecture/execution/tiers/)
> - [Event-Driven Architecture](../../docs/architecture/execution/event-driven/)
> - [Validation System Guide](#validation-system)

---

## 🧠 Execution Architecture Types

### Three-Tier Intelligence System

The shared package provides complete type definitions for Vrooli's three-tier architecture:

```typescript
// Tier 1: Coordination Intelligence
import { 
  SwarmState, 
  CoordinationEvent, 
  ResourceAllocation 
} from '@vrooli/shared';

// Tier 2: Process Intelligence
import { 
  RunState, 
  NavigatorType, 
  ProcessEvent 
} from '@vrooli/shared';

// Tier 3: Execution Intelligence
import { 
  ExecutionStrategy, 
  ToolOrchestration, 
  ExecutionEvent 
} from '@vrooli/shared';
```

### Key Type Categories

| Category | Description | Key Types |
|----------|-------------|-----------|
| **Core** | Base execution interfaces | `ExecutionContext`, `ResourceManager`, `StateTransition` |
| **Communication** | Inter-tier messaging | `TierMessage`, `CommandProtocol`, `ResponseChannel` |
| **Events** | Event bus definitions | `ExecutionEvent`, `CoordinationEvent`, `ProcessEvent` |
| **Security** | Validation and permissions | `SecurityContext`, `ValidationRule`, `PermissionGate` |
| **Strategies** | Execution approaches | `ConversationalStrategy`, `DeterministicStrategy`, `ReasoningStrategy` |

> 📖 **Implementation Details**: See [Execution Types Reference](../../docs/architecture/execution/types/README.md)

---

## 🔐 Validation System

### Comprehensive Entity Validation

The validation system provides **dual-purpose schemas** for both API validation and form handling:

```typescript
import { userValidation, routineValidation } from '@vrooli/shared';

// Server-side API validation
const validatedData = await userValidation.create().validate(userData);

// Client-side form validation with Formik
const UserForm = () => (
  <Formik
    validationSchema={userValidation.create()}
    // ... form logic
  />
);
```

### Validation Categories

| Category | Entities | Purpose |
|----------|----------|---------|
| **Core Entities** | User, Team, Project | Primary platform objects |
| **Communication** | Chat, Message, Comment | Social and collaboration features |
| **Workflows** | Routine, Run, Step | Execution and process management |
| **Resources** | Resource, Schedule, Report | Content and scheduling |
| **Metadata** | Tag, Bookmark, Notification | Organization and tracking |

### Advanced Validation Features

```typescript
// Conditional validation based on context
const routineSchema = routineValidation.create({
  format: 'api', // or 'form'
  isOwner: true,  // dynamic permissions
  hasCredits: true // resource constraints
});

// Relationship validation
const teamMemberSchema = memberValidation.create()
  .withTeamContext(teamId)
  .withRoleValidation();
```

> 📖 **Validation Guide**: See [Writing Validation Schemas](../../docs/testing/writing-tests.md#validation-testing)

---

## 🌊 Event System Integration

### Type-Safe Event Definitions

```typescript
import { 
  ExecutionEvent, 
  CoordinationEvent, 
  EventPayload 
} from '@vrooli/shared';

// Emit execution events
const event: ExecutionEvent = {
  type: 'STEP_COMPLETED',
  payload: {
    stepId: 'step_123',
    status: 'success',
    outputs: { result: 'processed' }
  },
  metadata: {
    timestamp: Date.now(),
    tier: 3,
    routineId: 'routine_456'
  }
};
```

### Event Categories

| Event Type | Trigger | Tier | Purpose |
|------------|---------|------|---------|
| **CoordinationEvent** | Swarm state changes | 1 | Resource allocation, goal updates |
| **ProcessEvent** | Routine transitions | 2 | Step progression, error handling |
| **ExecutionEvent** | Tool operations | 3 | Task completion, strategy updates |
| **SecurityEvent** | Validation checks | All | Threat detection, compliance |

> 📖 **Event Architecture**: See [Event Catalog](../../docs/architecture/execution/event-driven/event-catalog.md)

---

## 🔧 Utility Functions

### Core Utilities

```typescript
import { 
  formatDate, 
  validateEmail, 
  orDefault, 
  isEqual,
  LRUCache 
} from '@vrooli/shared';

// Date handling with timezone support
const formatted = formatDate(new Date(), 'America/New_York');

// Safe value handling
const value = orDefault(maybeUndefined, 'default');

// Deep equality checks
const hasChanged = !isEqual(prevData, newData);

// Performance caching
const cache = new LRUCache<string, UserData>(100);
```

### Specialized Utilities

| Category | Functions | Purpose |
|----------|-----------|---------|
| **Arrays** | `unique`, `chunk`, `groupBy` | Collection manipulation |
| **Objects** | `pick`, `omit`, `merge` | Object transformation |
| **Strings** | `toCamelCase`, `toSnakeCase` | Casing conversions |
| **Validation** | `isEmail`, `isUrl`, `isHandle` | Input validation |
| **Scheduling** | `parseSchedule`, `nextOccurrence` | Cron and time handling |

---

## 🌍 Internationalization System

### Multi-Language Support

```typescript
import { t, getCurrentLanguage, changeLanguage } from '@vrooli/shared';

// Translate with interpolation
const message = t('common:welcome', { name: 'John' });

// Language management
const currentLang = getCurrentLanguage(); // 'en', 'es', etc.
await changeLanguage('es');
```

### Translation Structure

```
translations/locales/
├── en/                 # English (default)
│   ├── common.json    # Shared UI text
│   ├── error.json     # Error messages
│   ├── validate.json  # Validation messages
│   └── service.json   # Service-specific text
├── es/                # Spanish
└── fr/                # French
```

---

## 🚀 Development & Usage

### Installation & Setup

```bash
# Install dependencies
pnpm install

# Build the package
pnpm build

# Run tests with coverage
pnpm test-coverage

# Type checking
pnpm type-check
```

### Import Patterns

```typescript
// Specific imports (recommended)
import { userValidation, ExecutionEvent } from '@vrooli/shared';

// Category imports
import { validation, execution, utils } from '@vrooli/shared';

// Full import (use sparingly)
import * as shared from '@vrooli/shared';
```

### TypeScript Configuration

The package uses **strict TypeScript** with:
- **ESM modules** with `.js` extensions in imports
- **Side-effect free** compilation for tree shaking
- **Type definitions** exported for IDE support

```typescript
// ✅ Correct import syntax
import { UserType } from "./types/user.js";

// ❌ Incorrect - missing .js extension
import { UserType } from "./types/user";
```

---

## 🧪 Testing & Quality

### Testing Framework

- **Vitest** for fast unit testing
- **100+ test files** with comprehensive coverage
- **Test fixtures** for realistic data scenarios
- **Validation testing** utilities for schema verification

```bash
# Run specific test categories
pnpm test -- validation
pnpm test -- utils
pnpm test -- execution

# Watch mode for development
pnpm test-watch

# Coverage report
pnpm coverage-report
```

### Quality Standards

- **>90% test coverage** requirement
- **ESLint** with TypeScript rules
- **Type-safe** exports and imports
- **Comprehensive fixtures** for testing

> 📖 **Testing Guide**: See [Writing Tests for Shared Package](../../docs/testing/writing-tests.md#shared-package-testing)

---

## 🔗 Package Integration

### Server Integration

```typescript
// packages/server/src/services/
import { 
  userValidation, 
  ExecutionEvent, 
  SwarmState 
} from '@vrooli/shared';

// API endpoint validation
app.post('/api/users', async (req, res) => {
  const validated = await userValidation.create().validate(req.body);
  // ... handle request
});
```

### UI Integration

```typescript
// packages/ui/src/components/
import { 
  userValidation, 
  formatDate, 
  t 
} from '@vrooli/shared';

// Form validation
const UserForm = () => (
  <Formik validationSchema={userValidation.create()}>
    {/* form fields */}
  </Formik>
);
```

### Background Jobs Integration

```typescript
// packages/jobs/src/
import { 
  ExecutionEvent, 
  ProcessEvent 
} from '@vrooli/shared';

// Event processing
export const processExecutionEvent = (event: ExecutionEvent) => {
  // handle event
};
```

---

## 📚 Related Documentation

### Architecture References
- **[Three-Tier Architecture](../../docs/architecture/execution/README.md)** - Complete architectural overview
- **[Event-Driven Architecture](../../docs/architecture/execution/event-driven/README.md)** - Event system details
- **[Security Architecture](../../docs/architecture/execution/security/README.md)** - Security validation framework

### Implementation Guides
- **[Implementation Mapping](../../docs/implementation-mapping.md)** - Map architecture to code
- **[Validation Guide](../../docs/testing/writing-tests.md#validation-testing)** - Writing validation schemas
- **[Type System Guide](../../docs/architecture/execution/types/README.md)** - Working with execution types

### Development Resources
- **[Development Workflows](../../docs/devops/development-workflows.md)** - Daily development patterns
- **[Testing Strategy](../../docs/testing/test-strategy.md)** - Testing approach and standards
- **[Contributing Guidelines](../../CONTRIBUTING.md)** - Code standards and practices

---

## 🎯 Key Features Summary

### **🏗️ Foundational Architecture**
- Complete type system for three-tier execution architecture
- Event-driven communication types and validation
- Shared context and state management interfaces

### **🔐 Robust Validation**
- Dual-purpose schemas for API and form validation
- 50+ entity validation schemas with comprehensive test coverage
- Advanced features like conditional validation and relationship checking

### **⚙️ Production-Ready Utilities**
- Performance-optimized utility functions with extensive test coverage
- Internationalization system with multi-language support
- Caching, scheduling, and data transformation utilities

### **🔬 Developer Experience**
- Type-safe imports with full TypeScript support
- Comprehensive test fixtures and validation utilities
- Clear separation between API and form validation contexts

---

*This package forms the foundation of Vrooli's type-safe, validated, and internationally accessible platform. Every improvement here benefits the entire ecosystem.*