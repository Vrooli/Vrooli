# 02 - Basic Automation

This section covers the core automation capabilities of Agent S2.

## Prerequisites

Complete the [01-getting-started](../01-getting-started/) examples first.

## Examples in this section

### mouse_control.py
Learn how to control the mouse:
- Move to specific positions
- Click, double-click, right-click
- Drag and drop
- Scroll

```bash
python mouse_control.py
```

### keyboard_input.py
Master keyboard automation:
- Type text with configurable speed
- Press special keys
- Use keyboard shortcuts
- Simulate hotkeys

```bash
python keyboard_input.py
```

### screenshot.py
Advanced screenshot techniques:
- Capture specific regions
- Different image formats and quality
- Continuous capture
- Performance benchmarking

```bash
python screenshot.py
```

### combined_automation.py
Combine mouse and keyboard for real tasks:
- Fill forms
- Navigate menus
- Interact with applications

```bash
python combined_automation.py
```

## Tips

1. **Timing is important** - Add small delays between actions for reliability
2. **Use absolute positions** - Screen coordinates are more reliable than relative
3. **Check screen size** - Ensure your coordinates work for the display resolution
4. **Handle errors** - Always check if actions succeeded

## What's Next?

Move on to [03-advanced-features](../03-advanced-features/) for:
- Complex automation sequences
- Error handling and recovery
- Performance optimization