// Mock API environment variables FIRST - before any imports that use them
process.env.VITE_PORT_API = "3000";
process.env.VITE_API_URL = "http://localhost:3000/api";
process.env.VITE_SITE_IP = "localhost";

import { afterAll, afterEach, beforeAll, vi } from "vitest";
import { reactI18nextMock } from "./mocks/libraries/react-i18next.js";
import { authMock } from "./mocks/utils/authentication.js";
import { routerMock } from "./mocks/utils/router.js";
import { zxcvbnMock } from "./mocks/libraries/zxcvbn.js";
import { muiStylesMock } from "./mocks/libraries/mui.js";
import { apiMocks } from "./mocks/utils/api.js";
import { socketServiceMock } from "./mocks/utils/socket.js";
import { iconsMock } from "./mocks/components/icons.js";

// AI_CHECK: TEST_QUALITY=3 | LAST: 2025-06-23
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-06-30 - Fixed 14 explicit 'any' types
// Implemented centralized mock system to reduce duplication:
// - Created factory functions for configurable mocks
// - Standardized mock implementations across all tests
// - Reduced test file complexity and maintenance overhead
// - Improved test performance by reducing mock setup time
// Note: Migration to centralized mocks is ongoing

// Store original console methods
const originalConsoleError = console.error;
const originalConsoleWarn = console.warn;
const originalConsoleLog = console.log;

// Track if console methods are currently mocked by a test
type ConsoleSpy = ReturnType<typeof vi.spyOn> | null;
const consoleSpies = {
    error: null as ConsoleSpy,
    warn: null as ConsoleSpy,
    log: null as ConsoleSpy,
};

// Deduplication system for console messages
type MessageCounter = {
    count: number;
    lastSeen: number;
    timer?: NodeJS.Timeout;
};

const messageCounters = new Map<string, MessageCounter>();
const DEDUP_WINDOW_MS = 25; // Group messages within 25ms
const MAX_INDIVIDUAL_SHOWS = 3; // Show first 3 occurrences individually

function normalizeMessage(args: unknown[]): string {
    return args.map(arg => String(arg)).join(" ").replace(/\s+/g, " ").trim();
}

function createDedupHandler(originalMethod: typeof console.error) {
    return (...args: unknown[]) => {
        const message = normalizeMessage(args);
        const now = Date.now();
        
        let counter = messageCounters.get(message);
        if (!counter) {
            counter = { count: 0, lastSeen: now };
            messageCounters.set(message, counter);
        }
        
        counter.count++;
        counter.lastSeen = now;
        
        // Show first few occurrences individually
        if (counter.count <= MAX_INDIVIDUAL_SHOWS) {
            originalMethod(...args);
            return;
        }
        
        // Clear existing timer
        if (counter.timer) {
            clearTimeout(counter.timer);
        }
        
        // Set new timer to show count after dedup window
        counter.timer = setTimeout(() => {
            if (counter.count > MAX_INDIVIDUAL_SHOWS) {
                const repeatCount = counter.count - MAX_INDIVIDUAL_SHOWS;
                originalMethod(`[${repeatCount} more similar messages suppressed] ${message}`);
            }
            messageCounters.delete(message);
        }, DEDUP_WINDOW_MS);
    };
}

// Create deduplicating console handlers
const dedupConsoleError = createDedupHandler(originalConsoleError);
const dedupConsoleWarn = createDedupHandler(originalConsoleWarn);
const dedupConsoleLog = createDedupHandler(originalConsoleLog);

