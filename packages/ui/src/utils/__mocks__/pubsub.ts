const mockInstance = {
    publish: jest.fn(),
    subscribe: jest.fn(),
    unsubscribe: jest.fn(),
};

export class PubSub {
    // Override the get method to return the mock instance
    static get() {
        return mockInstance;
    }
}
