# Competitor Change Monitor

## Purpose
Tracks changes in competitor websites, GitHub repositories, and public data sources. Automatically detects and analyzes changes in pricing, features, marketing, and legal content. Provides real-time alerts for high-impact changes that require strategic response.

## Cross-Scenario Impact
- **High Reusability**: Monitoring capabilities can be used by other business intelligence scenarios
- **API Integration**: Exposes change detection and analysis APIs for other scenarios to leverage
- **Data Provider**: Feeds competitive intelligence to product-manager-agent and roi-fit-analysis scenarios

## Key Features
- **Multi-Source Monitoring**: Tracks websites, GitHub repos, and public APIs
- **Intelligent Analysis**: Uses AI to categorize changes and assess business impact
- **Alert System**: Configurable thresholds for different types of changes
- **Historical Tracking**: Maintains database of all changes for trend analysis
- **Scheduled Scanning**: Automated periodic checks with configurable intervals

## Dependencies
### Resources
- **PostgreSQL**: Stores competitor data and change history
- **Ollama**: AI analysis of detected changes
- **Browserless**: Alternative web scraping when Huginn is unavailable
- **Agent-S2**: Fallback for complex scraping scenarios

### Workflow Logic
- Internal scheduler and API handlers manage scans, analysis, and alerting (no external workflow engine required)

## UI Style
Professional business intelligence dashboard with:
- Dark theme with accent colors for alert levels (green/yellow/red/critical)
- Clean data tables with sortable columns
- Timeline view of competitor changes
- Real-time notification panel

## Integration Points
### CLI Commands
```bash
competitor-monitor add <name> <url>
competitor-monitor list
competitor-monitor scan [competitor-id]
competitor-monitor analyze <competitor-id>
competitor-monitor alerts [--since date]
```

### API Endpoints
- `POST /api/competitors` - Add new competitor
- `GET /api/competitors` - List all competitors
- `POST /api/scan/{id}` - Trigger manual scan
- `GET /api/changes` - Get recent changes
- `GET /api/analysis/{id}` - Get change analysis

## Workflow Architecture
1. **Scheduler**: Periodically checks all monitored sources
2. **Website monitor**: Detects changes in web content
3. **GitHub monitor**: Tracks repository changes
4. **Change analyzer**: AI-powered analysis of detected changes
5. **Alert manager**: Sends notifications for high-impact changes

## Testing
- Validate with mock competitor data
- Test change detection with controlled website updates
- Verify alert thresholds trigger correctly
- Check database persistence of change history

## Business Value
- **Early Warning System**: Detect competitor moves before they impact market
- **Strategic Planning**: Data-driven competitive intelligence
- **Market Positioning**: Track how competitors position against us
- **Pricing Intelligence**: Monitor competitor pricing changes in real-time
