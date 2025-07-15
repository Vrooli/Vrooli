// AI_CHECK: TYPE_SAFETY=fixed-pubsub-mock-any-types | LAST: 2025-06-28
import { type PubSubInstance } from "../pubsub.js";

// Store subscribers for each topic
const subscribers = new Map<string, Set<(data: unknown) => void>>();

// Mock implementation of PubSub for testing
const mockInstance = {
    subscribe: (topic: string, callback: (data: unknown) => void) => {
        if (!subscribers.has(topic)) {
            subscribers.set(topic, new Set());
        }
        subscribers.get(topic)!.add(callback);
        
        // Return unsubscribe function
        return () => {
            const topicSubs = subscribers.get(topic);
            if (topicSubs) {
                topicSubs.delete(callback);
            }
        };
    },
    unsubscribe: () => {},
    publish: (topic: string, data: unknown) => {
        const topicSubs = subscribers.get(topic);
        if (topicSubs) {
            topicSubs.forEach(callback => callback(data));
        }
    },
};

export const PubSub: PubSubInstance & { resetMock: () => void } = {
    subscribe: mockInstance.subscribe,
    unsubscribe: mockInstance.unsubscribe,
    get: () => mockInstance,
    publish: mockInstance.publish,
    resetMock: () => {
        // Clear all subscribers
        subscribers.clear();
    },
};
