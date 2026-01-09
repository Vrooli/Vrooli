# Silent Founder OS - Troubleshooting Playbook

## App Won't Start

### Symptoms
- App icon clicked but nothing happens
- App starts then immediately closes
- Error message on launch

### Diagnostic Steps

**Check system requirements:**
- Windows 10+, macOS 11+, or Linux with glibc 2.28+
- At least 4 GB RAM available
- Sufficient disk space (500 MB minimum)

**Linux: Check dependencies:**
```bash
# For .deb installations
ldd /usr/lib/vrooli-ascension/vrooli-ascension | grep "not found"

# Common missing libraries
sudo apt install libgtk-3-0 libnotify4 libnss3 libxss1
```

**macOS: Gatekeeper issues:**
1. Right-click the app → Open (instead of double-click)
2. If blocked: System Preferences → Security & Privacy → "Open Anyway"
3. For Apple Silicon: ensure Rosetta is installed if prompted

**Windows: Missing runtime:**
- Install Visual C++ Redistributable if prompted
- Run as Administrator once to set up permissions

**Check logs:**
```bash
# Linux
~/.config/vrooli-ascension/logs/main.log

# macOS
~/Library/Logs/vrooli-ascension/main.log

# Windows
%APPDATA%\vrooli-ascension\logs\main.log
```

