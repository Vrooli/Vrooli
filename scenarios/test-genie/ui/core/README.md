# Test Genie Core Data Layer

This directory contains the core data layer modules that handle all data operations, state management, and communication for the Test Genie application.

## 📁 Module Overview

### `EventBus.js` (396 lines)
**Purpose**: Pub/sub event system for decoupled communication

**Key Features**:
- Event subscription/unsubscription
- Priority-based event handlers
- One-time event listeners
- Event propagation control
- Event history tracking
- Debug mode with detailed logging

**Usage**:
```javascript
import { eventBus, EVENT_TYPES } from './core/EventBus.js';

// Subscribe to events
const unsubscribe = eventBus.on(EVENT_TYPES.DATA_LOADED, (event) => {
    console.log('Data loaded:', event.data);
});

// Emit events
eventBus.emit(EVENT_TYPES.DATA_LOADED, { collection: 'suites', data: [...] });

// Unsubscribe
unsubscribe();

// One-time subscription
eventBus.once(EVENT_TYPES.SUITE_EXECUTED, (event) => {
    console.log('Suite executed once');
});
```

**Event Types**: 40+ predefined event types including:
- Data events: `DATA_LOADED`, `DATA_UPDATED`, `DATA_ERROR`
- Page events: `PAGE_CHANGED`, `PAGE_LOADED`
- Suite events: `SUITE_CREATED`, `SUITE_EXECUTED`, `SUITE_DELETED`
- Execution events: `EXECUTION_STARTED`, `EXECUTION_COMPLETED`
- WebSocket events: `WS_CONNECTED`, `WS_DISCONNECTED`, `WS_MESSAGE`
- System events: `HEALTH_CHANGED`, `ERROR_OCCURRED`

---

### `StateManager.js` (561 lines)
**Purpose**: Centralized reactive state management

**Key Features**:
- Nested path access (dot notation)
- State watchers for reactive updates
- Computed/derived state caching
- State history tracking
- Batch updates
- Convenience methods for common operations

**Usage**:
```javascript
import { stateManager } from './core/StateManager.js';

// Get state
const suites = stateManager.get('data.suites');
const activePage = stateManager.get('activePage');

// Set state
stateManager.set('activePage', 'dashboard');
stateManager.set('data.suites', [...]);

// Update (merge) state
stateManager.update('ui.loading', { suites: true });

// Watch for changes
const unwatch = stateManager.watch('data.suites', (newValue, oldValue) => {
    console.log('Suites changed:', newValue);
});

// Convenience methods
stateManager.setActivePage('dashboard');
stateManager.setData('suites', [...]);
stateManager.toggleSuiteSelection('suite-123');
stateManager.setLoading('suites', true);

// Batch updates (single event emission)
stateManager.batch(() => {
    stateManager.set('data.suites', [...]);
    stateManager.set('ui.loading.suites', false);
    stateManager.set('filters.suites.search', '');
});
```

**State Structure**:
```javascript
{
    activePage: 'dashboard',
    data: {
        suites: [],
        executions: [],
        metrics: {},
        coverage: [],
        scenarios: [],
        vaults: [],
        reports: { overview, trends, insights }
    },
    filters: {
        suites: { search: '', status: 'all' },
        executions: { search: '', status: 'all' }
    },
    selections: {
        selectedSuiteIds: Set(),
        selectedExecutionIds: Set()
    },
    ui: {
        activeSuiteDetailId: null,
        generateDialogOpen: false,
        sidebarOpen: false,
        loading: { suites, executions, coverage, reports, vaults }
    },
    system: {
        healthy: true,
        healthData: null,
        apiBaseUrl: '/api/v1'
    },
    settings: {
        coverageTarget: 80,
        phases: Set([...])
    }
}
```

---

### `ApiClient.js` (540 lines)
**Purpose**: Centralized HTTP API communication

**Key Features**:
- Request/response interceptors
- Automatic retry with exponential backoff
- Request timeout handling
- In-flight request tracking & cancellation
- Error handling & event emission
- Debug mode with request logging

