# Claude Code Anomaly Check Prompt

## Overview
This directory contains the prompt template used by the System Monitor's "Run Anomaly Check" feature to interact with Claude Code.

## Hot-Reloading Feature
The prompt file (`anomaly-check.md`) is **hot-reloadable**, meaning you can edit it while the System Monitor app is running and changes will take effect immediately on the next anomaly check - no restart required!

## File Location
- **Source**: `scenarios/system-monitor/initialization/claude-code/anomaly-check.md`
- **Deployed App**: `${HOME}/Vrooli/scenarios/system-monitor/initialization/claude-code/anomaly-check.md`

Both locations are checked by the API, so you can edit either file.

## How to Customize

1. **Edit the prompt file** while the app is running:
   ```bash
   # Edit in source location
   nano scenarios/system-monitor/initialization/claude-code/anomaly-check.md
   
   # OR edit in deployed location
   nano ~/Vrooli/scenarios/system-monitor/initialization/claude-code/anomaly-check.md
   ```

2. **Test your changes** immediately:
   - Open the System Monitor UI: http://localhost:3000
   - Click "RUN ANOMALY CHECK"
   - The new prompt will be used automatically

## Template Variables
The following placeholders are automatically replaced with real-time data:
- `{{CPU_USAGE}}` - Current CPU usage percentage
- `{{MEMORY_USAGE}}` - Current memory usage percentage
- `{{TCP_CONNECTIONS}}` - Number of active TCP connections
- `{{TIMESTAMP}}` - Current timestamp

## Prompt Structure
The prompt is organized into sections:
1. **System Metrics** - Current system state
2. **Investigation Objectives** - What to analyze
3. **Response Format** - How Claude should structure the response
4. **Guidelines** - Best practices for the investigation

## Tips for Customization
- Add specific log files or services relevant to your environment
- Adjust the response format to match your reporting needs
- Include custom checks for your specific use cases
- Add domain-specific knowledge or context
- Modify severity thresholds based on your system's normal behavior

## Example Customizations

### Add Docker container analysis:
```markdown
### 6. Container Analysis
- List all running Docker containers
- Check container resource usage
- Review container logs for errors
```

### Add database health checks:
```markdown
### 7. Database Health
- Check PostgreSQL connection count
- Review slow query logs
- Monitor replication lag
```

### Customize response format:
```markdown
**Alert Level**: [None/Info/Warning/Critical/Emergency]
**Ticket Required**: [Yes/No]
**Estimated Resolution Time**: [X minutes/hours]
```

## Development Workflow
1. Make changes to the prompt file
2. Save the file
3. Trigger an anomaly check in the UI
4. Review the results
5. Iterate on the prompt as needed

No compilation, no restarts, instant feedback! 🚀