/**
 * Shared test doubles and mock factories.
 *
 * Centralizes mocks that were previously duplicated across test files,
 * providing a single source of truth for test infrastructure.
 */
import { vi } from "vitest";
// ---------------------------------------------------------------------------
// @vrooli/api-base mock factory
// ---------------------------------------------------------------------------
/**
 * Returns the vi.mock factory object for @vrooli/api-base.
 *
 * Usage at the top of a test file (must be hoisted):
 *   vi.mock("@vrooli/api-base", () => apiBaseMock());
 *
 * Uses a deterministic fake URL so fetch assertions are stable.
 * The URL is intentionally non-routable (RFC 2606 .invalid TLD)
 * to avoid hardcoded localhost:PORT audit violations.
 */
export function apiBaseMock() {
    const apiBase = "http://test-api.invalid/api/v1";
    const wsBase = "ws://test-api.invalid/ws";
    return {
        resolveApiBase: () => apiBase,
        buildApiUrl: (path, opts) => `${opts.baseUrl}${path}`,
        resolveWsBase: () => wsBase,
        buildWsUrl: (path, opts) => `${opts.baseUrl}${path}`,
        // Connect-Web transport used by src/api/*.ts domain clients. Tests
        // that exercise those clients mock the domain module directly (so the
        // transport is never invoked); a no-op stub here just stops the
        // import-time `createScenarioConnectTransport` call from crashing.
        createScenarioConnectTransport: () => ({}),
    };
}
// ---------------------------------------------------------------------------
// FakeWebSocket — lightweight WebSocket test double
// ---------------------------------------------------------------------------
/**
 * Minimal fake WebSocket that mirrors the subset of the real API
 * used by useTerminalTransport. The test controls the lifecycle via
 * triggerOpen / triggerMessage / triggerClose.
 */
export class FakeWebSocket {
    constructor() {
        this.readyState = FakeWebSocket.CONNECTING;
        this.onopen = null;
        this.onmessage = null;
        this.onclose = null;
        this.sent = [];
        this.closed = false;
        /** Mirrors WebSocket.bufferedAmount; tests can set this directly to simulate back-pressure. */
        this.bufferedAmount = 0;
        /** When non-null, the next send() call will throw this error (tests set it explicitly). */
        this.sendError = null;
    }
    send(data) {
        if (this.sendError) {
            const err = this.sendError;
            this.sendError = null;
            throw err;
        }
        this.sent.push(data);
    }
    close() {
        this.closed = true;
    }
    triggerOpen() {
        this.readyState = FakeWebSocket.OPEN;
        this.onopen?.(new Event("open"));
    }
    triggerMessage(msg) {
        this.onmessage?.(new MessageEvent("message", { data: JSON.stringify(msg) }));
    }
    triggerClose(code) {
        this.readyState = FakeWebSocket.CLOSED;
        this.onclose?.(new CloseEvent("close", { code }));
    }
}
FakeWebSocket.CONNECTING = 0;
FakeWebSocket.OPEN = 1;
FakeWebSocket.CLOSED = 3;
/**
 * Creates a FakeWebSocket and a SocketFactory that returns it.
 * The factory is a vi.fn() so callers can assert on creation URL.
 */
export function createFakeSocketPair() {
    const fakeWs = new FakeWebSocket();
    const createSocket = vi.fn(() => fakeWs);
    return { fakeWs, createSocket };
}
/**
 * Creates a minimal xterm Terminal mock with controllable I/O.
 * Captures all write() calls and exposes simulateInput() to fire
 * onData callbacks. Includes scroll, selection, and buffer APIs
 * needed by useTerminalTouch.
 */
export function createMockTerminal() {
    const written = [];
    const dataCallbacks = [];
    const mockLine = {
        translateToString: vi.fn().mockReturnValue("hello world test line"),
    };
    return {
        cols: 80,
        rows: 24,
        write: vi.fn((data) => written.push(data)),
        onData: vi.fn((cb) => {
            dataCallbacks.push(cb);
            return { dispose: vi.fn() };
        }),
        written,
        simulateInput(data) {
            for (const cb of dataCallbacks)
                cb(data);
        },
        scrollLines: vi.fn(),
        scrollToBottom: vi.fn(),
        clear: vi.fn(),
        reset: vi.fn(),
        select: vi.fn(),
        selectAll: vi.fn(),
        getSelection: vi.fn().mockReturnValue(""),
        getSelectionPosition: vi.fn().mockReturnValue(undefined),
        clearSelection: vi.fn(),
        onSelectionChange: vi.fn(() => ({ dispose: vi.fn() })),
        focus: vi.fn(),
        buffer: {
            active: {
                viewportY: 0,
                baseY: 0,
                length: 24,
                getLine: vi.fn().mockReturnValue(mockLine),
            },
        },
    };
}
/**
 * Find a write() call containing `needle` in the first argument.
 * Returns the full string or undefined if not found.
 */
export function findWriteCall(mock, needle) {
    const calls = mock.mock.calls;
    const match = calls.find((c) => typeof c[0] === "string" && c[0].includes(needle));
    return match ? match[0] : undefined;
}
/**
 * Creates an array of session entries suitable for component props.
 * Each session gets a bash shell and default policy.
 */
export function makeSessions(...ids) {
    return ids.map((id) => ({
        session: {
            id,
            shell: "/bin/bash",
            created_at: "2026-01-15T14:30:00Z",
            cols: 80,
            rows: 24,
            backend: "standard",
            survives_restart: false,
            policy: { mode: "never" },
            busy: false,
            origin: "ui",
            owner: "",
            display_label: "",
        },
    }));
}
/**
 * Creates a single SessionInfo object with optional overrides.
 */
export function createMockSession(overrides = {}) {
    return {
        id: "test-session-id",
        shell: "/bin/bash",
        created_at: "2026-01-15T14:30:00Z",
        cols: 80,
        rows: 24,
        backend: "standard",
        survives_restart: false,
        policy: { mode: "never" },
        busy: false,
        origin: "ui",
        owner: "",
        display_label: "",
        ...overrides,
    };
}
// ---------------------------------------------------------------------------
// Fetch mock helpers
// ---------------------------------------------------------------------------
/**
 * Installs a successful fetch mock that returns the given JSON body.
 */
export function mockFetchSuccess(body) {
    globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(body),
    });
}
/**
 * Installs a failing fetch mock that returns the given status and JSON body.
 */
export function mockFetchError(status, body) {
    globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status,
        json: body !== undefined
            ? () => Promise.resolve(body)
            : () => Promise.reject(new Error("not json")),
    });
}
