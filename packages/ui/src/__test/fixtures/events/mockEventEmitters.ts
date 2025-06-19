/**
 * Mock event emitter utilities for testing real-time features
 */

import { type SocketEvent, type SocketEventPayloads, type SocketEventCallbackPayloads } from "@vrooli/shared/consts/socketEvents";

/**
 * Event sequence item for testing event flows
 */
export interface EventSequenceItem {
    event: string;
    data?: any;
    delay?: number;
    callback?: any;
}

/**
 * Mock socket emitter for testing WebSocket events
 */
export class MockSocketEmitter {
    private handlers: Map<string, Set<Function>> = new Map();
    private emitHistory: Array<{ event: string; data: any; timestamp: number }> = [];
    private connected = false;
    private rooms: Set<string> = new Set();

    constructor(autoConnect = true) {
        if (autoConnect) {
            this.connect();
        }
    }

    /**
     * Simulate connection
     */
    connect() {
        this.connected = true;
        this.emit("connect", undefined);
    }

    /**
     * Simulate disconnection
     */
    disconnect() {
        this.connected = false;
        this.rooms.clear();
        this.emit("disconnect", undefined);
    }

    /**
     * Check if connected
     */
    isConnected(): boolean {
        return this.connected;
    }

    /**
     * Register event handler
     */
    on<T extends SocketEvent>(event: T, handler: (data: SocketEventPayloads[T]) => void): this {
        if (!this.handlers.has(event)) {
            this.handlers.set(event, new Set());
        }
        this.handlers.get(event)!.add(handler);
        return this;
    }

    /**
     * Register one-time event handler
     */
    once<T extends SocketEvent>(event: T, handler: (data: SocketEventPayloads[T]) => void): this {
        const wrappedHandler = (data: SocketEventPayloads[T]) => {
            handler(data);
            this.off(event, wrappedHandler);
        };
        return this.on(event, wrappedHandler);
    }

    /**
     * Remove event handler
     */
    off<T extends SocketEvent>(event: T, handler?: Function): this {
        if (!handler) {
            this.handlers.delete(event);
        } else {
            const handlers = this.handlers.get(event);
            if (handlers) {
                handlers.delete(handler);
                if (handlers.size === 0) {
                    this.handlers.delete(event);
                }
            }
        }
        return this;
    }

    /**
     * Emit event
     */
    emit<T extends SocketEvent>(event: T, data: SocketEventPayloads[T], callback?: any): this {
        this.emitHistory.push({ event, data, timestamp: Date.now() });

        const handlers = this.handlers.get(event);
        if (handlers) {
            handlers.forEach(handler => {
                try {
                    handler(data);
                } catch (error) {
                    console.error(`Error in handler for event ${event}:`, error);
                }
            });
        }

        // Handle room events with callbacks
        if (callback && this.isRoomEvent(event)) {
            this.handleRoomCallback(event, data, callback);
        }

        return this;
    }

    /**
     * Emit event with callback (for room join/leave events)
     */
    emitWithAck<T extends keyof SocketEventCallbackPayloads>(
        event: T,
        data: SocketEventPayloads[T],
        callback: (response: SocketEventCallbackPayloads[T]) => void
    ): this {
        this.emit(event as SocketEvent, data);
        
        // Simulate async callback
        setTimeout(() => {
            const response = this.generateCallbackResponse(event, data);
            callback(response);
        }, 100);

        return this;
    }

    /**
     * Emit a sequence of events with delays
     */
    async emitSequence(sequence: EventSequenceItem[]): Promise<void> {
        for (const item of sequence) {
            if (item.delay) {
                await new Promise(resolve => setTimeout(resolve, item.delay));
            }
            if ("event" in item) {
                this.emit(item.event as SocketEvent, item.data, item.callback);
            }
        }
    }

    /**
     * Get emit history
     */
    getEmitHistory(): Array<{ event: string; data: any; timestamp: number }> {
        return [...this.emitHistory];
    }

    /**
     * Get events of a specific type from history
     */
    getEmitsByEvent(event: string): Array<{ data: any; timestamp: number }> {
        return this.emitHistory
            .filter(e => e.event === event)
            .map(({ data, timestamp }) => ({ data, timestamp }));
    }

    /**
     * Clear emit history
     */
    clearHistory(): void {
        this.emitHistory = [];
    }

    /**
     * Get count of handlers for an event
     */
    getHandlerCount(event: string): number {
        return this.handlers.get(event)?.size || 0;
    }

    /**
     * Check if event has handlers
     */
    hasHandlers(event: string): boolean {
        return this.getHandlerCount(event) > 0;
    }

    /**
     * Simulate room join
     */
    joinRoom(room: string): void {
        this.rooms.add(room);
    }

    /**
     * Simulate room leave
     */
    leaveRoom(room: string): void {
        this.rooms.delete(room);
    }

    /**
     * Check if in room
     */
    inRoom(room: string): boolean {
        return this.rooms.has(room);
    }

    /**
     * Get all joined rooms
     */
    getRooms(): string[] {
        return Array.from(this.rooms);
    }

    // Private helper methods

