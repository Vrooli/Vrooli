# Unified Resource Management System

The Unified Resource Management System provides comprehensive resource allocation, tracking, and optimization across Vrooli's three-tier execution architecture.

## Overview

The system manages various resource types across hierarchical scopes, ensuring efficient utilization and preventing resource exhaustion. It integrates with all three tiers to provide:

- **Hierarchical allocation** (User → Swarm → Run → Step)
- **Real-time usage tracking** across all tiers
- **Budget enforcement** with soft and hard limits
- **Resource pooling** and dynamic replenishment
- **Usage analytics** and optimization suggestions
- **Conflict resolution** for resource contention

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   Unified Resource Manager                   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────┐  │
│  │  Resource   │  │   Usage     │  │  Optimization    │  │
│  │   Pools     │  │  Tracking   │  │    Engine        │  │
│  └─────────────┘  └─────────────┘  └──────────────────┘  │
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────┐  │
│  │   Limit     │  │   Alert     │  │   Conflict       │  │
│  │ Enforcement │  │   System    │  │  Resolution      │  │
│  └─────────────┘  └─────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                 ┌────────────┴────────────┐
                 │                         │
        ┌────────▼────────┐      ┌────────▼────────┐
        │ Tier 1 Resource │      │ Tier 3 Resource │
        │    Manager      │      │    Manager      │
        └─────────────────┘      └─────────────────┘
```

## Resource Types

The system tracks and manages the following resource types:

- **CREDITS** - General purpose credits for operations
- **TOKENS** - LLM tokens for AI operations
- **API_CALLS** - External API call quotas
- **TIME** - Execution time limits
- **MEMORY** - Memory allocation limits
- **COMPUTE** - CPU/GPU compute resources
- **STORAGE** - Temporary storage allocation
- **NETWORK** - Network bandwidth allocation

## Hierarchical Scopes

Resources are managed at multiple levels:

1. **GLOBAL** - System-wide limits
2. **USER** - Per-user resource limits
3. **SESSION** - Per-session allocation
4. **SWARM** - Swarm-level budgets
5. **RUN** - Routine run allocations
6. **STEP** - Individual step limits

## Key Features

### 1. Resource Allocation

```typescript
const allocation = await resourceManager.allocateResources({
    requesterId: "swarm123",
    requesterType: "swarm",
    resources: [{
        resourceId: ResourceType.CREDITS,
        amount: 1000,
        required: true
    }],
    priority: AllocationPriority.HIGH,
    duration: 3600 // 1 hour
}, context);
```

### 2. Usage Tracking

```typescript
await resourceManager.trackUsage(allocationId, {
    resourceId: ResourceType.TOKENS,
    consumed: 5000,
    cost: 0.1,
    metadata: { model: "gpt-4" }
});
```

### 3. Limit Configuration

```typescript
await resourceManager.setResourceLimits({
    scope: LimitScope.USER,
    scopeId: "user123",
    limits: [{
        resourceType: ResourceType.CREDITS,
        limit: 10000,
        period: { unit: "day", value: 1 }
    }],
    enforcement: {
        mode: "hard",
        overageAllowed: false
    }
});
```

### 4. Usage Reports

```typescript
const report = await resourceManager.getUsageReport(
    LimitScope.USER,
    "user123",
    { start: startDate, end: endDate }
);
```

### 5. Optimization Suggestions

```typescript
const suggestions = await resourceManager.getOptimizationSuggestions(
    LimitScope.SWARM,
    "swarm123"
);
```

## Resource Pools

Resource pools provide dynamic capacity management:

- **Capacity** - Maximum available resources
- **Available** - Currently available for allocation
- **Reserved** - Allocated but not yet used
- **Replenishment** - Automatic refill rates

## Enforcement Modes

### Hard Limits
- Strict enforcement
- Allocation denied when limit reached
- No overage allowed

### Soft Limits
- Flexible enforcement
- Grace periods for temporary overage
- Overage charges with multipliers

## Event System Integration

The resource manager emits events for:

- **RESOURCE_USAGE** - Tracking all operations
- **RESOURCE_ALERT** - Threshold notifications
- **RESOURCE_EXHAUSTED** - Depletion warnings
- **RESOURCE_CONFLICT** - Contention notifications

## Conflict Resolution

When resource contention occurs, the system applies resolution strategies:

1. **Priority-based** - Higher priority wins
2. **Preemption** - Lower priority releases resources
3. **Sharing** - Resources split between requesters
4. **Queuing** - Requests queued until available
5. **Negotiation** - Dynamic reallocation

## Analytics and Reporting

The system provides comprehensive analytics:

- **Usage summaries** by resource type
- **Cost breakdowns** with categories
- **Efficiency metrics** and waste analysis
- **Utilization rates** and peak usage
- **Trend analysis** for capacity planning

## Best Practices

1. **Set appropriate limits** at each scope level
2. **Monitor usage patterns** for optimization
3. **Use soft limits** for flexible workloads
4. **Configure alerts** at 50%, 80%, and 100%
5. **Review optimization suggestions** regularly
6. **Implement resource pooling** for shared resources
7. **Use priority levels** appropriately

## Error Handling

The system handles various error scenarios:

- **Insufficient resources** - Clear denial reasons
- **Invalid allocations** - Validation errors
- **Expired allocations** - Automatic cleanup
- **System failures** - Graceful degradation

## Performance Considerations

- **Caching** - Resource states cached in Redis
- **Batch operations** - Bulk allocation support
- **Async tracking** - Non-blocking usage updates
- **Cleanup tasks** - Periodic expired allocation removal
- **Replenishment** - Efficient pool updates

## Integration with Tiers

### Tier 1 Integration
- Swarm-level resource budgets
- Team allocation strategies
- Performance-based distribution

### Tier 2 Integration
- Run-level resource tracking
- Step allocation management
- Checkpoint resource state

### Tier 3 Integration
- Execution-time enforcement
- Real-time usage tracking
- Strategy-based optimization

## Future Enhancements

- Machine learning for usage prediction
- Dynamic pricing models
- Resource marketplace
- Advanced scheduling algorithms
- Cross-tenant resource sharing
- Blockchain-based accounting