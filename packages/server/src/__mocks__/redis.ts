export class RedisClientMock {
    static instance: RedisClientMock | null = null;
    static instantiationCount = 0; // To ensure that only one instance is created
    static dataStore: { [key: string]: any } = {};
    eventHandlers: { [event: string]: ((...args: any[]) => void)[] } = {};
    url: string;
    isReady = false;
    shouldFail = false;

    constructor({ url }: { url: string }) {
        this.url = url;
        RedisClientMock.instantiationCount++;

        if (!RedisClientMock.instance) {
            RedisClientMock.instance = this;
        }

        return RedisClientMock.instance;
    }

    // A helper method to handle success or failure based on the shouldFail flag
    _handlePromise = (operation: () => any) => {
        return this.shouldFail ? Promise.reject(new Error("Redis operation failed")) : Promise.resolve(operation());
    };

    // Mocked Redis methods
    connect = jest.fn().mockImplementation(() => this._handlePromise(() => {
        this.isReady = true;
        return this;
    }));

    disconnect = jest.fn().mockImplementation(() => this._handlePromise(() => {
        this.isReady = false;
        return undefined;
    }));

    get = jest.fn((key: string) => this._handlePromise(() => RedisClientMock.dataStore[key] || null));

    set = jest.fn((key: string, value: any) => this._handlePromise(() => {
        RedisClientMock.dataStore[key] = value;
        return "OK";
    }));

    incrBy = jest.fn((key: string, increment: number) => {
        return this._handlePromise(() => {
            if (!RedisClientMock.dataStore[key]) RedisClientMock.dataStore[key] = 0;
            RedisClientMock.dataStore[key] += increment;
            return RedisClientMock.dataStore[key];
        });
    });

    incr = jest.fn((key: string) => {
        return this.incrBy(key, 1);
    });

    del = jest.fn((keys: string | string[]) => {
        return this._handlePromise(() => {
            const keysToDelete = Array.isArray(keys) ? keys : [keys];
            let count = 0;
            keysToDelete.forEach(key => {
                if (key in RedisClientMock.dataStore) {
                    delete RedisClientMock.dataStore[key];
                    count++;
                }
            });
            return count;
        });
    });

    // NOTE: Should use jest.useFakeTimers() to test this method
    expire = jest.fn((key: string, seconds: number) => {
        return this._handlePromise(() => {
            if (key in RedisClientMock.dataStore) {
                const milliseconds = seconds * 1000;
                setTimeout(() => {
                    delete RedisClientMock.dataStore[key];
                }, milliseconds);
                return 1; // Indicate success
            } else {
                return 0; // Key does not exist
            }
        });
    });

    hSet = jest.fn((key: string, fieldOrObject: any, value?: any) => {
        return this._handlePromise(() => {
            if (!RedisClientMock.dataStore[key]) RedisClientMock.dataStore[key] = {};

            if (typeof fieldOrObject === "object" && value === undefined) {
                Object.entries(fieldOrObject).forEach(([field, val]) => {
                    RedisClientMock.dataStore[key][field] = val;
                });
            } else if (typeof fieldOrObject === "string") {
                RedisClientMock.dataStore[key][fieldOrObject] = value;
            }
            return 1;
        });
    });

    // NOTE: The real zAdd would use a map or a set to store the members and their scores,
    // but this causes issues with serialization and deserialization in the tests that I don't 
    // feel like dealing with.
    zAdd = jest.fn((key: string, { score, value }: { score: number, value: string }) => {
        return this._handlePromise(() => {
            if (!RedisClientMock.dataStore[key]) {
                RedisClientMock.dataStore[key] = [];
            }
            const list = RedisClientMock.dataStore[key];
            if (!Array.isArray(list)) {
                throw new Error(`Expected an array for key ${key}, but found a different type`);
            }
            const index = list.findIndex((item) => item.value === value);
            if (index > -1) {
                list[index].score = score;
            } else {
                list.push({ score, value });
            }
            return 1;
        });
    });

    sAdd = jest.fn((key: string, members: any | any[]) => {
        return this._handlePromise(() => {
            if (!RedisClientMock.dataStore[key]) {
                RedisClientMock.dataStore[key] = [];
            } else if (!Array.isArray(RedisClientMock.dataStore[key])) {
                throw new Error(`Expected dataStore entry for key "${key}" to be an array`);
            }
            const membersArray = Array.isArray(members) ? members : [members];
            membersArray.forEach(member => {
                if (!RedisClientMock.dataStore[key].includes(member)) {
                    RedisClientMock.dataStore[key].push(member);
                }
            });
            return membersArray.length; // Return the number of members added
        });
    });

    sRem = jest.fn((key: string, member: any) => {
        return this._handlePromise(() => {
            if (RedisClientMock.dataStore[key] && Array.isArray(RedisClientMock.dataStore[key])) {
                const index = RedisClientMock.dataStore[key].indexOf(member);
                if (index > -1) {
                    RedisClientMock.dataStore[key].splice(index, 1);
                    return 1; // Member removed
                }
            }
            return 0; // Member not found
        });
    });

    sMembers = jest.fn((key: string) => {
        return this._handlePromise(() => {
            if (!RedisClientMock.dataStore[key]) {
                return [];
            } else if (!Array.isArray(RedisClientMock.dataStore[key])) {
                throw new Error(`Expected dataStore entry for key "${key}" to be an array`);
            }
            return [...RedisClientMock.dataStore[key]]; // Return a copy of the array
        });
    });

    zRem = jest.fn((key: string, member: string) => {
        return this._handlePromise(() => {
            if (!RedisClientMock.dataStore[key] || !Array.isArray(RedisClientMock.dataStore[key])) {
                return 0;
            }
            const index = RedisClientMock.dataStore[key].findIndex(item => item.value === member);
            if (index > -1) {
                RedisClientMock.dataStore[key].splice(index, 1);
                return 1; // Element was removed
            }
            return 0; // Element not found, nothing removed
        });
    });

    zRange = jest.fn((key: string, start: number, stop: number) => {
        return this._handlePromise(() => {
            if (!RedisClientMock.dataStore[key] || !Array.isArray(RedisClientMock.dataStore[key])) {
                return [];
            }
            // Sort the array by score in ascending order
            const sortedByScore = RedisClientMock.dataStore[key].slice().sort((a, b) => a.score - b.score);
            // If 'stop' is -1, set it to the last index of the array
            const endIndex = stop === -1 ? sortedByScore.length : stop + 1; // Adjust for 'stop' being inclusive
            // Slice the array to get the specified range, ensuring 'start' and 'endIndex' are within bounds
            const range = sortedByScore.slice(start, endIndex);
            // Return the values (members) of the elements in the range
            return range.map(item => item.value);
        });
    });

    hGetAll = jest.fn((key: string) => {
        return this._handlePromise(() => {
            if (!RedisClientMock.dataStore[key]) return {};
            return RedisClientMock.dataStore[key];
        });
    });

    // Utility methods for testing
    static simulateFailure(shouldFail: boolean) {
        if (RedisClientMock.instance) {
            RedisClientMock.instance.shouldFail = shouldFail;
        }
    }

    static __setMockData = (key: string, value: any) => {
        RedisClientMock.dataStore[key] = value;
    };

    static __setAllMockData = (data: { [key: string]: any }) => {
        RedisClientMock.dataStore = data;
    };

    static __getMockData = (key: string) => {
        return RedisClientMock.dataStore[key];
    };

    static __getAllMockData = () => {
        // Return a deep copy to prevent accidental modifications
        return JSON.parse(JSON.stringify(this.dataStore));
    };

    static __deleteMockData = (key: string) => {
        delete RedisClientMock.dataStore[key];
    };

    static __resetMockData = () => {
        RedisClientMock.dataStore = {};
    };

    static resetMock = () => {
        RedisClientMock.dataStore = {};
        if (RedisClientMock.instance) {
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
        RedisClientMock.simulateFailure(false);
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
