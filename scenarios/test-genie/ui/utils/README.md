# Test Genie Utilities

This directory contains reusable utility modules extracted from the monolithic `app.js` file.

## 📁 Module Overview

### `constants.js` (170 lines)
**Purpose**: Centralized configuration and constant values

**Exports**:
- API configuration
- Timing constants
- Default settings
- Status descriptors
- UI configuration
- Error/success messages
- Chart configuration

**Usage**:
```javascript
import { API_BASE_URL, DEFAULT_GENERATION_PHASES, STATUS_DESCRIPTORS } from './utils/constants.js';
```

---

### `formatters.js` (470 lines)
**Purpose**: Data formatting and transformation utilities

**Key Functions**:
- Time/date formatting: `formatTimestamp()`, `formatDateTime()`, `formatDateRange()`
- Duration formatting: `formatDurationSeconds()`
- Number formatting: `formatPercent()`, `clampPercentage()`
- String formatting: `formatLabel()`, `formatPhaseLabel()`
- Data normalization: `normalizeCollection()`, `normalizeId()`
- HTML safety: `escapeHtml()`

**Usage**:
```javascript
import { formatTimestamp, formatPercent, escapeHtml } from './utils/formatters.js';

const time = formatTimestamp(new Date());
const pct = formatPercent(85.5, 1); // "85.5%"
const safe = escapeHtml('<script>alert("xss")</script>');
```

---

### `validators.js` (190 lines)
**Purpose**: Input validation and data verification

**Key Functions**:
- Health checks: `isHealthPayloadHealthy()`
- Input validation: `isValidEmail()`, `isValidUrl()`, `isValidScenarioName()`
- Range validation: `normalizeCoverageTarget()`, `isValidPercentage()`
- Type checks: `isNonEmptyArray()`, `isNonEmptyObject()`
- Sanitization: `sanitizeInput()`

**Usage**:
```javascript
import { isValidEmail, normalizeCoverageTarget, isHealthPayloadHealthy } from './utils/validators.js';

if (isValidEmail(input)) {
  // proceed
}

const coverage = normalizeCoverageTarget(120); // Clamped to 100
```

---

### `domHelpers.js` (429 lines)
**Purpose**: DOM manipulation and UI interaction utilities

**Key Functions**:
- Focus management: `focusElement()`, `smoothScrollIntoView()`
- Visibility: `toggleElementVisibility()`, `isInViewport()`
- Scroll management: `enableDragScroll()`, `lockDialogScroll()`
- Event handling: `addManagedEventListener()`, `debounce()`, `throttle()`
- Sidebar: `toggleSidebar()`, `updateSidebarAccessibility()`
- Utilities: `copyToClipboard()`, `createElementFromHTML()`

**Usage**:
```javascript
import { focusElement, enableDragScroll, debounce } from './utils/domHelpers.js';

focusElement(myButton);
enableDragScroll(tableContainer);

const debouncedSearch = debounce((query) => {
  // search logic
}, 300);
```

---

### `index.js` (16 lines)
**Purpose**: Aggregated exports for convenient importing

**Usage**:
```javascript
// Import specific functions
import { formatTimestamp, isValidEmail, focusElement } from './utils/index.js';

// Or import everything
import * as utils from './utils/index.js';
```

---

## 🧪 Testing

Open `test.html` in a browser to run the test suite and verify all utilities work correctly.

```bash
# If you have a local server running:
open http://localhost:8080/utils/test.html

# Or use a simple HTTP server:
cd scenarios/test-genie/ui
python3 -m http.server 8080
# Then navigate to http://localhost:8080/utils/test.html
```

---

## 📝 Design Principles

### 1. Pure Functions
All functions are pure (no side effects) where possible, making them:
- Predictable
- Testable
- Reusable
- Composable

### 2. Single Responsibility
Each module has ONE clear purpose:
- `constants.js` → Configuration
- `formatters.js` → Data transformation
- `validators.js` → Input verification
- `domHelpers.js` → DOM manipulation

### 3. No Dependencies
Utilities have minimal dependencies on each other:
- `constants.js` → No dependencies
- `formatters.js` → Depends on `constants.js`
- `validators.js` → Depends on `constants.js`
- `domHelpers.js` → Depends on `constants.js`

### 4. Defensive Programming
All functions handle edge cases:
- Null/undefined checks
- Type validation
- Safe defaults
- Try/catch where appropriate

---

## 🔄 Migration from app.js

If you're updating code that uses `app.js` methods:

| Old (app.js) | New (utils) |
|-------------|-------------|
| `this.formatTimestamp(date)` | `formatTimestamp(date)` |
| `this.escapeHtml(str)` | `escapeHtml(str)` |
| `this.normalizeId(id)` | `normalizeId(id)` |
| `this.focusElement(el)` | `focusElement(el)` |
| `this.refreshIcons()` | `refreshIcons()` |

**Step 1**: Import the utilities
```javascript
import { formatTimestamp, escapeHtml, normalizeId } from './utils/index.js';
```

**Step 2**: Replace `this.` with direct function calls
```javascript
// Before
const time = this.formatTimestamp(execution.created_at);

// After
const time = formatTimestamp(execution.created_at);
```

---

## 🚀 Future Enhancements

Potential additions to the utilities:

### Formatters
- [ ] `formatFileSize()` - Convert bytes to KB/MB/GB
- [ ] `formatCurrency()` - Format numbers as currency
- [ ] `pluralize()` - Smart pluralization helper

### Validators
- [ ] `isValidPhoneNumber()` - Phone validation
- [ ] `isValidCreditCard()` - Credit card validation
- [ ] `isValidIPAddress()` - IP validation

### DOM Helpers
- [ ] `lazyLoad()` - Lazy load images/components
- [ ] `observeResize()` - ResizeObserver wrapper
- [ ] `delegateEvent()` - Event delegation helper

---

## 📚 Documentation

All functions include JSDoc comments with:
- Description
- Parameter types
- Return type
- Usage examples (where helpful)

Example:
```javascript
/**
 * Format duration in seconds to human-readable string (e.g., "2h 30m 15s")
 * @param {number} totalSeconds
 * @returns {string}
 */
export function formatDurationSeconds(totalSeconds) {
  // implementation
}
```

---

## 🤝 Contributing

When adding new utilities:

1. **Choose the right module**: Put functions in the appropriate file
2. **Follow naming conventions**: Use clear, descriptive names
3. **Add JSDoc comments**: Document parameters and return values
4. **Handle edge cases**: Null checks, type validation, safe defaults
5. **Keep it pure**: Avoid side effects when possible
6. **Update tests**: Add test cases to `test.html`
7. **Export from index.js**: Add to the aggregated exports

---

## 📊 Stats

| Metric | Value |
|--------|-------|
| Total Lines | 1,275 |
| Total Functions | 62 |
| Total Constants | 40+ |
| Test Coverage | Manual (test.html) |
| Dependencies | Minimal (self-contained) |

---

## 🎯 Benefits

### Before (Monolithic app.js)
- ❌ 5,701 lines in one file
- ❌ Hard to find specific functions
- ❌ Difficult to test
- ❌ High coupling
- ❌ Code duplication

### After (Modular utilities)
- ✅ Small, focused modules (<500 lines each)
- ✅ Easy to locate and modify
- ✅ Independently testable
- ✅ Low coupling
- ✅ Reusable across project

---

**Last Updated**: 2025-10-02
**Phase**: 1 of 7 (Foundation Complete)
