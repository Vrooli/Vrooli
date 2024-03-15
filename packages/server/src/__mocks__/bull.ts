// NOTE: Untested
export class BullQueueMock {
    static instance: BullQueueMock | null = null;
    static instantiationCount = 0;
    static jobs: any[] = [];
    name: string;
    eventHandlers: { [event: string]: ((...args: any[]) => void)[] } = {};

    constructor(name: string) {
        this.name = name;
        BullQueueMock.instantiationCount++;

        if (!BullQueueMock.instance) {
            BullQueueMock.instance = this;
        }

        return BullQueueMock.instance;
    }

    // Mocked Bull methods
    add = jest.fn((data: any, options?: any) => {
        const job = { id: BullQueueMock.jobs.length + 1, data, options };
        BullQueueMock.jobs.push(job);
        return Promise.resolve(job);
    });

    process = jest.fn((concurrency: number | Function, processor?: Function) => {
        let callback = processor;
        if (typeof concurrency === 'function') {
            callback = concurrency;
        }

        // Ensure callback is a function before invoking it
        if (typeof callback === 'function') {
            BullQueueMock.jobs.forEach(job => {
                // Call the callback with the job and done function
                callback!(job, () => { }); // The second argument simulates the "done" callback function
            });
        } else {
            // Handle the case where callback is undefined, e.g., log an error or throw an exception
            console.error('Process function is undefined.');
        }
    });

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
    };

    // Utility methods for testing
    static __addMockJob = (job: any) => {
        BullQueueMock.jobs.push(job);
    };

    static __getMockJobs = () => {
        // Return a deep copy to prevent accidental modifications
        return JSON.parse(JSON.stringify(BullQueueMock.jobs));
    };

    static __resetMockJobs = () => {
        BullQueueMock.jobs = [];
    };

    static resetMock = () => {
        BullQueueMock.jobs = [];
        if (BullQueueMock.instance) {
            BullQueueMock.instance.eventHandlers = {};
            BullQueueMock.instance.on.mockClear();
            BullQueueMock.instance.add.mockClear();
            BullQueueMock.instance.process.mockClear();
        }
        BullQueueMock.instance = null;
        BullQueueMock.instantiationCount = 0;
    };
}

export const Queue = (name: string) => {
    return new BullQueueMock(name);
};
