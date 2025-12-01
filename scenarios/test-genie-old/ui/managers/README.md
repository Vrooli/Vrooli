# Test Genie UI Managers

This directory contains the UI management layer modules that handle user interface interactions, navigation, dialogs, selections, and notifications for the Test Genie application.

## 📁 Module Overview

### `NavigationManager.js` (273 lines)
**Purpose**: Page navigation and routing system

**Key Features**:
- Browser history management (back/forward)
- URL hash-based routing
- Page transition handling
- Keyboard shortcuts (Ctrl+1-7 for quick navigation)
- Mobile sidebar auto-close on navigation
- Page validation and fallbacks

**Usage**:
```javascript
import { navigationManager } from './managers/NavigationManager.js';

// Navigate to a page
navigationManager.navigateTo('dashboard');
navigationManager.navigateTo('suites');

// Get current page
const currentPage = navigationManager.getCurrentPage();

// Check if page is active
if (navigationManager.isPageActive('suites')) {
    console.log('On suites page');
}

// Reload current page data
navigationManager.reloadCurrentPage();

// Get valid pages list
const validPages = navigationManager.getValidPages();
```

**Events Emitted**:
- `PAGE_CHANGED`: When page changes (includes previous and current page)
- `PAGE_LOAD_REQUESTED`: Requests page data to be loaded
- `UI_SIDEBAR_CLOSE`: Requests sidebar to close (mobile)
- `UI_SUITE_DETAIL_CLOSE`: Requests suite detail panel to close
- `UI_EXECUTION_DETAIL_CLOSE`: Requests execution detail panel to close

**Keyboard Shortcuts**:
- `Ctrl/Cmd + 1`: Dashboard
- `Ctrl/Cmd + 2`: Suites
- `Ctrl/Cmd + 3`: Executions
- `Ctrl/Cmd + 4`: Coverage
- `Ctrl/Cmd + 5`: Vault
- `Ctrl/Cmd + 6`: Reports
- `Ctrl/Cmd + 7`: Settings

---

### `DialogManager.js` (555 lines)
**Purpose**: Dialog and overlay management

**Key Features**:
- Open/close dialogs with focus management
- Scroll locking when dialogs open
- Escape key to close (priority-based)
- Return focus to trigger element on close
- Overlay click-to-close
- Multiple dialog types supported

**Dialog Types**:
- `generate`: Test suite generation dialog
- `vault`: Test vault creation dialog
- `health`: System health status dialog
- `coverageDetail`: Coverage detail overlay
- `suiteDetail`: Suite detail panel
- `executionDetail`: Execution detail panel

**Usage**:
```javascript
import { dialogManager } from './managers/DialogManager.js';

// Open dialogs
dialogManager.openGenerateDialog(button, 'my-scenario', false);
dialogManager.openVaultDialog(button);
dialogManager.openHealthDialog();

// Close dialogs
dialogManager.closeGenerateDialog();
dialogManager.closeVaultDialog();
dialogManager.closeHealthDialog();

// Generic methods
dialogManager.openDialog('generate', button, { scenarioName: 'my-app', isBulkMode: true });
dialogManager.closeDialog('vault');

// Check dialog state
if (dialogManager.isDialogOpen('generate')) {
    console.log('Generate dialog is open');
}

if (dialogManager.isAnyDialogOpen()) {
    console.log('At least one dialog is open');
}

// Close all
dialogManager.closeAllDialogs();

// Listen via EventBus
eventBus.on(EVENT_TYPES.UI_DIALOG_OPENED, (event) => {
    console.log('Dialog opened:', event.data.dialogType);
});

eventBus.on(EVENT_TYPES.UI_DIALOG_CLOSED, (event) => {
    console.log('Dialog closed:', event.data.dialogType);
});
```

**Events Emitted**:
- `UI_DIALOG_OPENED`: When a dialog opens (includes dialogType and options)
- `UI_DIALOG_CLOSED`: When a dialog closes (includes dialogType)
- `HEALTH_CHECK_REQUESTED`: When health dialog opens (to refresh data)

**Events Listened To**:
- `UI_DIALOG_OPEN`: Request to open a dialog
- `UI_DIALOG_CLOSE`: Request to close a dialog
- `UI_SUITE_DETAIL_CLOSE`: Request to close suite detail
- `UI_EXECUTION_DETAIL_CLOSE`: Request to close execution detail

**Escape Key Priority**:
1. Suite detail panel
2. Execution detail panel
3. Coverage detail dialog
4. Generate dialog
5. Vault dialog
6. Health dialog

---

### `SelectionManager.js` (430 lines)
**Purpose**: Multi-select checkbox management

**Key Features**:
- Track selected suites and executions
- Toggle individual selections
- Select/deselect all functionality
- Sync checkbox states with internal state
- Update action button visibility based on selection
- Prune invalid selections when data changes
- Indeterminate checkbox state for partial selections

