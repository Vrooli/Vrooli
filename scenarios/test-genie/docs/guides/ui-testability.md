# Writing Testable UIs for Browser Automation

> **Essential reading for scenario developers building UIs that will be tested with BAS workflows**

## Table of Contents
- [Overview](#overview)
- [Why Testability Matters](#why-testability-matters)
- [Element Identification](#element-identification)
- [State Management](#state-management)
- [Loading Indicators](#loading-indicators)
- [Error Handling](#error-handling)
- [Form Design](#form-design)
- [Navigation](#navigation)
- [Accessibility](#accessibility)
- [Performance](#performance)
- [Testing Checklist](#testing-checklist)
- [Common Pitfalls](#common-pitfalls)
- [See Also](#see-also)

## Overview

This guide outlines best practices for designing UIs that can be reliably tested using Vrooli Ascension (BAS) workflows. Following these guidelines ensures your scenarios are testable, maintainable, and production-ready.

**Key Principle**: Design UIs for automation from the start, not as an afterthought.

## Why Testability Matters

Vrooli scenarios are $10K-50K revenue applications. Automated testing ensures:
- **Production Quality** - Catch bugs before deployment
- **Confident Refactoring** - Change code without breaking functionality
- **Documentation** - Tests serve as living documentation
- **Requirement Validation** - Prove PRD requirements are met

**Remember**: AI agents write BAS workflows directly in JSON. Your UI must be machine-readable, not just human-usable.

## Element Identification

### Use `data-testid` Attributes

**Best Practice**: Add `data-testid` to all interactive elements and important containers.

```tsx
// ✅ GOOD: Unique, descriptive test IDs
<button data-testid="submit-order-btn" onClick={handleSubmit}>
  Submit Order
</button>

<input
  data-testid="email-input"
  id="email"
  name="email"
  type="email"
/>

<div data-testid="user-profile-card">
  <span data-testid="user-name">{user.name}</span>
  <button data-testid="edit-profile-btn">Edit</button>
</div>

// ❌ BAD: No identifiers, generic classes
<div className="btn" onClick={handleClick}>Submit</div>
<input className="field" />
```

### Naming Conventions

**Pattern**: `{component}-{element}-{action|role}`

```tsx
// Buttons
data-testid="create-project-btn"
data-testid="delete-account-btn"
data-testid="save-workflow-btn"

// Inputs
data-testid="username-input"
data-testid="password-input"
data-testid="search-query-input"

// Containers
data-testid="project-list-container"
data-testid="workflow-canvas"
data-testid="execution-history-panel"

// Status indicators
data-testid="loading-spinner"
data-testid="error-message"
data-testid="success-toast"
```

**Rules**:
- Use kebab-case: `project-modal-close` not `projectModalClose`
- Be descriptive: `delete-workflow-btn` not `btn1`
- Group related: `project-modal`, `project-modal-form`, `project-modal-submit`

### Semantic HTML

Use proper HTML elements for accessibility and testability:

```tsx
// ✅ GOOD: Semantic HTML
<nav data-testid="main-nav">
  <a href="/dashboard">Dashboard</a>
  <a href="/settings">Settings</a>
</nav>

<form data-testid="login-form" onSubmit={handleSubmit}>
  <label htmlFor="username">Username</label>
  <input id="username" name="username" />
  <button type="submit">Login</button>
</form>

// ❌ BAD: Divs for everything
<div onClick={navigate}>Dashboard</div>
<div onClick={handleSubmit}>Login</div>
```

## State Management

### Expose State via Attributes

Make UI state visible in the DOM for BAS workflows to detect:

```tsx
// ✅ GOOD: State is visible
<button
  data-testid="submit-btn"
  data-loading={isLoading}
  disabled={isLoading}
  aria-busy={isLoading}
>
  {isLoading ? 'Processing...' : 'Submit'}
</button>

<form
  data-testid="project-form"
  data-state={formState} // "idle" | "submitting" | "success" | "error"
>
  {/* form fields */}
</form>

<div
  data-testid="content-container"
  data-ready={isDataLoaded}
>
  {isDataLoaded ? <Content /> : <Skeleton />}
</div>

// ❌ BAD: State hidden in JavaScript
const [isLoading, setIsLoading] = useState(false); // No DOM representation
```

### Visual State Indicators

Use CSS classes that correspond to data attributes:

```css
/* Loading state */
[data-loading="true"] {
  opacity: 0.6;
  cursor: wait;
}

/* Disabled state */
button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Error state */
[data-state="error"] {
  border: 2px solid #ef4444;
}

/* Success state */
[data-state="success"] {
  border: 2px solid #10b981;
}
```

## Loading Indicators

### Predictable Loading Patterns

```tsx
// ✅ GOOD: Consistent loading pattern
{isLoading && (
  <div data-testid="loading-spinner" className="spinner">
    Loading...
  </div>
)}

// Content with loading state
<div data-testid="project-list" data-loading={isLoading}>
  {isLoading ? (
    <SkeletonLoader />
  ) : (
    projects.map(p => <ProjectCard key={p.id} project={p} />)
  )}
</div>
```

### Network Idle Patterns

Structure initial page loads to properly trigger `networkidle0`:

```tsx
// ✅ GOOD: Load data once, then poll
useEffect(() => {
  const loadInitialData = async () => {
    await fetchProjects();
    await fetchWorkflows();
    // Now start polling for updates
    const interval = setInterval(fetchUpdates, 5000);
    return () => clearInterval(interval);
  };
  loadInitialData();
}, []);

// ❌ BAD: Continuous polling from start
useEffect(() => {
  const interval = setInterval(fetchData, 1000); // Prevents networkidle0
  return () => clearInterval(interval);
}, []);
```

## Error Handling

### Visible Error Messages

```tsx
// ✅ GOOD: Error messages with clear identifiers
{error && (
  <div
    data-testid="error-message"
    role="alert"
    className="error-banner"
  >
    {error.message}
  </div>
)}

// Form field errors
<input
  data-testid="email-input"
  aria-invalid={!!emailError}
  aria-describedby="email-error"
/>
{emailError && (
  <span
    id="email-error"
    data-testid="email-error"
    className="field-error"
  >
    {emailError}
  </span>
)}
```

## Form Design

### Accessible, Testable Forms

```tsx
<form
  data-testid="create-project-form"
  onSubmit={handleSubmit}
>
  <label htmlFor="project-name">Project Name</label>
  <input
    id="project-name"
    name="projectName"
    data-testid="project-name-input"
    required
    aria-required="true"
  />

  <label htmlFor="description">Description</label>
  <textarea
    id="description"
    name="description"
    data-testid="project-description-input"
  />

  <button
    type="submit"
    data-testid="create-project-submit"
    disabled={!isValid}
  >
    Create Project
  </button>
</form>
```

**Key Points**:
- Use proper `<label>` elements with `htmlFor`
- Add `name` attributes for form data
- Use `type="submit"` for submit buttons
- Expose validation state via `aria-invalid`

## Navigation

### Predictable URLs

```tsx
// ✅ GOOD: Predictable, RESTful URLs
/dashboard
/projects
/projects/:id
/projects/:id/workflows
/projects/:id/workflows/:workflowId

// ❌ BAD: Random or session-based URLs
/app?session=abc123xyz&tab=2
/page_2847362
```

### Navigation State

```tsx
// ✅ GOOD: Active state is visible
<nav data-testid="main-nav">
  <a
    href="/dashboard"
    data-testid="nav-dashboard"
    data-active={pathname === '/dashboard'}
    aria-current={pathname === '/dashboard' ? 'page' : undefined}
  >
    Dashboard
  </a>
  <a
    href="/projects"
    data-testid="nav-projects"
    data-active={pathname.startsWith('/projects')}
    aria-current={pathname.startsWith('/projects') ? 'page' : undefined}
  >
    Projects
  </a>
</nav>
```

## Accessibility

### ARIA Labels and Roles

```tsx
// Icon-only buttons
<button
  data-testid="close-modal-btn"
  aria-label="Close dialog"
  onClick={handleClose}
>
  <X aria-hidden="true" />
</button>

// Regions
<nav
  data-testid="main-nav"
  role="navigation"
  aria-label="Main navigation"
>
  {/* nav items */}
</nav>

// Live regions for dynamic content
<div
  data-testid="notification-area"
  role="status"
  aria-live="polite"
>
  {notification}
</div>
```

### Keyboard Navigation

```css
/* Visible focus indicators */
button:focus-visible,
a:focus-visible,
input:focus-visible {
  outline: 2px solid #3b82f6;
  outline-offset: 2px;
}

/* Skip to main content */
.skip-link {
  position: absolute;
  top: -40px;
  left: 0;
  background: white;
  padding: 8px;
  z-index: 100;
}

.skip-link:focus {
  top: 0;
}
```

## Performance

### Optimize Initial Load

```tsx
// ✅ GOOD: Lazy load non-critical content
<img
  src={project.thumbnail}
  alt={project.name}
  loading="lazy"
  data-testid="project-thumbnail"
/>

// Signal app readiness
const App = () => {
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    // Perform critical initialization
    Promise.all([
      fetchUserData(),
      fetchConfiguration(),
    ]).then(() => {
      setIsReady(true);
    });
  }, []);

  return (
    <div data-testid="app-root" data-ready={isReady}>
      {isReady ? <MainApp /> : <InitialLoader />}
    </div>
  );
};
```

### Debounce User Input

```tsx
// ✅ GOOD: Debounce search to reduce requests
const [searchQuery, setSearchQuery] = useState('');
const debouncedSearch = useDebounce(searchQuery, 300);

useEffect(() => {
  if (debouncedSearch) {
    performSearch(debouncedSearch);
  }
}, [debouncedSearch]);

<input
  data-testid="search-input"
  value={searchQuery}
  onChange={(e) => setSearchQuery(e.target.value)}
  placeholder="Search..."
/>
```

## Testing Checklist

### Before Committing UI Code

- [ ] All interactive elements have `data-testid` attributes
- [ ] Forms have proper `<label>` elements with `htmlFor`
- [ ] Loading states visible via `data-loading` or similar
- [ ] Error messages have `data-testid="error-message"` and `role="alert"`
- [ ] Navigation URLs are predictable and RESTful
- [ ] State changes reflected in DOM attributes (`data-state`, `aria-busy`, etc.)
- [ ] No dynamic/random IDs (`id="item-${Math.random()}"`)
- [ ] Focus indicators visible on all interactive elements
- [ ] Initial page load doesn't have continuous polling
- [ ] App readiness detectable (`data-ready` attribute)

### Before Writing BAS Workflow

- [ ] Identify all selectors using `data-testid`
- [ ] Verify state transitions are visible
- [ ] Check loading indicators appear/disappear correctly
- [ ] Test error scenarios show error messages
- [ ] Navigation changes URL as expected
- [ ] Forms submit correctly and show feedback

## Common Pitfalls

### 1. Dynamic IDs

```tsx
// ❌ BAD: IDs change between sessions
<div id={`item-${Math.random()}`}>...</div>

// ✅ GOOD: Stable IDs
<div id={`item-${item.id}`} data-testid={`item-${item.id}`}>...</div>
```

### 2. Timing Issues

```tsx
// ❌ BAD: Fixed delays
await new Promise(resolve => setTimeout(resolve, 2000));

// ✅ GOOD: Wait for specific condition
await waitForElement('[data-testid="content"][data-ready="true"]');
```

### 3. Brittle Selectors

```tsx
// ❌ BAD: Complex CSS paths, nth-child
div > div:nth-child(3) > button.btn.primary
document.querySelector('.container .sidebar ul li:nth-child(2)')

// ✅ GOOD: data-testid
[data-testid="create-project-btn"]
[data-testid="sidebar-projects-link"]
```

### 4. Hidden State

```tsx
// ❌ BAD: State only in JavaScript
const [isProcessing, setIsProcessing] = useState(false);
// No way for BAS to detect this state

// ✅ GOOD: State in DOM
<div data-processing={isProcessing}>
  <button disabled={isProcessing} data-loading={isProcessing}>
    {isProcessing ? 'Processing...' : 'Submit'}
  </button>
</div>
```

### 5. Modal Dialogs

```tsx
// ✅ GOOD: Modal with predictable selectors
<Dialog
  open={isOpen}
  onClose={handleClose}
  data-testid="project-modal"
>
  <DialogTitle data-testid="project-modal-title">
    Create Project
  </DialogTitle>
  <DialogContent data-testid="project-modal-content">
    {/* form */}
  </DialogContent>
  <DialogActions>
    <Button
      data-testid="project-modal-cancel"
      onClick={handleClose}
    >
      Cancel
    </Button>
    <Button
      data-testid="project-modal-submit"
      type="submit"
    >
      Create
    </Button>
  </DialogActions>
</Dialog>
```

### 6. Single Page Apps

```tsx
// ✅ GOOD: URL changes reflect navigation
const navigate = useNavigate();

const handleProjectClick = (projectId) => {
  navigate(`/projects/${projectId}`); // URL changes
};

// ❌ BAD: Navigation without URL change
const handleProjectClick = (projectId) => {
  setCurrentView('project-detail'); // No URL change
  setSelectedProject(projectId);
};
```

## See Also

### Related Guides
- [Phases Overview](../phases/README.md) - How UI tests fit into the 11-phase architecture
- [Scenario Unit Testing](../phases/unit/scenario-unit-testing.md) - Unit testing with Vitest
- [Requirements Sync](../phases/business/requirements-sync.md) - Link UI tests to requirements

### Reference
- [Test Runners](../phases/unit/test-runners.md) - Node.js test runner configuration
- [Phases Overview](../phases/README.md) - Integration and business phase details
- [Presets](../reference/presets.md) - Test preset configurations

### External Resources
- [Testing Library Best Practices](https://testing-library.com/docs/queries/about/#priority)
- [ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)
- [WebAIM Checklist](https://webaim.org/standards/wcag/checklist)

---

**Remember**: A testable UI is an accessible, maintainable UI. Design for automation from the start.
