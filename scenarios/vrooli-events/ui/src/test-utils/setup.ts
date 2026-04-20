import "@testing-library/jest-dom";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";

// Automatic cleanup after each test
afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

// Mock ResizeObserver (not available in jsdom)
globalThis.ResizeObserver = vi.fn().mockImplementation(() => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
}));

// Mock EventSource (not available in jsdom).
// EventSource has both instance state (readyState) and static-class enum constants
// (CONNECTING/OPEN/CLOSED). vi.fn() alone produces a Mock<Procedure> without those
// enum props, so we attach them via Object.assign to satisfy the constructor shape.
const MockEventSource = Object.assign(
    vi.fn().mockImplementation(() => ({
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        close: vi.fn(),
        readyState: 0,
        CONNECTING: 0,
        OPEN: 1,
        CLOSED: 2,
    })),
    { CONNECTING: 0 as const, OPEN: 1 as const, CLOSED: 2 as const },
);
Object.defineProperty(globalThis, "EventSource", { value: MockEventSource, writable: true, configurable: true });
