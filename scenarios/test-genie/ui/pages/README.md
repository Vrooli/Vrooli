# Test Genie Page Modules

This directory contains page-specific logic modules that handle data loading, user interactions, and rendering for each major page in the Test Genie application.

## 📁 Module Overview

### `DashboardPage.js` (476 lines) ✅ COMPLETE
**Purpose**: Dashboard view with metrics and recent executions

**Key Features**:
- Load system metrics, suites, and executions in parallel
- Calculate and display header stats (active suites, running tests, avg coverage)
- Calculate and display dashboard metrics (total suites, tests generated, failed tests)
- Load and render recent executions table
- Real-time updates via EventBus (EXECUTION_UPDATED, EXECUTION_COMPLETED)
- Lookup suite names for executions
- View execution details

**Usage**:
```javascript
import { dashboardPage } from './pages/DashboardPage.js';

// Load dashboard (triggered automatically via PAGE_LOAD_REQUESTED event)
await dashboardPage.load();

// Refresh dashboard
await dashboardPage.refresh();

// View execution
dashboardPage.viewExecution('exec-123');
```

**Events Emitted**:
- `PAGE_LOADED`: When dashboard data is loaded
- `EXECUTION_VIEW_REQUESTED`: When user clicks to view execution

**Events Listened To**:
- `PAGE_LOAD_REQUESTED`: Triggers dashboard load
- `DATA_LOADED`: Recalculates metrics when data changes
- `EXECUTION_UPDATED`: Refreshes recent executions
- `EXECUTION_COMPLETED`: Refreshes recent executions

---

### `SuitesPage.js` (Planned: ~600 lines)
**Purpose**: Test suites management view

**Key Responsibilities**:
- Load test suites and scenarios from API
- Group suites by scenario
- Calculate suite statistics (test count, coverage, status)
- Render suites table with scenario grouping
- Handle suite filtering (search, status)
- Handle suite selection (individual, select all)
- Bulk operations (run selected suites)
- Open generate dialog for scenarios
- View suite details
- Execute individual suites

**Complexity Notes**:
- Most complex page module
- 200+ lines just for suite loading logic
- Complex scenario grouping and aggregation
- Multiple data transformations
- Filtering and sorting logic

**Key Methods** (from app.js):
```javascript
async loadTestSuites()           // ~200 lines - Load and group suites
applySuiteFilters(rows)          // Filter suites by search/status
renderSuitesTableWithCurrentData() // Render suites table
handleGenerateSubmit()           // Generate new test suite
viewSuite(suiteId)              // View suite details
async runSelectedSuites()        // Execute multiple suites
```

---

### `ExecutionsPage.js` (Planned: ~500 lines)
**Purpose**: Test executions history view

**Key Responsibilities**:
- Load test executions from API
- Render executions table with selection
- Handle execution filtering (search, status)
- Handle execution selection (individual, select all)
- View execution details
- Delete individual executions
- Bulk delete selected executions
- Clear all executions

**Key Methods** (from app.js):
```javascript
async loadExecutions()            // Load executions from API
applyExecutionFilters(rows)      // Filter executions
renderExecutionsTable(exec, opts) // Render table HTML
viewExecution(executionId)       // View execution details
deleteExecution(executionId)     // Delete single execution
async deleteSelectedExecutions() // Bulk delete
async clearAllExecutions()       // Clear all executions
```

---

### `CoveragePage.js` (Planned: ~400 lines)
**Purpose**: Code coverage analysis view

**Key Responsibilities**:
- Load coverage summaries from API
- Render coverage summaries table
- View detailed coverage analysis
- Generate coverage analysis for scenarios
- Handle coverage detail dialog
- Display file-level coverage data

**Key Methods** (from app.js):
```javascript
async loadCoverageSummaries()      // Load coverage summaries
loadCoverageAnalysis(scenarioName) // Load detailed analysis
generateCoverageAnalysis(config)   // Generate new analysis
renderCoverageTable(summaries)     // Render table
openCoverageDetail(scenarioName)   // Open detail dialog
```

