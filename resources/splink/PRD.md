# Product Requirements Document: Splink Record Linkage Library

## Executive Summary
**What**: UK government-developed Python library for probabilistic record linkage and deduplication at scale  
**Why**: Enable scenarios to perform entity resolution, data deduplication, and record matching across large datasets  
**Who**: Data-intensive scenarios requiring identity linking, customer data platforms, analytical data preparation  
**Value**: $30K+ per deployment - critical for data quality in enterprise systems  
**Priority**: P0 - Essential data processing capability for Vrooli ecosystem

## Requirements Checklist

### P0 Requirements (Must Have)
- [x] **Health Check Endpoint**: Service responds to health checks on port 8096
- [x] **Record Linkage API**: Execute probabilistic record linkage between datasets
- [x] **Deduplication API**: Identify and merge duplicate records within datasets
- [x] **DuckDB Backend**: Support local processing for datasets up to 10M records
- [x] **Unsupervised Learning**: Automatic parameter estimation without training data
- [x] **CLI Interface**: Command-line access to all linkage operations

### P1 Requirements (Should Have)
- [x] **Spark Integration**: Support for 100M+ record processing via Spark backend
- [x] **Interactive Visualization**: Web-based UI for exploring linkage results
- [x] **Batch Processing**: Queue-based batch job processing for large datasets
- [x] **PostgreSQL Integration**: Direct linkage of records in PostgreSQL databases

### P2 Requirements (Nice to Have)
- [x] **Real-time Linkage**: Stream processing for continuous record matching
- [x] **Custom Blocking Rules**: User-defined blocking strategies for performance optimization

## Technical Specifications

### Architecture
- **Core Engine**: Splink 3.x Python library with DuckDB backend
- **API Layer**: FastAPI for REST endpoints
- **Storage**: PostgreSQL for metadata, MinIO for dataset storage
- **Processing**: Async job queue with Redis for batch operations
- **Visualization**: Plotly-based interactive charts

### Dependencies
- Python 3.11+ runtime environment
- DuckDB for local SQL processing
- PostgreSQL for persistent storage
- Redis for job queue management
- MinIO for large dataset storage

### API Endpoints
```yaml
POST /linkage/link:
  description: Link records between two datasets
  params: dataset1_id, dataset2_id, settings
  
POST /linkage/deduplicate:
  description: Find duplicates within a dataset
  params: dataset_id, settings
  
GET /linkage/job/{job_id}:
  description: Check status of linkage job
  
GET /linkage/results/{job_id}:
  description: Retrieve linkage results
  
POST /linkage/estimate:
  description: Estimate parameters using EM algorithm
  
GET /health:
  description: Service health check
```

### Performance Requirements
- Link 1M records in <60 seconds on single machine
- Support datasets up to 10M records with DuckDB
- Health check response time <500ms
- API response time <2s for synchronous operations

## Success Metrics

### Completion Criteria
- All P0 requirements implemented and tested
- Documentation complete with usage examples
- Integration tests passing with 95%+ coverage
- Performance benchmarks meeting requirements

### Quality Metrics
- Linkage accuracy >95% on standard benchmarks
- False positive rate <1% for deduplication
- System uptime >99.9%
- Memory usage <4GB for 1M records

### Business Metrics
- Enable 5+ scenarios requiring record linkage
- Process 100M+ records monthly across deployments
- Save 100+ hours monthly in manual data cleaning
- Generate $30K+ value per enterprise deployment

## Implementation Notes

### Phase 1: Core Setup (Complete by Day 1)
- Docker container with Splink installation
- Basic health check endpoint
- Simple linkage API endpoint

### Phase 2: Integration (Complete by Day 3)
- PostgreSQL connectivity
- MinIO dataset storage
- Redis job queue

### Phase 3: Advanced Features (Complete by Day 5)
- Spark backend support
- Interactive visualizations
- Batch processing system

## Research Findings

### Similar Work
- **pandas-ai**: General data analysis but no record linkage
- **data-tools**: ETL focus without probabilistic matching
- **Qdrant**: Vector similarity but not Fellegi-Sunter model

### Template Selected
- Base: pandas-ai Python resource structure
- Extensions: Job queue from N8n patterns

