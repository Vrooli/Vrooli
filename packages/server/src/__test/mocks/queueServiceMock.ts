import { vi } from "vitest";

/**
 * Mock implementation of QueueService for testing
 * This mock ensures that email and other queue tasks return the expected success response
 * instead of undefined, preventing "Cannot read properties of undefined" errors in tests.
 */

// Create the mock QueueService
export const mockQueueService = {
    get: vi.fn().mockReturnValue({
        email: {
            addTask: vi.fn().mockResolvedValue({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: "mock-job-id" },
            }),
        },
        export: {
            addTask: vi.fn().mockResolvedValue({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: "mock-job-id" },
            }),
        },
        import: {
            addTask: vi.fn().mockResolvedValue({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: "mock-job-id" },
            }),
        },
        swarm: {
            addTask: vi.fn().mockResolvedValue({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: "mock-job-id" },
            }),
        },
        push: {
            addTask: vi.fn().mockResolvedValue({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: "mock-job-id" },
            }),
        },
        run: {
            addTask: vi.fn().mockResolvedValue({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: "mock-job-id" },
            }),
        },
        sandbox: {
            addTask: vi.fn().mockResolvedValue({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: "mock-job-id" },
            }),
        },
        sms: {
            addTask: vi.fn().mockResolvedValue({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: "mock-job-id" },
            }),
        },
        notification: {
            addTask: vi.fn().mockResolvedValue({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: "mock-job-id" },
            }),
        },
        addTask: vi.fn().mockResolvedValue({ 
            __typename: "Success" as const, 
            success: true,
            data: { id: "mock-job-id" },
        }),
        getTaskStatuses: vi.fn().mockResolvedValue([]),
        changeTaskStatus: vi.fn().mockResolvedValue({ 
            __typename: "Success" as const, 
            success: true, 
        }),
        initializeAllQueues: vi.fn().mockReturnValue({}),
        shutdown: vi.fn().mockResolvedValue(undefined),
    }),
    reset: vi.fn().mockResolvedValue(undefined),
};

/**
 * Setup function to be called at the beginning of test files that need QueueService mocking
 * Usage: 
 * import { setupQueueServiceMock } from "../../__test/mocks/queueServiceMock.js";
 * setupQueueServiceMock();
 */
export function setupQueueServiceMock() {
    vi.mock("../../tasks/queues.js", () => ({
        QueueService: mockQueueService,
    }));
}

/**
 * Reset all mock calls - useful in beforeEach blocks
 */
export function resetQueueServiceMock() {
    vi.clearAllMocks();
}
