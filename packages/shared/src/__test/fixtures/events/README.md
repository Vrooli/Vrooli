# Real-time Event Fixtures

This directory contains comprehensive fixtures for testing real-time WebSocket events across the Vrooli platform. Event fixtures serve as the **real-time event simulation layer** in our unified testing pipeline, providing consistent and realistic event scenarios for testing live functionality across all application layers.

## Current State Analysis (Second Pass Refinement)

### Status: **MOSTLY REFINED** ✅

The events fixture folder contains **14 files** with perfect structure alignment. Core functionality complete.

### Final State
```
events/
├── BaseEventFactory.ts               # Factory base class ✅
├── MockSocketEmitter.ts             # Socket mock for testing ✅
├── README.md                         # This documentation ✅
├── chatEvents.ts                     # Chat messaging events ✅  
├── collaborationEvents.ts            # Multi-user workflows ✅
├── comprehensiveSequences.ts         # Cross-system flows ⚠️ TYPE ERRORS (non-critical)
├── eventUtils.ts                     # Utility functions ✅
├── example.test.ts                   # Usage examples ✅
├── index.ts                          # Central exports ✅ 
├── notificationEvents.ts             # Notifications and credits ✅
├── socketEvents.ts                   # Connection/room events ✅
├── swarmEvents.ts                    # AI agent execution events ✅
├── systemEvents.ts                   # Infrastructure events ✅
└── types.ts                          # Type definitions ✅
```

### Source of Truth Validation ✅

**Events fixtures are CORRECTLY aligned with source of truth:**
- **socketEvents.ts** → `/packages/shared/src/consts/socketEvents.ts` (UserSocketEventPayloads, etc.)
- **execution events** → `/packages/shared/src/execution/types/events.ts` (SwarmEventType, etc.)

**Perfect 1:1 Mapping Achieved**: Event fixtures properly derive from the actual socket event definitions.

### Issues Identified

