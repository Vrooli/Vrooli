/**
 * Type definitions for the Event Fixture Factory system
 * Provides comprehensive typing for event simulation and testing
 */

// Base types for event handling
export interface BaseEvent {
    event: string;
    data: unknown;
}

// Timing support for realistic event simulation
export interface TimedEvent<T = unknown> extends BaseEvent {
    event: string;
    data: T;
    timing: {
        delay?: number;          // Delay before event (ms)
        jitter?: number;         // Random variance in delay (ms)
        timestamp?: number;      // Absolute timestamp
    };
}

// Event sequences for complex flows
export interface EventSequenceItem<T = unknown> {
    event?: string;
    data?: T;
    delay?: number;              // Delay before this event
    parallel?: EventSequenceItem<T>[];  // Events to emit in parallel
}

// Correlated events for tracking causality
export interface CorrelatedEvent<T = unknown> extends BaseEvent {
    event: string;
    data: T;
    metadata: {
        correlationId: string;
        sequence: number;
        timestamp: number;
        causedBy?: string;       // Event that triggered this
        causes?: string[];       // Events this will trigger
    };
}

// State tracking for event-driven state changes
export interface StatefulEvent<T = unknown> extends BaseEvent {
    event: string;
    data: T;
    state: {
        before: Record<string, unknown>;
        after: Record<string, unknown>;
        changed: string[];       // List of changed properties
    };
}

// Network simulation options
export interface NetworkCondition {
    latency: number;            // Base latency (ms)
    jitter: number;             // Latency variance (ms)
    loss: number;               // Packet loss percentage (0-100)
    bandwidth?: number;         // Bandwidth limit (bytes/s)
}

export interface SimulationOptions {
    network?: "fast" | "slow" | "flaky" | "offline" | NetworkCondition;
    timing?: "realtime" | "fast" | "instant";
    errors?: ErrorSimulation[];
    state?: Record<string, unknown>;
}

export interface ErrorSimulation {
    event: string;
    errorType: "network" | "timeout" | "validation" | "permission" | "system";
    probability: number;        // 0-1 chance of error
    details?: unknown;
}

// Event patterns for factory creation
export type EventPattern = 
    | "single"                  // One-time event
    | "burst"                   // Multiple events at once
    | "periodic"                // Regular intervals
    | "random"                  // Random timing
    | "escalating"              // Increasing frequency
    | "degrading";              // Decreasing frequency

// Test result from simulation
export interface TestResult {
    success: boolean;
    events: Array<{
        event: string;
        timestamp: number;
        data: unknown;
        error?: Error;
    }>;
    stateChanges: Array<{
        timestamp: number;
        property: string;
        oldValue: unknown;
        newValue: unknown;
    }>;
    errors: Error[];
    duration: number;
}

// Event effect for assertion
export interface EventEffect {
    type: "state" | "event" | "error" | "timing";
    property?: string;
    value?: unknown;
    event?: string;
    minDelay?: number;
    maxDelay?: number;
}

// State change tracking
export interface StateChangeLog {
    changes: Array<{
        event: string;
        timestamp: number;
        stateBefore: Record<string, unknown>;
        stateAfter: Record<string, unknown>;
        diff: Array<{
            property: string;
            before: unknown;
            after: unknown;
        }>;
    }>;
}

// Factory options for creating events
export interface EventFactoryOptions<T> {
    defaults?: Partial<T>;
    validation?: (event: T) => boolean | string;
    transform?: (event: T) => T;
}

// Main Event Fixture Factory interface
export interface EventFixtureFactory<TEvent, TData = unknown> {
    // Core event data
    single: TEvent;
    sequence: TEvent[];
    variants: Record<string, TEvent | TEvent[]>;
    
    // Factory methods
    create(overrides?: Partial<TData>): TEvent;
    createSequence(pattern: EventPattern, options?: {
        count?: number;
        interval?: number;
        data?: Partial<TData>;
    }): TEvent[];
    createCorrelated(correlationId: string, events: TEvent[]): CorrelatedEvent<TData>[];
    
    // Timing and simulation
    withTiming(events: TEvent[], intervals: number[]): TimedEvent<TData>[];
    withDelay(event: TEvent, delay: number): TimedEvent<TData>;
    withJitter(events: TEvent[], baseDelay: number, jitter: number): TimedEvent<TData>[];
    
    // State management
    withState(event: TEvent, state: { before: Record<string, unknown>; after: Record<string, unknown> }): StatefulEvent<TData>;
    trackStateChanges(events: TEvent[]): StateChangeLog;
    
    // Testing helpers
    validateEventOrder(events: TEvent[]): boolean;
    simulateEventFlow(events: TEvent[], options?: SimulationOptions): Promise<TestResult>;
    assertEventEffects(event: TEvent, expectedEffects: EventEffect[]): void;
}

// Socket-specific types
export interface SocketEventFixture<T = unknown> extends BaseEvent {
    event: string;
    data: T;
    room?: string;              // Room context for the event
    broadcast?: boolean;        // Whether event is broadcast
    acknowledgment?: unknown;   // Expected acknowledgment
}

// Helper type for creating fixture factories
export type CreateEventFixtureFactory<TEvent, TData = unknown> = (
    options?: EventFactoryOptions<TData>
) => EventFixtureFactory<TEvent, TData>;

// Utility type for extracting event data type
export type EventDataType<T> = T extends { data: infer D } ? D : never;

// Utility type for event names
export type EventName<T> = T extends { event: infer E } ? E : never;
