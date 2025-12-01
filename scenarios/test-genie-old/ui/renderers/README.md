# Test Genie Renderers

This directory contains reusable rendering modules that handle HTML generation for various UI components across the Test Genie application.

## 📁 Module Overview

### `TableRenderer.js` (600 lines) ✅ COMPLETE
**Purpose**: Unified table rendering across all pages

**Key Features**:
- Executions tables (dashboard + executions page)
- Suites tables with scenario grouping
- Coverage summaries tables
- Vault history tables
- Vault list rendering
- Selection support (checkboxes, select all)
- Action buttons (view, delete, execute)
- Empty/loading/error states

**Usage**:
```javascript
import { tableRenderer } from '../renderers/index.js';

// Render executions table with selection
const html = tableRenderer.renderExecutionsTable(executions, {
    selectable: true,
    isSelected: (id) => selectedIds.has(id),
    showDeleteButton: true
});
container.innerHTML = html;

// Render suites table
const html = tableRenderer.renderSuitesTable(scenarios, {
    isSelected: (id) => selectedIds.has(id)
});
container.innerHTML = html;

// Render coverage table
const html = tableRenderer.renderCoverageTable(coverages);
container.innerHTML = html;

// Render vault history
const html = tableRenderer.renderVaultHistoryTable(executions);
container.innerHTML = html;
```

---

### `ChartRenderer.js` (450 lines) ✅ COMPLETE
**Purpose**: Custom Canvas 2D chart rendering

**Key Features**:
- Trends charts (pass rate line + failure bars)
- High DPI display support (retina displays)
- Simple line charts
- Bar charts
- Grid lines with percentage labels
- X-axis date labels
- Responsive resizing

**Usage**:
```javascript
import { chartRenderer } from '../renderers/index.js';

// Render trends chart (ReportsPage)
chartRenderer.renderTrendsChart(canvas, series, {
    height: 220,
    margin: { top: 20, right: 24, bottom: 32, left: 40 },
    passLineColor: '#00ff41',
    failBarColor: 'rgba(255, 0, 64, 0.35)',
    gridColor: 'rgba(255, 255, 255, 0.08)'
});

// Render simple line chart
chartRenderer.renderLineChart(canvas, dataPoints, {
    height: 200,
    lineColor: '#00ff41',
    showMarkers: true
});

// Render bar chart
chartRenderer.renderBarChart(canvas, dataPoints, {
    height: 200,
    barColor: '#00ff41'
});

// Clear canvas
chartRenderer.clearCanvas(canvas);
```

---

### `CardRenderer.js` (400 lines) ✅ COMPLETE
**Purpose**: Card and metric rendering

**Key Features**:
- Metric cards (dashboard, reports overview)
- Insight cards with severity levels
- Suite cards (for cards view)
- Summary cards with key-value pairs
- Status badges with icons
- Progress bars
- Empty/loading/error cards

**Usage**:
```javascript
import { cardRenderer } from '../renderers/index.js';

// Render metric card
const html = cardRenderer.renderMetricCard({
    title: 'Total Suites',
    value: 42,
    subtitle: '8 active scenarios',
    icon: 'layers',
    trend: 'up'
});
container.innerHTML = html;

// Render insight card
const html = cardRenderer.renderInsightCard({
    title: 'Coverage Regression Detected',
    detail: 'Scenario X coverage dropped from 95% to 78%',
    severity: 'high',
    actions: ['Review recent changes', 'Add missing tests'],
    scenario_name: 'scenario-x'
});
container.innerHTML = html;

// Render suite card
const html = cardRenderer.renderSuiteCard(scenario, (id) => selectedIds.has(id));
container.innerHTML = html;

// Render status badge
const html = cardRenderer.renderStatusBadge('running', { showIcon: true });

// Render progress bar
const html = cardRenderer.renderProgressBar(75, {
    label: 'Code Coverage',
    color: 'success'
});
```

---

### `DetailPanelRenderer.js` (350 lines) ✅ COMPLETE
**Purpose**: Detail panel and complex component rendering

**Key Features**:
- Vault phase timelines with status indicators
- Test results detail panels
- Coverage detail panels
- Execution summary panels
- Phase configuration forms
- Scenario analytics rows
- Empty/loading/error states

**Usage**:
```javascript
import { detailPanelRenderer } from '../renderers/index.js';

// Render vault phase timeline
const html = detailPanelRenderer.renderVaultTimeline(
    phases,
    phaseResults,
    completedSet,
    failedSet,
    currentPhase
);
container.innerHTML = html;

// Render test results detail
const html = detailPanelRenderer.renderTestResultsDetail(execution, testResults);
container.innerHTML = html;

// Render coverage detail panel
const html = detailPanelRenderer.renderCoverageDetail(coverage);
container.innerHTML = html;

// Render execution summary
const html = detailPanelRenderer.renderExecutionSummary(execution);
container.innerHTML = html;

// Render phase config form
const html = detailPanelRenderer.renderPhaseConfigForm('test-generation', {
    timeout: 600,
    description: 'Generate comprehensive tests'
});
container.innerHTML = html;
```

---

## 🎯 Architecture Benefits

