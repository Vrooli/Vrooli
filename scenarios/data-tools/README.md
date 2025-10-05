# Data Tools - Enterprise Data Processing Platform

> **Comprehensive data processing and transformation toolkit for parsing, validating, querying, and streaming data**

## 🎯 Overview

Data Tools provides a complete data processing platform with multi-format parsing, schema validation, SQL query execution, and real-time streaming capabilities. Built as the foundational data manipulation layer for Vrooli scenarios.

### Value Proposition
Eliminates the need for custom data processing implementations by providing a unified platform for:
- **Data Parsing**: CSV, JSON, XML with automatic schema inference
- **Data Validation**: Quality assessment, anomaly detection, schema compliance
- **SQL Queries**: Execute queries on datasets with optimization
- **Stream Processing**: Real-time data ingestion and transformation

### Target Markets
- Data engineers building ETL pipelines
- Business intelligence teams needing data transformation
- Developers requiring data quality checks
- Analytics platforms processing structured data

### Revenue Potential
- **Range**: $25,000 - $75,000 per enterprise deployment
- **Market Demand**: High - every business needs data processing
- **Pricing Model**: Usage-based (per GB processed) + enterprise SLA tiers

## 🏗️ Architecture

### System Components
```
┌─────────────────┐     ┌─────────────────┐
│   CLI Tool      │────▶│   Go API Server │
│  (data-tools)   │     │  (Port: 19914)  │
└─────────────────┘     └─────────────────┘
                                │
                ┌───────────────┼───────────────┐
                ▼               ▼               ▼
        ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
        │  PostgreSQL │  │    Redis    │  │   Windmill  │
        │  (Storage)  │  │  (Cache)    │  │  (Workflows)│
        └─────────────┘  └─────────────┘  └─────────────┘
```

### Required Resources
- **PostgreSQL**: Dataset metadata, transformation history, query cache
- **Redis**: Query result caching, session state, streaming buffers
- **Windmill**: Workflow automation for data pipelines

### Optional Resources
- **MinIO**: Object storage for datasets > 100MB
- **Qdrant**: Vector storage for semantic search
- **Ollama**: Natural language query processing

## 📄 API Endpoints

Base URL: `http://localhost:19914/api/v1`

Authentication: `Authorization: Bearer data-tools-secret-token`

### Health Check
```bash
curl http://localhost:19914/health
```

### Data Parsing
```bash
curl -X POST http://localhost:19914/api/v1/data/parse \
  -H "Authorization: Bearer data-tools-secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "data": "name,age\nJohn,30\nJane,25",
    "format": "csv",
    "options": {"headers": true, "infer_types": true}
  }'
```

### Data Validation
```bash
curl -X POST http://localhost:19914/api/v1/data/validate \
  -H "Authorization: Bearer data-tools-secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "data": [{"name": "John", "age": 30}],
    "schema": {
      "columns": [
        {"name": "name", "type": "string"},
        {"name": "age", "type": "number"}
      ]
    }
  }'
```

### SQL Query Execution
```bash
curl -X POST http://localhost:19914/api/v1/data/query \
  -H "Authorization: Bearer data-tools-secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "sql": "SELECT * FROM dataset WHERE age > 25"
  }'
```

### Stream Creation
```bash
curl -X POST http://localhost:19914/api/v1/data/stream/create \
  -H "Authorization: Bearer data-tools-secret-token" \
  -H "Content-Type: application/json" \
  -d '{
    "source_config": {
      "type": "webhook",
      "connection": {"endpoint": "/webhooks/data"}
    },
    "processing_rules": [
      {"type": "validate", "parameters": {"schema": "..."}}
    ]
  }'
```

## 🚀 Quick Start

### 1. Setup and Build
```bash
# Navigate to scenario directory
cd scenarios/data-tools

# Start the scenario (builds API, installs CLI, initializes database)
make run

# Or use vrooli CLI
vrooli scenario start data-tools
```

### 2. Verify Installation
```bash
# Check API health
curl http://localhost:19914/health

# Verify CLI installation
data-tools --help

# Check scenario status
make status
```

### 3. Use the CLI
```bash
# Parse data from file or stdin
data-tools parse data.csv --format csv --headers
echo "name,age\nAlice,28" | data-tools parse - --format csv

# Validate data quality
data-tools validate dataset-id --schema schema.json

# Execute SQL queries
data-tools query "SELECT COUNT(*) FROM dataset"

# Transform data
data-tools transform dataset-id "SELECT * WHERE age > 25" --output filtered.json

# Manage streams
data-tools stream create --source kafka --config config.json
data-tools stream list
```

### 4. Access API Directly
```bash
# Health check (no auth required)
curl http://localhost:19914/health

# Parse CSV data
curl -X POST http://localhost:19914/api/v1/data/parse \
  -H "Authorization: Bearer data-tools-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"data":"name,age\nJohn,30","format":"csv"}'
```

## 📁 Project Structure

```
data-tools/
├── .vrooli/
│   └── service.json          # Service configuration
├── api/                      # Go API server
│   ├── cmd/server/
│   │   ├── main.go          # Server entry point
│   │   └── data_handlers.go # Data processing handlers
│   ├── internal/
│   │   └── dataprocessor/   # Data processing logic
│   ├── go.mod
│   └── data-tools-api       # Compiled binary
├── cli/                     # Command-line interface
│   ├── data-tools           # CLI script
│   ├── install.sh           # CLI installer
│   └── cli-tests.bats       # CLI test suite
├── initialization/
│   ├── automation/windmill/ # Windmill workflows
│   ├── configuration/       # App configuration
│   └── storage/postgres/    # Database schema
│       ├── schema.sql       # Table definitions
│       └── seed.sql         # Initial data
├── PRD.md                   # Product requirements
├── PROBLEMS.md              # Known issues
└── README.md                # This file
```

