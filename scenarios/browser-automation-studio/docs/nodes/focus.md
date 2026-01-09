# Focus Node

`[REQ:BAS-NODE-FOCUS-INPUT]` moves keyboard focus to a specific element so subsequent Type/Shortcut nodes interact with the correct control and accessibility flows can be validated.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Selector** | CSS selector to focus | Yes | Trimmed before dispatch.
| **Timeout (ms)** | Wait for the element | No | Defaults to 5 000; provide higher values for lazy-loaded forms.
| **Post-focus wait (ms)** | Delay after focusing | No | Useful when focus triggers validation or masking animations.

## Runtime Behavior

1. The automation compiler/executor forwards selector/timeout/wait values as-authored; validation runs in the workflow validator/UI.
2. Browserless scrolls the element into view (if needed) and calls DOM `focus()` via CDP, capturing any errors from the page.
3. Execution artifacts store the focused selector and timing for debugging.

## Use Cases

- Trigger onFocus/onBlur validation flows prior to typing.
- Bring hidden inputs (e.g., inside accordions) into focus before sending keyboard shortcuts.
- Validate accessibility by ensuring focus indicators appear where expected.

## Related Nodes

- **Type** – Usually follows Focus to enter text.
- **Blur** – Pair with Focus to assert onBlur validation.
- **Shortcut/Keyboard** – Fire keyboard commands once focus is in place.