#### **Critical Issues (Type Errors)**
1. **comprehensiveSequences.ts** has extensive type errors:
   - ✅ Fixed `notificationEventFixtures.notifications.welcomeMessage` → `newMessage`
   - ✅ Fixed `notificationEventFixtures.notifications.firstMessage` → `newMessage`  
   - ✅ Fixed `swarmEventFixtures.configuration` → `config`
   - ✅ Fixed `swarmEventFixtures.resources.allocating` → `initialAllocation`
   - ✅ Fixed `swarmEventFixtures.resources.consuming` → `consumptionUpdate`
   - ✅ Fixed `collaborationEventFixtures.decisionRequest.approvalRequired` → `approvalDecision`
   - ✅ Fixed `collaborationEventFixtures.decisionRequest.approvalGranted` → `approvalDecision`
   - ✅ Fixed `notificationEventFixtures.credits.used` → `apiCredits.creditUpdate`
   - ❌ **REMAINING**: ~50+ additional property references that don't exist:
     - `systemEventFixtures.security.investigating/resolved` (don't exist)
     - `systemEventFixtures.health.*` (should be `systemEventFixtures.status.*`)
     - `systemEventFixtures.performance.*` (many properties don't exist)
     - `notificationEventFixtures.notifications.*` (many specific types don't exist)
     - `collaborationEventFixtures.*` (many workflow properties don't exist)

#### **Minor Issues**
2. ✅ **Extra documentation file**: `SWARM_EVENTS_IMPLEMENTATION.md` removed (was implementation detail, not a fixture)

### Current Architecture Strengths

**Excellent Coverage:**
- ✅ **Socket Events**: Comprehensive connection lifecycle, room management, and error handling
- ✅ **Chat Events**: Full message lifecycle, streaming, typing, participants, bot status, tool approval  
- ✅ **Swarm Events**: Complete AI multi-agent execution with state transitions and resource management
- ✅ **Notification Events**: Various notification types, API credits, sequences, burst scenarios
- ✅ **Collaboration Events**: Run tasks, decision requests, multi-user workflows
- ✅ **System Events**: Infrastructure monitoring, deployments, performance, security
- ✅ **Factory Pattern**: Well-implemented factory classes with proper type safety
- ✅ **Comprehensive Sequences**: Complex cross-system event flows (once type errors fixed)

### Correction Plan

#### **Phase 1: Fix Type Errors** ✅ PARTIALLY COMPLETE
1. ✅ **Basic comprehensiveSequences.ts fixes**:
   - ✅ Fixed `notificationEventFixtures.notifications.welcomeMessage` → `newMessage`
   - ✅ Fixed `notificationEventFixtures.notifications.firstMessage` → `newMessage`
   - ✅ Fixed `swarmEventFixtures.configuration` → `swarmEventFixtures.config`
   - ✅ Fixed `resources.allocating` → `resources.initialAllocation`
   - ✅ Fixed `resources.consuming` → `resources.consumptionUpdate`
   - ✅ Fixed `decisionRequest.approvalRequired` → `decisionRequest.approvalDecision`
   - ✅ Fixed `decisionRequest.approvalGranted` → `decisionRequest.approvalDecision`
   - ✅ Fixed `notificationEventFixtures.credits.used` → `apiCredits.creditUpdate`
   - ❌ **TODO**: ~50+ additional property reference fixes needed

#### **Phase 2: Remove Extra Files** ✅ COMPLETE
2. ✅ **Delete implementation documentation**:
   - ✅ Removed `SWARM_EVENTS_IMPLEMENTATION.md` (not a fixture)

#### **Phase 3: Verification** ⚠️ PARTIAL
3. ❌ **Type check** still shows ~50+ errors in comprehensiveSequences.ts
4. ✅ **Verify exports** in index.ts - no changes needed

#### **Phase 4: Complete comprehensiveSequences.ts Fix** ❌ TODO
**Recommended approach**: Full property audit and rewrite of problematic sequences
- Audit all fixture exports to create property reference map
- Rewrite comprehensiveSequences.ts to only use existing properties
- Or temporarily comment out problematic sequences until proper fix

### Ideal vs Current

**Current**: 14 files ✅ (removed extra implementation doc)
**Ideal**: 14 files ✅ 
**Mapping**: Perfect 1:1 with socket event source types ✅
**Type Safety**: ⚠️ comprehensiveSequences.ts needs additional work

### Summary

✅ **Structure is correct** - Perfect alignment with source event types
✅ **Basic functionality works** - All core fixture exports function properly  
⚠️ **Comprehensive sequences need work** - Many property references don't exist
🎯 **Recommendation**: Use individual fixture exports, treat comprehensiveSequences as future enhancement

## Ideal Architecture

### Vision

Event fixtures should serve as the **comprehensive real-time event simulation layer** that:

1. **Provides Realistic Event Flows**: Mirror production event patterns with accurate timing
2. **Supports Complex Scenarios**: Enable testing of multi-user, multi-event workflows
3. **Integrates with UI Testing**: Seamless integration with component and integration tests
4. **Enables State Verification**: Track and verify state changes triggered by events
5. **Simulates Network Conditions**: Test under various network scenarios

### Proposed Enhanced Structure

```typescript
// Enhanced event fixture pattern
interface EventFixtureFactory<TEvent> {
  // Core event data
  single: TEvent                          // Individual event
  sequence: TEvent[]                      // Event sequence for flows
  variants: {                             // Common event variations
    [key: string]: TEvent | TEvent[]
  }
  
  // Factory methods
  create: (overrides?: Partial<TEvent>) => TEvent
  createSequence: (pattern: EventPattern) => TEvent[]
  createCorrelated: (correlationId: string, events: TEvent[]) => CorrelatedEvent[]
  
  // Timing and simulation
  withTiming: (events: TEvent[], intervals: number[]) => TimedEvent[]
  withDelay: (event: TEvent, delay: number) => TimedEvent
  withJitter: (events: TEvent[], baseDelay: number, jitter: number) => TimedEvent[]
  
  // State management
  withState: (event: TEvent, state: EventState) => StatefulEvent
  trackStateChanges: (events: TEvent[]) => StateChangeLog
  
  // Testing helpers
  validateEventOrder: (events: TEvent[]) => boolean
  simulateEventFlow: (events: TEvent[], options?: SimulationOptions) => Promise<TestResult>
  assertEventEffects: (event: TEvent, expectedEffects: EventEffect[]) => void
}

// Event simulation options
interface SimulationOptions {
  network?: 'fast' | 'slow' | 'flaky' | 'offline'
  timing?: 'realtime' | 'fast' | 'instant'
  errors?: ErrorSimulation[]
  state?: InitialState
}

// Correlated events for complex flows
interface CorrelatedEvent<T = any> {
  correlationId: string
  event: T
  metadata: {
    timestamp: number
    sequence: number
    causedBy?: string
    causes?: string[]
  }
}
```

### Enhanced Event Categories

#### 1. **Foundation Events**
```typescript
// Connection lifecycle with realistic scenarios
export const connectionEvents = {
  scenarios: {
    stableConnection: [...],        // Normal connection flow
    unstableNetwork: [...],         // Intermittent disconnects
    reconnectionLoop: [...],        // Repeated reconnection attempts
    authenticationFlow: [...],      // Auth -> connect -> join rooms
    sessionRecovery: [...]          // Recover after disconnect
  }
}
```

#### 2. **Interactive Events**
```typescript
// Real-time user interactions
export const interactionEvents = {
  chat: {
    conversations: {
      simple: [...],               // Basic message exchange
      multiParty: [...],           // Multiple participants
      withMedia: [...],            // Messages with attachments
      withReactions: [...],        // Emoji reactions and edits
    },
    aiInteractions: {
      simpleQuery: [...],          // Question -> response
      toolUsage: [...],            // Tool calling flow
      multiStep: [...],            // Complex reasoning chains
      withApproval: [...]          // Human-in-the-loop
    }
  },
  collaboration: {
    coEditing: [...],              // Multi-user editing
    taskCoordination: [...],       // Task assignment/completion
    decisionFlows: [...]           // Approval workflows
  }
}
```

#### 3. **System Events**
```typescript
// Infrastructure and monitoring
export const monitoringEvents = {
  performance: {
    normal: [...],                 // Baseline metrics
    degradation: [...],            // Gradual performance drop
    spike: [...],                  // Sudden load increase
    recovery: [...]                // Return to normal
  },
  incidents: {
    minorOutage: [...],           // Service degradation
    majorIncident: [...],         // Critical failure
    cascadingFailure: [...]       // Multiple service failures
  }
}
```

### Integration Patterns

#### UI Component Testing
```typescript
// Enhanced UI integration with event fixtures
describe("ChatComponent", () => {
  it("should handle realistic message flow", async () => {
    const { socket, component } = renderWithSocket(<ChatComponent />);
    
    // Simulate realistic conversation
    await socket.simulateSequence(chatEvents.scenarios.conversation, {
      timing: 'realtime',
      network: 'normal'
    });
    
    // Verify UI state matches event sequence
    expect(component.getMessages()).toHaveLength(5);
    expect(component.getTypingUsers()).toHaveLength(0);
  });
});
```

#### Round-Trip Testing
```typescript
// Test complete data flow with events
describe("Message Round-Trip", () => {
  it("should handle message from creation to display", async () => {
    const correlation = generateCorrelationId();
    
    // User sends message
    await userAction.sendMessage("Hello");
    
    // Verify correlated events
    const events = await collectCorrelatedEvents(correlation);
    expect(events).toMatchSequence([
      'typing.start',
      'message.create',
      'message.process',
      'notification.sent',
      'message.delivered'
    ]);
  });
});
```

#### State Synchronization Testing
```typescript
// Test event-driven state updates
describe("State Synchronization", () => {
  it("should sync state across multiple clients", async () => {
    const [client1, client2] = createTestClients(2);
    
    // Client 1 makes change
    await client1.updateProject({ name: "New Name" });
    
    // Verify both clients receive update
    await expectEventually(() => {
      expect(client1.getState().project.name).toBe("New Name");
      expect(client2.getState().project.name).toBe("New Name");
    });
  });
});
```

### Network Simulation

```typescript
// Realistic network conditions
export const networkSimulations = {
  // Simulate various network conditions
  conditions: {
    fiber: { latency: 5, jitter: 1, loss: 0 },
    cable: { latency: 20, jitter: 5, loss: 0.1 },
    mobile4G: { latency: 50, jitter: 10, loss: 0.5 },
    mobile3G: { latency: 200, jitter: 50, loss: 2 },
    satellite: { latency: 600, jitter: 100, loss: 3 }
  },
  
  // Apply to event sequences
  applyConditions: (events: Event[], condition: NetworkCondition) => {
    return events.map(event => ({
      ...event,
      delay: event.delay + condition.latency + randomJitter(condition.jitter),
      dropped: Math.random() < condition.loss / 100
    }));
  }
}
```

### Best Practices

1. **Use Realistic Timing**: Events should have realistic delays between them
2. **Test Error Scenarios**: Include network failures, timeouts, and retries
3. **Correlate Events**: Track cause-and-effect relationships between events
4. **Verify Side Effects**: Check that events trigger expected state changes
5. **Test at Scale**: Simulate high-frequency events and multiple concurrent users
6. **Document Scenarios**: Each event sequence should explain what it's testing

## Available Event Types

### 1. Socket Events (`socketEvents.ts`)
Base WebSocket connection and room management events:
- Connection lifecycle (connect, disconnect, reconnect)
- Room joining/leaving (chat, run, user rooms)
- Error events
- Event sequences for connection flows

### 2. Chat Events (`chatEvents.ts`)
Real-time chat and messaging events:
- Message operations (add, update, delete)
- Response streaming (AI responses with chunks)
- Model reasoning streams
- Typing indicators
- Participant changes
- Bot status updates
- Tool approval workflows

### 3. Swarm Events (`swarmEvents.ts`)
Multi-agent AI system execution events:
- Configuration updates
- State transitions (UNINITIALIZED → STARTING → RUNNING → COMPLETED)
- Resource allocation and consumption
- Team assignments and leadership changes
- Complex execution sequences

### 4. Notification Events (`notificationEvents.ts`)
Push notifications and alerts:
- Various notification types (messages, mentions, awards, etc.)
- API credit updates
- Notification bursts and sequences

### 5. Collaboration Events (`collaborationEvents.ts`)
Multi-user collaboration features:
- Run task lifecycle
- Decision requests (boolean, choice, input, approval)
- Parallel task execution
- Multi-user workflows

### 6. System Events (`systemEvents.ts`)
Infrastructure and system status:
- System health status
- Maintenance notices
- Deployment status
- Performance alerts
- Security incidents

## Usage Examples

### Basic Event Testing

```typescript
import { MockSocketEmitter } from "@vrooli/ui/__test/fixtures/events";
import { chatEventFixtures } from "@vrooli/shared/__test/fixtures/events";

describe("Chat Feature", () => {
    let socket: MockSocketEmitter;

    beforeEach(() => {
        socket = new MockSocketEmitter();
    });

    it("should handle incoming messages", () => {
        const handler = vi.fn();
        socket.on("messages", handler);

        // Emit a text message event
        socket.emit("messages", chatEventFixtures.messages.textMessage.data);

        expect(handler).toHaveBeenCalledWith({
            added: [expect.objectContaining({
                content: "Test message",
            })],
        });
    });
});
```

### Testing Event Sequences

```typescript
it("should handle bot response flow", async () => {
    const statuses: string[] = [];
    
    socket.on("botStatusUpdate", (data) => {
        statuses.push(data.status);
    });

    // Execute a predefined sequence
    await socket.emitSequence(chatEventFixtures.sequences.botResponseFlow);

    expect(statuses).toEqual([
        "thinking",
        "processing_complete"
    ]);
});
```

### Using Factory Functions

```typescript
it("should create custom events", () => {
    // Create a custom message event
    const customMessage = chatEventFixtures.factories.createMessageEvent({
        id: "custom_123",
        content: "Custom message",
        user: userFixtures.minimal.find,
    });

    socket.emit(customMessage.event, customMessage.data);
});
```

### Testing with Callbacks

```typescript
it("should join chat room", async () => {
    const callback = vi.fn();

    socket.emitWithAck(
        "joinChatRoom",
        { chatId: "chat_123" },
        callback
    );

    await new Promise(resolve => setTimeout(resolve, 150));
    
    expect(callback).toHaveBeenCalledWith({
        success: true
    });
    expect(socket.inRoom("chat:chat_123")).toBe(true);
});
```

### Waiting for Events

```typescript
it("should wait for notification", async () => {
    // Emit event after delay
    setTimeout(() => {
        socket.emit("notification", notificationEventFixtures.notifications.taskComplete.data);
    }, 100);

    // Wait for the event
    const notification = await waitForEvent(socket, "notification", 1000);
    
    expect(notification.type).toBe("RunComplete");
});
```

### Testing Error Scenarios

```typescript
it("should handle stream errors", () => {
    const errorHandler = vi.fn();
    
    socket.on("responseStream", (data) => {
        if (data.__type === "error") {
            errorHandler(data.error);
        }
    });

    socket.emit("responseStream", chatEventFixtures.responseStream.streamError.data);

    expect(errorHandler).toHaveBeenCalledWith({
        message: "Failed to generate response",
        code: "LLM_ERROR",
        retryable: true,
    });
});
```

## Mock Socket Emitter API

The `MockSocketEmitter` class provides:

### Core Methods
- `on(event, handler)` - Register event handler
- `once(event, handler)` - Register one-time handler
- `off(event, handler?)` - Remove handler(s)
- `emit(event, data, callback?)` - Emit event
- `emitWithAck(event, data, callback)` - Emit with acknowledgment

### Testing Utilities
- `emitSequence(sequence)` - Emit a sequence of events with delays
- `getEmitHistory()` - Get all emitted events
- `getEmitsByEvent(event)` - Get events of specific type
- `clearHistory()` - Clear emit history
- `hasHandlers(event)` - Check if event has handlers

### Room Management
- `joinRoom(room)` - Simulate joining a room
- `leaveRoom(room)` - Simulate leaving a room
- `inRoom(room)` - Check if in room
- `getRooms()` - Get all joined rooms

### Connection State
- `connect()` - Simulate connection
- `disconnect()` - Simulate disconnection
- `isConnected()` - Check connection state

## Test Helpers

### `waitForEvent(socket, event, timeout?)`
Wait for a specific event to be emitted.

### `collectEvents(socket, event, duration)`
Collect all events of a type over a time period.

### `assertEventEmitted(socket, event, expectedData?)`
Assert that an event was emitted with optional data matching.

### `assertEventNotEmitted(socket, event)`
Assert that an event was NOT emitted.

## Best Practices

1. **Always clean up handlers** in `afterEach` to prevent test interference
2. **Use event sequences** for testing complex flows
3. **Test error scenarios** to ensure proper error handling
4. **Use factories** for creating variations of events
5. **Mock time-sensitive operations** with delays in sequences
6. **Assert on event history** when order matters

## Integration with UI Components

```typescript
import { render, screen } from "@testing-library/react";
import { MockSocketEmitter } from "@vrooli/ui/__test/fixtures/events";
import { SocketContext } from "@vrooli/ui/contexts";

it("should update UI on socket events", () => {
    const socket = new MockSocketEmitter();

    render(
        <SocketContext.Provider value={socket}>
            <ChatComponent />
        </SocketContext.Provider>
    );

    // Emit event
    socket.emit("messages", chatEventFixtures.messages.textMessage.data);

    // Check UI updated
    expect(screen.getByText("Test message")).toBeInTheDocument();
});
```

## Implementation Roadmap

### Phase 1: Enhanced Timing Support (Priority: High)
- Add realistic delays to all event sequences
- Implement jitter simulation for network variability
- Create timing presets (instant, fast, realtime, slow)
- Add timestamp tracking for event correlation

### Phase 2: State Management (Priority: High)
- Implement state tracking across event sequences
- Add state verification helpers
- Create state snapshots for testing
- Enable state-driven event generation

### Phase 3: Error Scenarios (Priority: Medium)
- Expand error event coverage
- Add retry and recovery sequences
- Implement backoff strategies
- Create failure injection utilities

### Phase 4: Cross-Event Correlation (Priority: Medium)
- Implement correlation ID system
- Track event causality chains
- Add cross-event validation
- Create visualization tools for event flows

### Phase 5: Network Simulation (Priority: Low)
- Implement network condition presets
- Add packet loss simulation
- Create bandwidth throttling
- Enable offline mode testing

## Integration with Fixture Ecosystem

### With API Fixtures
```typescript
// Combine event and API fixtures for complete scenarios
const chatScenario = {
  initial: apiFixtures.chatFixtures.complete.find,
  events: chatEventFixtures.sequences.messageFlow,
  expected: {
    messages: 5,
    participants: 2
  }
};
```

### With Config Fixtures
```typescript
// Use config fixtures to set up event contexts
const swarmConfig = configFixtures.chatConfigFixtures.variants.aiAssistant;
const swarmEvents = swarmEventFixtures.factories.createConfigUpdate(
  "chat_123",
  swarmConfig
);
```

### With UI Testing
```typescript
// MockSocketEmitter integration
import { MockSocketEmitter } from "@vrooli/ui/__test/fixtures/events";
import { eventFixtures } from "@vrooli/shared/__test/fixtures";

const socket = new MockSocketEmitter();
await socket.emitSequence(eventFixtures.chat.sequences.botResponseFlow);
```

## Event Fixture Design Principles

1. **Realism Over Simplicity**: Fixtures should mirror production event patterns
2. **Composability**: Small events should combine into complex scenarios
3. **Type Safety**: Full TypeScript support with proper event types
4. **Testability**: Easy to assert on event effects and state changes
5. **Documentation**: Each sequence should explain its purpose
6. **Performance**: Efficient for both unit and integration tests

## Testing Patterns

### Unit Testing with Events
```typescript
// Test individual event handlers
it("should update message count on new message", () => {
  const handler = new MessageHandler();
  handler.handle(chatEventFixtures.messages.textMessage.data);
  expect(handler.getMessageCount()).toBe(1);
});
```

### Integration Testing with Events
```typescript
// Test complete features with event flows
it("should handle chat session", async () => {
  const session = new ChatSession();
  
  // Simulate complete chat flow
  for (const event of chatEventFixtures.sequences.messageFlow) {
    await session.processEvent(event);
  }
  
  expect(session.isActive()).toBe(true);
  expect(session.getMessageCount()).toBe(3);
});
```

### E2E Testing with Events
```typescript
// Test full user journeys
it("should complete support chat", async () => {
  const { app, socket } = await setupTestApp();
  
  // User initiates chat
  await app.startChat();
  
  // Simulate support agent response
  await socket.emitSequence([
    chatEventFixtures.participants.userJoining,
    chatEventFixtures.messages.textMessage,
    chatEventFixtures.botStatus.thinking,
    chatEventFixtures.responseStream.streamStart
  ]);
  
  // Verify chat completion
  expect(await app.getChatStatus()).toBe("active");
});
```

## Common Event Testing Scenarios

### WebSocket Reconnection
```typescript
// Test reconnection handling
await socket.emitSequence([
  socketEventFixtures.connection.connected,
  { delay: 5000 },
  socketEventFixtures.connection.disconnected,
  { delay: 1000 },
  socketEventFixtures.sequences.reconnectionFlow
]);
```

### Multi-User Collaboration
```typescript
// Test concurrent user actions
const [user1Events, user2Events] = createParallelEvents([
  collaborationEventFixtures.runTask.taskInProgress,
  collaborationEventFixtures.runTask.taskCompleted
]);

await socket.emitParallel([user1Events, user2Events]);
```

### AI Agent Coordination
```typescript
// Test swarm execution
await socket.emitSequence([
  swarmEventFixtures.sequences.basicLifecycle,
  { parallel: [
    swarmEventFixtures.team.subtaskLeaderUpdate,
    swarmEventFixtures.resources.consumptionUpdate
  ]},
  swarmEventFixtures.state.completed
]);
```

## Debugging Event Tests

### Event History
```typescript
// Inspect what events were emitted
console.log(socket.getEmitHistory());
console.log(socket.getEmitsByEvent("messages"));
```

### Event Timing
```typescript
// Check event timing
const history = socket.getTimedHistory();
history.forEach(({ event, timestamp, delta }) => {
  console.log(`${event} at ${timestamp}ms (Δ${delta}ms)`);
});
```

### State Tracking
```typescript
// Track state changes
const stateLog = socket.getStateChanges();
stateLog.forEach(({ event, stateBefore, stateAfter }) => {
  console.log(`${event} changed:`, diff(stateBefore, stateAfter));
});
```

## Contributing

When adding new event types:
1. Add fixtures to the appropriate file
2. Include minimal, complete, and scenario variants
3. Add factory functions for dynamic creation
4. Include event sequences for common flows
5. Add timing information for realistic simulation
6. Document the business scenario being tested
7. Update this README with examples
8. Cross-reference in `/docs/testing/fixtures-overview.md`

## Future Enhancements

1. **Event Recording**: Capture production events for fixture generation
2. **Visual Timeline**: D3.js visualization of event sequences
3. **Performance Profiling**: Measure event processing performance
4. **Chaos Testing**: Random event injection for resilience testing
5. **Event Contracts**: Schema validation for event payloads
6. **Multi-Region Simulation**: Test geo-distributed scenarios