### Unique Value
- Only probabilistic record linkage in Vrooli
- Scales from laptop to cluster seamlessly
- No training data required

### External References
- [Splink Documentation](https://moj-analytical-services.github.io/splink/)
- [Fellegi-Sunter Model](https://en.wikipedia.org/wiki/Record_linkage#Fellegi-Sunter_model)
- [UK Government Data Linking](https://www.gov.uk/government/publications/linked-data-in-government)
- [DuckDB Performance](https://duckdb.org/docs/guides/performance/benchmarks)
- [Record Linkage Best Practices](https://github.com/J535D165/recordlinkage)

### Security Notes
- No PII in logs or error messages
- Secure handling of sensitive linkage keys
- Rate limiting on API endpoints
- Input validation for all parameters

## Progress History
- 2025-01-10: Initial PRD created (0% → 5%)
- 2025-09-12 01:25: Scaffolding completed - Health endpoint working, CLI functional (5% → 20%)
- 2025-09-12 01:45: Full implementation completed - All P0 requirements functional (20% → 100%)
  - Implemented DuckDB-based deduplication engine
  - Created async job processing system
  - Added linkage and parameter estimation endpoints
  - Integrated simplified Splink-like algorithms
  - All tests passing (smoke, integration, unit)
- 2025-09-13: Major improvements completed - P1 requirements partially achieved
  - Integrated actual Splink Python library (v3.9.14) with fallback to simplified algorithms
  - Added PostgreSQL integration for data storage and result persistence
  - Implemented batch processing with priority queuing
  - Enhanced engine with save/load capabilities
  - Added support for CSV and database data sources
  - Maintained backward compatibility with simplified algorithms
- 2025-09-14: Interactive Visualization implemented - Additional P1 requirement achieved (75% P1 complete)
  - Added interactive web-based UI for exploring linkage results
  - Implemented multiple visualization types: dashboard, network graph, confidence distribution, processing metrics
  - Created job overview page showing all jobs with status and summary statistics
  - Integrated Plotly for interactive charts and graphs
  - Added CLI command `content visualize` for easy access to visualizations
  - Full HTML-based dashboards with responsive design
- 2025-09-14: Spark Integration completed - All P1 requirements achieved (100% P1 complete)
  - Implemented Apache Spark engine for processing 100M+ record datasets
  - Added PySpark support with optimized settings for large-scale processing
  - Created spark_engine.py module with SparkSession management
  - Integrated Spark backend selection in API endpoints (auto/duckdb/spark)
  - Added fallback logic from Spark to DuckDB when Spark unavailable
  - Implemented both native Splink-Spark and simplified Spark SQL processing
  - Added `/spark/info` endpoint to check Spark availability and configuration
  - Enhanced Docker container with Java runtime and PySpark dependencies
  - Created comprehensive Spark integration tests
  - Supports distributed processing with configurable executor memory and cores
- 2025-09-15: All P2 requirements completed - Full feature implementation achieved (100% complete)
  - **Real-time Stream Processing**: Implemented continuous record matching for streaming data
    - Created stream_processor.py module with Redis-based stream consumption
    - Added stream creation, monitoring, and management endpoints
    - Supports configurable batch sizes, timeouts, and deduplication windows
    - Tracks exact, probable, and possible matches with confidence scoring
    - Provides real-time statistics and match history
  - **Custom Blocking Rules Interface**: Implemented user-defined blocking strategies
    - Created blocking_rules.py module with 10 blocking strategies
    - Added rule sets with priority-based execution and combination logic
    - Includes pre-defined templates for person, address, and company matching
    - Supports rule validation, optimization suggestions, and SQL generation
    - Enables custom SQL expressions for advanced blocking scenarios
  - All existing tests passing with no regressions
  - Enhanced Docker configuration with Redis async support
- 2025-09-16: Production hardening and bug fixes
  - Fixed Redis connection handling to work gracefully when Redis unavailable
  - Corrected Redis port configuration (6380 instead of 6379)
  - Added proper timeout and error handling for stream processing
  - Fixed test suite container name detection
  - Improved visualization endpoint performance
  - All P0, P1, and P2 requirements verified and working