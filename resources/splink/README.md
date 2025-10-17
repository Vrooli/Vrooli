# Splink - Probabilistic Record Linkage at Scale

Splink is a Python library for probabilistic record linkage (entity resolution) and deduplication at scale, developed by the UK Ministry of Justice. It enables Vrooli scenarios to identify and link records that refer to the same entity across different datasets without requiring training data.

## Status
✅ **FULLY ENHANCED** - All P0 requirements and ALL P1 requirements implemented (100% P1 complete)
- Health endpoint: ✅ Working
- Deduplication API: ✅ Implemented with native Splink  
- Linkage API: ✅ Implemented with fallback support
- Parameter estimation: ✅ Implemented
- DuckDB backend: ✅ Integrated
- PostgreSQL integration: ✅ Added for data persistence
- Batch processing: ✅ Implemented with priority queuing
- Interactive Visualization: ✅ Web UI with Plotly charts
- **Spark Integration**: ✅ Full support for 100M+ record processing
- All tests passing: ✅ Confirmed

## Features

- **Probabilistic Matching**: Uses Fellegi-Sunter model for sophisticated record linkage
- **Scale**: Process 1M records in <1 minute on a laptop, 100M+ with Spark
- **Unsupervised**: No training data required - uses EM algorithm
- **Multiple Backends**: DuckDB (local), PostgreSQL (persistent), Spark (large-scale)
- **Batch Processing**: Submit multiple linkage jobs with priority queuing
- **Data Persistence**: Save/load datasets and results to PostgreSQL or CSV
- **Native Splink**: Uses actual Splink v3.9.14 library with automatic fallback
- **Interactive Visualization**: Web-based UI with network graphs, confidence distributions, and processing metrics

## Quick Start

```bash
# Install and start the service
vrooli resource splink manage install
vrooli resource splink manage start --wait

# Check service health
vrooli resource splink status

# Run deduplication on a dataset
vrooli resource splink content execute deduplicate --dataset customers.csv

# Link two datasets
vrooli resource splink content execute link --dataset1 orders.csv --dataset2 customers.csv

# View interactive visualizations
vrooli resource splink content visualize  # Show all jobs overview
vrooli resource splink content visualize JOB_ID  # View job dashboard
vrooli resource splink content visualize JOB_ID network  # View match network
```

## API Usage

### Deduplicate Records
```bash
# Standard deduplication with DuckDB (up to 10M records)
curl -X POST http://localhost:8096/linkage/deduplicate \
  -H "Content-Type: application/json" \
  -d '{
    "dataset_id": "customers_2024",
    "backend": "duckdb",
    "settings": {
      "blocking_rules": ["first_name", "last_name"],
      "comparison_columns": ["email", "phone", "address"]
    }
  }'

# Large-scale deduplication with Spark (100M+ records)
curl -X POST http://localhost:8096/linkage/deduplicate \
  -H "Content-Type: application/json" \
  -d '{
    "dataset_id": "customers_large",
    "backend": "spark",
    "settings": {
      "threshold": 0.85,
      "comparison_columns": ["first_name", "last_name", "email"]
    }
  }'
```

### Link Datasets
```bash
curl -X POST http://localhost:8096/linkage/link \
  -H "Content-Type: application/json" \
  -d '{
    "dataset1_id": "orders_2024",
    "dataset2_id": "customers_2024",
    "settings": {
      "link_type": "one_to_one",
      "threshold": 0.9
    }
  }'
```

### Batch Processing
```bash
curl -X POST http://localhost:8096/linkage/batch \
  -H "Content-Type: application/json" \
  -d '{
    "jobs": [
      {"job_type": "deduplicate", "dataset1_id": "customers", "priority": 10},
      {"job_type": "link", "dataset1_id": "orders", "dataset2_id": "customers", "priority": 5}
    ]
  }'
```

### Interactive Visualization

View results through the web UI:
```bash
# View all jobs overview
curl http://localhost:8096/visualization/jobs

# View specific job dashboard  
curl "http://localhost:8096/visualization/job/{job_id}"

# View specific chart type
curl "http://localhost:8096/visualization/job/{job_id}?chart_type=network"
```

Available visualization types:
- **dashboard**: Complete dashboard with all visualizations
- **network**: Interactive network graph of matched records
- **confidence**: Confidence score distribution histogram
- **metrics**: Processing metrics and performance indicators

## Use Cases

### Data Quality
- Remove duplicate customer records
- Merge fragmented user profiles
- Clean CRM databases

### Analytics
- Link transaction data to customer profiles
- Connect disparate data sources
- Build unified customer views

### Identity Resolution
- Match records across systems
- Resolve entity references
- Build identity graphs

## Configuration

Default configuration is in `config/defaults.sh`:
- Port: 8096
- Backend: DuckDB (local processing)
- Max dataset size: 10M records
- Memory limit: 4GB

### Spark Configuration

For large datasets (>10M records), use the Spark backend:

```bash
# Check Spark availability
curl http://localhost:8096/spark/info

# Configure Spark resources
export SPARK_EXECUTOR_MEMORY=4g  # Memory per executor
export SPARK_DRIVER_MEMORY=2g    # Driver memory
export SPARK_EXECUTOR_CORES=2    # Cores per executor

# Start with Spark backend
SPLINK_BACKEND=spark vrooli resource splink manage start --wait
```

Spark Features:
- **Distributed Processing**: Scales horizontally across multiple nodes
- **Adaptive Query Execution**: Optimizes queries based on runtime statistics
- **In-Memory Caching**: Speeds up iterative algorithms
- **Fault Tolerance**: Automatically recovers from node failures
- **Support for 100M+ Records**: Tested with billion-record datasets

## Testing

```bash
# Run all tests
vrooli resource splink test all

# Quick smoke test
vrooli resource splink test smoke

# Integration tests
vrooli resource splink test integration
```

## Dependencies

- Python 3.11+
- DuckDB (included)
- PostgreSQL (optional, for metadata)
- Redis (optional, for job queue)
- MinIO (optional, for large datasets)

## Documentation

- [Splink Official Docs](https://moj-analytical-services.github.io/splink/)
- [API Documentation](docs/api.md)
- [Configuration Guide](docs/configuration.md)
- [Examples](examples/)

## License

Splink is licensed under the MIT License. This resource wrapper follows Vrooli's standard licensing.