**Usage**:
```javascript
import { selectionManager } from './managers/SelectionManager.js';

// Toggle selections
selectionManager.toggleSuiteSelection('suite-123');
selectionManager.toggleExecutionSelection('exec-456');

// Select all
const allSuiteIds = suites.map(s => s.id);
selectionManager.selectAllSuites(allSuiteIds);

const allExecutionIds = executions.map(e => e.id);
selectionManager.selectAllExecutions(allExecutionIds);

// Clear selections
selectionManager.clearSuiteSelection();
selectionManager.clearExecutionSelection();
selectionManager.clearAllSelections();

// Get selected IDs
const selectedSuites = selectionManager.getSelectedSuiteIds();
const selectedExecutions = selectionManager.getSelectedExecutionIds();

// Check if item is selected
if (selectionManager.isSuiteSelected('suite-123')) {
    console.log('Suite is selected');
}

if (selectionManager.isExecutionSelected('exec-456')) {
    console.log('Execution is selected');
}

// Listen via EventBus
eventBus.on(EVENT_TYPES.SELECTION_CHANGED, (event) => {
    const { collection, selectedIds } = event.data;
    console.log(`${collection} selection changed:`, selectedIds);
});
```

**State Structure**:
```javascript
{
    selections: {
        selectedSuiteIds: Set(['suite-1', 'suite-2']),
        selectedExecutionIds: Set(['exec-1', 'exec-2'])
    }
}
```

**Events Emitted**:
- `SELECTION_CHANGED`: When selection changes (includes collection and selectedIds array)

**Events Listened To**:
- `DATA_LOADED`: Prunes invalid selections when data refreshes
- `SELECTION_CLEAR`: Request to clear selections

**Auto-Pruning**:
When data is reloaded, SelectionManager automatically removes selected IDs that no longer exist in the dataset. This prevents selecting "ghost" items that have been deleted.

---

### `NotificationManager.js` (319 lines)
**Purpose**: Toast notification system

**Key Features**:
- Display temporary toast notifications
- Multiple notification types (success, error, info, warning)
- Auto-dismiss after configurable duration
- Click to dismiss
- Stack multiple notifications with automatic positioning
- Maximum concurrent notifications limit
- Smooth animations (slide in from right)

**Usage**:
```javascript
import { notificationManager } from './managers/NotificationManager.js';

// Show notifications
notificationManager.showSuccess('Test suite executed successfully!');
notificationManager.showError('Failed to generate test suite');
notificationManager.showInfo('Test execution started');
notificationManager.showWarning('Coverage below target');

// Generic show with custom duration
notificationManager.show('Custom message', 'info', 10000); // 10 seconds

// Clear all notifications
notificationManager.clearAll();

// Configure durations
notificationManager.setDuration('success', 3000); // 3 seconds
notificationManager.setDuration('error', 8000); // 8 seconds

// Configure max concurrent notifications
notificationManager.setMaxNotifications(3);

// Trigger via EventBus
eventBus.emit(EVENT_TYPES.NOTIFICATION_SUCCESS, { message: 'Success!' });
eventBus.emit(EVENT_TYPES.NOTIFICATION_ERROR, { message: 'Error!' });
eventBus.emit(EVENT_TYPES.NOTIFICATION_INFO, { message: 'Info!' });
eventBus.emit(EVENT_TYPES.NOTIFICATION_SHOW, {
    message: 'Custom',
    type: 'warning',
    duration: 7000
});
```

**Default Durations**:
- Success: 5000ms (5 seconds)
- Error: 7000ms (7 seconds)
- Info: 5000ms (5 seconds)
- Warning: 6000ms (6 seconds)

**Events Listened To**:
- `NOTIFICATION_SHOW`: Display a notification
- `NOTIFICATION_SUCCESS`: Display success notification
- `NOTIFICATION_ERROR`: Display error notification
- `NOTIFICATION_INFO`: Display info notification

**Behavior**:
- Notifications stack vertically with 10px spacing
- Max 5 concurrent notifications by default
- Oldest notification removed when max reached
- Smooth slide-in animation from right
- Smooth slide-out animation when dismissed
- Click any notification to dismiss it immediately

---

## 🔄 Integration Pattern

All managers work together seamlessly through the EventBus and StateManager:

```javascript
import managers from './managers/index.js';
const { navigationManager, dialogManager, selectionManager, notificationManager } = managers;

// 1. User clicks navigation item
navigationManager.navigateTo('suites');

// 2. NavigationManager emits PAGE_CHANGED event
// 3. StateManager updates activePage
// 4. EventBus notifies all listeners

// 5. User opens generate dialog
dialogManager.openGenerateDialog(button, 'my-scenario');

// 6. DialogManager locks scroll and emits UI_DIALOG_OPENED
// 7. StateManager updates ui.generateDialogOpen = true

// 8. User selects suites
selectionManager.toggleSuiteSelection('suite-123');

// 9. SelectionManager updates state and emits SELECTION_CHANGED
// 10. UI updates "Run Selected" button visibility

// 11. Operation completes
notificationManager.showSuccess('Test suites executed!');

// 12. NotificationManager displays toast notification
```

---

## 🧪 Testing

