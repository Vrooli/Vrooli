# API Library

A comprehensive API discovery and knowledge management system that provides Vrooli with permanent institutional memory of external APIs, their capabilities, pricing, and integration patterns.

## 🎯 Purpose

The API Library transforms API discovery from ad-hoc searching to systematic knowledge retrieval. It enables scenarios to programmatically discover "what tool can solve X problem" using semantic search, while maintaining institutional knowledge of integration gotchas, pricing, and successful patterns.

## 🚀 Quick Start

### Prerequisites
- PostgreSQL for structured data storage
- Qdrant for semantic search capabilities (optional but recommended)
- Node.js for the UI

### Installation

1. **Install the CLI**:
```bash
cd cli
./install.sh
```

2. **Start the services**:
```bash
vrooli scenario run api-library
```

3. **Access the UI**:
Open http://localhost:3200 in your browser

## 💡 Core Features

### Semantic Search
- Natural language queries: "send SMS messages", "process payments"
- Relevance scoring based on capabilities
- Filter by configuration status, pricing, categories

### Knowledge Management
- **Notes & Gotchas**: Capture integration experiences
- **Metadata Tracking**: Source URLs, update timestamps
- **Relationship Mapping**: Alternative APIs, deprecation tracking
- **Configuration Status**: Know which APIs are ready to use
- **Version Tracking**: Monitor API version changes and breaking changes
- **Automatic Pricing Updates**: Refreshes pricing data every 24 hours

### Cost Optimization
- **Usage-Based Calculator**: Estimate costs based on your usage patterns
- **Alternative Recommendations**: Find cheaper APIs with similar capabilities
- **Savings Tips**: Get recommendations for reducing API costs

### Code Generation Integration
- **API Specifications**: Structured specs optimized for code generation
- **Language Templates**: Pre-built client templates for Python, JavaScript, Go
- **SDK Detection**: Identifies APIs with official SDKs available
- **Auth Patterns**: Provides authentication templates for different auth types

### Monitoring & Notifications
- **Webhook Subscriptions**: Get notified when APIs are added, updated, or deprecated
- **Health Monitoring**: Track API availability and response times
- **Usage Analytics**: Monitor your API usage patterns and trends
- **Tier Optimization**: Automatically recommend the best pricing tier

### Research Integration
- Request automated research for new APIs
- Integrates with research-assistant scenario
- Continuous discovery and library expansion

## 🔧 CLI Usage

### Search for APIs
```bash
# Find APIs for sending emails
api-library search "send emails"

# Get JSON output
api-library search "payment processing" --json
```

### View API Details
```bash
api-library show <api-id>
```

### Add Integration Notes
```bash
# Add a gotcha
api-library add-note <api-id> "Rate limits reset at midnight UTC" gotcha

# Add a tip
api-library add-note <api-id> "Use batch endpoints for better performance" tip
```

### List Configured APIs
```bash
api-library list-configured
```

### Request Research
```bash
api-library request-research "video transcription services with good accuracy"
```

## 🌐 API Endpoints

### Search
`POST /api/v1/search`
```json
{
  "query": "send notifications",
  "limit": 10,
  "filters": {
    "configured": true,
    "max_price": 0.01
  }
}
```

### Get API Details
`GET /api/v1/apis/:id`

### Add Note
`POST /api/v1/apis/:id/notes`
```json
{
  "content": "Webhook implementation is flaky, use polling instead",
  "type": "gotcha"
}
```

### Request Research
`POST /api/v1/request-research`
```json
{
  "capability": "OCR for handwritten documents",
  "requirements": {
    "accuracy": "high",
    "languages": ["en", "es", "fr"]
  }
}
```

### Calculate Cost
`POST /api/v1/calculate-cost`
```json
{
  "api_id": "api-uuid-here",
  "requests_per_month": 10000,
  "data_per_request_mb": 0.5
}
```

Returns optimized tier recommendation and alternatives.

### Track API Version
`POST /api/v1/apis/:id/versions`
```json
{
  "version": "2.0.0",
  "change_summary": "Breaking changes in authentication",
  "breaking_changes": true
}
```