### Before (Rendering in Page Modules)
- ❌ Duplicate rendering logic across pages
- ❌ ~1,000+ lines of HTML template code in pages
- ❌ Inconsistent styling and structure
- ❌ Hard to update UI patterns globally
- ❌ Difficult to test rendering logic

### After (Centralized Renderers)
- ✅ Single source of truth for each component type
- ✅ Consistent styling and behavior
- ✅ Easy to update UI patterns globally
- ✅ Independently testable rendering logic
- ✅ Reusable across all pages
- ✅ Reduced page module sizes

---

## 📊 Impact Metrics

### Code Reduction
- **Before**: ~1,000+ lines of rendering code across pages
- **After**: ~1,800 lines in 4 renderer modules
- **Net Increase**: +800 lines (one-time cost for reusability)
- **Future Savings**: Every new table/chart/card uses existing renderers

### Reusability
- **TableRenderer**: Used by 5 pages (Dashboard, Suites, Executions, Coverage, Vault)
- **ChartRenderer**: Used by Reports page, extensible for future charts
- **CardRenderer**: Used by Dashboard, Reports, Suites (cards view)
- **DetailPanelRenderer**: Used by Vault, Executions (detail views)

---

## 🔄 Integration Pattern

All renderers follow this pattern:

```javascript
import { tableRenderer, cardRenderer } from '../renderers/index.js';

class MyPage {
    renderContent() {
        // Use renderers instead of inline HTML templates
        const tableHtml = tableRenderer.renderExecutionsTable(data, options);
        const cardHtml = cardRenderer.renderMetricCard(metric);

        container.innerHTML = `
            <div class="page-header">
                ${cardHtml}
            </div>
            <div class="page-content">
                ${tableHtml}
            </div>
        `;
    }
}
```

---

## 🧪 Testing Strategy

Each renderer module is independently testable:

```javascript
import { tableRenderer } from './TableRenderer.js';

test('renderExecutionsTable() should handle empty data', () => {
    const html = tableRenderer.renderExecutionsTable([]);
    expect(html).toContain('No test executions found');
});

test('renderExecutionsTable() should render rows', () => {
    const executions = [
        { id: '1', suiteName: 'Test', status: 'completed', duration: 10, passed: 5, failed: 0, timestamp: '2025-01-01' }
    ];
    const html = tableRenderer.renderExecutionsTable(executions);
    expect(html).toContain('<table');
    expect(html).toContain('Test');
    expect(html).toContain('completed');
});

test('renderExecutionsTable() should support selection', () => {
    const executions = [...];
    const html = tableRenderer.renderExecutionsTable(executions, {
        selectable: true,
        isSelected: (id) => id === '1'
    });
    expect(html).toContain('data-execution-select-all');
    expect(html).toContain('checked');
});
```

---

## 🚀 Usage Guidelines

### When to Use Renderers

1. **Always use for tables** - `tableRenderer` handles all table types
2. **Always use for charts** - `chartRenderer` handles canvas rendering
3. **Always use for cards** - `cardRenderer` handles all card types
4. **Always use for detail panels** - `detailPanelRenderer` handles complex components

### When NOT to Use Renderers

1. **Simple inline HTML** - For 1-2 line HTML snippets, inline is fine
2. **Page-specific layouts** - Main page structure stays in page modules
3. **Form inputs** - Standard form elements don't need renderers

### Adding New Renderers

When adding new rendering logic:

1. Check if it fits in an existing renderer first
2. If it's used in 2+ places, extract to renderer
3. If it's page-specific, keep it in the page module
4. Follow singleton pattern like existing renderers

---

## 📝 Implementation Notes

### HTML Generation
- All renderers return **HTML strings**, not DOM elements
- Parent components set `innerHTML` to render
- Use `escapeHtml()` for ALL user-provided content
- Use template literals for readability

### State Management
- Renderers are **stateless** - pure functions
- No DOM manipulation inside renderers
- Pass all data and callbacks as parameters

### Styling
- Renderers use **semantic CSS classes**
- Actual styles defined in `styles.css`
- Renderers don't include `<style>` tags

### Icons
- All icons use **Lucide** via `data-lucide` attributes
- Parent component calls `refreshIcons()` after rendering
- Don't forget to refresh icons after setting `innerHTML`

---

## 🔧 Common Patterns

### Pattern 1: Conditional Rendering
```javascript
const html = hasData
    ? tableRenderer.renderExecutionsTable(data, options)
    : tableRenderer.renderEmptyState('No executions found');
```

### Pattern 2: Loading States
```javascript
// Show loading
container.innerHTML = tableRenderer.renderLoadingState('Loading executions...');

// Fetch data
const data = await api.getExecutions();

// Render actual content
container.innerHTML = tableRenderer.renderExecutionsTable(data, options);
```

### Pattern 3: Error Handling
```javascript
try {
    const data = await api.getData();
    container.innerHTML = tableRenderer.renderExecutionsTable(data, options);
} catch (error) {
    container.innerHTML = tableRenderer.renderErrorState('Failed to load data');
}
```

---

**Created**: 2025-10-02
**Phase**: 5 of 7 (Rendering Logic - Complete)
**Status**: All 4 renderer modules complete (~1,800 lines total)
