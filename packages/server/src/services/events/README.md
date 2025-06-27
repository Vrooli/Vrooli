# 🚀 Unified Event System

The Unified Event System is the backbone of Vrooli's emergent AI capabilities, enabling agents, routines, and swarms to coordinate through data-driven events rather than hardcoded logic.

## 📋 Overview

This event system replaces 8+ fragmented event components with a single, unified architecture that supports:

- **🔌 MQTT-Style Topics**: Hierarchical event namespaces (`swarm/*`, `tool/*`, `safety/*`)
- **📬 Delivery Guarantees**: Fire-and-forget, reliable, and barrier-sync modes
- **🚦 Blocking Events**: Tool approval workflows with timeout handling
- **🧩 Agent Extensibility**: Agents can propose new event types without code changes
- **📊 Event Discovery**: Catalog system for agents to discover available events
- **🔒 Type Safety**: Full TypeScript support with runtime validation

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Event Publishers                         │
│  (Tier 1: Coordination, Tier 2: Process, Tier 3: Execution)│
└────────────────────────┬───────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    Event Bus                                │
│  • Pattern-based routing (MQTT-style)                       │
│  • Delivery guarantees                                      │
│  • Barrier synchronization                                  │
└────────┬──────────────────────────────────┬────────────────┘
         │                                  │
         ▼                                  ▼
┌────────────────────┐            ┌────────────────────┐
│  Event Subscribers │            │  Approval System   │
│  • Agents          │            │  • Blocking events │
│  • Analytics       │            │  • User approvals  │
│  • UI Updates      │            │  • Timeouts        │
└────────────────────┘            └────────────────────┘
```

## 🔑 Key Components

### 1. **Event Types** (`types.ts`)
Core type definitions for all events in the system:

```typescript
// Base event that all events extend
interface BaseEvent {
  id: string;
  type: string;  // MQTT-style: "swarm/goal/created"
  timestamp: Date;
  source: EventSource;
  data: unknown;
  metadata?: EventMetadata;
}

// Classification types
interface SafetyEvent extends BaseEvent { /* blocking safety checks */ }
interface CoordinationEvent extends BaseEvent { /* Tier 1 strategic */ }
interface ProcessEvent extends BaseEvent { /* Tier 2 orchestration */ }
interface ExecutionEvent extends BaseEvent { /* Tier 3 execution */ }
```

### 2. **Event Bus** (`eventBus.ts`)
Central message broker with delivery guarantees:

```typescript
const eventBus = createEventBus(logger);

// Fire-and-forget (performance)
await eventBus.publish({
  type: "step/started",
  data: { stepId: "123" },
  metadata: { deliveryGuarantee: "fire-and-forget" }
});

// Reliable delivery (audit trail)
await eventBus.publish({
  type: "routine/completed",
  data: { routineId: "456", creditsUsed: "2500" },
  metadata: { deliveryGuarantee: "reliable" }
});

// Barrier sync (blocking approval)
const result = await eventBus.publishBarrierSync({
  type: "tool/approval_required",
  data: { toolName: "payment_api" },
  metadata: {
    deliveryGuarantee: "barrier-sync",
    barrierConfig: { quorum: 1, timeoutMs: 30000 }
  }
});
```

### 3. **Event Catalog** (`catalog.ts`)
Registry for event discovery and agent-proposed events:

```typescript
const catalog = createEventCatalog(logger);

// Agents can discover available events
const { coreEvents, emergentEvents } = await catalog.getAvailableEvents();

// Agents can propose new event types
const result = await catalog.proposeEmergentEvent(
  "market_analysis_agent",
  "market/trend/detected",
  { schema: trendEventSchema },
  "New event type for market trend detection patterns"
);
```

### 4. **Approval System** (`approval.ts`)
Tool approval workflow with blocking semantics:

```typescript
const approvalSystem = createApprovalSystem(eventBus, logger);

// Request approval (blocks until approved/rejected/timeout)
const result = await approvalSystem.processApprovalRequest({
  id: "tool_call_123",
  name: "execute_trade",
  parameters: { symbol: "AAPL", quantity: 100 },
  callerBotId: "trading_bot",
  approvalTimeoutMs: 30000,
  autoRejectOnTimeout: true,
  riskLevel: "high"
});

