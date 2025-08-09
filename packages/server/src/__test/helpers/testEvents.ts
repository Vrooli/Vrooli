/**
 * Test-specific event types and type guards
 * 
 * This module provides type definitions and type guards for test-only events.
 * These events are used in test scenarios to simulate various system behaviors
 * and should not be used in production code.
 */

import type { ServiceEvent } from "../../services/events/types.js";

/**
 * Simplified event interface for tests that doesn't inherit production constraints
 */
export interface TestServiceEvent<T = any> {
    /** Unique event identifier */
    id: string;
    /** Event type */
    type: string;
    /** When the event occurred */
    timestamp: Date;
    /** Event-specific payload data */
    data: T;
    /** Additional metadata for routing and processing */
    metadata?: {
        userId?: string;
        priority?: "low" | "medium" | "high" | "critical";
        [key: string]: any;
    };
}

/**
 * Type definitions for test-specific events
 * These events are used exclusively in test scenarios
 */
export interface TestEventTypeMap {
    // Basic test events
    "test/simple": { message: string };
    "test/filtered": { important: boolean };
    "test/indexed": { index: number };
    "test/complex": { 
        id: string;
        data: Record<string, unknown>;
        metadata?: Record<string, unknown>;
    };

    // Mock routine execution event for testing scenario runners
    "routine/executed": {
        routineId: string;
        routineLabel: string;
        input: Record<string, unknown>;
        output: Record<string, unknown>;
        duration: number;
        success: boolean;
        timestamp?: Date;
        metadata?: Record<string, unknown>;
    };

    // Mock chat events for testing
    "test/chat/message": { 
        message: string;
        chatId: string;
        userId?: string;
    };

    // Generic test events with arbitrary properties
    "test/arbitrary": Record<string, unknown>;
    "test/error": { error: string; code?: number };
    "test/retry": { attempt: number; maxAttempts: number };
    "test/timeout": { timeoutMs: number };
    "test/batch": { items: unknown[]; batchId: string };
    "test/ordering": { index: number };
    "test/barrier_ordering": { index: number };
}

/**
 * Type guard to check if an event is a test event
 * @param event - The service event to check
 * @param type - The specific test event type to match against
 * @returns Type predicate indicating if the event matches the test type
 */
export function isTestEvent<T extends keyof TestEventTypeMap>(
    event: TestServiceEvent<any>,
    type: T,
): event is TestServiceEvent<TestEventTypeMap[T]> {
    return event.type === type;
}

/**
 * Test-specific event type guards
 * These guards provide type-safe access to test event data
 */
export const TestEventGuards = {
    // Basic test events
    isTestSimple: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/simple"]> =>
        event.type === "test/simple",

    isTestFiltered: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/filtered"]> =>
        event.type === "test/filtered",

    isTestIndexed: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/indexed"]> =>
        event.type === "test/indexed",

    isTestComplex: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/complex"]> =>
        event.type === "test/complex",

    // Mock routine events
    isRoutineExecuted: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["routine/executed"]> =>
        event.type === "routine/executed",

    // Mock chat events
    isTestChatMessage: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/chat/message"]> =>
        event.type === "test/chat/message",

    // Generic test events
    isTestArbitrary: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/arbitrary"]> =>
        event.type === "test/arbitrary",

    isTestError: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/error"]> =>
        event.type === "test/error",

    isTestRetry: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/retry"]> =>
        event.type === "test/retry",

    isTestTimeout: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/timeout"]> =>
        event.type === "test/timeout",

    isTestBatch: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/batch"]> =>
        event.type === "test/batch",

    isTestOrdering: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/ordering"]> =>
        event.type === "test/ordering",

    isTestBarrierOrdering: (event: TestServiceEvent<any>): event is TestServiceEvent<TestEventTypeMap["test/barrier_ordering"]> =>
        event.type === "test/barrier_ordering",
};

/**
 * Helper function to create a test event with proper typing
 * @param type - The test event type
 * @param data - The event data matching the type
 * @returns A properly typed test service event for testing
 */
export function createTestEvent<T extends keyof TestEventTypeMap>(
    type: T,
    data: TestEventTypeMap[T],
): Omit<TestServiceEvent<TestEventTypeMap[T]>, "id" | "timestamp"> {
    return {
        type,
        data,
        metadata: {
            userId: "test-user",
            priority: "medium",
        },
    };
}

/**
 * Helper function to check if an event type is a test event
 * @param eventType - The event type string to check
 * @returns True if the event type is a test event
 */
export function isTestEventType(eventType: string): eventType is keyof TestEventTypeMap {
    return eventType.startsWith("test/") || eventType === "routine/executed";
}

/**
 * Create a mock routine execution event for testing
 * @param routineId - The routine ID
 * @param success - Whether the routine was successful
 * @param overrides - Additional properties to override defaults
 * @returns A test routine execution event
 */
export function createMockRoutineExecutedEvent(
    routineId: string,
    success = true,
    overrides: Partial<TestEventTypeMap["routine/executed"]> = {},
): Omit<ServiceEvent<TestEventTypeMap["routine/executed"]>, "id" | "timestamp"> {
    return createTestEvent("routine/executed", {
        routineId,
        routineLabel: `routine-${routineId}`,
        input: { testInput: "mock" },
        output: success ? { result: "success" } : { error: "mock failure" },
        duration: 100,
        success,
        timestamp: new Date(),
        ...overrides,
    });
}

/**
 * Create a mock chat message event for testing
 * @param message - The message content
 * @param chatId - The chat ID
 * @param userId - Optional user ID
 * @returns A test chat message event
 */
export function createMockChatMessageEvent(
    message: string,
    chatId: string,
    userId?: string,
): Omit<ServiceEvent<TestEventTypeMap["test/chat/message"]>, "id" | "timestamp"> {
    return createTestEvent("test/chat/message", {
        message,
        chatId,
        userId,
    });
}
