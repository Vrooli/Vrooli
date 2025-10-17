# Apache Airflow Resource

Enterprise-grade data pipeline orchestration with DAG-based workflow management for Vrooli.

## Overview

Apache Airflow provides production-ready workflow orchestration with visual DAG management, scheduled execution, and comprehensive monitoring. Version 3.0 brings DAG versioning, client-server architecture, and native ML/AI workflow support.

## Quick Start

```bash
# Install and start Airflow
vrooli resource apache-airflow manage install
vrooli resource apache-airflow manage start --wait

# Access the web UI
# URL: http://localhost:48080
# Username: admin
# Password: admin

# Check service health
vrooli resource apache-airflow test smoke
```

## Features

- **DAG-Based Orchestration**: Visual workflow design with dependency management
- **Production Scheduling**: Cron-based scheduling with backfilling and catchup
- **DAG Versioning**: Git-integrated versioning with audit trails (v3.0)
- **Remote Execution**: Client-server architecture for distributed tasks (v3.0)
- **Multi-System Integration**: 200+ native connectors for data sources
- **ML Pipeline Support**: Native support for ML workflows
- **Monitoring & Alerting**: Built-in monitoring with SLA management

## Architecture

```
Apache Airflow Resource
├── Webserver (Port 48080) - React UI and REST API
├── Scheduler - DAG scheduling and task orchestration
├── Workers - Celery workers for task execution
├── Triggerer - Async task management
├── PostgreSQL - Metadata storage
└── Redis - Message broker for Celery
```

## Configuration

Key environment variables:
- `AIRFLOW_WEBSERVER_PORT`: Web UI port (default: 48080)
- `AIRFLOW_EXECUTOR`: Executor type (default: CeleryExecutor)
- `AIRFLOW_WORKER_COUNT`: Number of workers (default: 2)
- `AIRFLOW_PARALLELISM`: Max parallel tasks (default: 32)
- `AIRFLOW_ENABLE_EXAMPLE_DAGS`: Load example DAGs (default: true)

## DAG Management

### Adding DAGs

```bash
# Add a DAG file
vrooli resource apache-airflow content add --file my_pipeline.py

# List available DAGs
vrooli resource apache-airflow content list

# Execute a DAG
vrooli resource apache-airflow content execute --dag my_pipeline
```

### Example DAG

```python
from airflow import DAG
from airflow.operators.python import PythonOperator
from datetime import datetime, timedelta

def process_data(**context):
    # Your processing logic here
    return "Processed"

dag = DAG(
    'example_pipeline',
    start_date=datetime(2025, 1, 1),
    schedule='@daily',
    catchup=False
)

task = PythonOperator(
    task_id='process',
    python_callable=process_data,
    dag=dag
)
```

## Integration with Vrooli

Airflow integrates seamlessly with other Vrooli resources:

- **Data Sources**: PostgreSQL, Redis, MongoDB
- **Processing**: Judge0, Pandas-AI for computation
- **Storage**: Minio, Qdrant for persistence
- **Monitoring**: Prometheus for metrics

## CLI Commands

```bash
# Lifecycle management
vrooli resource apache-airflow manage install
vrooli resource apache-airflow manage start [--wait]
vrooli resource apache-airflow manage stop
vrooli resource apache-airflow manage restart

# Testing
vrooli resource apache-airflow test smoke      # Quick health check
vrooli resource apache-airflow test integration # Full functionality
vrooli resource apache-airflow test all        # All tests

# DAG operations
vrooli resource apache-airflow content add --file <dag.py>
vrooli resource apache-airflow content list
vrooli resource apache-airflow content execute --dag <dag_id>
vrooli resource apache-airflow content remove --dag <dag_id>

# Monitoring
vrooli resource apache-airflow status [--json]
vrooli resource apache-airflow logs [service] [--follow]
```

## Troubleshooting

### Service Won't Start
- Check Docker is running: `docker ps`
- Verify ports are available: `netstat -tlnp | grep 48080`
- Check logs: `vrooli resource apache-airflow logs`

### DAGs Not Appearing
- Wait 30 seconds for DAG discovery
- Check DAG syntax: Python files must be valid
- Verify DAG directory: `/airflow-home/dags`

### Worker Issues
- Check Redis connection: `vrooli resource apache-airflow logs redis`
- Verify worker count: Default is 2 workers
- Monitor worker logs: `vrooli resource apache-airflow logs airflow-worker`

## Performance Tuning

- **Worker Count**: Increase `AIRFLOW_WORKER_COUNT` for more parallelism
- **Concurrency**: Adjust `AIRFLOW_PARALLELISM` for task throughput
- **DAG Concurrency**: Set `AIRFLOW_DAG_CONCURRENCY` for DAG runs
- **Database Pool**: Configure connection pooling for high load

## Security Considerations

- Change default admin password immediately
- Use environment variables for all secrets
- Enable TLS for production deployments
- Configure RBAC for multi-user access
- Audit DAG changes through version control

## Resources

- [Official Documentation](https://airflow.apache.org/docs/)
- [DAG Best Practices](https://airflow.apache.org/docs/apache-airflow/stable/best-practices.html)
- [Operator Reference](https://airflow.apache.org/docs/apache-airflow/stable/operators-and-hooks-ref.html)

## Development Status

Current implementation provides basic Airflow v3.0 functionality with Docker Compose deployment. Future improvements will add Kubernetes executor, advanced monitoring, and scenario-specific DAG templates.