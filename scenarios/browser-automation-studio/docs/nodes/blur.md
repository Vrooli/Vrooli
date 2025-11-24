# Blur Node

`[REQ:BAS-NODE-BLUR-VALIDATION]` triggers `blur` events on elements to drive client-side validation or state updates that occur when an input loses focus.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Selector** | CSS selector for the element to blur | Yes | Trimmed within the runtime.
| **Timeout (ms)** | Wait for the element before blurring | No | Defaults to 5 000.
| **Post-blur wait (ms)** | Delay after the blur event | No | Helpful when error banners fade in after validation.

## Runtime Behavior

1. Focus and Blur share the same payload shape (`focusConfig`). The automation compiler/executor forwards selector/timeout/wait values as-authored; validation happens in the workflow validator.
2. Browserless executes a small script that calls `element.blur()` inside the active frame, ensuring any bound `onBlur` handlers fire.
3. Execution telemetry records the selector plus the wait time so you can inspect validation timing in Execution Viewer.

## Use Cases

- Trigger form validation messages after typing text.
- Close dropdowns or editors that listen for blur events.
- Ensure UI reflects the same behavior a real user experiences when tabbing through fields.

## Related Nodes

- **Focus** – Typically precedes Blur; use Focus → Type → Blur sequences for realistic form entry.
- **Assert** – Verify validation messages after blur fires.
- **Wait** – Allow asynchronous validation calls to finish.
