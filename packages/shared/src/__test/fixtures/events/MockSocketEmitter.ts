/**
 * Enhanced Mock Socket Emitter for comprehensive event testing
 * Provides advanced features for simulating WebSocket behavior
 */

import {
    EventCorrelator,
    StateDiffer,
    TimingAnalyzer,
    applyNetworkDelay,
    delay,
    networkPresets,
    shouldDropPacket,
} from "./eventUtils.js";
import {
    type CorrelatedEvent,
    type EventSequenceItem,
    type NetworkCondition,
} from "./types.js";

export interface EmitHistoryItem {
    event: string;
    data: unknown;
    timestamp: number;
    room?: string;
    broadcast?: boolean;
    dropped?: boolean;
    error?: Error;
}

export interface MockSocketEmitterOptions {
    simulateNetwork?: boolean;
    networkCondition?: NetworkCondition | keyof typeof networkPresets;
    autoAcknowledge?: boolean;
    stateTracking?: boolean;
    correlationTracking?: boolean;
}

export class MockSocketEmitter {
    private handlers: Map<string, Set<Function>> = new Map();
    private rooms: Set<string> = new Set();
    private emitHistory: EmitHistoryItem[] = [];
    private connected = false;
    private options: MockSocketEmitterOptions;

    // Advanced features
    private stateTracker?: StateDiffer;
    private correlator?: EventCorrelator;
    private timingAnalyzer: TimingAnalyzer = new TimingAnalyzer();
    private state: Record<string, unknown> = {};

    constructor(options: MockSocketEmitterOptions = {}) {
        this.options = {
            simulateNetwork: false,
            autoAcknowledge: true,
            stateTracking: false,
            correlationTracking: false,
            ...options,
        };

        if (this.options.stateTracking) {
            this.stateTracker = new StateDiffer();
        }

        if (this.options.correlationTracking) {
            this.correlator = new EventCorrelator();
        }
    }

    // Core event emitter methods

    on(event: string, handler: Function): void {
        if (!this.handlers.has(event)) {
            this.handlers.set(event, new Set());
        }
        this.handlers.get(event)!.add(handler);
    }

    once(event: string, handler: Function): void {
        const onceHandler = (...args: unknown[]) => {
            handler(...args);
            this.off(event, onceHandler);
        };
        this.on(event, onceHandler);
    }

    off(event: string, handler?: Function): void {
        if (!handler) {
            this.handlers.delete(event);
        } else {
            this.handlers.get(event)?.delete(handler);
        }
    }

    async emit(event: string, data: unknown, callback?: Function): Promise<void> {
        // Apply network simulation if enabled
        if (this.options.simulateNetwork) {
            const networkDelay = await this.simulateNetworkConditions();
            if (networkDelay === null) {
                // Packet dropped
                this.recordEmit(event, data, { dropped: true });
                return;
            }
            await delay(networkDelay);
        }

        // Record emit
        this.recordEmit(event, data);

        // Track timing
        this.timingAnalyzer.record(event);

        // Handle correlated events
        if (this.correlator && this.isCorrelatedEvent(data)) {
            this.correlator.track(data as CorrelatedEvent);
        }

        // Update state if tracking
        if (this.stateTracker) {
            this.updateState(event, data);
        }

        // Trigger handlers
        const handlers = this.handlers.get(event);
        if (handlers) {
            const results = await Promise.all(
                Array.from(handlers).map(handler =>
                    this.safeExecuteHandler(handler, data),
                ),
            );

            // Auto-acknowledge if enabled
            if (callback && this.options.autoAcknowledge) {
                const success = results.every(r => r !== false);
                callback({ success });
            }
        } else if (callback && this.options.autoAcknowledge) {
            callback({ success: true });
        }
    }

    emitWithAck(event: string, data: unknown, callback: Function): void {
        this.emit(event, data, callback);
    }

    // Room management

    joinRoom(room: string): void {
        this.rooms.add(room);
        this.emit("room:joined", { room });
    }

    leaveRoom(room: string): void {
        this.rooms.delete(room);
        this.emit("room:left", { room });
    }

    inRoom(room: string): boolean {
        return this.rooms.has(room);
    }

    getRooms(): string[] {
        return Array.from(this.rooms);
    }

    // Connection management

    connect(): void {
        this.connected = true;
        this.emit("connect", {});
    }

    disconnect(): void {
        this.connected = false;
        this.rooms.clear();
        this.emit("disconnect", {});
    }

    isConnected(): boolean {
        return this.connected;
    }

    // Advanced testing features

    async emitSequence(sequence: EventSequenceItem[]): Promise<void> {
        for (const item of sequence) {
            if (item.delay) {
                await delay(item.delay);
            }

            if (item.event && item.data !== undefined) {
                await this.emit(item.event, item.data);
            }

            if (item.parallel) {
                await Promise.all(
                    item.parallel.map(pItem =>
                        pItem.event && pItem.data !== undefined
                            ? this.emit(pItem.event, pItem.data)
                            : Promise.resolve(),
                    ),
                );
            }
        }
    }

    async emitParallel(events: Array<{ event: string; data: unknown }>): Promise<void> {
        await Promise.all(
            events.map(({ event, data }) => this.emit(event, data)),
        );
    }

    // History and debugging

    getEmitHistory(): EmitHistoryItem[] {
        return [...this.emitHistory];
    }

    getEmitsByEvent(event: string): EmitHistoryItem[] {
        return this.emitHistory.filter(h => h.event === event);
    }

    getTimedHistory(): Array<EmitHistoryItem & { delta: number }> {
        return this.emitHistory.map((item, index) => ({
            ...item,
            delta: index > 0 ? item.timestamp - this.emitHistory[index - 1].timestamp : 0,
        }));
    }

