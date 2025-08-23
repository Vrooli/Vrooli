# App Debugger

## Purpose
Professional debugging and error analysis platform for Vrooli applications. Provides real-time monitoring, error analysis, performance profiling, and automated fix suggestions for all generated apps.

## Key Features
- **Real-time Error Monitoring**: Captures and analyzes errors from all running Vrooli apps
- **Intelligent Fix Suggestions**: AI-powered root cause analysis and solution recommendations
- **Performance Profiling**: Tracks performance metrics and identifies bottlenecks
- **Debug Session Management**: Stores complete debugging history with searchable logs
- **Cross-App Integration**: Works seamlessly with all Vrooli-generated applications

## Dependencies
### Required Resources
- **n8n**: Workflow automation for error analysis and fix generation
- **PostgreSQL**: Stores debug sessions, error logs, and performance metrics
- **Redis**: Real-time caching of debugging data and active sessions
- **QuestDB** (optional): Time-series storage for performance trends

### Shared Workflows
- `ollama.json`: AI reasoning for error analysis and fix suggestions
- `rate-limiter.json`: Prevents overwhelming resources during bulk error processing

## Cross-Scenario Impact
App Debugger is a foundational development tool that:
- **Accelerates Development**: Reduces debugging time for all scenario developers
- **Improves Quality**: Helps identify common error patterns across scenarios
- **Enables Learning**: Builds a knowledge base of fixes that benefit all future apps
- **Supports Production**: Can be deployed alongside production apps for monitoring

## Integration Points
### For Other Scenarios
- Call `/api/apps/{app_name}/errors` to retrieve error history
- Submit errors via `/api/errors` endpoint for analysis
- Use CLI: `app-debugger analyze <error-log-file>`
- Query fix suggestions: `app-debugger suggest-fix "<error-message>"`

### Internal APIs
- `GET /api/apps` - List all monitored applications
- `POST /api/errors` - Submit error for analysis
- `GET /api/errors/{id}/suggestions` - Get fix suggestions
- `GET /api/performance/{app_name}` - Get performance metrics
- `POST /api/debug-session` - Start new debug session

## UI Style
**Professional Development Theme**: Clean, modern interface with dark mode support. Organized into clear sections for errors, performance, and logs. Uses monospace fonts for code display, syntax highlighting for stack traces, and color-coded severity indicators.

## Workflows
1. **error-analyzer.json**: Analyzes error messages and stack traces
2. **fix-suggester.json**: Generates targeted fix recommendations
3. **log-monitor.json**: Monitors and categorizes application logs
4. **performance-profiler.json**: Tracks and analyzes performance metrics
5. **debug-orchestrator.json**: Coordinates complex debugging sessions

## Testing
```bash
# Test API endpoints
vrooli scenario test app-debugger

# Manual testing
app-debugger analyze "TypeError: Cannot read property 'map' of undefined"
app-debugger suggest-fix "ECONNREFUSED localhost:5432"
```

## Future Enhancements
- Integration with code-sleuth for automatic code navigation
- Historical trend analysis for recurring issues
- Automated fix application via git-manager
- Performance regression detection