// Handle user response
await approvalSystem.handleUserApprovalResponse({
  pendingId: "pending_456",
  approved: true,
  userId: "user_789",
  reason: "Trade parameters verified"
});
```

## 📊 Event Patterns

### Topic Hierarchy

Events follow MQTT-style hierarchical topics:

```
swarm/
  goal/
    created
    updated
    completed
    failed
  team/
    formed
    member_added

tool/
  called
  completed
  failed
  approval_required
  approval_granted

routine/
  started
  completed
  step_completed

safety/
  pre_action
  post_action

emergency/
  stop
  escalation
```

### Subscription Patterns

```typescript
// Subscribe to all swarm events
eventBus.subscribe("swarm/*", handler);

// Subscribe to all approval events
eventBus.subscribe("tool/approval_*", approvalHandler);

// Subscribe to multiple patterns
eventBus.subscribe(["routine/*", "step/*"], executionHandler);

// Subscribe to everything (use sparingly)
eventBus.subscribe("#", universalHandler);
```

## 🚀 Usage Examples

### 1. Publishing a Goal Event

```typescript
import { createUnifiedEventSystem, EventTypes, EventUtils } from './events';

const { eventBus } = createUnifiedEventSystem(logger);

// Create and publish a goal creation event
await eventBus.publish({
  id: nanoid(),
  type: EventTypes.SWARM_GOAL_CREATED,
  timestamp: new Date(),
  source: EventUtils.createEventSource(1, "goal-manager"),
  data: {
    swarmId: "swarm_123",
    goalDescription: "Analyze market trends for Q4",
    priority: "high",
    estimatedCredits: "50000"
  },
  metadata: EventUtils.createEventMetadata("fire-and-forget", "high", {
    conversationId: "conv_456"
  })
});
```

### 2. Implementing an Event Agent

```typescript
// Market analysis agent that subscribes to goal events
class MarketAnalysisAgent {
  async initialize(eventBus: IEventBus) {
    // Subscribe to goal creation events
    await eventBus.subscribe(
      "swarm/goal/created",
      this.handleGoalCreated.bind(this),
      {
        filter: (event) => event.data.goalDescription?.includes("market")
      }
    );
  }

  private async handleGoalCreated(event: GoalEvent) {
    // React to market-related goals
    if (event.data.priority === "high") {
      await this.startMarketAnalysis(event.data.swarmId);
    }
  }
}
```

### 3. Tool Approval Flow

```typescript
// In an LLM execution strategy
async function callExternalTool(toolName: string, params: any) {
  const approval = await approvalSystem.processApprovalRequest({
    id: nanoid(),
    name: toolName,
    parameters: params,
    callerBotId: "assistant_bot",
    conversationId: currentConversation.id,
    approvalTimeoutMs: 60000,
    autoRejectOnTimeout: false,
    estimatedCredits: "1000",
    riskLevel: calculateRiskLevel(toolName)
  });

  if (!approval.approved) {
    throw new Error(`Tool call rejected: ${approval.reason}`);
  }

  // Proceed with tool execution
  return await executeTool(toolName, params);
}
```

### 4. Proposing Emergent Events

```typescript
// Agent discovers need for new event type
const proposalResult = await catalog.proposeEmergentEvent(
  "sentiment_agent",
  "market/sentiment/shifted",
  {
    eventType: "market/sentiment/shifted",
    description: "Detects significant sentiment shifts in market data",
    schema: {
      type: "object",
      properties: {
        market: { type: "string" },
        previousSentiment: { type: "number", minimum: -1, maximum: 1 },
        currentSentiment: { type: "number", minimum: -1, maximum: 1 },
        confidence: { type: "number", minimum: 0, maximum: 1 }
      },
      required: ["market", "previousSentiment", "currentSentiment"]
    },
    examples: [{
      market: "NASDAQ",
      previousSentiment: 0.3,
      currentSentiment: -0.2,
      confidence: 0.85
    }]
  },
  "Detected recurring pattern of sentiment shifts that precede market movements. This event type would allow coordination between sentiment and trading agents."
);

