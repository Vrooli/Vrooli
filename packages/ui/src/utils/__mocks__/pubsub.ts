class PubSubMock {
    private static instance: PubSubMock | null = null;
    private subscribers = new Map<string, Map<symbol, (data: unknown) => void>>();

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    static get(): PubSubMock {
        if (!PubSubMock.instance) {
            PubSubMock.instance = new PubSubMock();
        }
        return PubSubMock.instance;
    }

    publish = jest.fn(<T extends string>(type: T, data: unknown = null) => {
        this.subscribers.get(type)?.forEach(subscriber => subscriber(data));
    });

    subscribe = jest.fn(<T extends string>(type: T, subscriber: (data: unknown) => void): () => void => {
        const token = Symbol(type);

        if (!this.subscribers.has(type)) {
            this.subscribers.set(type, new Map());
        }
        const subs = this.subscribers.get(type);
        if (subs) {
            subs.set(token, subscriber);
        }

        // Return an unsubscribe function
        return () => {
            const subscribersOfType = this.subscribers.get(type);
            if (subscribersOfType) {
                subscribersOfType.delete(token);
                if (subscribersOfType.size === 0) {
                    this.subscribers.delete(type);
                }
            }
        };
    });

    // Utility methods for testing
    static resetMock = () => {
        if (PubSubMock.instance) {
            PubSubMock.instance.subscribers.clear();
            PubSubMock.instance.publish.mockClear();
            PubSubMock.instance.subscribe.mockClear();
        }
        PubSubMock.instance = null;
    };
}

export const PubSub = PubSubMock;