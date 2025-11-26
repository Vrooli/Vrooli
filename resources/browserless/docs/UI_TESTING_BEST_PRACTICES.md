# UI Testing Best Practices for Browserless Automation

## Overview
This document outlines best practices for designing UIs that can be reliably and comprehensively tested using Browserless as the browser backend. Author and run automation flows through Browser Automation Studio (BAS); follow these guidelines so BAS+Browserless runs remain stable.

## Table of Contents
1. [Element Identification](#element-identification)
2. [State Management](#state-management)
3. [Loading Indicators](#loading-indicators)
4. [Error Handling](#error-handling)
5. [Form Design](#form-design)
6. [Navigation](#navigation)
7. [Accessibility](#accessibility)
8. [Performance](#performance)
9. [Building Workflows with BAS](#building-workflows-with-bas)

## Element Identification

### Use Semantic HTML and Unique Identifiers
```html
<!-- GOOD: Unique IDs and semantic elements -->
<button id="submit-order" data-testid="submit-order-btn">Submit Order</button>
<input id="email-input" name="email" type="email" />

<!-- BAD: Generic classes and no identifiers -->
<div class="btn">Submit</div>
<input class="field" />
```

### Data Attributes for Testing
Use `data-testid` attributes specifically for testing:
```html
<div data-testid="user-profile-card">
  <span data-testid="user-name">John Doe</span>
  <button data-testid="edit-profile">Edit</button>
</div>
```

### Consistent Naming Conventions
- Use kebab-case for IDs: `user-profile-form`
- Use descriptive names: `delete-account-btn` not `btn1`
- Group related elements: `checkout-form`, `checkout-submit`, `checkout-cancel`

## State Management

### Clear Visual States
Make UI states visually distinct:
```css
/* Loading state */
.loading { opacity: 0.6; cursor: wait; }

/* Disabled state */
button:disabled { opacity: 0.5; cursor: not-allowed; }

/* Error state */
.error { border: 2px solid red; }
```

### State Attributes
Use HTML attributes to indicate state:
```html
<button aria-busy="true" disabled>Processing...</button>
<div data-loading="true">Loading content...</div>
<form data-state="submitting">...</form>
```

## Loading Indicators

### Predictable Loading Patterns
```html
<!-- Show loading indicator with consistent class -->
<div class="spinner" data-testid="loading-spinner"></div>

<!-- Content container with loading state -->
<div data-testid="content" data-loading="true">
  <div class="skeleton-loader"></div>
</div>
```

### Network Idle Detection
Structure pages to properly trigger network idle:
```javascript
// Avoid continuous polling during initial load
window.addEventListener('load', () => {
  // Start polling only after initial content loads
  startDataPolling();
});
```

## Error Handling

### Visible Error Messages
```html
<!-- Error messages with clear identifiers -->
<div id="error-container" data-testid="error-message" role="alert">
  <span class="error-text">Invalid email format</span>
</div>
```

### Form Validation Feedback
```html
<input 
  id="email" 
  aria-invalid="true"
  aria-describedby="email-error"
/>
<span id="email-error" class="field-error">
  Please enter a valid email
</span>
```

## Form Design

### Accessible Form Elements
```html
<form id="login-form" data-testid="login-form">
  <label for="username">Username</label>
  <input 
    id="username" 
    name="username"
    required
    data-testid="username-input"
  />
  
  <label for="password">Password</label>
  <input 
    id="password" 
    name="password"
    type="password"
    required
    data-testid="password-input"
  />
  
  <button 
    type="submit"
    data-testid="login-submit"
  >
    Login
  </button>
</form>
```

### Clear Submit Actions
- Use proper form elements with `type="submit"`
- Avoid JavaScript-only form submissions when possible
- Provide clear success/failure feedback

## Navigation

### Predictable URLs
```javascript
// Good: Predictable URL patterns
/dashboard
/settings
/users/:id

// Bad: Random or session-based URLs
/app?session=abc123xyz
/page_2847362
```

### Navigation Indicators
```html
<!-- Active page indicator -->
<nav>
  <a href="/home" data-active="false">Home</a>
  <a href="/dashboard" data-active="true" aria-current="page">Dashboard</a>
</nav>
```

## Accessibility

### ARIA Labels and Roles
```html
<button aria-label="Close dialog" data-testid="close-dialog">
  <span aria-hidden="true">×</span>
</button>

<div role="navigation" aria-label="Main navigation">
  <!-- Navigation items -->
</div>
```

### Keyboard Navigation
Ensure all interactive elements are keyboard accessible:
```css
/* Visible focus indicators */
button:focus,
a:focus,
input:focus {
  outline: 2px solid blue;
  outline-offset: 2px;
}
```

## Performance

### Optimize Initial Load
```html
<!-- Lazy load non-critical content -->
<img loading="lazy" src="image.jpg" alt="Description" />

```

### App Ready Signal (Critical for Automation Speed)
- Expose a simple, global marker when the UI is truly usable (hydrated + critical data loaded). Recommended:
  - Set `document.documentElement.dataset.appReady = "true"` in your app bootstrap after initial render/hydration and first-screen data load.
  - Keep it dependency-less; this works across all scenarios.
- Automation can wait on `[data-app-ready="true"]`; fall back to DOM ready if absent.
- React example:
  ```tsx
  // ready.ts
  export const markAppReady = () => {
    try { document.documentElement.dataset.appReady = "true"; } catch (_) {}
  };

  // main.tsx
  import { createRoot } from "react-dom/client";
  import { App } from "./App";
  import { markAppReady } from "./ready";

  (async () => {
    // await bootstrap data if needed
    const root = createRoot(document.getElementById("root")!);
    root.render(<App />);
    markAppReady(); // set after hydration + critical data
  })();
  ```

### Debounce User Input
```javascript
// Debounce search to avoid excessive requests
let searchTimeout;
searchInput.addEventListener('input', (e) => {
  clearTimeout(searchTimeout);
  searchTimeout = setTimeout(() => {
    performSearch(e.target.value);
  }, 300);
});
```

## Building Workflows with BAS

Browserless now serves as the CDP backend while workflow authoring/runs live in Browser Automation Studio. Use these BAS docs to design and execute UI tests:
- `scenarios/browser-automation-studio/README.md` for scenario setup and CLI/API usage
- `scenarios/browser-automation-studio/docs/nodes/` for node-level capabilities when composing flows
- `scenarios/browser-automation-studio/docs/PROGRESS.md` for current feature coverage and gaps

## Testing Checklist

### Before Deployment
- [ ] All interactive elements have unique identifiers
- [ ] Forms have proper labels and validation messages
- [ ] Loading states are clearly indicated
- [ ] Error messages are visible and identifiable
- [ ] Navigation URLs are predictable
- [ ] Critical elements have `data-testid` attributes
- [ ] Page ready states are detectable
- [ ] Focus indicators are visible
- [ ] Content loads without continuous polling

### Workflow Testing
- Execute and validate flows via Browser Automation Studio; use the BAS CLI or UI replay tooling to confirm selectors, waits, and artifacts.

## Common Pitfalls to Avoid

1. **Dynamic IDs**: Avoid auto-generated IDs that change between sessions
2. **Timing Issues**: Use proper wait conditions instead of fixed delays
3. **Brittle Selectors**: Don't rely on complex CSS paths or nth-child
4. **Hidden State**: Ensure state changes are visible in the DOM
5. **Network Noise**: Minimize background requests during testing
6. **Modal Dialogs**: Ensure modals have predictable selectors
7. **Infinite Scroll**: Provide alternative pagination for testing
8. **Single Page Apps**: Ensure URL changes reflect navigation

## Resources

- `scenarios/browser-automation-studio/README.md` (BAS setup, CLI/API usage)
- `scenarios/browser-automation-studio/docs/nodes/` (workflow node catalog)
- [CSS Selector Best Practices](https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Selectors)
- [ARIA Guidelines](https://www.w3.org/WAI/ARIA/apg/)
- [Web Content Accessibility Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
