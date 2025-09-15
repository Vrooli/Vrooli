# Pandas AI Resource

## Overview
Pandas AI provides conversational AI-powered data analysis infrastructure for Vrooli, enabling natural language queries over structured data.

## Quick Start
```bash
# Install and start
vrooli resource pandas-ai manage install
vrooli resource pandas-ai manage start

# Check status
vrooli resource pandas-ai status

# Test the service
curl -X POST http://localhost:8095/analyze \
  -H "Content-Type: application/json" \
  -d '{"query": "describe the data", "data": {"sales": [100,200,300], "month": [1,2,3]}}'
```

## Key Features
- **Natural Language Analysis**: Query data using plain English
- **Direct Pandas Execution**: Execute raw pandas code for advanced operations
- **Multi-Source Support**: CSV, Excel, JSON, PostgreSQL, Redis, MongoDB
- **Automated Visualizations**: Generates charts and graphs automatically
- **Report Generation**: Creates professional data reports
- **Code Generation**: Converts queries to pandas code for transparency
- **Safety Controls**: Built-in safety checks for direct code execution

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

## API Usage Examples

### Basic Data Analysis
```bash
# Describe data
curl -X POST http://localhost:8095/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "query": "describe the data",
    "data": {
      "product": ["A", "B", "C"],
      "sales": [100, 150, 200],
      "profit": [20, 35, 45]
    }
  }'

# Calculate mean values
curl -X POST http://localhost:8095/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "query": "calculate mean",
    "data": {
      "temperature": [20, 22, 21, 23, 22],
      "humidity": [45, 50, 48, 52, 49]
    }
  }'

# Get summary statistics
curl -X POST http://localhost:8095/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "query": "summary",
    "data": {
      "scores": [85, 90, 78, 92, 88, 95, 82]
    }
  }'
```

### CSV File Analysis
```bash
# Analyze a CSV file
curl -X POST http://localhost:8095/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "query": "show top 5 rows",
    "csv_path": "/path/to/data.csv"
  }'
```

### Direct Pandas Code Execution (Enhanced)
```bash
# Execute raw pandas code for advanced operations
curl -X POST http://localhost:8095/pandas/execute \
  -H "Content-Type: application/json" \
  -d '{
    "code": "import pandas as pd\ndf = pd.DataFrame({\"A\": [1,2,3], \"B\": [4,5,6]})\nresult = df.corr()",
    "safe_mode": true
  }'

# With provided data
curl -X POST http://localhost:8095/pandas/execute \
  -H "Content-Type: application/json" \
  -d '{
    "data": [{"name": "Alice", "score": 85}, {"name": "Bob", "score": 90}],
    "code": "result = df.groupby(\"name\")[\"score\"].mean()",
    "safe_mode": true
  }'

# Generate matplotlib plots (returns base64 PNG)
curl -X POST http://localhost:8095/pandas/execute \
  -H "Content-Type: application/json" \
  -d '{
    "data": [{"x": 1, "y": 2}, {"x": 2, "y": 4}, {"x": 3, "y": 6}],
    "code": "import matplotlib.pyplot as plt\nplt.figure()\nplt.plot(df[\"x\"], df[\"y\"])\nplt.title(\"My Plot\")",
    "safe_mode": true
  }'

# Complex operations with visualizations
curl -X POST http://localhost:8095/pandas/execute \
  -H "Content-Type: application/json" \
  -d '{
    "data": {"dates": ["2024-01-01", "2024-01-02"], "values": [100, 150]},
    "code": "df[\"dates\"] = pd.to_datetime(df[\"dates\"])\ndf[\"rolling\"] = df[\"values\"].rolling(2).mean()\nresult = df",
    "safe_mode": true
  }'
```

#### Direct Execution Features
- **Safe Mode**: Enhanced safety checks (blocks file, network, subprocess operations)
- **Timeout Protection**: 10-second execution timeout prevents infinite loops
- **Plot Support**: Returns matplotlib figures as base64-encoded PNG images
- **Smart Caching**: Results cached for repeated queries (1-hour TTL)
- **Auto-detection**: Automatically detects output type (DataFrame, Series, scalar, plot)
- **Context Variables**: Pre-loaded with `pd`, `np`, `df`, `plt`, `sns`
- **Performance Tracking**: Returns execution time for optimization
- **Error Handling**: Detailed error messages with optional traceback

#### Cache Management
```bash
# Check cache statistics with hit rates
curl http://localhost:8095/cache/stats

# Clear cache and reset statistics
curl -X POST http://localhost:8095/cache/clear
```

