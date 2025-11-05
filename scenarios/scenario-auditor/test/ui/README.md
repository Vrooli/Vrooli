# UI Tests

## Overview
UI functionality is validated through:
- **Build verification** - Ensures TypeScript compiles without errors
- **Health checks** - Validates API connectivity and service status
- **Integration tests** - Tests UI-API communication
- **Manual testing** - Browser-based validation of user workflows

## Current Test Coverage

### Automated
- ✅ TypeScript compilation (`npm run build`)
- ✅ Health endpoint verification (`/health`)
- ✅ API connectivity checks
- ✅ Vite dev server startup

### Manual Validation Required
- UI component rendering
- Dashboard data display
- Rules manager functionality
- User interaction flows

## Running UI Tests

```bash
# Part of integration test suite
cd test && ./phases/test-integration.sh

# Manual UI build check
cd ui && npm run build

# Start UI for manual testing
vrooli scenario start scenario-auditor
# Then visit: http://localhost:36224
```

## UI Architecture

- **Framework**: React + TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **API Client**: Fetch API with health checks
- **Health Monitoring**: `/health` endpoint with API connectivity validation

## Adding Automated UI Tests

Future automated UI testing can use:
- **Browser Automation Studio** (`resource-browserless`) for screenshot testing
- **React Testing Library** for component tests
- **Playwright/Cypress** for end-to-end testing

See `docs/testing/architecture/PHASED_TESTING.md` for guidelines.

## Test Workflows

Current UI test coverage validates:
1. **Build Success**: TypeScript compiles cleanly
2. **Health Status**: UI reports healthy when API is reachable
3. **API Integration**: Proxy configuration and API calls work
4. **Error Handling**: Degraded status when API unavailable

See `test/phases/test-integration.sh` for implementation.
