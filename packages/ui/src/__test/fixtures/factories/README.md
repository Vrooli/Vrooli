# Project Fixture Factory

This directory contains fixture factories for UI testing, following the established pattern demonstrated by `userFixtureFactory.ts`.

## Recently Added

### ProjectFixtureFactory
- **File**: `projectFixtureFactory.ts`
- **Purpose**: Provides comprehensive fixtures for Project objects (which are Resources with resourceType="Project")
- **Features**:
  - Zero `any` types with proper TypeScript interfaces
  - UI-specific form data type with fields like `handle`, `name`, `description`, etc.
  - Complete CRUD test support with round-trip testing
  - MSW mock handlers for API integration
  - UI state management fixtures
  - Multiple scenarios: minimal, complete, invalid, privateProject, teamProject, completedProject

## Usage

```typescript
import { ProjectFixtureFactory } from './projectFixtureFactory.js';

// Create factory instance
const projectFactory = new ProjectFixtureFactory(apiClient, dbVerifier);

// Create form data
const formData = projectFactory.createFormData('complete');

// Test full round-trip
const result = await projectFactory.testRoundTrip(formData);

// Create mock responses
const mockProject = projectFactory.createMockResponse();

// Set up MSW handlers
projectFactory.setupMSW('success');
```

## Implementation Notes

- Follows the exact pattern of `UserFixtureFactory`
- Projects are implemented as Resources with specific resourceType
- Includes proper version and translation handling
- Supports team ownership and privacy settings
- Handles project completion states and configuration