### Data Profiling (NEW)
```bash
# Basic data profiling
curl -X POST http://localhost:8095/data/profile \
  -H "Content-Type: application/json" \
  -d '{
    "data": [
      {"id": 1, "value": 100, "category": "A"},
      {"id": 2, "value": 150, "category": "B"},
      {"id": 3, "value": null, "category": "A"}
    ],
    "profile_type": "basic"
  }'

# Advanced profiling with quality scoring
curl -X POST http://localhost:8095/data/profile \
  -H "Content-Type: application/json" \
  -d '{
    "csv_path": "/path/to/data.csv",
    "profile_type": "advanced"
  }'

# Correlation analysis
curl -X POST http://localhost:8095/data/profile \
  -H "Content-Type: application/json" \
  -d '{
    "data": {"x": [1,2,3,4,5], "y": [2,4,6,8,10], "z": [1,1,2,2,3]},
    "profile_type": "correlation"
  }'

# Distribution analysis
curl -X POST http://localhost:8095/data/profile \
  -H "Content-Type: application/json" \
  -d '{
    "data": {"values": [1,2,2,3,3,3,4,4,5]},
    "profile_type": "distribution"
  }'
```

#### Data Profiling Features
- **Basic Profile**: Column types, missing data, basic statistics
- **Advanced Profile**: Outlier detection, data quality scoring, cardinality analysis
- **Correlation Profile**: Correlation matrix, strong correlations identification
- **Distribution Profile**: Histogram bins, normality tests, skewness/kurtosis
- **Smart Recommendations**: Actionable insights for data improvement
- **Quality Scoring**: 0-100 scale automatic data quality assessment
- **Memory Profiling**: Dataset memory usage estimation

### Available Query Operations
- `describe` or `summary` - Get comprehensive data overview
- `mean` - Calculate mean values
- `sum` - Calculate sum of columns
- `count` - Count non-null values
- `max` - Find maximum values
- `min` - Find minimum values
- `head` - Show first 5 rows
- `tail` - Show last 5 rows
- `columns` - List column names

## CLI Reference

### Lifecycle Management
```bash
# Install the resource
vrooli resource pandas-ai manage install

# Start the service
vrooli resource pandas-ai manage start --wait

# Stop the service
vrooli resource pandas-ai manage stop

# Restart the service
vrooli resource pandas-ai manage restart

# Uninstall completely
vrooli resource pandas-ai manage uninstall
```

### Testing
```bash
# Run smoke tests (quick health check)
vrooli resource pandas-ai test smoke

# Run all tests
vrooli resource pandas-ai test all
```

### Status & Monitoring
```bash
# Check service status
vrooli resource pandas-ai status

# View logs
vrooli resource pandas-ai logs

# Get resource info
vrooli resource pandas-ai info --json
```

## Integration with Other Resources

### PostgreSQL Integration
```python
# Example: Analyzing PostgreSQL data
import requests

# Query data from PostgreSQL through Pandas-AI
response = requests.post('http://localhost:8095/analyze', json={
    'query': 'show customer revenue by month',
    'csv_path': '/path/to/exported/postgres/data.csv'
})
```

### Redis Integration
```python
# Caching analysis results in Redis
# Pandas-AI can read cached datasets from Redis exports
```

### SageMath Integration
```python
# Use Pandas-AI for data preparation before mathematical modeling
# 1. Analyze and clean data with Pandas-AI
# 2. Export results for SageMath processing
# 3. Combine statistical analysis with symbolic math
```

### Integration Best Practices
1. **Data Pipeline**: PostgreSQL → Pandas-AI → Visualization
2. **Caching Strategy**: Use Redis for frequently accessed analysis results
3. **Math Workflows**: Pandas-AI for statistics → SageMath for advanced math
4. **Report Generation**: Pandas-AI analysis → N8n for automated reports

## Performance Optimization

### Configuration
```bash
# Set analysis timeout (default: 10 seconds)
export PANDAS_AI_TIMEOUT=15

# Configure max workers for concurrent requests
export PANDAS_AI_MAX_WORKERS=4
```

### Tips for Large Datasets
1. **Use sampling**: For initial exploration, use a subset of data
2. **Chunk processing**: Break large files into smaller chunks
3. **Cache results**: Store frequently used analysis results
4. **Optimize queries**: Use specific operations (mean, sum) instead of generic "analyze"

## Troubleshooting

### Common Issues

**Service not responding**
```bash
# Check if service is running
vrooli resource pandas-ai status

# Restart if needed
vrooli resource pandas-ai manage restart
```

**Analysis timeout**
```bash
# Increase timeout for complex queries
export PANDAS_AI_TIMEOUT=30
vrooli resource pandas-ai manage restart
```

**Memory issues with large datasets**
```bash
# Monitor memory usage
htop

# Consider using data sampling or chunking
```
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
Service configuration is stored in `${HOME}/.pandas-ai/config.yaml`:
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