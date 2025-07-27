# Windmill Flow Examples

Flows are visual workflows that chain multiple scripts together with control logic, branching, error handling, and parallel execution. They enable complex automation scenarios beyond what individual scripts can achieve.

## Available Examples

### 1. Data Pipeline Flow (`data-pipeline-flow.json`)

A complete ETL (Extract, Transform, Load) pipeline demonstrating data processing best practices.

**Key Features:**
- **Sequential Processing**: Extract → Validate → Transform → Load
- **Error Handling**: Automatic retries with exponential backoff
- **Conditional Execution**: Skip steps based on validation results
- **Notifications**: Email alerts on completion or failure
- **Scheduling**: Runs daily at 2 AM UTC

**Flow Structure:**
```
1. Extract Data (API call with pagination)
   ↓
2. Validate Data (schema and business rules)
   ↓ (if valid)
3. Transform Data (enrichment and formatting)
   ↓ (if data exists)
4. Load to Database (bulk insert)
   ↓
5. Send Notification (success/failure)
```

**Use Cases:**
- Daily sales data synchronization
- Customer data aggregation
- Inventory updates
- Financial reporting pipelines

### 2. Expense Approval Workflow (`approval-workflow.json`)

Multi-level approval workflow with dynamic routing based on expense amount.

**Approval Levels:**
- **< $500**: Direct manager only
- **$500-$2000**: Manager + Department head
- **> $2000**: Manager + Department head + Finance director

**Key Features:**
- **Human-in-the-Loop**: Approval steps wait for human decision
- **Dynamic Routing**: Approval chain based on amount
- **Timeout Handling**: Auto-escalation after 48 hours
- **Audit Trail**: Complete history of approvals
- **Notifications**: Email at each step

**Flow Structure:**
```
1. Validate Request
   ↓
2. Determine Approvers (based on amount)
   ↓
3. Manager Approval
   ↓ (if approved)
4. Department Head Approval (if required)
   ↓ (if approved)
5. Finance Approval (if required)
   ↓ (if all approved)
6. Process Expense
   ↓
7. Notify Result
```

**Use Cases:**
- Expense reimbursements
- Purchase orders
- Leave requests
- Access requests

### 3. Multi-System Integration Workflow (`integration-workflow.json`)

Synchronizes customer data across CRM, billing, and support systems with conflict resolution.

**Key Features:**
- **Webhook Triggered**: Responds to real-time updates
- **Parallel Fetching**: Retrieves data from all systems simultaneously
- **Conflict Detection**: Identifies data inconsistencies
- **Smart Resolution**: Applies business rules for conflicts
- **Partial Failure Handling**: Continues despite individual system failures
- **Verification**: Confirms successful synchronization

**Flow Structure:**
```
1. Webhook Receiver
   ↓
2. Validate Signature
   ↓
3. Parallel Fetch Current Data
   ├── CRM System
   ├── Billing System
   └── Support System
   ↓
4. Detect Conflicts
   ↓ (if conflicts)
5. Resolve Conflicts
   ↓
6. Parallel Update Systems
   ├── Update CRM
   ├── Update Billing
   └── Update Support
   ↓
7. Verify Synchronization
   ↓
8. Log Audit Trail
```

**Use Cases:**
- Customer data synchronization
- Product catalog updates
- Order fulfillment
- Multi-channel inventory

## Creating Flows in Windmill

### Visual Flow Editor

1. **Access Flows**: Navigate to "Flows" in your workspace
2. **Create New Flow**: Click "New Flow"
3. **Add Steps**: Drag scripts from the library
4. **Connect Steps**: Draw connections between nodes
5. **Configure Logic**: Add conditions, loops, branches
6. **Set Error Handling**: Define retry policies
7. **Test Flow**: Run with sample data
8. **Deploy**: Save and optionally schedule

### Flow Components

#### Script Steps
- Execute individual scripts
- Map inputs from previous outputs
- Transform data between steps

