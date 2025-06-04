# Vrooli Shared

Shared TypeScript types, utilities, and configurations used across Vrooli packages.

## Overview

This package contains common code and utilities shared between different packages in the Vrooli monorepo. It includes TypeScript types, shared constants, utility functions, and common configurations to ensure consistency across the codebase.

## Directory Structure

```
shared/
├── src/
│   ├── __test/        # Tests setup
│   ├── __tools/        # Tools run separately of development/production
│   ├── ai/        # AI-related configuration, services, and utils
│   ├── api/        # API information
│   ├── consts/        # Constants
│   ├── forms/        # Form utils
│   ├── id/        # ID generation
│   ├── run/        # Run utils and state machine
│   ├── shape/        # Shape to convert between form and API types, and generic shaping utils
    ├── translations/        # i18n translations
    ├── utils/        # Generic utility functions
    └── validation/   # Form and API validation
```

## Usage

### Installation

```bash
pnpm install
```

### Importing

```typescript
import { UserType, TaskType, formatDate, validateEmail } from '@local/shared';
```

## Development

### Building

```bash
pnpm build
```

### Testing

```bash
pnpm test
```

## Contributing

Please refer to the main project's [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## Documentation

- [Architecture Overview](../../ARCHITECTURE.md)
- [API Documentation](../docs/api/README.md)

## Best Practices

- Keep types simple and focused
- Document all public types and utilities
- Write comprehensive tests
- Maintain backward compatibility
- Follow consistent naming conventions 