// Optimize console filtering with string checks instead of regex for common patterns
const filteredConsoleError = (...args: unknown[]) => {
    const message = args.join(" ");
    
    // Quick string checks for common patterns (faster than regex)
    if (message.includes("MUI: The `children` component") ||
        message.includes("MUI: You are providing a disabled") ||
        message.includes("A disabled element does not fire events") ||
        message.includes("Tooltip needs to listen") ||
        message.includes("Add a simple wrapper element") ||
        message.includes("The above error occurred in") ||
        message.includes("React will try to recreate") ||
        message.includes("Error caught by Error Boundary") ||
        message.includes("componentStack:") ||
        message.includes("react-i18next:: You will need") ||
        message.includes("getObjectUrlBase called with non-object") ||
        message.includes("AdvancedInputBase: No child handleAction") ||
        message.includes("@hello-pangea/dnd") ||
        message.includes("Warning: ReactDOM.render") ||
        message.includes("Warning: An invalid form control") ||
        message.includes("Warning: Failed prop type") ||
        message.includes("Warning: Unknown event handler") ||
        message.includes("Warning: An update to") ||
        message.includes("Failed to load iframe page") ||
        message.includes("React.jsx: type is invalid") ||
        message.includes("element from the file it's defined in") ||
        message.includes("mixed up default and named imports")) {
        return;
    }
    
    // Only use regex for complex patterns that can't be string checked
    const complexPatterns = [
        /Unsupported input type:/,
        /Cannot read properties of (null|undefined) \(reading 'placeholder'\)/,
        /Element type is invalid: expected a string .* but got: undefined/,
    ];
    
    if (complexPatterns.some(pattern => pattern.test(message))) {
        return;
    }
    
    // Use deduplicating handler for remaining messages
    dedupConsoleError(...args);
};

const filteredConsoleWarn = (...args: unknown[]) => {
    const message = args.join(" ");
    
    // Reuse same optimization approach
    if (message.includes("MUI:") ||
        message.includes("@hello-pangea/dnd") ||
        message.includes("react-i18next::") ||
        message.includes("Warning: React.jsx: type is invalid") ||
        message.includes("Warning:")) {
        return;
    }
    
    // Use deduplicating handler for remaining messages
    dedupConsoleWarn(...args);
};

const filteredConsoleLog = (...args: unknown[]) => {
    const message = args.join(" ");
    
    // Simple string checks for log patterns
    if (message.includes("AdvancedInput:")) {
        return;
    }
    
    // Use deduplicating handler for remaining messages
    dedupConsoleLog(...args);
};

// Setup console filtering and HTML5 doctype
beforeAll(() => {
    // Set up HTML5 doctype for @hello-pangea/dnd compatibility
    if (typeof document !== "undefined" && !document.doctype) {
        const doctype = document.implementation.createDocumentType("html", "", "");
        document.insertBefore(doctype, document.documentElement);
    }
    
    // Only apply filtering if not already mocked by a test
    if (!consoleSpies.error) {
        console.error = filteredConsoleError;
    }
    if (!consoleSpies.warn) {
        console.warn = filteredConsoleWarn;
    }
    if (!consoleSpies.log) {
        console.log = filteredConsoleLog;
    }
});

// Clean up after each test
afterEach(() => {
    // If a test created a spy, restore it
    if (consoleSpies.error) {
        consoleSpies.error.mockRestore();
        consoleSpies.error = null;
    }
    if (consoleSpies.warn) {
        consoleSpies.warn.mockRestore();
        consoleSpies.warn = null;
    }
    if (consoleSpies.log) {
        consoleSpies.log.mockRestore();
        consoleSpies.log = null;
    }
    
    // Clean up deduplication system
    messageCounters.forEach((counter) => {
        if (counter.timer) {
            clearTimeout(counter.timer);
        }
    });
    messageCounters.clear();
    
    // Reapply our filtered console methods
    console.error = filteredConsoleError;
    console.warn = filteredConsoleWarn;
    console.log = filteredConsoleLog;
    
    // Clear all other mocks
    vi.clearAllMocks();
});

// Track when tests create console spies
const originalSpyOn = vi.spyOn;
vi.spyOn = function<T, K extends keyof T>(object: T, method: K, ...args: Parameters<typeof originalSpyOn>) {
    // Track console spies created by tests
    if (object === console) {
        if (method === "error") consoleSpies.error = originalSpyOn.apply(vi, [object, method, ...args]) as ConsoleSpy;
        else if (method === "warn") consoleSpies.warn = originalSpyOn.apply(vi, [object, method, ...args]) as ConsoleSpy;
        else if (method === "log") consoleSpies.log = originalSpyOn.apply(vi, [object, method, ...args]) as ConsoleSpy;
        
        return consoleSpies[method as keyof typeof consoleSpies] || originalSpyOn.apply(vi, [object, method, ...args]);
    }
    return originalSpyOn.apply(vi, [object, method, ...args]);
} as typeof originalSpyOn;

