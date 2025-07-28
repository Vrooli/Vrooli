import { vi } from "vitest";
import { generatePK } from "@vrooli/shared";

/**
 * Mock implementation of QueueService for testing
 * This mock ensures that email and other queue tasks return the expected success response
 * instead of undefined, preventing "Cannot read properties of undefined" errors in tests.
 */

// Create the mock QueueService
export const mockQueueService = {
    get: vi.fn().mockReturnValue({
        email: {
            addTask: vi.fn().mockImplementation(async () => ({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: generatePK().toString() },
            })),
        },
        export: {
            addTask: vi.fn().mockImplementation(async () => ({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: generatePK().toString() },
            })),
        },
        import: {
            addTask: vi.fn().mockImplementation(async () => ({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: generatePK().toString() },
            })),
        },
        swarm: {
            addTask: vi.fn().mockImplementation(async () => ({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: generatePK().toString() },
            })),
        },
        push: {
            addTask: vi.fn().mockImplementation(async () => ({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: generatePK().toString() },
            })),
        },
        run: {
            addTask: vi.fn().mockImplementation(async () => ({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: generatePK().toString() },
            })),
        },
        sandbox: {
            addTask: vi.fn().mockImplementation(async () => ({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: generatePK().toString() },
            })),
        },
        sms: {
            addTask: vi.fn().mockImplementation(async () => ({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: generatePK().toString() },
            })),
        },
        notification: {
            addTask: vi.fn().mockImplementation(async () => ({ 
                __typename: "Success" as const, 
                success: true,
                data: { id: generatePK().toString() },
            })),
        },
        addTask: vi.fn().mockImplementation(async () => ({ 
            __typename: "Success" as const, 
            success: true,
            data: { id: generatePK().toString() },
        })),
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
