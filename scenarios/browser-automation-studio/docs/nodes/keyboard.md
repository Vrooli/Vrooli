# Keyboard Node

`[REQ:BAS-NODE-KEYBOARD-DISPATCH]` emits low-level keydown/keypress/keyup events with fine-grained control over modifiers, making it ideal for form validation, key navigation, or custom event testing.

## Configuration

| Field | Description | Required | Notes |
| --- | --- | --- | --- |
| **Key** | Key value (`Enter`, `a`, `ArrowDown`, etc.) | Yes | Matches Chrome DevTools event keys.
| **Event type** | `keydown`, `keypress`, or `keyup` | Yes | Determines which event to dispatch.
| **Modifiers** | Ctrl, Shift, Alt, Meta toggles | No | Multiple modifiers supported.
| **Delay (ms)** | Optional delay before dispatch | No | Useful for chaining multiple keyboard nodes.
| **Timeout (ms)** | Wait for focusable element when `focusSelector` provided | No |
| **Focus selector** | Optional element to focus before dispatch | No | Allows sending keys to specific inputs.

## Runtime Behavior

1. The automation compiler forwards key/event/modifier/focus/timeout params as-is; validation occurs in the workflow validator.
2. Browserless focuses the optional selector, then dispatches the requested key event through CDP’s `Input.dispatchKeyEvent` with the correct modifier state.
3. Execution artifacts capture the key/event/modifiers for debugging.

## Example

```json
{
  "type": "keyboard",
  "data": {
    "key": "Tab",
    "keyEventType": "keydown",
    "keyModifiers": ["shift"],
    "focusSelector": "input[name=password]"
  }
}
```

## Tips

- Chain multiple keyboard nodes to simulate complex sequences (e.g., Shift+Tab, followed by Enter). For simpler combos, consider the Shortcut node.
- When testing accessibility, use keyboard nodes to navigate between elements instead of clicks.
- If no focus selector is provided, keys dispatch to whatever element currently has focus; use Focus nodes for determinism.

## Related Nodes

- **Shortcut** – Higher-level combos for productivity commands.
- **Focus** – Set the target element before sending keys.
- **Type** – Use for text entry, then keyboard nodes for navigation.
