# Test Playbooks

This directory contains automated workflow tests for the visited-tracker scenario, organized by capability and user journey.

## Structure

```
playbooks/
├── capabilities/           # Feature-specific tests organized by operational target
│   ├── 01-campaign-tracking/
│   │   ├── api/           # API integration tests for campaign tracking
│   │   └── cli/           # CLI workflow tests
│   └── 02-web-interface/
│       ├── api/           # API integration tests for web features
│       └── ui/            # UI workflow tests using browser-automation-studio
├── journeys/              # End-to-end user journey tests
│   └── 01-new-user/       # First-time user onboarding and basic workflows
└── registry.json          # Automated registry of all playbooks
```

## Capabilities

### 01-campaign-tracking (P0)
Tests for core campaign management, file tracking, staleness scoring, and CLI operations.

**Surfaces**:
- `api/`: HTTP API endpoint tests
- `cli/`: CLI command workflow tests

**Key Workflows**:
- Campaign creation and management
- File visit recording
- Staleness score calculation
- Prioritization queries (least-visited, most-stale)

### 02-web-interface (P1)
Tests for web interface functionality, visual components, and user interactions.

**Surfaces**:
- `api/`: HTTP API tests for UI-specific endpoints
- `ui/`: Browser-based UI workflow tests

**Key Workflows**:
- Campaign dashboard navigation
- File list viewing and sorting
- Manual visit recording
- Export/import operations

## Journeys

### 01-new-user
Complete end-to-end workflows for first-time users getting started with visited-tracker.

**Scenarios**:
- Create first campaign
- Track file visits
- View staleness metrics
- Use prioritization features

## Running Tests

### All playbooks
```bash
make test
```

### Specific capability
```bash
# Run P0 campaign tracking tests
vrooli scenario test visited-tracker --filter="01-campaign-tracking"

# Run UI tests only
vrooli scenario test visited-tracker --filter="ui"
```

### Individual workflow
```bash
# Run specific workflow via browser-automation-studio
browser-automation-studio execute bas/cases/01-campaign-tracking/api/campaign-lifecycle.json
```

## Test Data

Test fixtures and sample data are located in `test/fixtures/`.

## Selectors

UI test selectors are centralized in `ui/src/consts/selectors.ts` for maintainability.