if (proposalResult.accepted) {
  // Start emitting the new event type
  await eventBus.publish({
    type: "market/sentiment/shifted",
    data: { /* sentiment data */ }
  });
}
```

## 🔒 Delivery Guarantees

### 1. **Fire-and-Forget** (Default)
- Fastest performance
- No delivery confirmation
- Best for: UI updates, metrics, non-critical notifications

### 2. **Reliable**
- Guaranteed delivery with retries
- Delivery confirmation
- Best for: Audit trails, completion events, error reports

### 3. **Barrier-Sync**
- Blocks until responses received
- Timeout handling
- Best for: Approvals, safety checks, consensus operations

## 🧪 Testing

```typescript
// Test event flow
const testEvent = EventUtils.createBaseEvent(
  "test/example",
  { message: "test" },
  EventUtils.createEventSource(3, "test-component")
);

// Test subscription
const receivedEvents: BaseEvent[] = [];
const subId = await eventBus.subscribe("test/*", async (event) => {
  receivedEvents.push(event);
});

// Publish and verify
await eventBus.publish(testEvent);
expect(receivedEvents).toHaveLength(1);
expect(receivedEvents[0].type).toBe("test/example");

// Cleanup
await eventBus.unsubscribe(subId);
```

## 🔍 Monitoring

```typescript
// Get event bus metrics
const metrics = eventBus.getMetrics();
console.log({
  totalEventsPublished: metrics.totalEventsPublished,
  activeSubscriptions: metrics.activeSubscriptions,
  eventHistory: metrics.eventHistory
});

// Get approval system metrics
const approvalMetrics = approvalSystem.getMetrics();
console.log({
  pendingApprovals: approvalMetrics.pendingApprovals,
  approvalRate: approvalMetrics.approvalRate,
  averageApprovalTime: approvalMetrics.averageApprovalTime
});

// Get catalog statistics
const eventStats = await catalog.getEventStatistics();
console.log({
  topEventTypes: eventStats.topEventTypes,
  recentProposals: eventStats.recentProposals
});
```

## 📚 Best Practices

### 1. **Event Naming**
- Use hierarchical naming: `category/subcategory/action`
- Be specific: `tool/payment/initiated` not `tool/action`
- Use past tense for completed actions: `completed` not `complete`

### 2. **Event Data**
- Keep payloads focused and minimal
- Include correlation IDs for related events
- Use consistent field names across similar events

### 3. **Subscriptions**
- Unsubscribe when no longer needed
- Use specific patterns over wildcards when possible
- Implement error handling in all handlers

### 4. **Performance**
- Use fire-and-forget for high-volume events
- Batch related events when possible
- Monitor subscription count per pattern

### 5. **Testing**
- Test timeout scenarios for barrier events
- Verify cleanup of subscriptions
- Test pattern matching edge cases

## 🚨 Common Pitfalls

1. **Over-subscribing**: Don't use `#` pattern unless absolutely necessary
2. **Missing unsubscribe**: Always clean up subscriptions to prevent memory leaks
3. **Blocking unnecessarily**: Only use barrier-sync for true blocking requirements
4. **Large payloads**: Keep event data focused; use references for large objects
5. **Tight coupling**: Events should be self-contained, not require external context

## 🔮 Future Enhancements

- **Event Replay**: Replay historical events for debugging/analysis
- **Event Sourcing**: Build state from event history
- **Distributed Events**: Cross-instance event propagation
- **Event Schemas UI**: Visual tool for browsing event schemas
- **Performance Analytics**: Detailed metrics on event flow patterns

## 📖 Related Documentation

- [Event Catalog Specification](../../../docs/architecture/execution/event-driven/event-catalog.md)
- [Tool Approval System](../../../docs/architecture/execution/systems/tool-approval-system.md)
- [Three-Tier AI Architecture](../../../docs/architecture/execution/README.md)
- [Emergent Capabilities](../../../docs/architecture/execution/emergent-capabilities/README.md)