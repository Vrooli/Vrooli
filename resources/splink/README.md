# Splink - Probabilistic Record Linkage at Scale

Splink is a Python library for probabilistic record linkage (entity resolution) and deduplication at scale, developed by the UK Ministry of Justice. It enables Vrooli scenarios to identify and link records that refer to the same entity across different datasets without requiring training data.

## Status
✅ **FUNCTIONAL** - All P0 requirements implemented and tested
- Health endpoint: ✅ Working
- Deduplication API: ✅ Implemented  
- Linkage API: ✅ Implemented
- Parameter estimation: ✅ Implemented
- DuckDB backend: ✅ Integrated
- All tests passing: ✅ Confirmed

## Features

- **Probabilistic Matching**: Uses Fellegi-Sunter model for sophisticated record linkage
- **Scale**: Process 1M records in <1 minute on a laptop, 100M+ with Spark
- **Unsupervised**: No training data required - uses EM algorithm
- **Multiple Backends**: DuckDB (local), Spark, Athena, SQLite
- **Interactive Visualizations**: Explore linkage results with built-in charts

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
```

## API Usage

### Deduplicate Records
```bash
curl -X POST http://localhost:8096/linkage/deduplicate \
  -H "Content-Type: application/json" \
  -d '{
    "dataset_id": "customers_2024",
    "settings": {
      "blocking_rules": ["first_name", "last_name"],
      "comparison_columns": ["email", "phone", "address"]
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

For large datasets (>10M records), configure Spark backend:
```bash
export SPLINK_BACKEND=spark
export SPARK_MASTER_URL=spark://master:7077
```

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