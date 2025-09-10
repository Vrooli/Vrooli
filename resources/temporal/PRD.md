# Product Requirements Document (PRD) - Temporal

## 🎯 Infrastructure Definition

### Core Infrastructure Capability
**What permanent infrastructure capability does this resource add to Vrooli?**
Temporal provides durable execution infrastructure for building mission-critical distributed applications with strong consistency guarantees. It offers state management, automatic retries, saga patterns, and exactly-once execution semantics for complex, long-running workflows that must never fail.

### System Amplification
**How does this resource make the entire Vrooli system more capable?**
- **Guaranteed Execution** - Workflows continue exactly where they left off after failures, crashes, or restarts
- **Temporal Durability** - Workflow state persists indefinitely, surviving any infrastructure failure
- **Distributed Transactions** - Implements saga patterns for multi-service coordination with compensation
- **Time-based Operations** - Native support for delays, timeouts, cron schedules, and temporal queries
- **Workflow Versioning** - Safe deployment of workflow changes without breaking running instances
- **Observability** - Built-in workflow history, state inspection, and debugging capabilities

### Enabling Value
**What new scenarios become possible when this resource is available?**
1. **Financial Transactions**: Multi-step payment processing, reconciliation, regulatory compliance workflows
2. **Order Fulfillment**: E-commerce order orchestration, inventory management, shipping coordination
3. **Data Pipelines**: ETL/ELT with exactly-once processing, data quality checks, rollback capabilities
4. **Human-in-the-Loop**: Approval workflows, document processing, customer onboarding with manual steps
5. **Infrastructure Automation**: Provisioning, deployment pipelines, disaster recovery orchestration
6. **AI/ML Workflows**: Training pipelines, model deployment, A/B testing with guaranteed completion

## 📊 Infrastructure Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Temporal server with PostgreSQL backend for persistence
  - [ ] Web UI for workflow visualization and debugging (port 7233)
  - [ ] gRPC API endpoint for workflow execution (port 7234)
  - [ ] Worker registration and management system
  - [ ] Health monitoring and readiness checks
  - [ ] Standard v2.0 CLI interface (resource-temporal)
  - [ ] Workflow namespace isolation and management
  
- **Should Have (P1)**
  - [ ] Elasticsearch integration for advanced visibility
  - [ ] SDK examples for Go, TypeScript, Python
  - [ ] Workflow templates and common patterns library
  - [ ] Metrics export to Prometheus
  
- **Nice to Have (P2)**
  - [ ] Multi-cluster deployment support
  - [ ] Advanced authorization with mTLS
  - [ ] Archival to S3/Minio for long-term storage

### Performance Targets
- **Workflow Throughput**: 1000+ workflows/second
- **State Transitions**: 10,000+ events/second
- **Query Latency**: <100ms for workflow state queries
- **Recovery Time**: <5 seconds after crash
- **Storage Efficiency**: Automatic history archival after 30 days

### Security Requirements
- [ ] **No hardcoded credentials** - All configuration from environment
- [ ] **Namespace isolation** - Workflows cannot access other namespaces
- [ ] **TLS support** - Optional TLS for gRPC connections
- [ ] **Authentication** - Basic auth for Web UI access
- [ ] **Input validation** - Sanitize all workflow inputs

## 🛠 Technical Specifications

### Architecture Overview
```
┌─────────────────────────────────────────────┐
│            Temporal Resource                 │
├─────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐          │
│  │  Frontend   │  │   History   │          │
│  │   Service   │  │   Service   │          │
│  │  (Port 7233)│  │             │          │
│  └──────┬──────┘  └──────┬──────┘          │
│         │                 │                  │
│  ┌──────▼─────────────────▼──────┐          │
│  │      PostgreSQL Database       │          │
│  │   (Workflow State & History)   │          │
│  └────────────────────────────────┘          │
│                                              │
│  ┌─────────────┐  ┌─────────────┐          │
│  │   Matching  │  │   Worker    │          │
│  │   Service   │  │   Service   │          │
│  │             │  │  (Port 7234)│          │
│  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────┤
                External Integrations           │
  ┌─────────────┐  ┌─────────────┐            │
  │   Vrooli    │  │  Scenario   │            │
  │   Workers   │  │   Workers   │            │
  └─────────────┘  └─────────────┘            │
```

### Dependencies
- **PostgreSQL**: Persistence layer for workflow state
- **Optional**: Elasticsearch for enhanced visibility
- **Optional**: Prometheus for metrics collection
- **Optional**: Minio for archival storage

### API Specifications

#### Workflow Operations
```bash
# Start workflow
tctl workflow start --task-queue my-queue --type MyWorkflow --input '{"data": "value"}'

# Query workflow state
tctl workflow query --workflow-id wf-123 --query-type getStatus

# Signal running workflow
tctl workflow signal --workflow-id wf-123 --name updateData --input '{"update": "value"}'

# Cancel workflow
tctl workflow cancel --workflow-id wf-123

# List workflows
tctl workflow list --query 'WorkflowType="OrderProcessing" AND ExecutionStatus="Running"'
```

#### Namespace Management
```bash
# Create namespace
tctl namespace register --name production --retention 30d

# Update namespace
tctl namespace update --name production --retention 90d

# Describe namespace
tctl namespace describe --name production
```