## 🧪 Testing & Validation

### Run All Tests
```bash
# Run comprehensive test suite
make test

# This executes:
# - Go compilation test
# - Go unit tests
# - API health check
# - CLI command tests
```

### Manual Testing
```bash
# Test API endpoints
curl http://localhost:19914/health
curl -H "Authorization: Bearer data-tools-secret-token" \
     -X POST http://localhost:19914/api/v1/data/parse \
     -d '{"data":"test,data\n1,2","format":"csv"}'

# Test CLI commands
data-tools parse test.csv --format csv
data-tools validate dataset-id
data-tools query "SELECT * FROM dataset"
```

### View Logs
```bash
# View scenario logs
make logs

# View specific lifecycle step logs
vrooli scenario logs data-tools --step start-api
```

## 📊 Performance Expectations

### Response Times
- **Parse < 10MB**: < 200ms
- **Validate dataset**: < 100ms
- **SQL queries**: < 50ms (simple), < 500ms (complex)
- **Stream latency**: < 100ms end-to-end

### Throughput
- **Processing**: 100,000 rows/second
- **Concurrent queries**: 50+ simultaneous
- **Memory efficiency**: < 2x dataset size in RAM

### Resource Usage
- **API Server**: ~100MB RAM, minimal CPU
- **Database**: ~500MB with metadata
- **Redis Cache**: ~50MB typical

## 🔒 Security & Compliance

### Built-in Security
- Bearer token authentication for API
- SQL injection prevention
- Input validation on all endpoints
- Audit logging for transformations
- Column-level encryption support

### Production Checklist
- [ ] Change default API token (`DATA_TOOLS_API_TOKEN`)
- [ ] Enable HTTPS/TLS for API
- [ ] Set up database backups
- [ ] Configure Redis persistence
- [ ] Enable comprehensive audit logging
- [ ] Review data retention policies

## 🔧 CLI Commands Reference

```bash
# Core Commands
data-tools parse <input> --format <csv|json|xml>    # Parse and analyze data
data-tools transform <dataset> <operation>          # Transform datasets
data-tools validate <dataset> --schema <file>       # Validate data quality
data-tools query <sql>                              # Execute SQL queries
data-tools stream <create|list|start|stop>          # Manage streams

# Management Commands
data-tools health                                   # Check service health
data-tools list                                     # List datasets
data-tools get <dataset-id>                         # Get dataset details
data-tools delete <dataset-id>                      # Delete dataset
data-tools docs                                     # Show API documentation

# Options
--help, -h       Show help for any command
--json           Output in JSON format
```

## 💡 Usage Examples

### Parse CSV with Schema Inference
```bash
data-tools parse sales.csv --format csv --headers --infer-types
```

### Validate Data Quality
```bash
data-tools validate dataset-123 \
  --schema schema.json \
  --report quality-report.json
```

### Execute SQL Query
```bash
data-tools query "SELECT product, SUM(amount) FROM sales GROUP BY product"
```

### Transform and Export
```bash
data-tools transform dataset-123 \
  "SELECT * WHERE date >= '2025-01-01'" \
  --output filtered.json --format json
```

### Create Streaming Source
```bash
data-tools stream create \
  --source webhook \
  --config '{"endpoint":"/data/events"}' \
  --processing '{"validate":true,"schema":"event-schema.json"}'
```

## 🛟 Support & Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| API won't start | Check port 19914 not in use: `lsof -i :19914` |
| CLI not found | Re-run setup: `cd cli && ./install.sh` |
| Database errors | Verify PostgreSQL running: `vrooli resource status postgres` |
| Auth failures | Set token: `export DATA_TOOLS_API_TOKEN=your-token` |
| Parse failures | Check data format, try `--format auto` |

### View Logs
```bash
# Scenario logs
make logs

# Specific step logs
vrooli scenario logs data-tools --step start-api --follow

# Check health
data-tools health
```

### Database Access
```bash
# Connect to PostgreSQL
resource-postgres execute "SELECT * FROM datasets LIMIT 10"

# Check dataset metadata
resource-postgres execute "SELECT id, name, row_count FROM datasets"
```

## 🎯 Next Steps

### For Development
1. Review [PRD.md](PRD.md) for complete feature list
2. Check [PROBLEMS.md](PROBLEMS.md) for known limitations
3. Explore API endpoints with `data-tools docs`
4. Review database schema in `initialization/storage/postgres/schema.sql`

### For Integration
Use data-tools in your scenarios:
```bash
# From CLI
data-tools parse input.csv | your-scenario process

# From API
curl -X POST http://localhost:19914/api/v1/data/parse \
  -H "Authorization: Bearer data-tools-secret-token" \
  -d @request.json
```

### For Production
1. Update authentication token
2. Configure SSL/TLS
3. Set up monitoring and alerts
4. Plan data retention strategy
5. Configure backup procedures

## 📈 Roadmap

### Current (v1.0)
- ✅ Multi-format data parsing
- ✅ Schema validation and inference
- ✅ SQL query execution
- ✅ Data quality assessment
- ✅ Streaming data processing

### Planned (v2.0)
- [ ] MinIO integration for large datasets
- [ ] Advanced analytics (correlation, regression)
- [ ] Natural language queries via Ollama
- [ ] Distributed processing
- [ ] Advanced data governance

---

**Last Updated**: 2025-10-03
**Status**: Production Ready
**API Version**: v1
**Port**: 19914