**Usage**:
```javascript
import { apiClient } from './core/ApiClient.js';

// Test Suite operations
const suites = await apiClient.getTestSuites();
const suite = await apiClient.getTestSuite('suite-123');
const result = await apiClient.generateTestSuite({
    scenario_name: 'my-app',
    test_types: ['unit', 'integration'],
    coverage_target: 80
});

// Execute suite
await apiClient.executeTestSuite('suite-123', {
    timeout: 300
});

// Execution operations
const executions = await apiClient.getTestExecutions({ limit: 10 });
const execution = await apiClient.getTestExecution('exec-456');
await apiClient.deleteTestExecution('exec-456');

// Coverage operations
const coverages = await apiClient.getCoverageSummaries();
const analysis = await apiClient.getCoverageAnalysis('my-app');

// Vault operations
const vaults = await apiClient.getTestVaults();
await apiClient.createTestVault({
    vault_name: 'Full Test Vault',
    scenario_name: 'my-app',
    phases: ['setup', 'develop', 'test']
});

// Reports
const reports = await apiClient.getAllReports(30); // 30 days window

// System
const health = await apiClient.checkHealth();

// Interceptors
apiClient.addRequestInterceptor(async (url, options) => {
    options.headers['X-Custom-Header'] = 'value';
    return options;
});

apiClient.addResponseInterceptor(async (data, response) => {
    console.log('Response received:', data);
    return data;
});
```

**API Methods**:
- **System**: `checkHealth()`, `getSystemMetrics()`
- **Suites**: `getTestSuites()`, `getTestSuite()`, `generateTestSuite()`, `executeTestSuite()`
- **Executions**: `getTestExecutions()`, `getTestExecution()`, `deleteTestExecution()`, `clearAllExecutions()`
- **Coverage**: `getCoverageSummaries()`, `getCoverageAnalysis()`, `generateCoverageAnalysis()`
- **Vaults**: `getTestVaults()`, `getTestVault()`, `createTestVault()`, `getVaultExecutions()`
- **Scenarios**: `getScenarios()`
- **Reports**: `getReportsOverview()`, `getReportsTrends()`, `getReportsInsights()`, `getAllReports()`

---

### `WebSocketClient.js` (497 lines)
**Purpose**: Real-time bidirectional communication

**Key Features**:
- Automatic reconnection with exponential backoff
- Message queuing when disconnected
- Heartbeat/ping mechanism
- Topic subscription management
- Connection metrics tracking
- Debug mode with message logging

**Usage**:
```javascript
import { wsClient, WS_MESSAGE_TYPES } from './core/WebSocketClient.js';

// Connect (auto-detects URL)
await wsClient.connect();

// Or specify URL
await wsClient.connect('wss://example.com/ws');

// Subscribe to topics
wsClient.subscribe(['executions', 'test_completion', 'system_status']);

// Send messages
wsClient.send({
    type: 'custom_action',
    payload: { data: 'value' }
});

// Listen for events via EventBus
eventBus.on(EVENT_TYPES.EXECUTION_UPDATED, (event) => {
    console.log('Execution updated:', event.data);
});

eventBus.on(EVENT_TYPES.WS_CONNECTED, () => {
    console.log('WebSocket connected');
});

eventBus.on(EVENT_TYPES.WS_DISCONNECTED, () => {
    console.log('WebSocket disconnected');
});

// Check connection status
if (wsClient.isReady()) {
    console.log('WebSocket is connected and ready');
}

// Get metrics
const metrics = wsClient.getMetrics();
console.log('Messages received:', metrics.messagesReceived);
console.log('Reconnections:', metrics.reconnections);

// Disconnect
wsClient.disconnect();
```

**WebSocket Message Types**:
- `SUBSCRIBE` / `UNSUBSCRIBE`: Topic subscription
- `EXECUTION_UPDATE`: Real-time execution updates
- `TEST_COMPLETION`: Test completion notifications
- `SYSTEM_STATUS`: System health updates
- `VAULT_UPDATE`: Vault execution updates
- `COVERAGE_UPDATE`: Coverage analysis updates

---

## 🔄 Integration Pattern

All modules work together seamlessly through the EventBus:

```javascript
import core from './core/index.js';
const { eventBus, stateManager, apiClient, wsClient } = core;

// 1. ApiClient fetches data and emits events
const suites = await apiClient.getTestSuites();
// Automatically emits: DATA_LOADED event

// 2. StateManager listens and updates state
eventBus.on(EVENT_TYPES.DATA_LOADED, (event) => {
    if (event.data.collection === 'suites') {
        stateManager.setData('suites', event.data.data);
    }
});

// 3. StateManager emits when state changes
stateManager.set('data.suites', suites);
// Automatically emits: DATA_UPDATED event

// 4. UI components listen and re-render
eventBus.on(EVENT_TYPES.DATA_UPDATED, (event) => {
    if (event.data.path.startsWith('data.suites')) {
        renderSuitesList();
    }
});

// 5. WebSocketClient broadcasts real-time updates
wsClient.connect();
// Automatically emits: WS_CONNECTED event

// 6. Real-time updates flow through EventBus
eventBus.on(EVENT_TYPES.EXECUTION_UPDATED, (event) => {
    // Update UI in real-time
    updateExecutionStatus(event.data);
});
```

