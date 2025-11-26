# Temporal Resource

Durable execution platform for building highly reliable distributed applications with guaranteed workflow completion.

## Overview

Temporal provides infrastructure for mission-critical workflows that must never fail. Unlike visual workflow builders (n8n) or lightweight scriptable automation platforms, Temporal offers:
- **Durable Execution**: Workflows survive crashes, restarts, and infrastructure failures
- **Exactly-Once Semantics**: Perfect for financial transactions and data integrity
- **Time Travel Debugging**: Complete execution history with replay capabilities
- **Distributed Transactions**: Saga pattern implementation across services

## Quick Start

```bash
# Install and start Temporal
vrooli resource temporal manage install
vrooli resource temporal manage start

# Check health
vrooli resource temporal status

# View Web UI
open http://localhost:7233

# Stop when done
vrooli resource temporal manage stop
```

## Architecture

Temporal consists of several services:
- **Frontend Service** (Port 7233): Web UI and workflow management
- **History Service**: Manages workflow state and history
- **Matching Service**: Routes tasks to workers
- **Worker Service** (Port 7234): gRPC endpoint for workers
- **PostgreSQL**: Persistence layer for durability

## Usage Examples

### Starting a Workflow
```bash
# Using tctl (Temporal CLI)
docker exec temporal-server tctl workflow start \
  --task-queue my-queue \
  --type MyWorkflow \
  --input '{"orderId": "12345"}'
```

### Querying Workflow State
```bash
docker exec temporal-server tctl workflow query \
  --workflow-id wf-123 \
  --query-type getStatus
```

### Worker Development
```javascript
// Example TypeScript worker
import { Worker } from '@temporalio/worker';

const worker = await Worker.create({
  workflowsPath: './workflows',
  activities,
  taskQueue: 'my-queue',
  connection: {
    address: 'localhost:7234'
  }
});

await worker.run();
```

## When to Use Temporal

Use Temporal for:
- ✅ Financial transactions requiring exactly-once processing
- ✅ Multi-day workflows with human approvals
- ✅ Distributed sagas across multiple services
- ✅ Workflows that must survive any failure
- ✅ Complex state machines with versioning

Use n8n instead for:
- ❌ Simple automations and integrations
- ❌ Visual workflow design requirements
- ❌ Quick prototypes and experiments
- ❌ Non-critical scheduled tasks

## Configuration

Environment variables:
```bash
TEMPORAL_PORT=7233           # Web UI port
TEMPORAL_GRPC_PORT=7234      # Worker gRPC port
TEMPORAL_DB_PASSWORD=secret  # PostgreSQL password
TEMPORAL_NAMESPACE=default   # Default namespace
```

## Testing

```bash
# Run all tests
vrooli resource temporal test all

# Quick health check
vrooli resource temporal test smoke

# Integration tests
vrooli resource temporal test integration
```

## Troubleshooting

### Service Won't Start
- Check PostgreSQL is running: `docker ps | grep postgres`
- Verify ports are free: `lsof -i :7233`
- Check logs: `vrooli resource temporal logs`

### Workers Can't Connect
- Ensure gRPC port 7234 is accessible
- Check namespace exists: `docker exec temporal-server tctl namespace describe`
- Verify task queue matches between worker and workflow

### High Memory Usage
- Workflow history grows over time
- Configure archival for old workflows
- Adjust PostgreSQL connection pool settings

## Resources
- [Temporal Documentation](https://docs.temporal.io/)
- [SDK Examples](https://github.com/temporalio/samples)
- [Best Practices](https://docs.temporal.io/best-practices)