#### Control Flow
- **Branches**: If/else conditions
- **Loops**: For each, while conditions
- **Parallel**: Execute multiple steps simultaneously
- **Try/Catch**: Error handling blocks

#### Special Steps
- **Approval**: Wait for human input
- **Wait**: Delay execution
- **HTTP Request**: Direct API calls
- **Database Query**: Direct SQL execution

### Input/Output Mapping

Windmill uses a powerful expression language for data mapping:

```javascript
// Reference previous step output
${nodes.step_name.output}

// Access specific fields
${nodes.extract_data.output.records[0].id}

// Use flow inputs
${flow.input.customer_id}

// Conditional values
${nodes.validate.output.is_valid ? 'proceed' : 'stop'}

// Array operations
${nodes.fetch_data.output.map(item => item.id)}
```

## Best Practices

### Flow Design

1. **Modular Steps**: Keep each step focused on one task
2. **Clear Naming**: Use descriptive names for steps
3. **Error Boundaries**: Add error handling at critical points
4. **Logging**: Include logging steps for debugging
5. **Idempotency**: Design flows to be safely re-runnable

### Performance

1. **Parallel Execution**: Use parallel steps when possible
2. **Batch Processing**: Process data in chunks
3. **Timeouts**: Set appropriate timeouts for each step
4. **Resource Limits**: Consider memory and CPU usage

### Error Handling

1. **Retry Policies**: Configure retries for transient failures
2. **Fallback Logic**: Define alternative paths
3. **Notifications**: Alert on critical failures
4. **Partial Success**: Handle scenarios where some steps fail

### Security

1. **Input Validation**: Validate all external inputs
2. **Secret Management**: Use Windmill resources for secrets
3. **Audit Logging**: Log all critical actions
4. **Access Control**: Set appropriate permissions

## Advanced Flow Patterns

### 1. Fan-Out/Fan-In
Process multiple items in parallel, then aggregate results:
```
Split Array → Parallel Process Each → Aggregate Results
```

### 2. Circuit Breaker
Prevent cascading failures:
```
Check Health → If Healthy: Process → If Unhealthy: Skip/Alert
```

### 3. Saga Pattern
Distributed transactions with compensations:
```
Action 1 → Action 2 → Action 3
   ↓ (on failure)
Compensate 3 → Compensate 2 → Compensate 1
```

### 4. Event Sourcing
Capture all changes as events:
```
Receive Event → Validate → Store Event → Update State → Publish
```

## Testing Flows

### Test Strategies

1. **Unit Testing**: Test individual scripts separately
2. **Integration Testing**: Test complete flow with real connections
3. **Mock Testing**: Use mock data for external systems
4. **Load Testing**: Verify performance under load

### Debug Tools

- **Step Outputs**: View output of each step
- **Execution Timeline**: See timing and dependencies
- **Log Viewer**: Centralized logging
- **Replay**: Re-run flows with previous inputs

## Monitoring Flows

### Metrics to Track

- **Success Rate**: Percentage of successful runs
- **Execution Time**: Average and P95 duration
- **Error Frequency**: Common failure points
- **Resource Usage**: CPU and memory consumption

### Alerting

Set up alerts for:
- Flow failures
- SLA violations
- Unusual patterns
- Resource exhaustion

## Deployment Patterns

### 1. Blue-Green Deployment
Maintain two versions, switch traffic:
- Deploy new version alongside old
- Test new version
- Switch traffic
- Keep old version for rollback

### 2. Canary Deployment
Gradual rollout:
- Route small percentage to new version
- Monitor metrics
- Gradually increase traffic
- Full deployment or rollback

### 3. Feature Flags
Toggle functionality:
- Add condition nodes checking flags
- Enable/disable features without deployment
- A/B testing capabilities

## Next Steps

1. **Import Examples**: Load example flows into Windmill
2. **Customize**: Modify flows for your use cases
3. **Build New Flows**: Create workflows for your processes
4. **Combine with Apps**: Create UIs that trigger flows
5. **Schedule**: Set up automated execution

For script examples, see [Scripts](../scripts/). For UI applications, see [Apps](../apps/).