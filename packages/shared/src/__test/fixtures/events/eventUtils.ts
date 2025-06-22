/**
 * Utility functions for event fixture testing
 * Provides helpers for timing, state tracking, correlations, and testing
 */

import {
    type CorrelatedEvent,
    type EventSequenceItem,
    type NetworkCondition,
    type TimedEvent,
} from "./types.js";

/**
 * Generate a unique correlation ID for event tracking
 */
export function generateCorrelationId(): string {
    return `corr_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * Create a delay promise for event timing
 */
export function delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Apply network conditions to a delay value
 */
export function applyNetworkDelay(
    baseDelay: number,
    condition: NetworkCondition,
): number {
    const jitterValue = (Math.random() - 0.5) * 2 * condition.jitter;
    return Math.max(0, baseDelay + condition.latency + jitterValue);
}

/**
 * Check if an event should be dropped based on packet loss
 */
export function shouldDropPacket(lossPercentage: number): boolean {
    return Math.random() * 100 < lossPercentage;
}

/**
 * Convert event sequence items to timed events
 */
export function sequenceToTimedEvents<T>(
    sequence: EventSequenceItem<T>[],
): TimedEvent<T>[] {
    const timedEvents: TimedEvent<T>[] = [];
    let cumulativeDelay = 0;

    for (const item of sequence) {
        if (item.delay) {
            cumulativeDelay += item.delay;
        }

        if (item.event && item.data) {
            timedEvents.push({
                event: item.event,
                data: item.data,
                timing: {
                    delay: cumulativeDelay,
                    timestamp: Date.now() + cumulativeDelay,
                },
            });
        }

        // Handle parallel events
        if (item.parallel) {
            const parallelTimed = sequenceToTimedEvents(item.parallel);
            parallelTimed.forEach(pe => {
                pe.timing.delay = cumulativeDelay;
                pe.timing.timestamp = Date.now() + cumulativeDelay;
                timedEvents.push(pe);
            });
        }
    }

    return timedEvents;
}

/**
 * Create a state differ for tracking changes
 */
export class StateDiffer {
    private snapshots: Map<string, Record<string, unknown>> = new Map();

    /**
     * Take a snapshot of current state
     */
    snapshot(id: string, state: Record<string, unknown>): void {
        this.snapshots.set(id, JSON.parse(JSON.stringify(state)));
    }

    /**
     * Compare current state with a snapshot
     */
    diff(id: string, currentState: Record<string, unknown>): {
        added: string[];
        removed: string[];
        changed: Array<{ key: string; before: unknown; after: unknown }>;
    } {
        const snapshot = this.snapshots.get(id);
        if (!snapshot) {
            throw new Error(`No snapshot found with id: ${id}`);
        }

        const snapshotKeys = Object.keys(snapshot);
        const currentKeys = Object.keys(currentState);
        const snapshotKeySet = new Set(snapshotKeys);
        const currentKeySet = new Set(currentKeys);

        const added = currentKeys.filter(k => !snapshotKeySet.has(k));
        const removed = snapshotKeys.filter(k => !currentKeySet.has(k));
        const changed: Array<{ key: string; before: unknown; after: unknown }> = [];

        for (const key of snapshotKeys) {
            if (currentKeySet.has(key) && snapshot[key] !== currentState[key]) {
                changed.push({
                    key,
                    before: snapshot[key],
                    after: currentState[key],
                });
            }
        }

        return { added, removed, changed };
    }

    /**
     * Clear all snapshots
     */
    clear(): void {
        this.snapshots.clear();
    }
}

/**
 * Event correlation tracker
 */
export class EventCorrelator {
    private correlations: Map<string, CorrelatedEvent[]> = new Map();
    private eventChains: Map<string, string[]> = new Map();

    /**
     * Track a correlated event
     */
    track<T>(event: CorrelatedEvent<T>): void {
        const { correlationId } = event.metadata;

        if (!this.correlations.has(correlationId)) {
            this.correlations.set(correlationId, []);
            this.eventChains.set(correlationId, []);
        }

        this.correlations.get(correlationId)!.push(event);
        this.eventChains.get(correlationId)!.push(event.event);

        // Update causality links
        if (event.metadata.causedBy) {
            this.linkCausality(correlationId, event.metadata.causedBy, event.event);
        }
    }

    /**
     * Get all events for a correlation ID
     */
    getCorrelatedEvents(correlationId: string): CorrelatedEvent[] {
        return this.correlations.get(correlationId) || [];
    }

    /**
     * Get event chain for a correlation
     */
    getEventChain(correlationId: string): string[] {
        return this.eventChains.get(correlationId) || [];
    }

    /**
     * Check if event chain matches expected pattern
     */
    matchesPattern(correlationId: string, pattern: string[]): boolean {
        const chain = this.getEventChain(correlationId);
        if (chain.length !== pattern.length) return false;

        return chain.every((event, index) => {
            if (pattern[index] === "*") return true; // Wildcard
            return event === pattern[index];
        });
    }

    /**
     * Get causality graph
     */
    getCausalityGraph(correlationId: string): Map<string, string[]> {
        const graph = new Map<string, string[]>();
        const events = this.getCorrelatedEvents(correlationId);

        events.forEach(event => {
            if (event.metadata.causes && event.metadata.causes.length > 0) {
                graph.set(event.event, event.metadata.causes);
            }
        });

        return graph;
    }

    private linkCausality(correlationId: string, causeEvent: string, effectEvent: string): void {
        const events = this.correlations.get(correlationId);
        if (!events) return;

        const causeEventObj = events.find(e => e.event === causeEvent);
        if (causeEventObj && causeEventObj.metadata.causes) {
            if (!causeEventObj.metadata.causes.includes(effectEvent)) {
                causeEventObj.metadata.causes.push(effectEvent);
            }
        }
    }
}

/**
 * Event timing analyzer
 */
export class TimingAnalyzer {
    private events: Array<{ event: string; timestamp: number }> = [];

    /**
     * Record an event occurrence
     */
    record(event: string): void {
        this.events.push({ event, timestamp: Date.now() });
    }

    /**
     * Get timing statistics for events
     */
    getStats(): {
        totalDuration: number;
        eventCount: number;
        averageInterval: number;
        minInterval: number;
        maxInterval: number;
        eventFrequency: Map<string, number>;
    } {
        if (this.events.length === 0) {
            return {
                totalDuration: 0,
                eventCount: 0,
                averageInterval: 0,
                minInterval: 0,
                maxInterval: 0,
                eventFrequency: new Map(),
            };
        }

        const sorted = [...this.events].sort((a, b) => a.timestamp - b.timestamp);
        const totalDuration = sorted[sorted.length - 1].timestamp - sorted[0].timestamp;

        const intervals: number[] = [];
        for (let i = 1; i < sorted.length; i++) {
            intervals.push(sorted[i].timestamp - sorted[i - 1].timestamp);
        }

        const eventFrequency = new Map<string, number>();
        this.events.forEach(e => {
            eventFrequency.set(e.event, (eventFrequency.get(e.event) || 0) + 1);
        });

        return {
            totalDuration,
            eventCount: this.events.length,
            averageInterval: intervals.length > 0 ? intervals.reduce((a, b) => a + b, 0) / intervals.length : 0,
            minInterval: intervals.length > 0 ? Math.min(...intervals) : 0,
            maxInterval: intervals.length > 0 ? Math.max(...intervals) : 0,
            eventFrequency,
        };
    }

    /**
     * Check if events occurred within expected time window
     */
    withinWindow(event1: string, event2: string, maxDelay: number): boolean {
        const e1 = this.events.find(e => e.event === event1);
        const e2 = this.events.find(e => e.event === event2);

        if (!e1 || !e2) return false;

        return Math.abs(e2.timestamp - e1.timestamp) <= maxDelay;
    }

    /**
     * Clear recorded events
     */
    clear(): void {
        this.events = [];
    }
}

/**
 * Wait for an event with timeout
 */
export function waitForEvent<T>(
    emitter: { once: (event: string, handler: (data: T) => void) => void },
    event: string,
    timeout = 5000,
): Promise<T> {
    return new Promise((resolve, reject) => {
        const timer = setTimeout(() => {
            reject(new Error(`Timeout waiting for event: ${event}`));
        }, timeout);

        emitter.once(event, (data: T) => {
            clearTimeout(timer);
            resolve(data);
        });
    });
}

/**
 * Collect events over a time period
 */
export async function collectEvents<T>(
    emitter: {
        on: (event: string, handler: (data: T) => void) => void;
        off: (event: string, handler: (data: T) => void) => void;
    },
    event: string,
    duration: number,
): Promise<T[]> {
    const events: T[] = [];

    const handler = (data: T) => {
        events.push(data);
    };

    emitter.on(event, handler);
    await delay(duration);
    emitter.off(event, handler);

    return events;
}

/**
 * Network condition presets
 */
export const networkPresets = {
    fiber: { latency: 5, jitter: 1, loss: 0 },
    cable: { latency: 20, jitter: 5, loss: 0.1 },
    dsl: { latency: 40, jitter: 10, loss: 0.2 },
    mobile4G: { latency: 50, jitter: 10, loss: 0.5 },
    mobile3G: { latency: 200, jitter: 50, loss: 2 },
    mobile2G: { latency: 400, jitter: 100, loss: 5 },
    satellite: { latency: 600, jitter: 100, loss: 3 },
    congested: { latency: 100, jitter: 200, loss: 10 },
};

/**
 * Create event assertion helpers
 */
export function createEventAssertions<T>(events: T[]) {
    return {
        /**
         * Assert event was emitted
         */
        emitted(predicate: (event: T) => boolean, message?: string): void {
            const found = events.some(predicate);
            if (!found) {
                throw new Error(message || "Expected event was not emitted");
            }
        },

        /**
         * Assert event was NOT emitted
         */
        notEmitted(predicate: (event: T) => boolean, message?: string): void {
            const found = events.some(predicate);
            if (found) {
                throw new Error(message || "Unexpected event was emitted");
            }
        },

        /**
         * Assert event count
         */
        count(expectedCount: number, predicate?: (event: T) => boolean): void {
            const filtered = predicate ? events.filter(predicate) : events;
            if (filtered.length !== expectedCount) {
                throw new Error(
                    `Expected ${expectedCount} events, but got ${filtered.length}`,
                );
            }
        },

        /**
         * Assert event order
         */
        inOrder(predicates: Array<(event: T) => boolean>): void {
            let eventIndex = 0;

            for (const predicate of predicates) {
                const foundIndex = events.findIndex((e, i) => i >= eventIndex && predicate(e));
                if (foundIndex === -1) {
                    throw new Error("Events did not occur in expected order");
                }
                eventIndex = foundIndex + 1;
            }
        },
    };
}