---

## 🧪 Testing

Open `test.html` in a browser to run the comprehensive test suite:

```bash
# Start local server
cd scenarios/test-genie/ui
python3 -m http.server 8080

# Navigate to
open http://localhost:8080/core/test.html
```

**Test Coverage**:
- ✓ EventBus subscription/emission
- ✓ EventBus priority handling
- ✓ StateManager get/set operations
- ✓ StateManager watchers
- ✓ StateManager batch updates
- ✓ ApiClient configuration
- ✓ ApiClient interceptors
- ✓ WebSocketClient initialization
- ✓ Integration between modules

---

## 📊 Architecture Benefits

### Before (Monolithic app.js)
- ❌ API calls scattered throughout code
- ❌ State accessed directly via `this.currentData`
- ❌ Direct method calls (tight coupling)
- ❌ No real-time update system
- ❌ Difficult to test

### After (Modular Core)
- ✅ All API calls centralized in ApiClient
- ✅ State managed by StateManager
- ✅ Components decoupled via EventBus
- ✅ Real-time updates via WebSocketClient
- ✅ Each module independently testable

---

## 🔧 Advanced Usage

### Custom Event Types

```javascript
// Define custom events
const CUSTOM_EVENTS = {
    EXPORT_STARTED: 'custom:export:started',
    EXPORT_COMPLETED: 'custom:export:completed'
};

// Use them
eventBus.emit(CUSTOM_EVENTS.EXPORT_STARTED, { format: 'csv' });
```

### State Computed Values

```javascript
// Define computed property
const totalTests = stateManager.getComputed('totalTests', (state) => {
    return state.data.suites.reduce((sum, suite) => {
        return sum + (suite.test_count || 0);
    }, 0);
});

// Invalidate cache when data changes
eventBus.on(EVENT_TYPES.DATA_UPDATED, (event) => {
    if (event.data.path === 'data.suites') {
        stateManager.invalidateComputed('totalTests');
    }
});
```

### Request Cancellation

```javascript
// Cancel all pending requests (e.g., on page navigation)
apiClient.cancelAllRequests();
```

### WebSocket Reconnection Control

```javascript
// Disable automatic reconnection
wsClient.shouldReconnect = false;

// Manually trigger reconnection
await wsClient.connect();
```

---

## 🎯 Performance Considerations

### EventBus
- Events are synchronous but fast
- Use priority for critical handlers
- Consider `stopPropagation()` for expensive operations
- Event history limited to last 100 events

### StateManager
- Watchers have minimal overhead
- Batch updates for multiple changes
- Computed values are cached until invalidated
- State history limited to last 50 changes

### ApiClient
- Automatic retry reduces manual error handling
- Request cancellation prevents memory leaks
- Interceptors run for every request (keep them fast)

### WebSocketClient
- Message queue prevents data loss during disconnects
- Exponential backoff prevents server hammering
- Heartbeat keeps connection alive

---

## 📝 Migration from app.js

| Old Pattern | New Pattern |
|------------|-------------|
| `this.currentData.suites` | `stateManager.get('data.suites')` |
| `this.fetchWithErrorHandling(...)` | `apiClient.get(...)` |
| `this.wsConnection.send(...)` | `wsClient.send(...)` |
| Direct method calls | `eventBus.emit(EVENT_TYPES....)` |
| Manual state updates | `stateManager.set(...)` |

---

## 📚 Best Practices

1. **Always use EventBus for cross-component communication** - Never directly call methods from other modules
2. **Use StateManager as single source of truth** - Don't maintain separate state in components
3. **Let ApiClient handle errors** - It automatically retries and emits error events
4. **Subscribe to events for real-time updates** - Don't poll when you can listen
5. **Use batch updates** - When changing multiple state values at once
6. **Clean up subscriptions** - Always call unsubscribe/unwatch when done
7. **Enable debug mode during development** - `eventBus.setDebug(true)`

---

## 🚀 Next Steps

Phase 3 will build on this foundation:
- `managers/NavigationManager.js` - Uses EventBus & StateManager
- `managers/DialogManager.js` - Uses EventBus & StateManager
- Page-specific modules - Use ApiClient for data, StateManager for state, EventBus for coordination

---

**Created**: 2025-10-02
**Phase**: 2 of 7 (Data Layer Complete)
**Total Lines**: 2,018 lines across 4 core modules
