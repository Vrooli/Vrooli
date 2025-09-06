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