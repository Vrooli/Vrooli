const actualWorker = jest.requireActual("worker_threads");

type Worker = {
    on: jest.Mock;
    postMessage: jest.Mock;
    terminate: jest.Mock;
}

function createWorkerMockInstance(filename, options) {
    const realWorker = new actualWorker.Worker(filename, options);

    // Wrap real methods with Jest functions for spying and custom logic
    const mockOn = jest.fn((...args) => realWorker.on(...args));
    const mockPostMessage = jest.fn((...args) => realWorker.postMessage(...args));
    const mockTerminate = jest.fn(() => realWorker.terminate());
    const mockRemoveListener = jest.fn((...args) => realWorker.removeListener(...args));

    const mockInstance = {
        ...realWorker,
        on: mockOn,
        postMessage: mockPostMessage,
        terminate: mockTerminate,
        removeListener: mockRemoveListener,
    };

    return mockInstance;
}

class WorkerMock {
    static instances: Worker[] = [];
    static instantiationCount = 0;

    constructor(filename, options) {
        const instance = createWorkerMockInstance(filename, options);
        WorkerMock.instances.push(instance);
        WorkerMock.instantiationCount++;
        return instance;
    }

    static resetMock = () => {
        WorkerMock.instances.forEach(instance => {
            instance.on.mockClear();
            instance.postMessage.mockClear();
            instance.terminate.mockClear();
        });
        WorkerMock.instances = [];
        WorkerMock.instantiationCount = 0;
    };

    static getInstances = () => WorkerMock.instances;
}

// Export the WorkerMock as Worker
export const Worker = WorkerMock;

// Default export remains the same
const worker_threads = {
    Worker: WorkerMock,
};

export default worker_threads;
