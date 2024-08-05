// import { Worker as RealWorker } from "worker_threads";
import { Worker as RealWorker } from "../../../../node_modules/jest-worker/build/index";

function createWorkerMockInstance(filename, options) {
    const realWorker = new RealWorker(filename, options);

    // Wrap real methods with Jest functions for spying and custom logic
    const mockOn = jest.fn(realWorker.on.bind(realWorker));
    const mockPostMessage = jest.fn(realWorker.postMessage.bind(realWorker));
    const mockTerminate = jest.fn(() => realWorker.terminate());

    // Create a mock instance object
    const mockInstance = {
        realWorker,
        filename,
        workerData: options?.workerData,
        on: mockOn,
        postMessage: mockPostMessage,
        terminate: mockTerminate,
    };

    console.log("Mock worker created:", filename, options?.workerData);

    return mockInstance;
}

class WorkerMock {
    static instances = [];
    static instantiationCount = 0;

    constructor(filename, options) {
        const instance = createWorkerMockInstance(filename, options);
        WorkerMock.instances.push(instance);
        WorkerMock.instantiationCount++;
        return instance.realWorker; // Return the actual Worker object wrapped with mocks
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
