# Core Automation Examples

This directory contains examples demonstrating Agent S2's core automation capabilities - the fundamental building blocks that power all automation tasks, including those orchestrated by AI.

## 🎯 Overview

Core automation provides direct, deterministic control over:
- Mouse movements and clicks
- Keyboard input and shortcuts  
- Screenshot capture
- Multi-step automation sequences

These are the reliable, precise functions that the AI layer uses to execute its plans.

## 📁 Examples

### 1. **screenshot.py** - Screenshot Capture
Demonstrates various screenshot capabilities:
- Full screen capture
- Region-based screenshots
- Different image formats (PNG, JPEG)
- Quality settings
- Continuous capture
- Performance benchmarking

```bash
python screenshot.py
```

### 2. **screenshot-demo.sh** - Basic Screenshot Script
Simple bash script showing:
- API health checking
- Screenshot capture via curl
- Base64 image handling
- Error handling patterns

```bash
./screenshot-demo.sh
```

### 3. **mouse_control.py** - Mouse Control
Comprehensive mouse automation:
- Precise positioning
- Click operations (left, right, middle)
- Drag and drop
- Drawing patterns (squares, circles)
- Smooth movement control
- Hover demonstrations

```bash
python mouse_control.py
```

### 4. **keyboard_input.py** - Keyboard Control  
Full keyboard automation capabilities:
- Text typing at various speeds
- Special key presses
- Keyboard shortcuts (Ctrl+C, Ctrl+V, etc.)
- Function keys
- Multi-line text input
- Special characters and symbols

```bash
python keyboard_input.py
```

### 5. **automation_sequences.py** - Complex Workflows
Multi-step automation sequences:
- Document creation workflows
- Copy/paste operations
- Form filling
- Window management
- Conditional workflows
- Looped operations

```bash
python automation_sequences.py
```

## 🚀 Getting Started

1. **Ensure Agent S2 is running:**
```bash
../../manage.sh --action status
```

2. **Connect to VNC to watch automation:**
- URL: `vnc://localhost:5900`
- Password: `agents2vnc`

3. **Run any example:**
```bash
cd core/
python [example_name].py
```

## 🔧 API Endpoints Used

These examples demonstrate the core API endpoints:

- `GET /health` - Service health check
- `GET /capabilities` - Available capabilities
- `POST /screenshot` - Capture screenshots
- `POST /execute` - Execute automation tasks
- `GET /mouse/position` - Get mouse position

## 💡 Key Concepts

### Precision and Timing
Core automation provides pixel-perfect precision and exact timing control:
```python
# Move to exact position over 0.5 seconds
move_to_position(x=500, y=300, duration=0.5)
```

### Building Blocks
Simple operations combine into complex workflows:
```python
steps = [
    {"type": "mouse_move", "parameters": {"x": 100, "y": 100}},
    {"type": "click", "parameters": {}},
    {"type": "type_text", "parameters": {"text": "Hello"}},
    {"type": "key_press", "parameters": {"keys": ["Return"]}}
]
```

### Deterministic Behavior
Core automation is completely predictable - the same inputs always produce the same outputs, making it the reliable foundation for AI orchestration.

## 🤝 Integration with AI

When AI is enabled, it uses these core functions to execute its plans:

```
AI: "Click on the Chrome icon"
  ↓
AI analyzes screen → finds Chrome at (150, 450)
  ↓  
AI calls: execute_click(x=150, y=450)
  ↓
Core automation performs the click
```

## 📂 Output Files

Generated files are organized in the `../test-outputs/` directory:
- **Screenshots:** Saved to `../test-outputs/screenshots/`
- **Logs:** Execution traces in `../test-outputs/logs/`
- **Temporary files:** Stored in `../test-outputs/temp/`

This keeps the example code clean and test outputs organized.

## 📝 Notes

- All examples include error handling
- Examples are designed to be educational and demonstrate best practices
- Timing delays are included for visibility - adjust as needed
- Generated files are automatically organized in test-outputs directories

## 🛠️ Troubleshooting

If examples fail:
1. Check Agent S2 is running: `../../manage.sh --action status`
2. Verify ports 4113 and 5900 are accessible
3. Ensure no other applications are using these ports
4. Check container logs: `../../manage.sh --action logs`