---

### `VaultPage.js` (Planned: ~700 lines)
**Purpose**: Test vault management view

**Key Responsibilities**:
- Setup vault page initialization
- Load vault list from API
- Load vault scenario options
- Create new test vaults
- View vault details and execution history
- Manage vault phases
- Handle vault phase drafts
- Display vault activity timeline

**Complexity Notes**:
- Second most complex page module
- Vault phase management is intricate
- Phase draft system with local state
- Activity timeline rendering
- Multiple API endpoints

**Key Methods** (from app.js):
```javascript
setupVaultPage()                  // Initialize vault page
async loadVaultList()            // Load vaults
async loadVaultScenarioOptions() // Load scenario options
async handleVaultSubmit()        // Create new vault
viewVault(vaultId)              // View vault details
loadVaultExecutions(vaultId)    // Load vault execution history
setupVaultPhaseSelector()       // Setup phase selection UI
```

---

### `ReportsPage.js` (Planned: ~600 lines)
**Purpose**: Analytics and reports view

**Key Responsibilities**:
- Load reports overview, trends, and insights
- Render overview metrics
- Render trends chart (Chart.js integration)
- Render insights list
- Handle time window selection (7, 30, 90 days)
- Refresh reports
- Chart data transformation

**Key Methods** (from app.js):
```javascript
async loadReports()            // Load all report data
renderReportsOverview(data)    // Render overview section
renderReportsTrends(data)      // Render trends chart
renderReportsInsights(data)    // Render insights
updateReportsTimeWindow(days)  // Change time window
```

---

### `SettingsPage.js` (Planned: ~400 lines)
**Purpose**: Application settings view

**Key Responsibilities**:
- Load settings from localStorage
- Render settings UI
- Handle coverage target slider
- Handle phase checkboxes
- Save settings to localStorage
- Apply default settings to other dialogs
- Settings validation

**Key Methods** (from app.js):
```javascript
initializeDefaultSettings()        // Load from localStorage
loadStoredDefaultSettings()       // Parse stored settings
saveDefaultSettings()             // Save to localStorage
applyDefaultSettingsToUI()        // Update UI controls
applyDefaultSettingsToDialogs()   // Apply to generate dialog
updateSettingsCoverageDisplay()   // Update coverage slider display
```

---

## 🔄 Common Integration Pattern

All page modules follow this pattern:

```javascript
import { eventBus, EVENT_TYPES } from '../core/EventBus.js';
import { stateManager } from '../core/StateManager.js';
import { apiClient } from '../core/ApiClient.js';
import { notificationManager } from '../managers/NotificationManager.js';

export class PageModule {
    constructor() {
        this.eventBus = eventBus;
        this.stateManager = stateManager;
        this.apiClient = apiClient;
        this.notificationManager = notificationManager;

        this.initialize();
    }

    initialize() {
        this.setupDOMReferences();
        this.setupEventListeners();
    }

    setupEventListeners() {
        // Listen for page load requests
        this.eventBus.on(EVENT_TYPES.PAGE_LOAD_REQUESTED, (event) => {
            if (event.data.page === 'my-page') {
                this.load();
            }
        });
    }

    async load() {
        try {
            this.stateManager.setLoading('myPage', true);

            // Load data via apiClient
            const data = await this.apiClient.getData();

            // Update state
            this.stateManager.setData('myData', data);

            // Render
            this.render();

            // Emit success
            this.eventBus.emit(EVENT_TYPES.PAGE_LOADED, { page: 'my-page' });

        } catch (error) {
            console.error('Load failed:', error);
            this.notificationManager.showError('Failed to load data');
        } finally {
            this.stateManager.setLoading('myPage', false);
        }
    }

    render() {
        // Render page content
    }
}

export const pageModule = new PageModule();
export default pageModule;
```

