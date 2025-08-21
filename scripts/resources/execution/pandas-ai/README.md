# Pandas AI Resource

## Overview
Pandas AI provides conversational AI-powered data analysis infrastructure for Vrooli, enabling natural language queries over structured data.

## Quick Start
```bash
# Install and start
vrooli resource install pandas-ai
vrooli resource start pandas-ai

# Check status
resource-pandas-ai status

# Analyze data
resource-pandas-ai analyze --file data.csv --query "Show me the top 5 products by revenue"
```

## Key Features
- **Natural Language Analysis**: Query data using plain English
- **Multi-Source Support**: CSV, Excel, JSON, PostgreSQL, Redis, MongoDB
- **Automated Visualizations**: Generates charts and graphs automatically
- **Report Generation**: Creates professional data reports
- **Code Generation**: Converts queries to pandas code for transparency

## Architecture
```
┌─────────────────────────────────┐
│       Pandas AI Service         │
├─────────────────────────────────┤
│  • Natural Language Processing  │
│  • Code Generation Engine       │
│  • Visualization Generator      │
│  • Data Connector Layer         │
└─────────────────────────────────┘
           │
    ┌──────┴──────┐
    ▼             ▼
┌────────┐   ┌────────┐
│Database│   │  Files │
│Sources │   │ (CSV,  │
│(PG,Redis)  │ Excel) │
└────────┘   └────────┘
```

## CLI Reference

### Basic Commands
- `start` - Start the Pandas AI service
- `stop` - Stop the service
- `status` - Check service health and configuration
- `install` - Install Pandas AI and dependencies
- `restart` - Restart the service

### Content Management
- `content add --file <script.py>` - Add Python analysis script
- `content list` - List stored scripts
- `content get --name <script>` - Retrieve a script
- `content remove --name <script>` - Remove a script
- `content execute --name <script>` - Execute stored script

### Analysis Commands
- `analyze --file <data> --query <question>` - Analyze data with natural language
- `analyze --db postgres --table <table> --query <question>` - Query database tables
- `visualize --file <data> --type <chart>` - Generate visualizations

## Configuration
Service configuration is stored in `/home/matthalloran8/.pandas-ai/config.yaml`:
```yaml
port: 8095
host: 0.0.0.0
api_key: ${PANDAS_AI_API_KEY}
default_llm: openai/gpt-3.5-turbo
cache_enabled: true
max_retries: 3
```

## Integration Examples

### With PostgreSQL
```bash
# Query database tables
resource-pandas-ai analyze \
  --db postgres \
  --table sales \
  --query "What were the top selling products last month?"
```

### With CSV Files
```bash
# Analyze CSV data
resource-pandas-ai analyze \
  --file sales_data.csv \
  --query "Show monthly revenue trends with a line chart"
```

### With Other Resources
```bash
# Combine with Qdrant for semantic search
resource-qdrant search --collection products --query "electronics" | \
  resource-pandas-ai analyze --query "Summarize pricing patterns"
```

## Troubleshooting

### Service Won't Start
- Check port 8095 is available: `ss -tlnp | grep 8095`
- Verify Docker is running: `docker ps`
- Check logs: `resource-pandas-ai logs`

### Analysis Errors
- Ensure data file exists and is readable
- Verify database credentials are configured
- Check API key is set for LLM provider

### Performance Issues
- Monitor memory usage: `docker stats pandas-ai`
- Consider chunking large datasets
- Enable caching in configuration

## Security Considerations
- API keys are stored in Vault when available
- Database connections use secure credentials
- Analysis scripts are sandboxed in Docker container
- Network access restricted to localhost by default

## Related Resources
- [PostgreSQL](../../../storage/postgres/) - Primary data source
- [Redis](../../../storage/redis/) - Cache and session storage
- [Qdrant](../../../storage/qdrant/) - Vector search integration
- [OpenRouter](../../../ai/openrouter/) - LLM provider
- [Ollama](../../../ai/ollama/) - Local LLM alternative

## Support
For issues or questions, check:
- Service logs: `resource-pandas-ai logs`
- Docker status: `docker ps | grep pandas-ai`
- PRD requirements: [PRD.md](./PRD.md)