### Resolution
- Most startup issues are resolved by reinstalling the app
- If logs show permission errors, check that the app has access to its data directory
- For persistent issues, submit a bug report at [vrooli.com/feedback](https://vrooli.com/feedback) with the log file attached

---

## Workflow Fails Mid-Run

### Symptoms
- Execution stops unexpectedly
- Step shows "failed" status
- Error message in execution log

### Common Causes and Fixes

**Selector not found:**
- The target element may have changed (different class, ID, or structure)
- Open the failing step and update the selector
- Use more stable selectors: `[data-testid="..."]` or unique IDs over classes

**Page not loaded:**
- The previous navigation may not have completed
- Add a "Wait for element" step before the failing action
- Increase timeout in step settings

**Element not visible/clickable:**
- The element may be off-screen or behind a modal
- Add a scroll action before clicking
- Check for cookie banners or popups blocking the element

**Network timeout:**
- The target site may be slow or unresponsive
- Increase timeout settings for navigate and wait steps
- Check your internet connection

**Session expired:**
- If automating authenticated flows, the session may have timed out
- Add a login workflow as a prerequisite
- Check for "session expired" messages in screenshots

### Debugging Workflow

1. **View execution details:** Click the failed execution to see step-by-step results
2. **Check screenshots:** The last successful screenshot shows the page state before failure
3. **Review logs:** Console and network logs may reveal errors
4. **Test manually:** Run the failing step in isolation to reproduce

---

## Assertions Failing

### Symptoms
- Workflow stops at assertion step
- Assertion shows "failed" with comparison details

### Diagnostic Steps

**Check the expected value:**
- Text content may have changed (dates, counts, dynamic content)
- Consider using partial match or regex instead of exact match

**Verify the selector:**
- The element may exist but with a different selector
- Use the selector picker to update the assertion target

**Timing issues:**
- The element may not be populated yet when the assertion runs
- Add a "Wait for element" step before the assertion
- Increase assertion timeout

**Dynamic content:**
- Avoid asserting on timestamps, session IDs, or random values
- Use pattern matching for variable content: `contains` instead of `equals`

### Assertion Types Reference

| Type | Use When |
|------|----------|
| Element exists | Verify page structure, presence of features |
| Text contains | Check for specific content (partial match) |
| Text equals | Verify exact content (labels, static text) |
| Attribute equals | Check form values, data attributes, hrefs |
| Element not exists | Verify something was removed or hidden |

---

## Performance Issues

### Symptoms
- App runs slowly
- High memory usage
- Browser pane unresponsive

### Optimization Steps

**Clear old data:**
- Executions accumulate screenshots and logs over time
- Archive or delete old executions you no longer need
- Check disk space: `df -h` (Linux/macOS) or Disk Management (Windows)

**Reduce recording scope:**
- If capturing high-frequency actions, the timeline grows large
- Use shorter recording sessions for workflows
- Split complex flows into multiple smaller workflows

**Check background processes:**
- Close other memory-intensive applications
- Check for browser extensions in the built-in browser (should be minimal)

**System resources:**
- Ensure 8 GB+ RAM for smooth operation
- Use an SSD for faster screenshot I/O
- Close unused tabs in the built-in browser

**Disable unnecessary capture:**
- Network logging can be disabled for simpler workflows
- Console logging can be filtered to errors only

---

## Where Logs and Screenshots Are Stored

### Default Locations

**Linux:**
```
~/.config/vrooli-ascension/
├── logs/                    # Application logs
├── data/
│   ├── workflows/           # Workflow definitions
│   ├── executions/          # Execution history
│   └── screenshots/         # Captured screenshots
└── settings.json            # User preferences
```

**macOS:**
```
~/Library/Application Support/vrooli-ascension/
├── logs/
├── data/
│   ├── workflows/
│   ├── executions/
│   └── screenshots/
└── settings.json
```

**Windows:**
```
%APPDATA%\vrooli-ascension\
├── logs\
├── data\
│   ├── workflows\
│   ├── executions\
│   └── screenshots\
└── settings.json
```

### Configuring Storage Location

The `BAS_SCREENSHOTS_ROOT` environment variable overrides the default screenshot directory:

```bash
export BAS_SCREENSHOTS_ROOT=/path/to/custom/screenshots
```

For object storage (advanced), configure MinIO with `BAS_SCREENSHOT_STORAGE=minio` and appropriate `MINIO_*` environment variables.

---

## Finding Documentation

### In-App Documentation
- Click the **Help** menu → **Documentation**
- Context-sensitive help is available via **?** icons throughout the interface

### Online Documentation
- **Main repository:** [github.com/Vrooli/Vrooli](https://github.com/Vrooli/Vrooli)
- **Vrooli Ascension docs:** [github.com/Vrooli/Vrooli/tree/master/scenarios/browser-automation-studio/docs](https://github.com/Vrooli/Vrooli/tree/master/scenarios/browser-automation-studio/docs)

### Key Documentation Files

| File | Contents |
|------|----------|
| `README.md` | Overview, quick start, architecture |
| `docs/CONTROL_SURFACE.md` | API and CLI reference |
| `docs/ENVIRONMENT.md` | Environment variables and configuration |
| `docs/NODE_INDEX.md` | All workflow action types explained |
| `docs/nodes/*.md` | Detailed documentation for each action type |

---

## When to Escalate

### Self-Service First
Most issues can be resolved using:
1. This troubleshooting guide
2. Documentation in the GitHub repository
3. Application logs for error details

### Submit Feedback When
- You've followed troubleshooting steps but the issue persists
- You encounter a crash or data loss
- You find a security vulnerability
- The documentation doesn't cover your situation

### How to Escalate

1. Go to [vrooli.com/feedback](https://vrooli.com/feedback)
2. Select the appropriate category:
   - **Bug report** for technical issues
   - **Feature request** for missing functionality
   - **Refund request** for billing issues
   - **General feedback** for other matters
3. Include:
   - Operating system and version
   - Vrooli Ascension version (Help → About)
   - Steps to reproduce the issue
   - Relevant log snippets (see "Where Logs Are Stored")
   - Screenshots of error messages

### Response Times
- Bug reports and feature requests: reviewed weekly
- Refund requests: processed within 5 business days
- Security issues: prioritized for immediate review

---

## Quick Troubleshooting Checklist

- [ ] Is the app the latest version? (Help → Check for Updates)
- [ ] Is there enough disk space for screenshots?
- [ ] Are selectors still valid for failing workflows?
- [ ] Have you checked the execution logs and screenshots?
- [ ] Have you consulted the documentation?
- [ ] Can you reproduce the issue consistently?
- [ ] Have you checked for similar issues in the repository?

If all else fails, submit at [vrooli.com/feedback](https://vrooli.com/feedback) with details.
