import { vi, expect, beforeAll, afterEach, afterAll } from 'vitest';

// AI_CHECK: TEST_QUALITY=2 | LAST: 2025-06-18
// Improved test quality in UI package by:
// - Refactoring ErrorBoundary tests to focus on user behavior rather than implementation
// - Improving useFetch tests with better descriptions and behavior-driven testing
// - Enhancing useSocketChat tests with clearer real-time scenarios
// - Added console warning suppression for cleaner test output
// Note: Some tests like activeChatStore and TextInput still need quality improvements

// Store original console methods
const originalConsoleError = console.error;
const originalConsoleWarn = console.warn;
const originalConsoleLog = console.log;

// Patterns to suppress in test output
const suppressedPatterns = [
    // MUI Tooltip warnings
    /MUI: The `children` component of the Tooltip is not forwarding its props correctly/,
    /MUI: You are providing a disabled `button` child to the Tooltip component/,
    /A disabled element does not fire events/,
    /Tooltip needs to listen to the child element's events/,
    /Add a simple wrapper element, such as a `span`/,
    // React ErrorBoundary expected errors
    /The above error occurred in the <.*> component:/,
    /React will try to recreate this component tree/,
    // i18next warnings
    /react-i18next:: You will need to pass in an i18next instance/,
    // getObjectUrlBase warnings
    /getObjectUrlBase called with non-object/,
    // RichInputBase warnings
    /RichInputBase: No child handleAction function found/,
    // Other common test noise
    /Warning: ReactDOM.render is no longer supported/,
    /Warning: An invalid form control/,
    /Warning: Failed prop type/,
    /Warning: Unknown event handler property/,
];

// Setup console filtering
beforeAll(() => {
    // Filter console.error
    console.error = (...args) => {
        const message = args.join(' ');
        const shouldSuppress = suppressedPatterns.some(pattern => pattern.test(message));
        
        if (!shouldSuppress) {
            originalConsoleError(...args);
        }
    };

    // Filter console.warn
    console.warn = (...args) => {
        const message = args.join(' ');
        const shouldSuppress = suppressedPatterns.some(pattern => pattern.test(message));
        
        if (!shouldSuppress) {
            originalConsoleWarn(...args);
        }
    };

    // Filter console.log
    console.log = (...args) => {
        const message = args.join(' ');
        
        // Patterns to suppress in logs
        const logSuppressPatterns = [
            /AdvancedInput: Click detected/,
            /AdvancedInput: Click on interactive element/,
            /AdvancedInput: Focusing input/,
        ];
        
        const shouldSuppress = logSuppressPatterns.some(pattern => pattern.test(message));
        
        if (!shouldSuppress) {
            originalConsoleLog(...args);
        }
    };
});

// Restore console after each test to ensure clean state
afterEach(() => {
    // Clear any mocked console methods that individual tests might have created
    if (console.error !== originalConsoleError && !console.error.toString().includes('shouldSuppress')) {
        console.error = originalConsoleError;
    }
    if (console.warn !== originalConsoleWarn && !console.warn.toString().includes('shouldSuppress')) {
        console.warn = originalConsoleWarn;
    }
    if (console.log !== originalConsoleLog) {
        console.log = originalConsoleLog;
    }
});

