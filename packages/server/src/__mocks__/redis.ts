export class RedisClientMock {
    static instance: RedisClientMock | null = null;
    static instantiationCount = 0; // To ensure that only one instance is created
    dataStore: { [key: string]: any } = {};
    eventHandlers: { [event: string]: ((...args: any[]) => void)[] } = {};
    url: string;
    isReady = false;

    constructor({ url }: { url: string }) {
        this.url = url;
        RedisClientMock.instantiationCount++;

        if (!RedisClientMock.instance) {
            RedisClientMock.instance = this;
        }

        return RedisClientMock.instance;
    }

    // Mocked Redis methods
    connect = jest.fn().mockImplementation(() => {
        this.isReady = true; // Set isReady to true when connected
        return Promise.resolve(this);
    });

    disconnect = jest.fn().mockImplementation(() => {
        this.isReady = false; // Set isReady to false when disconnected
        return Promise.resolve(undefined);
    });

    get = jest.fn((key: string) => {
        return Promise.resolve(this.dataStore[key] || null);
    });

    set = jest.fn((key: string, value: any) => {
        this.dataStore[key] = value;
        return Promise.resolve("OK");
    });

    incrBy = jest.fn((key: string, increment: number) => {
        if (!this.dataStore[key]) this.dataStore[key] = 0;
        this.dataStore[key] += increment;
        return Promise.resolve(this.dataStore[key]);
    });

    incr = jest.fn((key: string) => this.incrBy(key, 1));

    del = jest.fn((key: string) => {
        const existed = key in this.dataStore;
        delete this.dataStore[key];
        return Promise.resolve(existed ? 1 : 0);
    });

    // NOTE: Should use jest.useFakeTimers() to test this method
    expire = jest.fn((key: string, seconds: number) => {
        if (key in this.dataStore) {
            const milliseconds = seconds * 1000;

            // Schedule the deletion of the key
            setTimeout(() => {
                delete this.dataStore[key];
            }, milliseconds);

            return Promise.resolve(1); // Indicate success
        } else {
            return Promise.resolve(0); // Key does not exist
        }
    });

    hSet = jest.fn((key: string, field: string, value: any) => {
        if (!this.dataStore[key]) this.dataStore[key] = {};
        this.dataStore[key][field] = value;
        return Promise.resolve(1); // Assuming successful hSet operation
    });

    zAdd = jest.fn((key: string, score: number, member: string) => {
        if (!this.dataStore[key]) this.dataStore[key] = new Map();
        this.dataStore[key].set(member, score);
        return Promise.resolve(1); // Assuming one member was added
    });

    sAdd = jest.fn((key: string, member: any) => {
        if (!this.dataStore[key]) this.dataStore[key] = new Set();
        this.dataStore[key].add(member);
        return Promise.resolve(1); // Assuming one member was added
    });

    zRem = jest.fn((key: string, member: string) => {
        if (!this.dataStore[key]) return Promise.resolve(0);
        const removed = this.dataStore[key].delete(member);
        return Promise.resolve(removed ? 1 : 0);
    });

    zRange = jest.fn((key: string, start: number, stop: number) => {
        if (!this.dataStore[key]) return Promise.resolve([]);
        const members = Array.from(this.dataStore[key].keys());
        return Promise.resolve(members.slice(start, stop + 1)); // +1 because stop is inclusive
    });

    hGetAll = jest.fn((key: string) => {
        if (!this.dataStore[key]) return Promise.resolve({});
        return Promise.resolve(this.dataStore[key]);
    });

    // Utility methods for testing
    __setMockData = (key: string, value: any) => {
        this.dataStore[key] = value;
    };

    __getMockData = (key: string) => {
        return this.dataStore[key];
    };

    __resetMockData = () => {
        this.dataStore = {};
    };

    static resetMock = () => {
        if (RedisClientMock.instance) {
            RedisClientMock.instance.dataStore = {};
            RedisClientMock.instance.eventHandlers = {};
            RedisClientMock.instance.isReady = false;
            RedisClientMock.instance.connect.mockClear();
            RedisClientMock.instance.disconnect.mockClear();
            RedisClientMock.instance.get.mockClear();
            RedisClientMock.instance.set.mockClear();
            RedisClientMock.instance.incr.mockClear();
            RedisClientMock.instance.incrBy.mockClear();
            RedisClientMock.instance.del.mockClear();
            RedisClientMock.instance.expire.mockClear();
            RedisClientMock.instance.hSet.mockClear();
            RedisClientMock.instance.zAdd.mockClear();
            RedisClientMock.instance.sAdd.mockClear();
            RedisClientMock.instance.zRem.mockClear();
            RedisClientMock.instance.zRange.mockClear();
            RedisClientMock.instance.hGetAll.mockClear();
        }
        RedisClientMock.instance = null;
        RedisClientMock.instantiationCount = 0;
    };

    on = jest.fn((event: string, callback: (...args: any[]) => void) => {
        if (!this.eventHandlers[event]) {
            this.eventHandlers[event] = [];
        }
        this.eventHandlers[event].push(callback);
    });

    // Utility function to simulate emitting events
    __emit = (event: string, ...args: any[]) => {
        if (this.eventHandlers[event]) {
            this.eventHandlers[event].forEach(callback => callback(...args));
        }
        if (event === "end") {
            this.isReady = false;
        }
    };
}

export const createClient = ({ url }: { url: string }) => {
    return new RedisClientMock({ url });
};