---

## 📊 Architecture Benefits

### Before (Monolithic app.js)
- ❌ All page logic mixed together (5,701 lines)
- ❌ Page methods scattered throughout class
- ❌ Difficult to find page-specific code
- ❌ Hard to test individual pages
- ❌ No clear separation between pages

### After (Modular Pages)
- ✅ Each page in its own module (400-700 lines each)
- ✅ Clear page boundaries and responsibilities
- ✅ Easy to find and modify page logic
- ✅ Each page independently testable
- ✅ Can lazy-load pages on demand

---

## 🚀 Implementation Status

| Page | Status | Lines | Complexity | Priority |
|------|--------|-------|------------|----------|
| DashboardPage | ✅ Complete | 490 | Medium | High |
| SuitesPage | ✅ Complete | 771 | Very High | High |
| ExecutionsPage | ✅ Complete | 502 | Medium | High |
| CoveragePage | ✅ Complete | 398 | Low | Medium |
| VaultPage | ✅ Complete | 801 | High | Medium |
| ReportsPage | ✅ Complete | 649 | Medium | Low |
| SettingsPage | ✅ Complete | 381 | Low | Low |

**Total Completed**: 3,992 lines across 7 page modules
**Phase 4 Complete**: 2025-10-02

---

## ✅ Phase 4 Complete (2025-10-02)

All 7 page modules have been successfully implemented:
- **DashboardPage.js** (490 lines) - Dashboard with metrics and recent executions
- **SuitesPage.js** (771 lines) - Test suites management (most complex)
- **ExecutionsPage.js** (502 lines) - Execution history and management
- **CoveragePage.js** (398 lines) - Code coverage analysis
- **VaultPage.js** (801 lines) - Vault management with phase configuration
- **ReportsPage.js** (649 lines) - Analytics with custom canvas charts
- **SettingsPage.js** (381 lines) - Application settings with localStorage

**Total**: 3,992 lines of modular, event-driven page code

## 🎯 Next Steps - Phase 5: Rendering Extraction

- Extract rendering logic to `renderers/` modules
- TableRenderer, ChartRenderer, CardRenderer
- Reduce duplication across pages

---

## 📝 Implementation Guidelines

When implementing remaining pages:

1. **Extract Methods from app.js**:
   - Find all methods related to the page
   - Group by functionality (load, render, actions, helpers)
   - Preserve logic but update to use new modules

2. **Use Core Modules**:
   - `apiClient` for all API calls
   - `stateManager` for all state access
   - `eventBus` for all events
   - Managers for UI operations

3. **Follow Event-Driven Pattern**:
   - Listen to `PAGE_LOAD_REQUESTED`
   - Emit `PAGE_LOADED` when complete
   - Listen to relevant data events

4. **Maintain State**:
   - Store data in `stateManager`
   - Use watchers for reactive updates
   - Emit events on state changes

5. **Handle Errors**:
   - Use try/catch blocks
   - Show notifications via `notificationManager`
   - Log errors for debugging

---

## 🔧 Testing Strategy

Each page module should be testable:

```javascript
// Mock dependencies
const mockEventBus = { on: jest.fn(), emit: jest.fn() };
const mockStateManager = { get: jest.fn(), set: jest.fn() };
const mockApiClient = { getTestSuites: jest.fn() };

// Create page instance
const page = new SuitesPage(mockEventBus, mockStateManager, mockApiClient);

// Test load method
test('load() should fetch and display suites', async () => {
    mockApiClient.getTestSuites.mockResolvedValue([...]);
    await page.load();
    expect(mockStateManager.setData).toHaveBeenCalledWith('suites', [...]);
});
```

---

**Created**: 2025-10-02
**Updated**: 2025-10-02
**Phase**: 4 of 7 (Page-Specific Logic - Complete)
**Status**: All 7 page modules complete (3,992 lines total)
