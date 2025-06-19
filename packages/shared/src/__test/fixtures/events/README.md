# Real-time Event Fixtures

This directory contains comprehensive fixtures for testing real-time WebSocket events across the Vrooli platform.

## Overview

The event fixtures provide:
- Pre-configured event payloads for all socket event types
- Event sequences for testing complex flows
- Factory functions for creating custom events
- Mock socket emitter utilities for testing

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

## Contributing

When adding new event types:
1. Add fixtures to the appropriate file
2. Include minimal, complete, and scenario variants
3. Add factory functions for dynamic creation
4. Include event sequences for common flows
5. Update this README with examples