# Browser Operations Libraries

## Overview

There are two browser operations libraries in this directory:

## `browser-ops.sh` - Core Operations
- **21 core functions** for basic browser automation
- Direct atomic operations (navigate, click, type, screenshot, etc.)
- Fixed screenshots with proper viewport and timing
- Combined operations like `browser::navigate_and_screenshot()`
- **Use this** for most standard browser automation tasks

## `browser-ops-enhanced.sh` - Enhanced Operations  
- **8 enhanced wrapper functions** with retry logic and debugging
- Sources `browser-ops.sh` and adds retry/debug layers
- Automatic screenshots on failures when `BROWSER_DEBUG=true`
- Configurable retry attempts and delays
- **Use this** for complex workflows that need resilience

## Usage Recommendation

1. **Default**: Use `browser-ops.sh` for standard automation
2. **Complex workflows**: Use `browser-ops-enhanced.sh` for added resilience
3. **Debugging issues**: Use `browser-ops-enhanced.sh` with `BROWSER_DEBUG=true`

Both libraries work with the atomic operations approach (no compilation needed).