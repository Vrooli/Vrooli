import { vi } from "vitest";

// Create mock factories that return fresh instances
export function createMockEmailAddTask(): ReturnType<typeof vi.fn> {
    return vi.fn().mockResolvedValue(undefined);
}
export function createMockBusPublish(): ReturnType<typeof vi.fn> {
    return vi.fn().mockResolvedValue(undefined);
}
export function createMockSocketEmit(): ReturnType<typeof vi.fn> {
    return vi.fn();
}
export function createMockNotifyPush(): ReturnType<typeof vi.fn> {
    return vi.fn().mockReturnValue({ toUser: vi.fn().mockResolvedValue(undefined) });
}
export function createMockNotifyPushScheduleReminder(): ReturnType<typeof vi.fn> {
    return vi.fn().mockReturnValue({ toUsers: vi.fn().mockResolvedValue(undefined) });
}

// Create a factory for service mocks to ensure isolation
export function createServiceMocks(): {
    mockEmailAddTask: ReturnType<typeof vi.fn>;
    mockBusPublish: ReturnType<typeof vi.fn>;
    mockSocketEmit: ReturnType<typeof vi.fn>;
    mockNotifyPush: ReturnType<typeof vi.fn>;
    mockNotifyPushScheduleReminder: ReturnType<typeof vi.fn>;
    mockQueueService: object;
    mockBusService: object;
    mockSocketService: object;
    mockNotify: ReturnType<typeof vi.fn>;
} {
    const mockEmailAddTask = createMockEmailAddTask();
    const mockBusPublish = createMockBusPublish();
    const mockSocketEmit = createMockSocketEmit();
    const mockNotifyPush = createMockNotifyPush();
    const mockNotifyPushScheduleReminder = createMockNotifyPushScheduleReminder();

    const mockQueueService = {
        get: () => ({
            email: {
                addTask: mockEmailAddTask,
            },
        }),
        reset: vi.fn(),
    };

    const mockBusService = {
        get: () => ({
            getBus: () => ({
                publish: mockBusPublish,
            }),
        }),
        reset: vi.fn(),
    };

    const mockSocketService = {
        get: () => ({
            emitSocketEvent: mockSocketEmit,
        }),
    };

    const mockNotify = vi.fn(() => ({
        pushFreeCreditsReceived: mockNotifyPush,
        pushScheduleReminder: mockNotifyPushScheduleReminder,
    }));

    return {
        mockEmailAddTask,
        mockBusPublish,
        mockSocketEmit,
        mockNotifyPush,
        mockNotifyPushScheduleReminder,
        mockQueueService,
        mockBusService,
        mockSocketService,
        mockNotify,
    };
}

// For backward compatibility with existing tests
const serviceMocks = createServiceMocks();
export const mockEmailAddTask = serviceMocks.mockEmailAddTask;
export const mockBusPublish = serviceMocks.mockBusPublish;
export const mockSocketEmit = serviceMocks.mockSocketEmit;
export const mockNotifyPush = serviceMocks.mockNotifyPush;
export const mockNotifyPushScheduleReminder = serviceMocks.mockNotifyPushScheduleReminder;
export const mockQueueService = serviceMocks.mockQueueService;
export const mockBusService = serviceMocks.mockBusService;
export const mockSocketService = serviceMocks.mockSocketService;
export const mockNotify = serviceMocks.mockNotify;

// Reset all mocks function for backward compatibility
export function resetAllMocks(): void {
    mockEmailAddTask.mockClear();
    mockBusPublish.mockClear();
    mockSocketEmit.mockClear();
    mockNotifyPush.mockClear();
    mockNotifyPushScheduleReminder.mockClear();
    mockNotify.mockClear();
}