Open `test.html` in a browser to run the test suite:

```bash
# Start local server
cd scenarios/test-genie/ui
python3 -m http.server 8080

# Navigate to
open http://localhost:8080/managers/test.html
```

**Test Coverage** (to be implemented):
- ✓ NavigationManager page transitions
- ✓ NavigationManager keyboard shortcuts
- ✓ DialogManager open/close
- ✓ DialogManager escape key handling
- ✓ SelectionManager toggle selections
- ✓ SelectionManager select/deselect all
- ✓ NotificationManager display notifications
- ✓ NotificationManager auto-dismiss

---

## 📊 Architecture Benefits

### Before (Monolithic app.js)
- ❌ Navigation logic scattered in init and event handlers
- ❌ Dialog methods mixed with business logic
- ❌ Selection state tracked in class properties
- ❌ Notifications created inline with business logic
- ❌ Difficult to test UI interactions

### After (Modular Managers)
- ✅ Navigation centralized in NavigationManager
- ✅ Dialog logic isolated in DialogManager
- ✅ Selections managed by SelectionManager
- ✅ Notifications handled by NotificationManager
- ✅ Each manager independently testable
- ✅ Clear separation of concerns
- ✅ Event-driven coordination

---

## 🔧 Advanced Usage

### Custom Navigation Handlers

```javascript
// Listen for page changes
eventBus.on(EVENT_TYPES.PAGE_CHANGED, (event) => {
    const { previous, current } = event.data;
    console.log(`Navigated from ${previous} to ${current}`);

    // Run custom logic
    if (current === 'dashboard') {
        loadDashboardCharts();
    }
});
```

### Dialog Coordination

```javascript
// Open dialog via event bus
eventBus.emit(EVENT_TYPES.UI_DIALOG_OPEN, {
    dialogType: 'generate',
    trigger: buttonElement,
    options: {
        scenarioName: 'my-app',
        isBulkMode: true
    }
});

// Close dialog via event bus
eventBus.emit(EVENT_TYPES.UI_DIALOG_CLOSE, {
    dialogType: 'generate'
});
```

### Selection Actions

```javascript
// Get selected items and perform action
const selectedSuiteIds = selectionManager.getSelectedSuiteIds();

if (selectedSuiteIds.length > 0) {
    await Promise.all(
        selectedSuiteIds.map(id => apiClient.executeTestSuite(id))
    );

    selectionManager.clearSuiteSelection();
    notificationManager.showSuccess(`Executed ${selectedSuiteIds.length} test suites`);
}
```

### Notification Chaining

```javascript
async function performOperation() {
    try {
        notificationManager.showInfo('Starting operation...');

        await longRunningTask();

        notificationManager.showSuccess('Operation completed successfully!');
    } catch (error) {
        notificationManager.showError(`Operation failed: ${error.message}`);
    }
}
```

---

## 🎯 Performance Considerations

### NavigationManager
- Minimal overhead - only updates active classes and URL
- No data loading (emits events for page modules to handle)
- Browser history integrated (no custom implementation)

### DialogManager
- Lazy initialization - only creates references when needed
- No polling - event-driven open/close
- Focus management prevents tab-trapping

### SelectionManager
- Uses Set for O(1) selection lookups
- Auto-prunes invalid selections to prevent memory leaks
- Updates UI only when selections change

### NotificationManager
- Limits concurrent notifications (prevents spam)
- Auto-cleanup after dismiss animation
- Smooth CSS animations (GPU-accelerated)

---

## 📝 Migration from app.js

| Old Pattern | New Pattern |
|------------|-------------|
| `this.navigateTo(page)` | `navigationManager.navigateTo(page)` |
| `this.openGenerateDialog(...)` | `dialogManager.openGenerateDialog(...)` |
| `this.selectedSuiteIds.add(id)` | `selectionManager.toggleSuiteSelection(id)` |
| `this.showSuccess(message)` | `notificationManager.showSuccess(message)` |
| `this.activePage` | `stateManager.get('activePage')` |

---

## 📚 Best Practices

1. **Always use managers for UI operations** - Don't directly manipulate DOM or state
2. **Listen to events for UI updates** - Don't poll or use intervals
3. **Let managers handle accessibility** - Focus management and ARIA handled automatically
4. **Use StateManager as single source of truth** - Don't maintain duplicate state
5. **Emit events for cross-module communication** - Don't call methods directly
6. **Clean up event listeners** - Always unsubscribe when done
7. **Enable debug mode during development** - `manager.setDebug(true)`

---

## 🚀 Next Steps

Phase 4 will build on this foundation:
- `pages/DashboardPage.js` - Uses all managers for dashboard functionality
- `pages/SuitesPage.js` - Uses NavigationManager, DialogManager, SelectionManager
- `pages/ExecutionsPage.js` - Uses SelectionManager, NotificationManager
- Page-specific modules will coordinate via EventBus

---

**Created**: 2025-10-02
**Phase**: 3 of 7 (UI Management Complete)
**Total Lines**: 1,577 lines across 4 manager modules