// Vitest has built-in DOM assertions - no custom matchers needed

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
            DAYS_1_MS: 86400000, // 24 hours in milliseconds
            
            // Essential enums
            InputType: { Text: 'Text', JSON: 'JSON' },
            TaskStatus: { Running: 'Running', Completed: 'Completed' },
            FormStructureType: { Tip: 'Tip' },
            ResourceSubType: {
                RoutineInternalAction: 'RoutineInternalAction',
                RoutineApi: 'RoutineApi',
                RoutineCode: 'RoutineCode',
                RoutineData: 'RoutineData',
                RoutineGenerate: 'RoutineGenerate',
                RoutineInformational: 'RoutineInformational',
                RoutineMultiStep: 'RoutineMultiStep',
                RoutineSmartContract: 'RoutineSmartContract',
                RoutineWeb: 'RoutineWeb',
                CodeDataConverter: 'CodeDataConverter',
                CodeSmartContract: 'CodeSmartContract',
                StandardDataStructure: 'StandardDataStructure',
                StandardPrompt: 'StandardPrompt',
            },
            
            // Essential functions
            nanoid: () => Math.random().toString(36).substring(2, 9),
            noop: () => {},
            parseSearchParams: () => ({}),
            stringifySearchParams: () => '',
            generatePK: () => Math.random().toString(36).substring(2, 15),
            
            // Regex exports needed by tests
            urlRegex: /^(?:(?:https?|ftp):\/\/)(?:\S+(?::\S*)?@)?(?:(?!(?:10|127)(?:\.\d{1,3}){3})(?!(?:169\.254|192\.168)(?:\.\d{1,3}){2})(?!172\.(?:1[6-9]|2\d|3[0-1])(?:\.\d{1,3}){2})(?:[1-9]\d?|1\d\d|2[01]\d|22[0-3])(?:\.(?:1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.(?:[1-9]\d?|1\d\d|2[0-4]\d|25[0-4]))|(?:(?:[a-z\u00a1-\uffff0-9]-*)*[a-z\u00a1-\uffff0-9]+)(?:\.(?:[a-z\u00a1-\uffff0-9]-*)*[a-z\u00a1-\uffff0-9]+)*(?:\.(?:[a-z\u00a1-\uffff]{2,}))\.?)(?::\d{2,5})?(?:[/?#]\S*)?$/i,
            urlRegexDev: /^(?:(?:https?|ftp):\/\/)(?:\S+(?::\S*)?@)?(?:(?:localhost|127\.0\.0\.1)(?::\d{2,5})?|(?:[a-z\u00a1-\uffff0-9]+(?:-[a-z\u00a1-\uffff0-9]+)*)(?:\.(?:[a-z\u00a1-\uffff0-9]+(?:-[a-z\u00a1-\uffff0-9]+)*))*\.(?:[a-z\u00a1-\uffff]{2,})\.?)(?::\d{2,5})?(?:[/?#]\S*)?$/i,
            walletAddressRegex: /^addr1[a-zA-Z0-9]{98}$/,
            handleRegex: /^(?=.*[a-zA-Z0-9])[a-zA-Z0-9_]{3,16}$/,
            hexColorRegex: /^#([0-9A-F]{3}([0-9A-F]{3})?)$/i,
        };
    }
});

// Essential mocks only - no fancy stuff
vi.mock('../utils/display/device.js', () => ({
    keyComboToString: () => 'Ctrl+S',
    getDeviceInfo: () => ({ isMobile: false, deviceOS: 'Windows' }),
    DeviceOS: {
        IOS: 'iOS',
        MacOS: 'MacOS',
        Windows: 'Windows',
        Android: 'Android',
        Linux: 'Linux',
    },
}));
vi.mock('../utils/display/chatTools.js', () => ({
    taskToTaskInfo: () => ({ name: 'Test Task', description: 'Test Description' }),
    isTaskStale: vi.fn(() => false),
    STALE_TASK_THRESHOLD_MS: 600000,
}));
vi.mock('@uiw/react-codemirror', () => ({ default: () => null }));

// Mock only the most problematic lexical components with realistic implementations
vi.mock('../components/inputs/AdvancedInput/lexical/AdvancedInputLexical.js', () => ({
    AdvancedInputLexical: () => {
        const React = require('react');
        return React.createElement('div', { 'data-testid': 'lexical-editor' });
    },
    default: () => {
        const React = require('react');
        return React.createElement('div', { 'data-testid': 'lexical-editor' });
    },
}));
vi.mock('../components/inputs/RichInput/RichInput.js', () => ({
    RichInput: () => null,
    default: () => null,
}));
vi.mock('../components/inputs/TagSelector/TagSelector.js', () => ({
    TagSelector: () => null,
    default: () => null,
}));
// NOTE: Removed AdvancedInput mock to allow proper testing
vi.mock('../utils/pubsub.js', () => ({
    PubSub: {
        get: () => ({
            subscribe: vi.fn(() => vi.fn()),
            publish: vi.fn(),
            hasSubscribers: () => false,
        }),
    },
}));

// Mock API environment variables to prevent actual connections
process.env.VITE_PORT_API = '0'; // Use port 0 to prevent actual connections
process.env.VITE_API_URL = 'http://localhost:0/api'; // Mock API URL
process.env.VITE_SITE_IP = 'localhost';

// Mock fetch to prevent any actual HTTP requests
global.fetch = vi.fn(() => Promise.reject(new Error('Network requests are not allowed in tests')));

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

// Database setup for integration tests (only initialized if needed)
let prisma: any;
let DbProvider: any;

// Lazy load database utilities only when tests need them
export const getTestDatabaseClient = () => {
    if (!prisma) {
        console.warn('Database client not initialized. Only available for integration tests.');
    }
    return prisma;
};

export const createTestTransaction = async (callback: (tx: any) => Promise<void>) => {
    if (!prisma) {
        throw new Error("Database client not available. This function only works in integration tests.");
    }
    
    return await prisma.$transaction(async (tx: any) => {
        await callback(tx);
        // Transaction will be automatically rolled back after test
        throw new Error("ROLLBACK"); // Force rollback for test isolation
    }).catch((error: any) => {
        if (error.message === "ROLLBACK") {
            // Expected rollback for test isolation
            return;
        }
        throw error;
    });
};

export const createTestAPIClient = (baseUrl?: string) => {
    const url = baseUrl || process.env.VITE_SERVER_URL || 'http://localhost:5329';
    
    return {
        async get(endpoint: string, options?: RequestInit) {
            const response = await fetch(`${url}${endpoint}`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                    ...options?.headers,
                },
                ...options,
            });
            return {
                data: await response.json(),
                status: response.status,
                ok: response.ok,
            };
        },
        
        async post(endpoint: string, data: any, options?: RequestInit) {
            const response = await fetch(`${url}${endpoint}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    ...options?.headers,
                },
                body: JSON.stringify(data),
                ...options,
            });
            return {
                data: await response.json(),
                status: response.status,
                ok: response.ok,
            };
        },
        
        async put(endpoint: string, data: any, options?: RequestInit) {
            const response = await fetch(`${url}${endpoint}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                    ...options?.headers,
                },
                body: JSON.stringify(data),
                ...options,
            });
            return {
                data: await response.json(),
                status: response.status,
                ok: response.ok,
            };
        },
        
        async delete(endpoint: string, options?: RequestInit) {
            const response = await fetch(`${url}${endpoint}`, {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json',
                    ...options?.headers,
                },
                ...options,
            });
            return {
                data: response.status === 204 ? null : await response.json(),
                status: response.status,
                ok: response.ok,
            };
        },
    };
};

// Only initialize database if this is an integration test
const isIntegrationTest = process.env.DB_URL && process.env.REDIS_URL;

if (isIntegrationTest) {
    beforeAll(async () => {
        console.log("=== Database available for integration tests ===");
        
        try {
            // Initialize database client for test verification
            const serverModule = await import("@vrooli/server");
            if (serverModule.prisma) {
                prisma = serverModule.prisma;
                console.log("✓ Prisma client available");
            }
            
            if (serverModule.DbProvider) {
                DbProvider = serverModule.DbProvider;
                await DbProvider.init();
                console.log("✓ DbProvider initialized");
            }
        } catch (error) {
            console.warn("Database initialization skipped:", error);
        }
    }, 300000); // 5 minute timeout
    
    afterAll(async () => {
        try {
            if (DbProvider && DbProvider.reset) {
                await DbProvider.reset();
            }
            
            if (prisma && prisma.$disconnect) {
                await prisma.$disconnect();
            }
        } catch (error) {
            console.error("Database cleanup error:", error);
        }
    });
}