### Get Version History
`GET /api/v1/apis/:id/versions`

Returns all version changes with timestamps and breaking change flags.

### Webhook Management
`POST /api/v1/webhooks`
```json
{
  "url": "https://your-endpoint.com/webhook",
  "events": ["api.created", "api.updated", "api.deprecated"],
  "description": "Notify on API changes"
}
```

### Code Generation Support

#### Get API Specification for Code Generation
`GET /api/v1/codegen/apis/:id`

Returns structured API specification with endpoints, snippets, and generation hints.

#### Search APIs for Code Generation
`POST /api/v1/codegen/search`
```json
{
  "capability": "payment processing",
  "languages": ["python", "javascript"],
  "max_price": 0.10
}
```

#### Get Language Templates
`GET /api/v1/codegen/templates/:language`

Available languages: `python`, `javascript`, `go`

Returns client class templates and authentication patterns for the specified language.

`GET /api/v1/webhooks` - List all webhook subscriptions
`DELETE /api/v1/webhooks/:id` - Remove a webhook subscription

### Health Monitoring
`GET /api/v1/apis/:id/health` - Get latest health check results
`POST /api/v1/apis/:id/health/check` - Trigger immediate health check

## 🏗️ Architecture

### Data Storage
- **PostgreSQL**: Structured metadata, pricing, relationships
- **Qdrant**: Vector embeddings for semantic search
- **Redis** (optional): Caching layer for frequently accessed data

### Components
- **Go API**: RESTful service with full CRUD operations
- **CLI**: Bash-based tool wrapping API endpoints
- **React UI**: Professional interface for browsing and management

## 🔄 Integration with Other Scenarios

### Provides To
- **All scenarios**: API discovery and metadata
- **auto-integration-builder**: API specifications for code generation
- **cost-optimizer**: Pricing data for optimization decisions

### Consumes From
- **research-assistant**: Automated API discovery and research

## 📊 Value Proposition

### Technical Value
- Eliminates redundant API research across scenarios
- 10-20 hours saved per scenario development
- Instant discovery of appropriate APIs for any capability

### Business Value
- $30K-50K typical value per enterprise deployment
- Reduces integration failures through captured gotchas
- Enables cost optimization through pricing transparency

## 🎨 UI Features

The web interface provides:
- **Search-first design**: Prominent search bar with semantic capabilities
- **Card-based browsing**: Visual API cards with key information
- **Detailed views**: Complete API information with documentation links
- **Notes management**: Add and view integration experiences
- **Configuration tracking**: See which APIs are ready to use

## 🧪 Testing

Run the test suite:
```bash
vrooli scenario test api-library
```

This validates:
- Database schema initialization
- API endpoint functionality
- CLI command execution
- Semantic search accuracy

## 📈 Future Enhancements

### Version 2.0
- Automatic pricing updates from source URLs
- API health monitoring and uptime tracking
- Integration code generation
- Cost calculator based on usage patterns

### Long-term Vision
- Central nervous system for all external integrations
- Self-updating through continuous research
- Predictive API recommendations
- Multi-region API suggestions

## 🔒 Security Notes

- API credentials are never stored directly
- Configuration status tracked separately from secrets
- All modifications logged with timestamps
- Read-only access by default

## 📝 Contributing

When adding new APIs:
1. Include complete metadata (description, capabilities, tags)
2. Add pricing information if available
3. Document any known gotchas or tips
4. Set appropriate category and status

## 🆘 Troubleshooting

### Search not returning results
- Check PostgreSQL full-text search indexes
- Verify search_vector column is populated
- Try simpler search terms

### Qdrant connection issues
- Semantic search will fall back to PostgreSQL full-text search
- Check Qdrant service is running on correct port

### Research requests not processing
- Verify research-assistant scenario is available
- Check research_requests table for status

## 📚 References

- [PRD.md](PRD.md) - Complete product requirements
- [API Documentation](docs/api.md) - Full API specification
- [CLI Reference](docs/cli.md) - Detailed CLI commands