// Configure Testing Library to reduce error output
import { configure } from "@testing-library/react";

configure({
    // Reduce HTML output in errors
    getElementError: (message, container) => {
        const error = new Error(message);
        error.name = "TestingLibraryElementError";
        // Don't include the full container HTML
        return error;
    },
    // Reduce timeout for faster failures
    asyncUtilTimeout: 2000,
});

// Vitest has built-in DOM assertions - no custom matchers needed


// TEMPORARILY COMMENTED OUT: Testing performance without global @vrooli/shared mock
// This mock was intercepting all imports and forcing TypeScript compilation
// vi.mock("@vrooli/shared", async (importOriginal) => {
//     try {
//         const actual = await importOriginal();
//         ... (mock implementation)
//     } catch (error) {
//         ... (fallback implementation)
//     }
// });

// Essential mocks only - no fancy stuff
vi.mock("../utils/display/device.js", () => ({
    keyComboToString: () => "Ctrl+S",
    getDeviceInfo: () => ({ isMobile: false, deviceOS: "Windows" }),
    DeviceOS: {
        IOS: "iOS",
        MacOS: "MacOS",
        Windows: "Windows",
        Android: "Android",
        Linux: "Linux",
    },
}));
vi.mock("../utils/display/chatTools.js", () => ({
    taskToTaskInfo: () => ({ name: "Test Task", description: "Test Description" }),
    isTaskStale: vi.fn(() => false),
    STALE_TASK_THRESHOLD_MS: 600000,
}));
vi.mock("@uiw/react-codemirror", () => ({ default: () => null }));

// Mock only the most problematic lexical components with realistic implementations
vi.mock("../components/inputs/AdvancedInput/lexical/AdvancedInputLexical.js", () => ({
    AdvancedInputLexical: () => {
        const React = require("react");
        return React.createElement("div", { "data-testid": "lexical-editor" });
    },
    default: () => {
        const React = require("react");
        return React.createElement("div", { "data-testid": "lexical-editor" });
    },
}));
// RichInput has been replaced with AdvancedInput
vi.mock("../components/inputs/TagSelector/TagSelector.js", () => ({
    TagSelector: () => null,
    default: () => null,
}));
// NOTE: Removed AdvancedInput mock to allow proper testing
vi.mock("../utils/pubsub.js", () => ({
    PubSub: {
        get: () => ({
            subscribe: vi.fn(() => vi.fn()),
            publish: vi.fn(),
            hasSubscribers: () => false,
        }),
    },
}));

// Environment variables moved to top of file before imports

