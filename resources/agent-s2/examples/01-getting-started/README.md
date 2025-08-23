# 01 - Getting Started with Agent S2

Welcome to Agent S2! This section contains the simplest examples to help you get started.

## Prerequisites

1. Agent S2 must be running:
   ```bash
   ./manage.sh --action start
   ```

2. Python 3 with the Agent S2 client library:
   ```bash
   pip install -e ../../
   ```

## Examples in this section

### hello_screenshot.py
The simplest example - takes a screenshot and saves it to a file.

```bash
python hello_screenshot.py
```

### check_health.py
Shows how to check if Agent S2 is running and get basic information.

```bash
python check_health.py
```

### simple_automation.py
A basic example that combines screenshot, mouse movement, and typing.

```bash
python simple_automation.py
```

## What's Next?

Once you're comfortable with these basics, move on to:
- [02-basic-automation](../02-basic-automation/) - Learn mouse and keyboard control
- [03-advanced-features](../03-advanced-features/) - Explore advanced capabilities
- [04-ai-integration](../04-ai-integration/) - Use AI for intelligent automation