    private isRoomEvent(event: string): event is keyof SocketEventCallbackPayloads {
        return [
            "joinChatRoom", "leaveChatRoom",
            "joinRunRoom", "leaveRunRoom",
            "joinUserRoom", "leaveUserRoom",
            "requestCancellation"
        ].includes(event);
    }

    private handleRoomCallback(event: string, data: any, callback: Function): void {
        setTimeout(() => {
            const response = this.generateCallbackResponse(event as keyof SocketEventCallbackPayloads, data);
            callback(response);
        }, 100);
    }

    private generateCallbackResponse<T extends keyof SocketEventCallbackPayloads>(
        event: T,
        data: any
    ): SocketEventCallbackPayloads[T] {
        // Simulate successful responses by default
        switch (event) {
            case "joinChatRoom":
                this.joinRoom(`chat:${data.chatId}`);
                return { success: true } as SocketEventCallbackPayloads[T];
            case "leaveChatRoom":
                this.leaveRoom(`chat:${data.chatId}`);
                return { success: true } as SocketEventCallbackPayloads[T];
            case "joinRunRoom":
                this.joinRoom(`run:${data.runId}`);
                return { success: true } as SocketEventCallbackPayloads[T];
            case "leaveRunRoom":
                this.leaveRoom(`run:${data.runId}`);
                return { success: true } as SocketEventCallbackPayloads[T];
            case "joinUserRoom":
                this.joinRoom(`user:${data.userId}`);
                return { success: true } as SocketEventCallbackPayloads[T];
            case "leaveUserRoom":
                this.leaveRoom(`user:${data.userId}`);
                return { success: true } as SocketEventCallbackPayloads[T];
            case "requestCancellation":
                return { success: true } as SocketEventCallbackPayloads[T];
            default:
                return { success: false, error: "Unknown event" } as SocketEventCallbackPayloads[T];
        }
    }
}

/**
 * Create a mock socket with pre-configured behavior
 */
export function createMockSocket(config?: {
    autoConnect?: boolean;
    failRoomJoins?: boolean;
    simulateDisconnects?: boolean;
}): MockSocketEmitter {
    const socket = new MockSocketEmitter(config?.autoConnect ?? true);

    if (config?.failRoomJoins) {
        // Override callback responses to simulate failures
        const originalEmit = socket.emit.bind(socket);
        socket.emit = function(event: any, data: any, callback?: any) {
            if (event.includes("join") && callback) {
                setTimeout(() => {
                    callback({ success: false, error: "Simulated failure" });
                }, 100);
                return this;
            }
            return originalEmit(event, data, callback);
        };
    }

    if (config?.simulateDisconnects) {
        // Randomly disconnect after some time
        setTimeout(() => {
            if (socket.isConnected()) {
                socket.disconnect();
                // Reconnect after a delay
                setTimeout(() => {
                    socket.connect();
                }, 2000);
            }
        }, Math.random() * 10000 + 5000);
    }

    return socket;
}

/**
 * Utility to wait for an event
 */
export function waitForEvent<T extends SocketEvent>(
    socket: MockSocketEmitter,
    event: T,
    timeout = 5000
): Promise<SocketEventPayloads[T]> {
    return new Promise((resolve, reject) => {
        const timer = setTimeout(() => {
            socket.off(event, handler);
            reject(new Error(`Timeout waiting for event: ${event}`));
        }, timeout);

        const handler = (data: SocketEventPayloads[T]) => {
            clearTimeout(timer);
            socket.off(event, handler);
            resolve(data);
        };

        socket.on(event, handler);
    });
}

/**
 * Utility to collect events over a time period
 */
export async function collectEvents<T extends SocketEvent>(
    socket: MockSocketEmitter,
    event: T,
    duration: number
): Promise<SocketEventPayloads[T][]> {
    const events: SocketEventPayloads[T][] = [];

    const handler = (data: SocketEventPayloads[T]) => {
        events.push(data);
    };

    socket.on(event, handler);

    await new Promise(resolve => setTimeout(resolve, duration));

    socket.off(event, handler);

    return events;
}

/**
 * Test helper to assert event was emitted
 */
export function assertEventEmitted(
    socket: MockSocketEmitter,
    event: string,
    expectedData?: any
): void {
    const emitted = socket.getEmitsByEvent(event);
    
    if (emitted.length === 0) {
        throw new Error(`Event "${event}" was not emitted`);
    }

    if (expectedData !== undefined) {
        const found = emitted.some(e => 
            JSON.stringify(e.data) === JSON.stringify(expectedData)
        );
        
        if (!found) {
            throw new Error(
                `Event "${event}" was emitted but with different data. ` +
                `Expected: ${JSON.stringify(expectedData)}, ` +
                `Actual: ${JSON.stringify(emitted.map(e => e.data))}`
            );
        }
    }
}

/**
 * Test helper to assert event was NOT emitted
 */
export function assertEventNotEmitted(
    socket: MockSocketEmitter,
    event: string
): void {
    const emitted = socket.getEmitsByEvent(event);
    
    if (emitted.length > 0) {
        throw new Error(
            `Event "${event}" was emitted ${emitted.length} time(s) ` +
            `but should not have been`
        );
    }
}