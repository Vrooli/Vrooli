import { vi } from "vitest";
import type { IEventBus, BaseEvent, EventSubscription, EventQuery } from "@vrooli/shared";

export class MockEventBus implements IEventBus {
    private subscribers = new Map<string, Array<(event: BaseEvent) => Promise<void>>>();

    async publish<T extends BaseEvent>(eventType: string, event: T): Promise<void> {
        const handlers = this.subscribers.get(eventType) || [];
        await Promise.all(handlers.map(handler => handler(event)));
    }

    subscribe<T extends BaseEvent>(
        eventType: string, 
        handler: (event: T) => Promise<void>
    ): EventSubscription {
        if (!this.subscribers.has(eventType)) {
            this.subscribers.set(eventType, []);
        }
        this.subscribers.get(eventType)!.push(handler as (event: BaseEvent) => Promise<void>);
        
        return {
            unsubscribe: () => {
                const handlers = this.subscribers.get(eventType);
                if (handlers) {
                    const index = handlers.indexOf(handler as (event: BaseEvent) => Promise<void>);
                    if (index > -1) {
                        handlers.splice(index, 1);
                    }
                }
            }
        };
    }

    async query<T extends BaseEvent>(query: EventQuery): Promise<T[]> {
        // Mock implementation - return empty array
        return [];
    }

    async shutdown(): Promise<void> {
        this.subscribers.clear();
    }
}