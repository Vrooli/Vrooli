import { vi } from "vitest";

// Create mock instances that can be reused
export const mockEmailAddTask = vi.fn().mockResolvedValue(undefined);
export const mockBusPublish = vi.fn().mockResolvedValue(undefined);
export const mockSocketEmit = vi.fn();
export const mockNotifyPush = vi.fn().mockReturnValue({ toUser: vi.fn().mockResolvedValue(undefined) });
export const mockNotifyPushScheduleReminder = vi.fn().mockReturnValue({ toUsers: vi.fn().mockResolvedValue(undefined) });

// Export the mock structures
export const mockQueueService = {
    get: () => ({
        email: {
            addTask: mockEmailAddTask,
        },
    }),
    reset: vi.fn(),
};

export const mockBusService = {
    get: () => ({
        getBus: () => ({
            publish: mockBusPublish,
        }),
    }),
    reset: vi.fn(),
};

export const mockSocketService = {
    get: () => ({
        emitSocketEvent: mockSocketEmit,
    }),
};

export const mockNotify = vi.fn(() => ({
    pushFreeCreditsReceived: mockNotifyPush,
    pushScheduleReminder: mockNotifyPushScheduleReminder,
}));

// Reset all mocks
export const resetAllMocks = () => {
    mockEmailAddTask.mockClear();
    mockBusPublish.mockClear();
    mockSocketEmit.mockClear();
    mockNotifyPush.mockClear();
    mockNotifyPushScheduleReminder.mockClear();
    mockNotify.mockClear();
};
