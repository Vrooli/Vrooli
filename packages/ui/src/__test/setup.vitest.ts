import { vi } from 'vitest';

// Mock @vrooli/shared using importOriginal to keep most functionality
vi.mock('@vrooli/shared', async (importOriginal) => {
    try {
        const actual = await importOriginal();
        
        // Override only problematic functions that cause circular dependencies
        return {
            ...actual,
            // Mock functions that might cause issues during test setup
            parseSearchParams: () => {
                // Mock based on current URL search for tests
                const urlParams = new URLSearchParams(global.window?.location?.search || '');
                const result = {};
                urlParams.forEach((value, key) => {
                    try {
                        result[key] = JSON.parse(value);
                    } catch {
                        result[key] = value;
                    }
                });
                return result;
            },
            stringifySearchParams: (params) => {
                if (!params || Object.keys(params).length === 0) return '';
                const urlParams = new URLSearchParams();
                Object.entries(params).forEach(([key, value]) => {
                    urlParams.append(key, `"${value}"`);
                });
                return '?' + urlParams.toString();
            },
        };
    } catch (error) {
        // Fallback if the import fails (e.g., during transpilation issues)
        return {
            // Essential constants
            DUMMY_ID: 'dummy-id',
            SERVER_VERSION: 'v2',
            
            // Essential enums
            InputType: { Text: 'Text', JSON: 'JSON' },
            TaskStatus: { Running: 'Running', Completed: 'Completed' },
            
            // Essential functions
            nanoid: () => Math.random().toString(36).substring(2, 9),
            noop: () => {},
            parseSearchParams: () => ({}),
            stringifySearchParams: () => '',
            generatePK: () => Math.random().toString(36).substring(2, 15),
        };
    }
});

// Essential mocks only - no fancy stuff
vi.mock('../utils/display/device.js', () => ({
    keyComboToString: () => 'Ctrl+S',
    getDeviceInfo: () => ({ isMobile: false }),
}));
vi.mock('../utils/display/chatTools.js', () => ({
    taskToTaskInfo: () => ({ name: 'Test Task', description: 'Test Description' }),
    STALE_TASK_THRESHOLD_MS: 600000,
}));
vi.mock('@uiw/react-codemirror', () => ({ default: () => null }));
vi.mock('../utils/pubsub.js', () => ({
    PubSub: {
        get: () => ({
            subscribe: vi.fn(() => vi.fn()),
            publish: vi.fn(),
            hasSubscribers: () => false,
        }),
    },
}));

// Minimal browser mocks
Object.defineProperty(window, 'matchMedia', {
    value: () => ({ matches: false, addListener: () => {}, removeListener: () => {} }),
});

global.IntersectionObserver = vi.fn(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
}));

global.ResizeObserver = vi.fn(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(), 
    disconnect: vi.fn(),
}));