### Configuration Schema
```yaml
persistence:
  defaultStore: postgres
  visibilityStore: postgres
  datastores:
    postgres:
      sql:
        pluginName: postgres
        databaseName: temporal
        connectAddr: localhost:5432
        connectProtocol: tcp
        user: temporal
        password: ${TEMPORAL_DB_PASSWORD}
        maxConns: 20
        maxIdleConns: 20

clusterMetadata:
  enableGlobalNamespace: false
  failoverVersionIncrement: 10
  masterClusterName: "active"
  currentClusterName: "active"

services:
  frontend:
    rpc:
      grpcPort: 7233
      membershipPort: 6933
      bindOnIP: "0.0.0.0"
  
  matching:
    rpc:
      grpcPort: 7235
      membershipPort: 6935
  
  history:
    rpc:
      grpcPort: 7234
      membershipPort: 6934
  
  worker:
    rpc:
      grpcPort: 7239
      membershipPort: 6939
```

## 🔄 Integration Points

### Scenario Integration
```javascript
// Example: Order processing scenario using Temporal
import { WorkflowClient } from '@temporalio/client';

const client = new WorkflowClient({
  address: 'localhost:7234',
});

// Start order fulfillment workflow
const handle = await client.start('orderFulfillment', {
  taskQueue: 'orders',
  args: [{
    orderId: '12345',
    customerId: 'cust-789',
    items: [...],
    payment: {...}
  }],
  workflowId: `order-${orderId}`,
});

// Query workflow status
const status = await handle.query('getOrderStatus');
console.log('Order status:', status);
```

### Comparison with Existing Workflow Tools

| Feature | Temporal | n8n | Windmill | Use Case |
|---------|----------|-----|----------|----------|
| Visual Builder | ❌ | ✅ | ✅ | Temporal for code-first |
| Durability | ✅✅✅ | ✅ | ✅ | Temporal for mission-critical |
| Distributed | ✅✅✅ | ❌ | ✅ | Temporal for multi-service |
| Exactly-Once | ✅✅✅ | ❌ | ❌ | Temporal for financial |
| Time Travel | ✅✅✅ | ❌ | ❌ | Temporal for debugging |
| Learning Curve | High | Low | Medium | n8n for citizen developers |

**When to use Temporal over n8n/Windmill:**
- Financial transactions requiring exactly-once semantics
- Multi-day workflows with human approvals
- Distributed sagas across multiple services
- Workflows that must survive infrastructure failures
- Complex state machines with versioning requirements

## 📈 Success Metrics

### Adoption Metrics
- Number of mission-critical workflows migrated to Temporal
- Reduction in workflow failure rates (target: <0.01%)
- Developer productivity improvement (target: 2x for complex workflows)

### Technical Metrics
- Workflow completion rate: >99.99%
- Recovery time from failure: <5 seconds
- Query response time: <100ms p99
- Resource utilization: <2GB RAM for 1000 concurrent workflows

### Business Impact
- **Revenue Protection**: $500K+ from preventing transaction failures
- **Operational Savings**: $200K+ from reduced manual intervention
- **Compliance Value**: $300K+ from audit trail and guaranteed execution
- **Total Value**: $1M+ annually from reliability improvements

## 🔍 Research Findings

### Similar Work Analysis
- **Airbyte's Internal Temporal**: Used specifically for Airbyte's sync orchestration, not exposed as general resource
- **n8n**: Visual workflow builder, good for simple automations but lacks durability guarantees
- **Windmill**: Developer-focused but missing Temporal's strong consistency and time-travel debugging

### Unique Value Proposition
Temporal is the only workflow orchestration solution that provides:
1. **True durability** - Workflows survive any failure including total data center loss
2. **Time travel debugging** - Complete execution history with ability to replay
3. **Exactly-once semantics** - Critical for financial and data integrity scenarios
4. **Code-first workflows** - Full programming language power vs limited visual builders

### External References
- [Temporal Official Documentation](https://docs.temporal.io/)
- [Uber's Cadence (Temporal predecessor)](https://cadenceworkflow.io/)
- [Temporal vs Airflow Comparison](https://temporal.io/blog/temporal-vs-airflow)
- [Building Reliable Distributed Systems](https://temporal.io/blog/building-reliable-distributed-systems)
- [Saga Pattern Implementation](https://temporal.io/blog/saga-pattern)

### Security Considerations
- Workflow data encryption at rest via PostgreSQL
- Optional mTLS for service-to-service communication
- Namespace isolation for multi-tenancy
- Audit logging for compliance requirements

## 📝 Implementation Notes

### Phase 1: Core Setup (This Generator)
- Basic Temporal server with PostgreSQL
- Web UI accessibility
- Health checks and monitoring
- CLI wrapper for v2.0 compliance

### Phase 2: SDK Integration (Future Improver)
- Go, TypeScript, Python SDK examples
- Common workflow patterns library
- Integration with existing scenarios

### Phase 3: Advanced Features (Future Improver)
- Elasticsearch for visibility
- Prometheus metrics export
- Multi-cluster setup
- Archival configuration

## 🎯 Why Temporal Matters for Vrooli

Temporal transforms Vrooli from a platform that can automate tasks to one that can guarantee business-critical operations. While n8n and Windmill excel at visual automation and rapid development, Temporal provides the infrastructure for scenarios where failure is not an option. This includes financial transactions, regulatory compliance, data consistency, and any workflow where "mostly working" isn't good enough.

The combination of Temporal with existing automation tools creates a complete spectrum:
- **n8n**: Rapid visual prototyping and simple integrations
- **Windmill**: Developer-friendly scripting and UI generation
- **Temporal**: Mission-critical workflows with guaranteed execution

Together, they enable Vrooli to tackle any automation challenge from simple notifications to complex distributed transactions.