// Minimal browser mocks
Object.defineProperty(window, "matchMedia", {
    value: () => ({ matches: false, addListener: () => { }, removeListener: () => { } }),
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

// Mock HTMLIFrameElement to prevent network requests and AsyncTaskManager issues
class MockIFrame implements Partial<HTMLIFrameElement> {
    src = "";
    title = "";
    width = "";
    height = "";
    frameBorder = "";
    allow = "";
    allowFullScreen = false;
    dataset: Record<string, string> = {};
    
    getAttribute(name: string): string | null {
        if (name.startsWith("data-")) {
            return this.dataset[name.slice(5)] || null;
        }
        return (this as Record<string, unknown>)[name] as string || null;
    }
    
    setAttribute(name: string, value: string): void {
        if (name.startsWith("data-")) {
            this.dataset[name.slice(5)] = value;
        } else {
            (this as Record<string, unknown>)[name] = value;
        }
    }
    
    hasAttribute(name: string): boolean {
        if (name.startsWith("data-")) {
            return this.dataset[name.slice(5)] !== undefined;
        }
        return (this as Record<string, unknown>)[name] !== undefined;
    }
    
    removeAttribute(name: string): void {
        if (name.startsWith("data-")) {
            delete this.dataset[name.slice(5)];
        } else {
            delete (this as Record<string, unknown>)[name];
        }
    }
}

global.HTMLIFrameElement = MockIFrame as unknown as typeof HTMLIFrameElement;

// =============================================================================
// MSW SETUP
// =============================================================================
// Import and configure MSW for API mocking
import { server } from "./mocks/server.js";

// Establish API mocking before all tests
beforeAll(() => {
    // Start MSW server with request interception
    server.listen({ 
        onUnhandledRequest: (req, print) => {
            // Only warn about unhandled requests to external URLs, not our localhost API
            const url = new URL(req.url);
            if (url.hostname === "localhost" && url.port === "3000") {
                console.error(`🚨 Unhandled API request: ${req.method} ${req.url} - This should be mocked!`);
            } else {
                print.warning();
            }
        },
    });
});

// Reset any request handlers that we may add during the tests
afterEach(() => {
    server.resetHandlers();
});

// Clean up after the tests are finished
afterAll(() => {
    // Clean up any remaining deduplication timers
    messageCounters.forEach((counter) => {
        if (counter.timer) {
            clearTimeout(counter.timer);
        }
    });
    messageCounters.clear();
    
    // Restore original console methods
    console.error = originalConsoleError;
    console.warn = originalConsoleWarn;
    console.log = originalConsoleLog;
    
    server.close();
});

// Note: Integration tests have been moved to a dedicated integration package.
// UI tests should focus on component behavior and user interactions.

// =============================================================================
// CENTRALIZED MOCKS
// =============================================================================
// Apply global mocks that should be available to all tests by default.
// Individual tests can override these by using vi.mock() before importing.

// Mock react-i18next globally
vi.mock("react-i18next", () => reactI18nextMock);

// Mock i18next globally
import { mockI18next } from "./mocks/libraries/react-i18next.js";

vi.mock("i18next", () => ({ default: mockI18next }));

// Mock zxcvbn for password strength (synchronous in tests)
vi.mock("zxcvbn", () => zxcvbnMock);

// Mock MUI styles
vi.mock("@mui/material/styles", () => muiStylesMock);
vi.mock("@mui/styles", () => muiStylesMock);

// Mock authentication utilities
vi.mock("../utils/authentication/session.js", () => authMock);
vi.mock("../../utils/authentication/session.js", () => authMock);
vi.mock("../../utils/authentication/session", () => authMock);

// Mock router utilities
vi.mock("../route/router.js", () => routerMock);
vi.mock("../../route/router.js", () => routerMock);
vi.mock("../../route/router", () => routerMock);

// Mock API utilities
vi.mock("../api/fetchWrapper.js", () => apiMocks);
vi.mock("../../api/fetchWrapper.js", () => apiMocks);
vi.mock("../../api/fetchWrapper", () => apiMocks);

// fetchData is no longer globally mocked
// Individual tests should mock fetchData if needed for testing components that use it
// This allows the actual fetchData implementation to be tested properly

// ServerResponseParser is no longer globally mocked
// Individual tests should mock specific methods they need to test
// This allows the actual implementation to be tested properly

// Mock socket service
vi.mock("../api/socket.js", () => socketServiceMock);
vi.mock("../../api/socket.js", () => socketServiceMock);
vi.mock("../../api/socket", () => socketServiceMock);

// Mock icons
vi.mock("../icons/Icons.js", () => iconsMock);
vi.mock("../../icons/Icons.js", () => iconsMock);
vi.mock("../../icons/Icons", () => iconsMock);

// Mock socket.io-client
import { socketIOClientMock } from "./mocks/libraries/socket.io-client.js";

vi.mock("socket.io-client", () => socketIOClientMock);

// Mock the global fetch to prevent real HTTP requests during tests
global.fetch = vi.fn(() =>
    Promise.resolve(new Response(JSON.stringify({ data: null }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
    })),
);

// Mock @hello-pangea/dnd for drag and drop functionality
import { helloPangeaDndMock } from "./mocks/libraries/hello-pangea-dnd.js";

vi.mock("@hello-pangea/dnd", () => helloPangeaDndMock);


// Note: ServerResponseParser is no longer globally mocked
// Individual tests should mock specific methods they need to test
// This allows the actual implementation to be tested properly