    clearHistory(): void {
        this.emitHistory = [];
        this.timingAnalyzer.clear();
    }

    hasHandlers(event: string): boolean {
        return this.handlers.has(event) && this.handlers.get(event)!.size > 0;
    }

    getHandlerCount(event?: string): number {
        if (event) {
            return this.handlers.get(event)?.size || 0;
        }

        let total = 0;
        this.handlers.forEach(handlers => {
            total += handlers.size;
        });
        return total;
    }

    // State management

    setState(state: Record<string, unknown>): void {
        this.state = { ...state };
        if (this.stateTracker) {
            this.stateTracker.snapshot("current", this.state);
        }
    }

    getState(): Record<string, unknown> {
        return { ...this.state };
    }

    getStateChanges(): Array<{
        event: string;
        stateBefore: Record<string, unknown>;
        stateAfter: Record<string, unknown>;
        diff: Array<{ property: string; before: unknown; after: unknown }>;
    }> {
        if (!this.stateTracker) {
            throw new Error("State tracking not enabled");
        }

        // This would need to be enhanced to track full history
        return [];
    }

    // Correlation tracking

    getCorrelatedEvents(correlationId: string): CorrelatedEvent[] {
        if (!this.correlator) {
            throw new Error("Correlation tracking not enabled");
        }
        return this.correlator.getCorrelatedEvents(correlationId);
    }

    getEventChain(correlationId: string): string[] {
        if (!this.correlator) {
            throw new Error("Correlation tracking not enabled");
        }
        return this.correlator.getEventChain(correlationId);
    }

    // Timing analysis

    getTimingStats() {
        return this.timingAnalyzer.getStats();
    }

    // Error injection

    injectError(event: string, error: Error): void {
        this.recordEmit(event, null, { error });
        const handlers = this.handlers.get(event);
        if (handlers) {
            handlers.forEach(handler => {
                try {
                    handler(null, error);
                } catch (e) {
                    // Handler threw on error
                }
            });
        }
    }

    // Network simulation

    setNetworkCondition(condition: NetworkCondition | keyof typeof networkPresets): void {
        this.options.networkCondition = condition;
        this.options.simulateNetwork = true;
    }

    disableNetworkSimulation(): void {
        this.options.simulateNetwork = false;
    }

    // Private helper methods

    private recordEmit(
        event: string,
        data: unknown,
        extra: Partial<EmitHistoryItem> = {},
    ): void {
        this.emitHistory.push({
            event,
            data,
            timestamp: Date.now(),
            room: this.getCurrentRoom(),
            broadcast: this.isBroadcast(event),
            ...extra,
        });
    }

    private async simulateNetworkConditions(): Promise<number | null> {
        const condition = this.getNetworkCondition();

        // Check packet loss
        if (shouldDropPacket(condition.loss)) {
            return null;
        }

        // Apply latency with jitter
        return applyNetworkDelay(0, condition);
    }

    private getNetworkCondition(): NetworkCondition {
        const { networkCondition } = this.options;

        if (!networkCondition) {
            return networkPresets.fiber;
        }

        if (typeof networkCondition === "string") {
            return networkPresets[networkCondition] || networkPresets.fiber;
        }

        return networkCondition;
    }

    private isCorrelatedEvent(data: unknown): boolean {
        return (
            typeof data === "object" &&
            data !== null &&
            "metadata" in data &&
            typeof (data as CorrelatedEvent).metadata === "object" &&
            "correlationId" in (data as CorrelatedEvent).metadata
        );
    }

    private updateState(event: string, data: unknown): void {
        // Override in specific implementations to define state mutations
        this.state = {
            ...this.state,
            lastEvent: event,
            lastEventData: data,
            lastEventTime: Date.now(),
        };
    }

    private async safeExecuteHandler(handler: Function, data: unknown): Promise<unknown> {
        try {
            const result = handler(data);
            if (result instanceof Promise) {
                return await result;
            }
            return result;
        } catch (error) {
            console.error("Handler error:", error);
            return false;
        }
    }

    private getCurrentRoom(): string | undefined {
        // Return the first room (simplified - real implementation might be different)
        return Array.from(this.rooms)[0];
    }

    private isBroadcast(event: string): boolean {
        // Simple heuristic - events starting with 'broadcast:' are broadcasts
        return event.startsWith("broadcast:");
    }

    // Test assertion helpers

    assertEventEmitted(event: string, data?: unknown): void {
        const found = this.emitHistory.some(h => {
            if (h.event !== event) return false;
            if (data === undefined) return true;
            return JSON.stringify(h.data) === JSON.stringify(data);
        });

        if (!found) {
            throw new Error(`Event "${event}" was not emitted${data ? " with expected data" : ""}`);
        }
    }

    assertEventNotEmitted(event: string): void {
        const found = this.emitHistory.some(h => h.event === event);
        if (found) {
            throw new Error(`Event "${event}" was unexpectedly emitted`);
        }
    }

    assertEventCount(event: string, expectedCount: number): void {
        const actualCount = this.getEmitsByEvent(event).length;
        if (actualCount !== expectedCount) {
            throw new Error(
                `Expected ${expectedCount} "${event}" events, but got ${actualCount}`,
            );
        }
    }

    assertEventOrder(expectedOrder: string[]): void {
        const actualOrder = this.emitHistory.map(h => h.event);

        let actualIndex = 0;
        for (const expectedEvent of expectedOrder) {
            const foundIndex = actualOrder.indexOf(expectedEvent, actualIndex);
            if (foundIndex === -1) {
                throw new Error(
                    `Expected event "${expectedEvent}" not found in correct order`,
                );
            }
            actualIndex = foundIndex + 1;
        }
    }
}
